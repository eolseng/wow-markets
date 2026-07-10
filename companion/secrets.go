package main

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const (
	keyringService           = "WoW Markets Companion"
	legacyKeyringService     = "Wow Market Scan"
	keyringInstallationToken = "installation-token"
)

func loadInstallationToken() (string, error) {
	token, err := keyring.Get(keyringService, keyringInstallationToken)
	if err == nil {
		return token, nil
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		return "", err
	}

	legacyToken, legacyErr := keyring.Get(legacyKeyringService, keyringInstallationToken)
	if errors.Is(legacyErr, keyring.ErrNotFound) {
		return "", nil
	}
	if legacyErr != nil {
		return "", legacyErr
	}
	if err := keyring.Set(keyringService, keyringInstallationToken, legacyToken); err != nil {
		return "", err
	}
	if err := keyring.Delete(legacyKeyringService, keyringInstallationToken); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return legacyToken, err
	}
	return legacyToken, nil
}

func saveInstallationToken(token string) error {
	return keyring.Set(keyringService, keyringInstallationToken, token)
}

func deleteInstallationToken() error {
	legacyErr := keyring.Delete(legacyKeyringService, keyringInstallationToken)
	if legacyErr != nil && !errors.Is(legacyErr, keyring.ErrNotFound) {
		return legacyErr
	}
	err := keyring.Delete(keyringService, keyringInstallationToken)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
