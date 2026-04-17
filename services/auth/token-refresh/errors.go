package main

import "github.com/aws/aws-lambda-go/events"

func handleCognitoError(err error) (events.APIGatewayProxyResponse, error) {
	errMsg := err.Error()

	switch {
	case contains(errMsg, "NotAuthorizedException"):
		return errorResponse(401, "Invalid or expired refresh token", err)
	case contains(errMsg, "TooManyRequestsException"):
		return errorResponse(429, "Too many requests. Please try again later.", err)
	default:
		return errorResponse(500, "Token refresh failed", err)
	}
}
