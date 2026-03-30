// Single Lambda entry point — routes API Gateway requests to auth service packages.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	_ "github.com/lib/pq"

	"github.com/explorer-agent/auth/login"
	passwordlessrequest "github.com/explorer-agent/auth/passwordless-request"
	passwordlessverify "github.com/explorer-agent/auth/passwordless-verify"
	"github.com/explorer-agent/auth/signup"
	tokenrefresh "github.com/explorer-agent/auth/token-refresh"
)

// ServiceHandler is the signature every auth service package exposes.
type ServiceHandler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

var (
	cognitoClient *cognitoidentityprovider.Client
	sesClient     *ses.Client
	db            *sql.DB
	clientID      string
	fromEmail     string
	appURL        string

	// routes maps "METHOD /resource/path" → service handler.
	routes map[string]ServiceHandler
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	cognitoClient = cognitoidentityprovider.NewFromConfig(cfg)
	sesClient = ses.NewFromConfig(cfg)

	clientID = os.Getenv("CLIENT_ID")
	fromEmail = os.Getenv("FROM_EMAIL")
	appURL = os.Getenv("APP_URL")

	if clientID == "" {
		log.Fatal("CLIENT_ID environment variable must be set")
	}

	// Database — only needed for passwordless flows
	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	if dbHost != "" && dbName != "" && dbUser != "" && dbPass != "" {
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

	// Register routes — each points to its service package.
	routes = map[string]ServiceHandler{
		"POST /auth/signup": func(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			return signup.Handle(ctx, r, cognitoClient, clientID)
		},
		"POST /auth/login": func(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			return login.Handle(ctx, r, cognitoClient, clientID)
		},
		"POST /auth/passwordless/request": func(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			return passwordlessrequest.Handle(ctx, r, db, sesClient, fromEmail, appURL)
		},
		"POST /auth/passwordless/verify": func(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			return passwordlessverify.Handle(ctx, r, db, cognitoClient, clientID)
		},
		"POST /auth/refresh": func(ctx context.Context, r events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			return tokenrefresh.Handle(ctx, r, cognitoClient, clientID)
		},
	}
}

func router(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	path := req.Resource
	if path == "" {
		path = req.Path
	}
	key := req.HTTPMethod + " " + path

	log.Printf("Auth router: %s", key)

	if handler, ok := routes[key]; ok {
		return handler(ctx, req)
	}

	body, _ := json.Marshal(map[string]string{"error": "NotFound", "message": "Route not found: " + key})
	return events.APIGatewayProxyResponse{
		StatusCode: 404,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}, nil
}

func main() {
	lambda.Start(router)
}
