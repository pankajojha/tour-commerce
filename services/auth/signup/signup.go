package signup

import (
	"context"
	"encoding/json"
	"log"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

type Request struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
	Phone    string `json:"phone,omitempty"`
}

type Response struct {
	UserSub          string `json:"userSub"`
	Message          string `json:"message"`
	CodeDeliveryDest string `json:"codeDeliveryDestination,omitempty"`
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func Handle(ctx context.Context, req events.APIGatewayProxyRequest, cognitoClient *cognitoidentityprovider.Client, clientID string) (events.APIGatewayProxyResponse, error) {
	var body Request
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return errResp(400, "Invalid request body")
	}
	if !emailRegex.MatchString(body.Email) {
		return errResp(400, "Invalid email format")
	}
	if len(body.Password) < 8 {
		return errResp(400, "Password must be at least 8 characters")
	}

	attrs := []types.AttributeType{
		{Name: strPtr("email"), Value: strPtr(body.Email)},
	}
	if body.Name != "" {
		attrs = append(attrs, types.AttributeType{Name: strPtr("name"), Value: strPtr(body.Name)})
	}
	if body.Phone != "" {
		attrs = append(attrs, types.AttributeType{Name: strPtr("phone_number"), Value: strPtr(body.Phone)})
	}

	result, err := cognitoClient.SignUp(ctx, &cognitoidentityprovider.SignUpInput{
		ClientId:       strPtr(clientID),
		Username:       strPtr(body.Email),
		Password:       strPtr(body.Password),
		UserAttributes: attrs,
	})
	if err != nil {
		return mapCognitoErr(err)
	}

	resp := Response{
		UserSub: *result.UserSub,
		Message: "User registered successfully. Please check your email for verification code.",
	}
	if result.CodeDeliveryDetails != nil && result.CodeDeliveryDetails.Destination != nil {
		resp.CodeDeliveryDest = *result.CodeDeliveryDetails.Destination
	}
	return okResp(201, resp)
}

func mapCognitoErr(err error) (events.APIGatewayProxyResponse, error) {
	msg := err.Error()
	log.Printf("Cognito error: %v", msg)
	switch {
	case contains(msg, "UsernameExistsException"):
		return errResp(409, "User with this email already exists")
	case contains(msg, "InvalidPasswordException"):
		return errResp(400, "Password does not meet requirements")
	case contains(msg, "InvalidParameterException"):
		return errResp(400, "Invalid parameters provided")
	default:
		return errResp(500, "Failed to register user")
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func okResp(status int, body any) (events.APIGatewayProxyResponse, error) {
	j, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{StatusCode: status, Headers: headers(), Body: string(j)}, nil
}

func errResp(status int, message string) (events.APIGatewayProxyResponse, error) {
	j, _ := json.Marshal(map[string]string{"error": "SignupError", "message": message})
	return events.APIGatewayProxyResponse{StatusCode: status, Headers: headers(), Body: string(j)}, nil
}

func headers() map[string]string {
	return map[string]string{"Content-Type": "application/json", "Access-Control-Allow-Origin": "*"}
}

func strPtr(s string) *string { return &s }
