package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func decodeState(raw string) (oauthState, error) {
	if raw == "" {
		return oauthState{}, fmt.Errorf("empty state")
	}

	decodedBytes, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return oauthState{}, err
	}

	var state oauthState
	if err := json.Unmarshal(decodedBytes, &state); err != nil {
		return oauthState{}, err
	}

	state.Provider = strings.ToLower(strings.TrimSpace(state.Provider))
	return state, nil
}
