package main

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
