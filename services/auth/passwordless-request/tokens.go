package main

import (
	"crypto/rand"
	"math/big"
)

func generateCode(length int) (string, error) {
	const digits = "0123456789"
	code := make([]byte, length)

	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}

	return string(code), nil
}

func generateToken(length int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	token := make([]byte, length)

	for i := range token {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		token[i] = chars[num.Int64()]
	}

	return string(token), nil
}

func generateAuthToken(tokenType string) (string, error) {
	if tokenType == "code" {
		return generateCode(6)
	}
	return generateToken(32)
}
