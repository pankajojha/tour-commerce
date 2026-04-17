package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"golang.org/x/crypto/bcrypt"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var verifyReq VerifyRequest
	if err := json.Unmarshal([]byte(request.Body), &verifyReq); err != nil {
		return errorResponse(400, "Invalid request body", err)
	}

	if verifyReq.TokenID == "" || verifyReq.Token == "" {
		return errorResponse(400, "TokenID and token are required", nil)
	}

	userEmail, tokenHash, expiresAt, attempts, err := getTokenRecord(ctx, verifyReq.TokenID)
	if err == sql.ErrNoRows {
		return errorResponse(404, "Invalid or expired token", nil)
	}
	if err != nil {
		return errorResponse(500, "Failed to retrieve token", err)
	}

	if time.Now().After(expiresAt) {
		return errorResponse(401, "Token has expired", nil)
	}

	if attempts >= 3 {
		return errorResponse(429, "Too many verification attempts", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(tokenHash), []byte(verifyReq.Token)); err != nil {
		incrementAttempts(ctx, verifyReq.TokenID)
		return errorResponse(401, "Invalid token", nil)
	}

	if err := deleteToken(ctx, verifyReq.TokenID); err != nil {
		log.Printf("Failed to delete token: %v", err)
	}

	authInput := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeCustomAuth,
		ClientId: stringPtr(clientID),
		AuthParameters: map[string]string{
			"USERNAME": userEmail,
		},
	}

	result, err := cognitoClient.InitiateAuth(ctx, authInput)
	if err != nil {
		log.Printf("Cognito auth error: %v", err)
		return errorResponse(500, "Authentication failed", err)
	}

	if result.AuthenticationResult == nil {
		return errorResponse(401, "Authentication failed", nil)
	}

	response := VerifyResponse{
		AccessToken:  *result.AuthenticationResult.AccessToken,
		IdToken:      *result.AuthenticationResult.IdToken,
		RefreshToken: *result.AuthenticationResult.RefreshToken,
		ExpiresIn:    result.AuthenticationResult.ExpiresIn,
		TokenType:    *result.AuthenticationResult.TokenType,
	}

	return successResponse(200, response)
}
