package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshResponse struct {
	AccessToken string `json:"accessToken"`
	IdToken     string `json:"idToken"`
	ExpiresIn   int32  `json:"expiresIn"`
	TokenType   string `json:"tokenType"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

var (
	cognitoClient *cognitoidentityprovider.Client
	clientID      string
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	cognitoClient = cognitoidentityprovider.NewFromConfig(cfg)
	clientID = os.Getenv("CLIENT_ID")

	if clientID == "" {
		log.Fatal("CLIENT_ID environment variable must be set")
	}
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse request body
	var refreshReq RefreshRequest
	if err := json.Unmarshal([]byte(request.Body), &refreshReq); err != nil {
		return errorResponse(400, "Invalid request body", err)
	}

	// Validate input
	if refreshReq.RefreshToken == "" {
		return errorResponse(400, "Refresh token is required", nil)
	}

	// Refresh tokens with Cognito
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

	// Check if refresh was successful
	if result.AuthenticationResult == nil {
		return errorResponse(401, "Token refresh failed", nil)
	}

	// Prepare response
	response := RefreshResponse{
		AccessToken: *result.AuthenticationResult.AccessToken,
		IdToken:     *result.AuthenticationResult.IdToken,
		ExpiresIn:   result.AuthenticationResult.ExpiresIn,
		TokenType:   *result.AuthenticationResult.TokenType,
	}

	return successResponse(200, response)
}

func handleCognitoError(err error) (events.APIGatewayProxyResponse, error) {
	errMsg := err.Error()
	log.Printf("Cognito error: %v", errMsg)

	// Map Cognito errors to user-friendly messages
	switch {
	case contains(errMsg, "NotAuthorizedException"):
		return errorResponse(401, "Invalid or expired refresh token", err)
	case contains(errMsg, "TooManyRequestsException"):
		return errorResponse(429, "Too many requests. Please try again later.", err)
	default:
		return errorResponse(500, "Token refresh failed", err)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func successResponse(statusCode int, body interface{}) (events.APIGatewayProxyResponse, error) {
	jsonBody, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(jsonBody),
	}, nil
}

func errorResponse(statusCode int, message string, err error) (events.APIGatewayProxyResponse, error) {
	errResp := ErrorResponse{
		Error:   "RefreshError",
		Message: message,
	}

	if err != nil {
		log.Printf("Error: %v", err)
	}

	jsonBody, _ := json.Marshal(errResp)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(jsonBody),
	}, nil
}

func stringPtr(s string) *string {
	return &s
}

func main() {
	lambda.Start(handler)
}
