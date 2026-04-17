package main

import (
	"log"
	"os"
	"strings"
)

var (
	frontendSuccessURL string
	frontendCancelURL  string
)

func init() {
	frontendSuccessURL = strings.TrimSpace(os.Getenv("FRONTEND_AUTH_CALLBACK_URL"))
	frontendCancelURL = strings.TrimSpace(os.Getenv("FRONTEND_AUTH_CANCEL_URL"))

	if frontendSuccessURL == "" || frontendCancelURL == "" {
		log.Fatal("FRONTEND_AUTH_CALLBACK_URL and FRONTEND_AUTH_CANCEL_URL environment variables must be set")
	}
}
