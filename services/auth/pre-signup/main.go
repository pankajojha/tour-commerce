package main

import (
	"context"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

var cognitoClient *cognitoidentityprovider.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}
	cognitoClient = cognitoidentityprovider.NewFromConfig(cfg)
}

// handler implements the "User Pool First" account-linking pattern.
//
// When a social provider (Google, Facebook, Apple) sign-up arrives and a
// native Cognito account with the same email already exists, the social
// identity is linked to the existing account rather than creating a
// duplicate.  New social sign-ups are auto-confirmed with a verified email.
func handler(ctx context.Context, event events.CognitoEventUserPoolsPreSignup) (events.CognitoEventUserPoolsPreSignup, error) {
	log.Printf("Pre-signup trigger: source=%s user=%s", event.TriggerSource, event.UserName)

	switch event.TriggerSource {
	case "PreSignUp_ExternalProvider":
		return handleExternalProvider(ctx, event)
	default:
		// Native sign-ups go through normal verification
		return event, nil
	}
}

func handleExternalProvider(ctx context.Context, event events.CognitoEventUserPoolsPreSignup) (events.CognitoEventUserPoolsPreSignup, error) {
	email, ok := event.Request.UserAttributes["email"]
	if !ok || email == "" {
		log.Println("No email in social sign-up attributes, skipping link")
		event.Response.AutoConfirmUser = true
		event.Response.AutoVerifyEmail = true
		return event, nil
	}

	// Look up existing native users with the same email
	listOut, err := cognitoClient.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{
		UserPoolId: aws.String(event.UserPoolID),
		Filter:     aws.String(`email = "` + email + `"`),
		Limit:      aws.Int32(10),
	})
	if err != nil {
		log.Printf("ListUsers error: %v", err)
		// Don't block sign-up on lookup failure
		event.Response.AutoConfirmUser = true
		event.Response.AutoVerifyEmail = true
		return event, nil
	}

	// Find a native (non-federated) user to link to
	for _, user := range listOut.Users {
		username := aws.ToString(user.Username)
		if isFederatedUsername(username) {
			continue
		}

		// Parse provider info from the federated username (e.g. "Google_123456")
		providerName, providerSub, ok := parseFederatedUsername(event.UserName)
		if !ok {
			log.Printf("Could not parse federated username: %s", event.UserName)
			break
		}

		_, err := cognitoClient.AdminLinkProviderForUser(ctx, &cognitoidentityprovider.AdminLinkProviderForUserInput{
			UserPoolId: aws.String(event.UserPoolID),
			DestinationUser: &types.ProviderUserIdentifierType{
				ProviderName:           aws.String("Cognito"),
				ProviderAttributeValue: aws.String(username),
			},
			SourceUser: &types.ProviderUserIdentifierType{
				ProviderName:           aws.String(providerName),
				ProviderAttributeName:  aws.String("Cognito_Subject"),
				ProviderAttributeValue: aws.String(providerSub),
			},
		})
		if err != nil {
			log.Printf("AdminLinkProviderForUser error: %v", err)
		} else {
			log.Printf("Linked %s identity to existing user %s", providerName, username)
		}
		break
	}

	event.Response.AutoConfirmUser = true
	event.Response.AutoVerifyEmail = true
	return event, nil
}

// isFederatedUsername returns true for usernames created by social providers.
func isFederatedUsername(username string) bool {
	prefixes := []string{"google_", "facebook_", "signinwithapple_"}
	lower := strings.ToLower(username)
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

// parseFederatedUsername splits "Google_123456" into ("Google", "123456", true).
func parseFederatedUsername(username string) (provider, sub string, ok bool) {
	idx := strings.Index(username, "_")
	if idx < 1 || idx >= len(username)-1 {
		return "", "", false
	}
	return username[:idx], username[idx+1:], true
}

func main() {
	lambda.Start(handler)
}
