package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type VerifyRequest struct {
	TokenID string `json:"tokenId"`
	Token   string `json:"token"`
}

type VerifyResponse struct {
	AccessToken  string `json:"accessToken"`
	IdToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int32  `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

var (
	cognitoClient *cognitoidentityprovider.Client
	db            *sql.DB
	dbHost        string
	dbName        string
	dbUser        string
	dbPass        string
	clientID      string
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	cognitoClient = cognitoidentityprovider.NewFromConfig(cfg)
	clientID = os.Getenv("CLIENT_ID")

	// Database connection
	dbHost = os.Getenv("DB_HOST")
	dbName = os.Getenv("DB_NAME")
	dbUser = os.Getenv("DB_USER")
	dbPass = os.Getenv("DB_PASSWORD")

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
	var verifyReq VerifyRequest
	if err := json.Unmarshal([]byte(request.Body), &verifyReq); err != nil {
		return errorResponse(400, "Invalid request body", err)
	}

	// Validate input
	if verifyReq.TokenID == "" || verifyReq.Token == "" {
		return errorResponse(400, "TokenID and token are required", nil)
	}

	// Retrieve token from database
	var userEmail, tokenHash string
	var expiresAt time.Time
	var attempts int

	query := `
		SELECT user_email, token_hash, expires_at, attempts
		FROM passwordless_tokens
		WHERE token_id = $1
	`

	err := db.QueryRowContext(ctx, query, verifyReq.TokenID).Scan(&userEmail, &tokenHash, &expiresAt, &attempts)
	if err == sql.ErrNoRows {
		return errorResponse(404, "Invalid or expired token", nil)
	}
	if err != nil {
		return errorResponse(500, "Failed to retrieve token", err)
	}

	// Check if token has expired
	if time.Now().After(expiresAt) {
		return errorResponse(401, "Token has expired", nil)
	}

	// Check attempts (max 3)
	if attempts >= 3 {
		return errorResponse(429, "Too many verification attempts", nil)
	}

	// Verify token hash
	err = bcrypt.CompareHashAndPassword([]byte(tokenHash), []byte(verifyReq.Token))
	if err != nil {
		// Increment attempts
		updateQuery := `UPDATE passwordless_tokens SET attempts = attempts + 1 WHERE token_id = $1`
		db.ExecContext(ctx, updateQuery, verifyReq.TokenID)

		return errorResponse(401, "Invalid token", nil)
	}

	// Delete used token
	deleteQuery := `DELETE FROM passwordless_tokens WHERE token_id = $1`
	_, err = db.ExecContext(ctx, deleteQuery, verifyReq.TokenID)
	if err != nil {
		log.Printf("Failed to delete token: %v", err)
	}

	// Authenticate user with Cognito using custom auth flow
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

	// Check if authentication was successful
	if result.AuthenticationResult == nil {
		return errorResponse(401, "Authentication failed", nil)
	}

	// Prepare response
	response := VerifyResponse{
		AccessToken:  *result.AuthenticationResult.AccessToken,
		IdToken:      *result.AuthenticationResult.IdToken,
		RefreshToken: *result.AuthenticationResult.RefreshToken,
		ExpiresIn:    result.AuthenticationResult.ExpiresIn,
		TokenType:    *result.AuthenticationResult.TokenType,
	}

	return successResponse(200, response)
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
		Error:   "VerificationError",
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
