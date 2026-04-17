package main

import (
	"fmt"
	"net/url"
	"strings"
)

func normalizedCognitoAuthorizeEndpoint(domain string) (string, error) {
	trimmed := strings.TrimSpace(domain)
	if trimmed == "" {
		return "", fmt.Errorf("empty cognito domain")
	}

	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}

	parsed.Path = "/oauth2/authorize"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func buildAuthorizeURL(providerName, encodedState string) (string, error) {
	authorizeBase, err := normalizedCognitoAuthorizeEndpoint(cognitoHost)
	if err != nil {
		return "", err
	}

	authorizeURL, err := url.Parse(authorizeBase)
	if err != nil {
		return "", err
	}

	query := authorizeURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", clientID)
	query.Set("redirect_uri", callbackURL)
	query.Set("identity_provider", providerName)
	query.Set("scope", defaultScope)
	query.Set("state", encodedState)
	authorizeURL.RawQuery = query.Encode()
	return authorizeURL.String(), nil
}
