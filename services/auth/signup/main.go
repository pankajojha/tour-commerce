package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
	Phone    string `json:"phone,omitempty"`
}

type SignupResponse struct {
	UserSub          string `json:"userSub"`
	Message          string `json:"message"`
	CodeDeliveryDest string `json:"codeDeliveryDestination,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

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

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse request body
	var signupReq SignupRequest
	if err := json.Unmarshal([]byte(request.Body), &signupReq); err != nil {
		return errorResponse(400, "Invalid request body", err)
	}

	// Validate email
	if !emailRegex.MatchString(signupReq.Email) {
		return errorResponse(400, "Invalid email format", nil)
	}

	// Validate password (basic check - Cognito will enforce full policy)
	if len(signupReq.Password) < 8 {
		return errorResponse(400, "Password must be at least 8 characters", nil)
	}

	// Prepare user attributes
	userAttributes := []types.AttributeType{
		{
			Name:  stringPtr("email"),
			Value: stringPtr(signupReq.Email),
		},
	}

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

	// Sign up user in Cognito
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

	// Prepare response
	response := SignupResponse{
		UserSub: *result.UserSub,
		Message: "User registered successfully. Please check your email for verification code.",
	}

	if result.CodeDeliveryDetails != nil && result.CodeDeliveryDetails.Destination != nil {
		response.CodeDeliveryDest = *result.CodeDeliveryDetails.Destination
	}

	return successResponse(201, response)
}

func handleCognitoError(err error) (events.APIGatewayProxyResponse, error) {
	errMsg := err.Error()

	// Check for specific Cognito errors
	if regexp.MustCompile(`UsernameExistsException`).MatchString(errMsg) {
		return errorResponse(409, "User with this email already exists", err)
	}
	if regexp.MustCompile(`InvalidPasswordException`).MatchString(errMsg) {
		return errorResponse(400, "Password does not meet requirements", err)
	}
	if regexp.MustCompile(`InvalidParameterException`).MatchString(errMsg) {
		return errorResponse(400, "Invalid parameters provided", err)
	}

	return errorResponse(500, "Failed to register user", err)
}

func successResponse(statusCode int, body interface{}) (events.APIGatewayProxyResponse, error) {
	jsonBody, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(jsonBody),
	}, nil
}

func errorResponse(statusCode int, message string, err error) (events.APIGatewayProxyResponse, error) {
	errResp := ErrorResponse{
		Error:   "SignupError",
		Message: message,
	}

	if err != nil {
		log.Printf("Error: %v", err)
	}

	jsonBody, _ := json.Marshal(errResp)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(jsonBody),
	}, nil
}

func stringPtr(s string) *string {
	return &s
}

func main() {
	lambda.Start(handler)
}
