package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var loginReq LoginRequest
	if err := json.Unmarshal([]byte(request.Body), &loginReq); err != nil {
		return errorResponse(400, "Invalid request body", err)
	}

	if loginReq.Email == "" || loginReq.Password == "" {
		return errorResponse(400, "Email and password are required", nil)
	}

	authInput := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeUserPasswordAuth,
		ClientId: stringPtr(clientID),
		AuthParameters: map[string]string{
			"USERNAME": loginReq.Email,
			"PASSWORD": loginReq.Password,
		},
	}

	result, err := cognitoClient.InitiateAuth(ctx, authInput)
	if err != nil {
		return handleCognitoError(err)
	}

	if result.AuthenticationResult == nil {
		return errorResponse(401, "Authentication failed", nil)
	}

	response := LoginResponse{
		AccessToken:  *result.AuthenticationResult.AccessToken,
		IdToken:      *result.AuthenticationResult.IdToken,
		RefreshToken: *result.AuthenticationResult.RefreshToken,
		ExpiresIn:    result.AuthenticationResult.ExpiresIn,
		TokenType:    *result.AuthenticationResult.TokenType,
	}

	return successResponse(200, response)
}
