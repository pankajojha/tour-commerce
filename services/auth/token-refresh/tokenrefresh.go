package tokenrefresh

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

type Request struct {
	RefreshToken string `json:"refreshToken"`
}

type Response struct {
	AccessToken string `json:"accessToken"`
	IdToken     string `json:"idToken"`
	ExpiresIn   int32  `json:"expiresIn"`
	TokenType   string `json:"tokenType"`
}

func Handle(ctx context.Context, req events.APIGatewayProxyRequest, cognitoClient *cognitoidentityprovider.Client, clientID string) (events.APIGatewayProxyResponse, error) {
	var body Request
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return errResp(400, "Invalid request body")
	}
	if body.RefreshToken == "" {
		return errResp(400, "Refresh token is required")
	}

	result, err := cognitoClient.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow:       types.AuthFlowTypeRefreshTokenAuth,
		ClientId:       &clientID,
		AuthParameters: map[string]string{"REFRESH_TOKEN": body.RefreshToken},
	})
	if err != nil {
		return mapCognitoErr(err)
	}
	if result.AuthenticationResult == nil {
		return errResp(401, "Token refresh failed")
	}

	return okResp(200, Response{
		AccessToken: *result.AuthenticationResult.AccessToken,
		IdToken:     *result.AuthenticationResult.IdToken,
		ExpiresIn:   result.AuthenticationResult.ExpiresIn,
		TokenType:   *result.AuthenticationResult.TokenType,
	})
}

func mapCognitoErr(err error) (events.APIGatewayProxyResponse, error) {
	msg := err.Error()
	log.Printf("Cognito error: %v", msg)
	switch {
	case strings.Contains(msg, "NotAuthorizedException"):
		return errResp(401, "Invalid or expired refresh token")
	case strings.Contains(msg, "TooManyRequestsException"):
		return errResp(429, "Too many requests. Please try again later.")
	default:
		return errResp(500, "Token refresh failed")
	}
}

func okResp(status int, body any) (events.APIGatewayProxyResponse, error) {
	j, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{StatusCode: status, Headers: headers(), Body: string(j)}, nil
}

func errResp(status int, message string) (events.APIGatewayProxyResponse, error) {
	j, _ := json.Marshal(map[string]string{"error": "RefreshError", "message": message})
	return events.APIGatewayProxyResponse{StatusCode: status, Headers: headers(), Body: string(j)}, nil
}

func headers() map[string]string {
	return map[string]string{"Content-Type": "application/json", "Access-Control-Allow-Origin": "*"}
}
