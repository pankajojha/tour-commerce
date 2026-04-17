package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var signupReq SignupRequest
	if err := json.Unmarshal([]byte(request.Body), &signupReq); err != nil {
		return errorResponse(400, "Invalid request body", err)
	}

	if !emailRegex.MatchString(signupReq.Email) {
		return errorResponse(400, "Invalid email format", nil)
	}

	if len(signupReq.Password) < 8 {
		return errorResponse(400, "Password must be at least 8 characters", nil)
	}

	userAttributes := []types.AttributeType{{
		Name:  stringPtr("email"),
		Value: stringPtr(signupReq.Email),
	}}

	if signupReq.Name != "" {
		userAttributes = append(userAttributes, types.AttributeType{
			Name:  stringPtr("name"),
			Value: stringPtr(signupReq.Name),
		})
	}

	if signupReq.Phone != "" {
		userAttributes = append(userAttributes, types.AttributeType{
			Name:  stringPtr("phone_number"),
			Value: stringPtr(signupReq.Phone),
		})
	}

	signUpInput := &cognitoidentityprovider.SignUpInput{
		ClientId:       stringPtr(clientID),
		Username:       stringPtr(signupReq.Email),
		Password:       stringPtr(signupReq.Password),
		UserAttributes: userAttributes,
	}

	result, err := cognitoClient.SignUp(ctx, signUpInput)
	if err != nil {
		return handleCognitoError(err)
	}

	response := SignupResponse{
		UserSub: *result.UserSub,
		Message: "User registered successfully. Please check your email for verification code.",
	}

	if result.CodeDeliveryDetails != nil && result.CodeDeliveryDetails.Destination != nil {
		response.CodeDeliveryDest = *result.CodeDeliveryDetails.Destination
	}

	return successResponse(201, response)
}
