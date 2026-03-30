package passwordlessrequest

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Request struct {
	Email string `json:"email"`
	Type  string `json:"type"` // "magic_link" or "code"
}

type Response struct {
	Message string `json:"message"`
	TokenID string `json:"tokenId"`
}

func Handle(ctx context.Context, req events.APIGatewayProxyRequest, db *sql.DB, sesClient *ses.Client, fromEmail, appURL string) (events.APIGatewayProxyResponse, error) {
	if db == nil {
		return errResp(503, "Passwordless auth not configured")
	}

	var body Request
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return errResp(400, "Invalid request body")
	}
	if body.Email == "" {
		return errResp(400, "Email is required")
	}
	if body.Type == "" {
		body.Type = "magic_link"
	}

	// Rate limit: max 3 in 15 min
	var count int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM passwordless_tokens WHERE user_email = $1 AND created_at > NOW() - INTERVAL '15 minutes'`,
		body.Email).Scan(&count)
	if err != nil {
		return errResp(500, "Rate limit check failed")
	}
	if count >= 3 {
		return errResp(429, "Too many requests. Please try again later.")
	}

	tokenID := uuid.New().String()
	var token string
	if body.Type == "code" {
		token, err = generateCode(6)
	} else {
		token, err = generateToken(32)
	}
	if err != nil {
		return errResp(500, "Failed to generate token")
	}

	tokenHash, _ := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	expiresAt := time.Now().Add(10 * time.Minute)

	_, err = db.ExecContext(ctx,
		`INSERT INTO passwordless_tokens (token_id, user_email, token_hash, token_type, expires_at, attempts, created_at) VALUES ($1,$2,$3,$4,$5,0,NOW())`,
		tokenID, body.Email, string(tokenHash), body.Type, expiresAt)
	if err != nil {
		return errResp(500, "Failed to store token")
	}

	if err := sendEmail(ctx, sesClient, fromEmail, appURL, body.Email, token, body.Type); err != nil {
		return errResp(500, "Failed to send email")
	}

	return okResp(200, Response{Message: "Verification sent to your email", TokenID: tokenID})
}

func generateCode(length int) (string, error) {
	const digits = "0123456789"
	code := make([]byte, length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[n.Int64()]
	}
	return string(code), nil
}

func generateToken(length int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	token := make([]byte, length)
	for i := range token {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		token[i] = chars[n.Int64()]
	}
	return string(token), nil
}

func sendEmail(ctx context.Context, sesClient *ses.Client, fromEmail, appURL, email, token, tokenType string) error {
	var subject, body string
	if tokenType == "code" {
		subject = "Your verification code"
		body = fmt.Sprintf(`<html><body><h2>Your Verification Code</h2><p>Your code is: <strong>%s</strong></p><p>Expires in 10 minutes.</p></body></html>`, token)
	} else {
		link := fmt.Sprintf("%s/auth/verify?token=%s", appURL, token)
		subject = "Your magic link"
		body = fmt.Sprintf(`<html><body><h2>Sign in</h2><p><a href="%s">Sign in now</a></p><p>Expires in 10 minutes.</p></body></html>`, link)
	}
	_, err := sesClient.SendEmail(ctx, &ses.SendEmailInput{
		Source:      &fromEmail,
		Destination: &sestypes.Destination{ToAddresses: []string{email}},
		Message: &sestypes.Message{
			Subject: &sestypes.Content{Data: &subject},
			Body:    &sestypes.Body{Html: &sestypes.Content{Data: &body}},
		},
	})
	return err
}

func okResp(status int, body any) (events.APIGatewayProxyResponse, error) {
	j, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{StatusCode: status, Headers: headers(), Body: string(j)}, nil
}

func errResp(status int, message string) (events.APIGatewayProxyResponse, error) {
	j, _ := json.Marshal(map[string]string{"error": "PasswordlessError", "message": message})
	return events.APIGatewayProxyResponse{StatusCode: status, Headers: headers(), Body: string(j)}, nil
}

func headers() map[string]string {
	return map[string]string{"Content-Type": "application/json", "Access-Control-Allow-Origin": "*"}
}
