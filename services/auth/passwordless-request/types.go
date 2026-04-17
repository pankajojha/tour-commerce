package main

type PasswordlessRequest struct {
	Email string `json:"email"`
	Type  string `json:"type"`
}

type PasswordlessResponse struct {
	Message string `json:"message"`
	TokenID string `json:"tokenId"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
