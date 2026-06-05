package ssologin

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCachePathDeterministic(t *testing.T) {
	path1, err := CachePath("https://example.awsapps.com/start")
	if err != nil {
		t.Fatal(err)
	}
	path2, err := CachePath("https://example.awsapps.com/start")
	if err != nil {
		t.Fatal(err)
	}
	if path1 != path2 {
		t.Errorf("cache paths differ: %s vs %s", path1, path2)
	}
	if filepath.Ext(path1) != ".json" {
		t.Errorf("expected .json extension, got %s", path1)
	}
}

func TestStoreAndLoadToken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	startURL := "https://test.awsapps.com/start"
	expires := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)

	token := TokenCache{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		ExpiresAt:    expires,
	}

	if err := StoreToken(startURL, token); err != nil {
		t.Fatalf("StoreToken: %v", err)
	}

	loaded, err := LoadToken(startURL)
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected token, got nil")
	}
	if loaded.AccessToken != token.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, token.AccessToken)
	}
}

func TestLoadTokenExpired(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	startURL := "https://expired.awsapps.com/start"
	expires := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)

	if err := StoreToken(startURL, TokenCache{
		AccessToken: "old",
		ExpiresAt:   expires,
	}); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadToken(startURL)
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for expired token")
	}
}

func TestIsValid(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	startURL := "https://valid.awsapps.com/start"
	if IsValid(startURL) {
		t.Error("expected invalid before store")
	}

	if err := StoreToken(startURL, TokenCache{
		AccessToken: "token",
		ExpiresAt:   time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	if !IsValid(startURL) {
		t.Error("expected valid after store")
	}

	// Ensure cache dir was created with restricted permissions.
	path, _ := CachePath(startURL)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("token file mode = %o, want 0600", info.Mode().Perm())
	}
}
