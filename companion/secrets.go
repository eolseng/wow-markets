package main

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const (
	keyringService           = "Wow Market Scan"
	keyringInstallationToken = "installation-token"
	keyringRefreshToken      = "refresh-token"
)

func loadInstallationToken() (string, error) {
	token, err := keyring.Get(keyringService, keyringInstallationToken)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	return token, err
}

func saveInstallationToken(token string) error {
	return keyring.Set(keyringService, keyringInstallationToken, token)
}

func deleteInstallationToken() error {
	err := keyring.Delete(keyringService, keyringInstallationToken)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

func loadRefreshToken() (string, error) {
	token, err := keyring.Get(keyringService, keyringRefreshToken)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	return token, err
}

func saveRefreshToken(token string) error {
	return keyring.Set(keyringService, keyringRefreshToken, token)
}

func deleteRefreshToken() error {
	err := keyring.Delete(keyringService, keyringRefreshToken)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
