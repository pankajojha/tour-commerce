package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var pwdlessReq PasswordlessRequest
	if err := json.Unmarshal([]byte(request.Body), &pwdlessReq); err != nil {
		return errorResponse(400, "Invalid request body", err)
	}

	if pwdlessReq.Email == "" {
		return errorResponse(400, "Email is required", nil)
	}

	if pwdlessReq.Type == "" {
		pwdlessReq.Type = "magic_link"
	}

	if err := checkRateLimit(ctx, pwdlessReq.Email); err != nil {
		return errorResponse(429, "Too many requests. Please try again later.", err)
	}

	tokenID := uuid.New().String()
	token, err := generateAuthToken(pwdlessReq.Type)
	if err != nil {
		return errorResponse(500, "Failed to generate token", err)
	}

	tokenHash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return errorResponse(500, "Failed to hash token", err)
	}

	expiresAt := time.Now().Add(10 * time.Minute)
	if err := saveToken(ctx, tokenID, pwdlessReq.Email, string(tokenHash), pwdlessReq.Type, expiresAt); err != nil {
		return errorResponse(500, "Failed to store token", err)
	}

	if err := sendEmail(ctx, pwdlessReq.Email, token, pwdlessReq.Type); err != nil {
		return errorResponse(500, "Failed to send email", err)
	}

	response := PasswordlessResponse{
		Message: "Verification code sent to your email",
		TokenID: tokenID,
	}
	return successResponse(200, response)
}
