package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

func sendEmail(ctx context.Context, email, token, tokenType string) error {
	var subject, body string

	if tokenType == "code" {
		subject = "Your verification code"
		body = fmt.Sprintf(`
			<html>
			<body>
				<h2>Your Verification Code</h2>
				<p>Your verification code is: <strong>%s</strong></p>
				<p>This code will expire in 10 minutes.</p>
				<p>If you didn't request this code, please ignore this email.</p>
			</body>
			</html>
		`, token)
	} else {
		magicLink := fmt.Sprintf("%s/auth/verify?token=%s", appURL, token)
		subject = "Your magic link"
		body = fmt.Sprintf(`
			<html>
			<body>
				<h2>Sign in to Explorer Agent</h2>
				<p>Click the link below to sign in:</p>
				<p><a href="%s">Sign in now</a></p>
				<p>This link will expire in 10 minutes.</p>
				<p>If you didn't request this link, please ignore this email.</p>
			</body>
			</html>
		`, magicLink)
	}

	input := &ses.SendEmailInput{
		Source: &fromEmail,
		Destination: &types.Destination{
			ToAddresses: []string{email},
		},
		Message: &types.Message{
			Subject: &types.Content{Data: &subject},
			Body: &types.Body{
				Html: &types.Content{Data: &body},
			},
		},
	}

	_, err := sesClient.SendEmail(ctx, input)
	return err
}
