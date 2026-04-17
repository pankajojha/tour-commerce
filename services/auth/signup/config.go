package main

import (
	"context"
	"log"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

var (
	cognitoClient *cognitoidentityprovider.Client
	userPoolID    string
	clientID      string
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	cognitoClient = cognitoidentityprovider.NewFromConfig(cfg)
	userPoolID = os.Getenv("USER_POOL_ID")
	clientID = os.Getenv("CLIENT_ID")

	if userPoolID == "" || clientID == "" {
		log.Fatal("USER_POOL_ID and CLIENT_ID environment variables must be set")
	}
}
