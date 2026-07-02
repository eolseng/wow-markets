package companionauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxResponseBytes = 1 << 20

type Client struct {
	apiURL string
	client *http.Client
}

type AuthSession struct {
	AccessToken      string `json:"access_token"`
	AccessExpiresIn  int    `json:"access_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	User             User   `json:"user"`
}

type User struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	ID          string `json:"id"`
	IsAdmin     bool   `json:"is_admin"`
}

type Installation struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	TokenPrefix string `json:"token_prefix"`
}

type InstallationCreateResponse struct {
	Installation Installation `json:"installation"`
	Token        string       `json:"token"`
}

type problemResponse struct {
	Detail  string          `json:"detail"`
	Error   string          `json:"error"`
	Message json.RawMessage `json:"message"`
}

func New(apiURL string) (*Client, error) {
	apiURL = strings.TrimRight(strings.TrimSpace(apiURL), "/")
	if apiURL == "" {
		return nil, errors.New("API URL is required")
	}
	return &Client{
		apiURL: apiURL,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (client *Client) Login(
	ctx context.Context,
	email string,
	password string,
) (AuthSession, error) {
	return postJSON[AuthSession](ctx, client.client, client.apiURL+"/v1/auth/login", "", map[string]string{
		"email":    strings.TrimSpace(email),
		"password": password,
	})
}

func (client *Client) Refresh(
	ctx context.Context,
	refreshToken string,
) (AuthSession, error) {
	return postJSON[AuthSession](ctx, client.client, client.apiURL+"/v1/auth/refresh", "", map[string]string{
		"refresh_token": refreshToken,
	})
}

func (client *Client) CreateInstallation(
	ctx context.Context,
	accessToken string,
	name string,
) (InstallationCreateResponse, error) {
	return postJSON[InstallationCreateResponse](
		ctx,
		client.client,
		client.apiURL+"/v1/me/installations",
		accessToken,
		map[string]string{"name": strings.TrimSpace(name)},
	)
}

func postJSON[T any](
	ctx context.Context,
	httpClient *http.Client,
	url string,
	bearer string,
	body any,
) (T, error) {
	var zero T
	payload, err := json.Marshal(body)
	if err != nil {
		return zero, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return zero, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "WowMarketScanCompanion/0.1")
	if strings.TrimSpace(bearer) != "" {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearer))
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return zero, err
	}
	defer response.Body.Close()

	responsePayload, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return zero, err
	}
	if len(responsePayload) > maxResponseBytes {
		return zero, errors.New("API response exceeds size limit")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return zero, apiError(response.StatusCode, responsePayload)
	}

	var result T
	decoder := json.NewDecoder(bytes.NewReader(responsePayload))
	if err := decoder.Decode(&result); err != nil {
		return zero, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return zero, errors.New("API response contained trailing JSON")
	}
	return result, nil
}

func apiError(status int, payload []byte) error {
	var problem problemResponse
	if err := json.Unmarshal(payload, &problem); err == nil {
		if problem.Detail != "" {
			return fmt.Errorf("API returned %d: %s", status, problem.Detail)
		}
		if problem.Error != "" {
			return fmt.Errorf("API returned %d: %s", status, problem.Error)
		}
		if message := problemMessage(problem.Message); message != "" {
			return fmt.Errorf("API returned %d: %s", status, message)
		}
	}
	return fmt.Errorf("API returned HTTP %d", status)
}

func problemMessage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var message string
	if err := json.Unmarshal(raw, &message); err == nil {
		return message
	}
	var messages []string
	if err := json.Unmarshal(raw, &messages); err == nil {
		return strings.Join(messages, "; ")
	}
	return ""
}
