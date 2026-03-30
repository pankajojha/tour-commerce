package login

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
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Response struct {
	AccessToken  string `json:"accessToken"`
	IdToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int32  `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
}

func Handle(ctx context.Context, req events.APIGatewayProxyRequest, cognitoClient *cognitoidentityprovider.Client, clientID string) (events.APIGatewayProxyResponse, error) {
	var body Request
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return errResp(400, "Invalid request body")
	}
	if body.Email == "" || body.Password == "" {
		return errResp(400, "Email and password are required")
	}

	result, err := cognitoClient.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeUserPasswordAuth,
		ClientId: &clientID,
		AuthParameters: map[string]string{
			"USERNAME": body.Email,
			"PASSWORD": body.Password,
		},
	})
	if err != nil {
		return mapCognitoErr(err)
	}
	if result.AuthenticationResult == nil {
		return errResp(401, "Authentication failed")
	}

	return okResp(200, Response{
		AccessToken:  *result.AuthenticationResult.AccessToken,
		IdToken:      *result.AuthenticationResult.IdToken,
		RefreshToken: *result.AuthenticationResult.RefreshToken,
		ExpiresIn:    result.AuthenticationResult.ExpiresIn,
		TokenType:    *result.AuthenticationResult.TokenType,
	})
}

func mapCognitoErr(err error) (events.APIGatewayProxyResponse, error) {
	msg := err.Error()
	log.Printf("Cognito error: %v", msg)
	switch {
	case strings.Contains(msg, "NotAuthorizedException"), strings.Contains(msg, "UserNotFoundException"):
		return errResp(401, "Incorrect email or password")
	case strings.Contains(msg, "UserNotConfirmedException"):
		return errResp(403, "User email not verified")
	case strings.Contains(msg, "TooManyRequestsException"):
		return errResp(429, "Too many login attempts. Please try again later.")
	case strings.Contains(msg, "PasswordResetRequiredException"):
		return errResp(403, "Password reset required")
	default:
		return errResp(500, "Login failed")
	}
}

func okResp(status int, body any) (events.APIGatewayProxyResponse, error) {
	j, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{StatusCode: status, Headers: headers(), Body: string(j)}, nil
}

func errResp(status int, message string) (events.APIGatewayProxyResponse, error) {
	j, _ := json.Marshal(map[string]string{"error": "LoginError", "message": message})
	return events.APIGatewayProxyResponse{StatusCode: status, Headers: headers(), Body: string(j)}, nil
}

func headers() map[string]string {
	return map[string]string{"Content-Type": "application/json", "Access-Control-Allow-Origin": "*"}
}
