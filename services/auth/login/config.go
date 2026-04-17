package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

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
