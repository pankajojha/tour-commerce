package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var refreshReq RefreshRequest
	if err := json.Unmarshal([]byte(request.Body), &refreshReq); err != nil {
		return errorResponse(400, "Invalid request body", err)
	}

	if refreshReq.RefreshToken == "" {
		return errorResponse(400, "Refresh token is required", nil)
	}

	authInput := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeRefreshTokenAuth,
		ClientId: stringPtr(clientID),
		AuthParameters: map[string]string{
			"REFRESH_TOKEN": refreshReq.RefreshToken,
		},
	}

	result, err := cognitoClient.InitiateAuth(ctx, authInput)
	if err != nil {
		return handleCognitoError(err)
	}

	if result.AuthenticationResult == nil {
		return errorResponse(401, "Token refresh failed", nil)
	}

	response := RefreshResponse{
		AccessToken: *result.AuthenticationResult.AccessToken,
		IdToken:     *result.AuthenticationResult.IdToken,
		ExpiresIn:   result.AuthenticationResult.ExpiresIn,
		TokenType:   *result.AuthenticationResult.TokenType,
	}

	return successResponse(200, response)
}
