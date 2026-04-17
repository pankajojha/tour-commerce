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

	providerKey := strings.ToLower(strings.TrimSpace(request.PathParameters["provider"]))
	if providerKey == "" {
		providerKey = strings.ToLower(strings.TrimSpace(request.QueryStringParameters["provider"]))
	}

	providerName, ok := mapProvider(providerKey)
	if !ok {
		return errorResponse(400, "Unsupported OAuth provider", nil)
	}

	nonce, err := randomNonce(24)
	if err != nil {
		return errorResponse(500, "Failed to prepare OAuth state", err)
	}

	encodedState, err := encodeState(oauthState{Nonce: nonce, Provider: providerKey})
	if err != nil {
		return errorResponse(500, "Failed to prepare OAuth state", err)
	}

	location, err := buildAuthorizeURL(providerName, encodedState)
	if err != nil {
		return errorResponse(500, "Failed to construct authorize URL", err)
	}

	return redirectResponse(location), nil
}
