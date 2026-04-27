package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// ─────────────────────────────────────────────────────────────────────────────
// Configuration
// ─────────────────────────────────────────────────────────────────────────────

var (
	// AI_GATEWAY_URL is the base URL of the tourplannerai-api (AI runtime API Gateway).
	// Example: https://<api-id>.execute-api.<region>.amazonaws.com/production
	aiGatewayURL = strings.TrimRight(os.Getenv("AI_GATEWAY_URL"), "/")

	httpClient = &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        50,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
)

// ─────────────────────────────────────────────────────────────────────────────
// Route allowlist
//
// Only METHOD + path combinations listed here will be forwarded to the AI
// gateway. Every other request returns 404. This provides an explicit security
// boundary — the proxy never blindly forwards arbitrary paths.
//
// Grouped by product domain:
//   Health          GET  /health
//   Excursions      POST /excursions/search
//                   GET  /excursions/{id}
//                   POST /excursions/availability
//                   POST /excursions/quote
//   Destination     GET  /destination/context
//   Cruise          GET  /cruise/schedule
//   Planning        POST /planning/recommend
//   Intake          POST /trips/intake
//   Sharing         POST /trips/share
//   Bookings        GET  /trips/bookings/{id}
//   Synthesis       POST /synthesis/shared-trip
// ─────────────────────────────────────────────────────────────────────────────

func isAllowed(method, path string) bool {
	m := strings.ToUpper(method)
	switch {
	// Health
	case m == "GET" && path == "/health":
		return true

	// Excursions — search, detail by id, availability, quote
	case m == "POST" && path == "/excursions/search":
		return true
	case m == "GET" && isExcursionDetail(path):
		return true
	case m == "POST" && path == "/excursions/availability":
		return true
	case m == "POST" && path == "/excursions/quote":
		return true

	// Destination context
	case m == "GET" && path == "/destination/context":
		return true

	// Cruise schedule
	case m == "GET" && path == "/cruise/schedule":
		return true

	// AI planning recommendation
	case m == "POST" && path == "/planning/recommend":
		return true

	// Collaborative trip intake + share (AI-processed)
	case m == "POST" && path == "/trips/intake":
		return true
	case m == "POST" && path == "/trips/share":
		return true

	// Booking status check (AI-tracked)
	case m == "GET" && isBookingDetail(path):
		return true

	// AI synthesis of a shared collaborative trip
	case m == "POST" && path == "/synthesis/shared-trip":
		return true

	// CORS preflight — forwarded so the AI gateway can set its own CORS headers
	case m == "OPTIONS":
		return true

	default:
		return false
	}
}

// isExcursionDetail returns true for /excursions/{id} where id is non-empty.
func isExcursionDetail(path string) bool {
	// Must be /excursions/<something> but NOT /excursions/search, /excursions/availability,
	// /excursions/quote (those are POST, so this guard is belt-and-suspenders).
	if !strings.HasPrefix(path, "/excursions/") {
		return false
	}
	suffix := strings.TrimPrefix(path, "/excursions/")
	return suffix != "" && !strings.Contains(suffix, "/")
}

// isBookingDetail returns true for /trips/bookings/{id} where id is non-empty.
func isBookingDetail(path string) bool {
	if !strings.HasPrefix(path, "/trips/bookings/") {
		return false
	}
	id := strings.TrimPrefix(path, "/trips/bookings/")
	return id != "" && !strings.Contains(id, "/")
}

// ─────────────────────────────────────────────────────────────────────────────
// Lambda handler
// ─────────────────────────────────────────────────────────────────────────────

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if aiGatewayURL == "" {
		log.Print("AI_GATEWAY_URL is not configured")
		return jsonError(http.StatusServiceUnavailable, "AI gateway not configured"), nil
	}

	if !isAllowed(req.HTTPMethod, req.Path) {
		return jsonError(http.StatusNotFound, "route not found"), nil
	}

	return proxyRequest(ctx, req)
}

// ─────────────────────────────────────────────────────────────────────────────
// Proxy logic
// ─────────────────────────────────────────────────────────────────────────────

func proxyRequest(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	targetURL := buildTargetURL(req)

	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	upstreamReq, err := http.NewRequestWithContext(ctx, req.HTTPMethod, targetURL, bodyReader)
	if err != nil {
		log.Printf("failed to build upstream request: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to build upstream request"), nil
	}

	forwardHeaders(req, upstreamReq)

	resp, err := httpClient.Do(upstreamReq)
	if err != nil {
		log.Printf("upstream request to %s failed: %v", targetURL, err)
		return jsonError(http.StatusBadGateway, "upstream AI service unavailable"), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read upstream response: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to read upstream response"), nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: resp.StatusCode,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(body),
	}, nil
}

// buildTargetURL constructs the full upstream URL including query parameters.
func buildTargetURL(req events.APIGatewayProxyRequest) string {
	target := aiGatewayURL + req.Path

	// Prefer multi-value query parameters to preserve repeated keys (e.g. ?tag=a&tag=b).
	if len(req.MultiValueQueryStringParameters) > 0 {
		vals := url.Values{}
		for k, vs := range req.MultiValueQueryStringParameters {
			for _, v := range vs {
				vals.Add(k, v)
			}
		}
		return target + "?" + vals.Encode()
	}

	if len(req.QueryStringParameters) > 0 {
		vals := url.Values{}
		for k, v := range req.QueryStringParameters {
			vals.Set(k, v)
		}
		return target + "?" + vals.Encode()
	}

	return target
}

// forwardHeaders copies relevant inbound headers to the upstream request and
// injects caller identity from the JWT authorizer context.
func forwardHeaders(req events.APIGatewayProxyRequest, upstream *http.Request) {
	// Normalise header keys to lowercase for consistent lookup.
	headers := make(map[string]string, len(req.Headers))
	for k, v := range req.Headers {
		headers[strings.ToLower(k)] = v
	}

	// Authorization — bearer token for the AI gateway's Cognito authorizer.
	if auth := headers["authorization"]; auth != "" {
		upstream.Header.Set("Authorization", auth)
	}

	// Content-Type — default to JSON for requests with a body.
	if ct := headers["content-type"]; ct != "" {
		upstream.Header.Set("Content-Type", ct)
	} else if req.Body != "" {
		upstream.Header.Set("Content-Type", "application/json")
	}

	// Correlation ID — pass through for end-to-end tracing.
	if corr := headers["x-correlation-id"]; corr != "" {
		upstream.Header.Set("X-Correlation-ID", corr)
	}

	// User ID — extracted from the JWT authorizer context and forwarded as
	// X-User-ID so the AI runtime knows the calling user without re-parsing the token.
	if sub, ok := req.RequestContext.Authorizer["sub"].(string); ok && sub != "" {
		upstream.Header.Set("X-User-ID", sub)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func jsonError(code int, msg string) events.APIGatewayProxyResponse {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(b),
	}
}

func main() {
	lambda.Start(handler)
}
