package main

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type oauthState struct {
	Nonce    string `json:"nonce"`
	Provider string `json:"provider"`
}
