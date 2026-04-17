package main

import (
	"context"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

func handler(_ context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod == "OPTIONS" {
		return corsOnlyResponse(200), nil
	}

	if request.HTTPMethod != "GET" {
		return errorResponse(405, "Method not allowed", nil)
	}

	stateRaw := strings.TrimSpace(request.QueryStringParameters["state"])
	decodedState, _ := decodeState(stateRaw)

	provider := decodedState.Provider
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(request.QueryStringParameters["provider"]))
	}

	oauthErr := strings.TrimSpace(request.QueryStringParameters["error"])
	oauthErrDesc := strings.TrimSpace(request.QueryStringParameters["error_description"])
	code := strings.TrimSpace(request.QueryStringParameters["code"])

	if oauthErr != "" {
		location, err := withQuery(frontendCancelURL, map[string]string{
			"error":             oauthErr,
			"error_description": oauthErrDesc,
			"provider":          provider,
			"state":             stateRaw,
		})
		if err != nil {
			return errorResponse(500, "Failed to build cancel redirect URL", err)
		}
		return redirectResponse(location), nil
	}

	if code == "" {
		location, err := withQuery(frontendCancelURL, map[string]string{
			"error":             "invalid_request",
			"error_description": "missing code",
			"provider":          provider,
			"state":             stateRaw,
		})
		if err != nil {
			return errorResponse(500, "Failed to build missing-code redirect URL", err)
		}
		return redirectResponse(location), nil
	}

	location, err := withQuery(frontendSuccessURL, map[string]string{
		"code":     code,
		"provider": provider,
		"state":    stateRaw,
	})
	if err != nil {
		return errorResponse(500, "Failed to build success redirect URL", err)
	}

	return redirectResponse(location), nil
}
