package main

import (
	"regexp"

	"github.com/aws/aws-lambda-go/events"
)

var (
	usernameExistsRegex  = regexp.MustCompile(`UsernameExistsException`)
	invalidPasswordRegex = regexp.MustCompile(`InvalidPasswordException`)
	invalidParamRegex    = regexp.MustCompile(`InvalidParameterException`)
)

func handleCognitoError(err error) (events.APIGatewayProxyResponse, error) {
	errMsg := err.Error()

	if usernameExistsRegex.MatchString(errMsg) {
		return errorResponse(409, "User with this email already exists", err)
	}
	if invalidPasswordRegex.MatchString(errMsg) {
		return errorResponse(400, "Password does not meet requirements", err)
	}
	if invalidParamRegex.MatchString(errMsg) {
		return errorResponse(400, "Invalid parameters provided", err)
	}

	return errorResponse(500, "Failed to register user", err)
}
