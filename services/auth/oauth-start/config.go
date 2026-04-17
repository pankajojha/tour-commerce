package main

import (
	"log"
	"os"
	"strings"
)

var (
	clientID     string
	cognitoHost  string
	callbackURL  string
	defaultScope string
)

func init() {
	clientID = strings.TrimSpace(os.Getenv("CLIENT_ID"))
	cognitoHost = strings.TrimSpace(os.Getenv("COGNITO_DOMAIN"))
	callbackURL = strings.TrimSpace(os.Getenv("OAUTH_CALLBACK_URL"))
	defaultScope = strings.TrimSpace(os.Getenv("OAUTH_SCOPE"))

	if defaultScope == "" {
		defaultScope = "openid email profile"
	}

	if clientID == "" || cognitoHost == "" || callbackURL == "" {
		log.Fatal("CLIENT_ID, COGNITO_DOMAIN, and OAUTH_CALLBACK_URL environment variables must be set")
	}
}
