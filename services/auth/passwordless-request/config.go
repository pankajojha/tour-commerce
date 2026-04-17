package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	_ "github.com/lib/pq"
)

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

	dbHost = os.Getenv("DB_HOST")
	dbName = os.Getenv("DB_NAME")
	dbUser = os.Getenv("DB_USER")
	dbPass = os.Getenv("DB_PASSWORD")
	fromEmail = os.Getenv("FROM_EMAIL")
	appURL = os.Getenv("APP_URL")

	if dbHost == "" || dbName == "" || dbUser == "" || dbPass == "" {
		log.Fatal("Database environment variables must be set")
	}

	connStr := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=require", dbHost, dbUser, dbPass, dbName)

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
}
