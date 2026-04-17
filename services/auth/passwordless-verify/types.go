package main

type VerifyRequest struct {
	TokenID string `json:"tokenId"`
	Token   string `json:"token"`
}

type VerifyResponse struct {
	AccessToken  string `json:"accessToken"`
	IdToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int32  `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type tokenRecord struct {
	userEmail string
	tokenHash string
	expiresAt int64
	attempts  int
}
