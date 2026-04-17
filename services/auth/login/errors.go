package main

import (
	"github.com/aws/aws-lambda-go/events"
)

func handleCognitoError(err error) (events.APIGatewayProxyResponse, error) {
	errMsg := err.Error()

	switch {
	case contains(errMsg, "NotAuthorizedException"):
		return errorResponse(401, "Incorrect email or password", err)
	case contains(errMsg, "UserNotConfirmedException"):
		return errorResponse(403, "User email not verified. Please check your email for verification code.", err)
	case contains(errMsg, "UserNotFoundException"):
		return errorResponse(401, "Incorrect email or password", err)
	case contains(errMsg, "TooManyRequestsException"):
		return errorResponse(429, "Too many login attempts. Please try again later.", err)
	case contains(errMsg, "PasswordResetRequiredException"):
		return errorResponse(403, "Password reset required", err)
	default:
		return errorResponse(500, "Login failed", err)
	}
}
