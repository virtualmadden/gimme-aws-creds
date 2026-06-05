package ssologin

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TokenCache matches the AWS CLI SSO token cache file format.
type TokenCache struct {
	AccessToken           string `json:"accessToken"`
	RefreshToken          string `json:"refreshToken,omitempty"`
	ClientID              string `json:"clientId"`
	ClientSecret          string `json:"clientSecret"`
	RegistrationExpiresAt string `json:"registrationExpiresAt,omitempty"`
	ExpiresAt             string `json:"expiresAt"`
}

// CachePath returns the AWS CLI-compatible cache file path for a start URL.
func CachePath(startURL string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	hash := sha1.Sum([]byte(startURL))
	filename := hex.EncodeToString(hash[:]) + ".json"
	return filepath.Join(home, ".aws", "sso", "cache", filename), nil
}

// LoadToken reads a cached SSO token if present and not expired.
func LoadToken(startURL string) (*TokenCache, error) {
	path, err := CachePath(startURL)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read SSO cache: %w", err)
	}

	var token TokenCache
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("parse SSO cache: %w", err)
	}

	if token.ExpiresAt != "" {
		expires, err := time.Parse(time.RFC3339, token.ExpiresAt)
		if err == nil && time.Now().After(expires) {
			return nil, nil
		}
	}

	if token.AccessToken == "" {
		return nil, nil
	}

	return &token, nil
}

// StoreToken writes a token to the AWS CLI-compatible cache location.
func StoreToken(startURL string, token TokenCache) error {
	path, err := CachePath(startURL)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create SSO cache directory: %w", err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal SSO token: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write SSO cache: %w", err)
	}

	return nil
}

// IsValid reports whether a cached token exists and is not expired.
func IsValid(startURL string) bool {
	token, err := LoadToken(startURL)
	return err == nil && token != nil
}
