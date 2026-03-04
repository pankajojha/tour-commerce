package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type PasswordlessRequest struct {
	Email string `json:"email"`
	Type  string `json:"type"` // "magic_link" or "code"
}

type PasswordlessResponse struct {
	Message string `json:"message"`
	TokenID string `json:"tokenId"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

var (
	sesClient *ses.Client
	db        *sql.DB
	dbHost    string
	dbName    string
	dbUser    string
	dbPass    string
	fromEmail string
	appURL    string
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	sesClient = ses.NewFromConfig(cfg)

	// Database connection
	dbHost = os.Getenv("DB_HOST")
	dbName = os.Getenv("DB_NAME")
	dbUser = os.Getenv("DB_USER")
	dbPass = os.Getenv("DB_PASSWORD")
	fromEmail = os.Getenv("FROM_EMAIL")
	appURL = os.Getenv("APP_URL")

	if dbHost == "" || dbName == "" || dbUser == "" || dbPass == "" {
		log.Fatal("Database environment variables must be set")
	}

	connStr := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=require",
		dbHost, dbUser, dbPass, dbName)

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse request body
	var pwdlessReq PasswordlessRequest
	if err := json.Unmarshal([]byte(request.Body), &pwdlessReq); err != nil {
		return errorResponse(400, "Invalid request body", err)
	}

	// Validate email
	if pwdlessReq.Email == "" {
		return errorResponse(400, "Email is required", nil)
	}

	// Default to magic_link if type not specified
	if pwdlessReq.Type == "" {
		pwdlessReq.Type = "magic_link"
	}

	// Check rate limiting (max 3 attempts in last 15 minutes)
	if err := checkRateLimit(ctx, pwdlessReq.Email); err != nil {
		return errorResponse(429, "Too many requests. Please try again later.", err)
	}

	// Generate token
	tokenID := uuid.New().String()
	var token string
	var err error

	if pwdlessReq.Type == "code" {
		token, err = generateCode(6)
	} else {
		token, err = generateToken(32)
	}

	if err != nil {
		return errorResponse(500, "Failed to generate token", err)
	}

	// Hash token for storage
	tokenHash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return errorResponse(500, "Failed to hash token", err)
	}

	// Store token in database with 10-minute expiration
	expiresAt := time.Now().Add(10 * time.Minute)
	query := `
		INSERT INTO passwordless_tokens (token_id, user_email, token_hash, token_type, expires_at, attempts, created_at)
		VALUES ($1, $2, $3, $4, $5, 0, NOW())
	`

	_, err = db.ExecContext(ctx, query, tokenID, pwdlessReq.Email, string(tokenHash), pwdlessReq.Type, expiresAt)
	if err != nil {
		return errorResponse(500, "Failed to store token", err)
	}

	// Send email
	if err := sendEmail(ctx, pwdlessReq.Email, token, pwdlessReq.Type); err != nil {
		return errorResponse(500, "Failed to send email", err)
	}

	response := PasswordlessResponse{
		Message: "Verification code sent to your email",
		TokenID: tokenID,
	}

	return successResponse(200, response)
}

func checkRateLimit(ctx context.Context, email string) error {
	query := `
		SELECT COUNT(*) FROM passwordless_tokens
		WHERE user_email = $1 AND created_at > NOW() - INTERVAL '15 minutes'
	`

	var count int
	err := db.QueryRowContext(ctx, query, email).Scan(&count)
	if err != nil {
		return err
	}

	if count >= 3 {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}

func generateCode(length int) (string, error) {
	const digits = "0123456789"
	code := make([]byte, length)

	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}

	return string(code), nil
}

func generateToken(length int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	token := make([]byte, length)

	for i := range token {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		token[i] = chars[num.Int64()]
	}

	return string(token), nil
}

func sendEmail(ctx context.Context, email, token, tokenType string) error {
	var subject, body string

	if tokenType == "code" {
		subject = "Your verification code"
		body = fmt.Sprintf(`
			<html>
			<body>
				<h2>Your Verification Code</h2>
				<p>Your verification code is: <strong>%s</strong></p>
				<p>This code will expire in 10 minutes.</p>
				<p>If you didn't request this code, please ignore this email.</p>
			</body>
			</html>
		`, token)
	} else {
		magicLink := fmt.Sprintf("%s/auth/verify?token=%s", appURL, token)
		subject = "Your magic link"
		body = fmt.Sprintf(`
			<html>
			<body>
				<h2>Sign in to Explorer Agent</h2>
				<p>Click the link below to sign in:</p>
				<p><a href="%s">Sign in now</a></p>
				<p>This link will expire in 10 minutes.</p>
				<p>If you didn't request this link, please ignore this email.</p>
			</body>
			</html>
		`, magicLink)
	}

	input := &ses.SendEmailInput{
		Source: &fromEmail,
		Destination: &types.Destination{
			ToAddresses: []string{email},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data: &subject,
			},
			Body: &types.Body{
				Html: &types.Content{
					Data: &body,
				},
			},
		},
	}

	_, err := sesClient.SendEmail(ctx, input)
	return err
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
		Error:   "PasswordlessError",
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

func main() {
	lambda.Start(handler)
}
