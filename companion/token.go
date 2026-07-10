package main

import (
	"encoding/base64"
	"errors"
	"strings"
)

const (
	installationTokenVersionPrefix = "wms1_"
	installationTokenSecretBytes   = 32
	installationTokenHintLength    = len(installationTokenVersionPrefix) + 8
)

func normalizeInstallationToken(value string) (string, error) {
	token := strings.TrimSpace(value)
	if !strings.HasPrefix(token, installationTokenVersionPrefix) {
		return "", errors.New("enter an installation token beginning with wms1_")
	}
	encoded := strings.TrimPrefix(token, installationTokenVersionPrefix)
	secret, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(secret) != installationTokenSecretBytes || base64.RawURLEncoding.EncodeToString(secret) != encoded {
		return "", errors.New("this installation token is not valid; copy the complete token from WoW Markets")
	}
	return token, nil
}

func installationTokenPrefix(token string) string {
	if len(token) <= installationTokenHintLength {
		return token
	}
	return token[:installationTokenHintLength]
}
