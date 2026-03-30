package passwordlessverify

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"golang.org/x/crypto/bcrypt"
)

type Request struct {
	TokenID string `json:"tokenId"`
	Token   string `json:"token"`
}

type Response struct {
	AccessToken  string `json:"accessToken"`
	IdToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int32  `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
}

func Handle(ctx context.Context, req events.APIGatewayProxyRequest, db *sql.DB, cognitoClient *cognitoidentityprovider.Client, clientID string) (events.APIGatewayProxyResponse, error) {
	if db == nil {
		return errResp(503, "Passwordless auth not configured")
	}

	var body Request
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return errResp(400, "Invalid request body")
	}
	if body.TokenID == "" || body.Token == "" {
		return errResp(400, "TokenID and token are required")
	}

	var userEmail, tokenHash string
	var expiresAt time.Time
	var attempts int

	err := db.QueryRowContext(ctx,
		`SELECT user_email, token_hash, expires_at, attempts FROM passwordless_tokens WHERE token_id = $1`,
		body.TokenID).Scan(&userEmail, &tokenHash, &expiresAt, &attempts)
	if err == sql.ErrNoRows {
		return errResp(404, "Invalid or expired token")
	}
	if err != nil {
		return errResp(500, "Failed to retrieve token")
	}
	if time.Now().After(expiresAt) {
		return errResp(401, "Token has expired")
	}
	if attempts >= 3 {
		return errResp(429, "Too many verification attempts")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(tokenHash), []byte(body.Token)); err != nil {
		db.ExecContext(ctx, `UPDATE passwordless_tokens SET attempts = attempts + 1 WHERE token_id = $1`, body.TokenID)
		return errResp(401, "Invalid token")
	}

	db.ExecContext(ctx, `DELETE FROM passwordless_tokens WHERE token_id = $1`, body.TokenID)

	result, err := cognitoClient.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow:       types.AuthFlowTypeCustomAuth,
		ClientId:       &clientID,
		AuthParameters: map[string]string{"USERNAME": userEmail},
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
	case strings.Contains(msg, "NotAuthorizedException"):
		return errResp(401, "Authentication failed")
	default:
		return errResp(500, "Authentication failed")
	}
}

func okResp(status int, body any) (events.APIGatewayProxyResponse, error) {
	j, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{StatusCode: status, Headers: headers(), Body: string(j)}, nil
}

func errResp(status int, message string) (events.APIGatewayProxyResponse, error) {
	j, _ := json.Marshal(map[string]string{"error": "VerificationError", "message": message})
	return events.APIGatewayProxyResponse{StatusCode: status, Headers: headers(), Body: string(j)}, nil
}

func headers() map[string]string {
	return map[string]string{"Content-Type": "application/json", "Access-Control-Allow-Origin": "*"}
}
