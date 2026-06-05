package awsconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProfileWithSSOSession(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config := `[profile dev]
sso_session = corp
sso_account_id = 123456789012
sso_role_name = AdministratorAccess
region = us-west-2

[sso-session corp]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access
`
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_CONFIG_FILE", configPath)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))

	profile, _, err := LoadProfile("dev")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}

	if profile.SSOStartURL != "https://example.awsapps.com/start" {
		t.Errorf("SSOStartURL = %q", profile.SSOStartURL)
	}
	if profile.SSORegion != "us-east-1" {
		t.Errorf("SSORegion = %q", profile.SSORegion)
	}
	if profile.SSOAccountID != "123456789012" {
		t.Errorf("SSOAccountID = %q", profile.SSOAccountID)
	}
	if profile.SSORoleName != "AdministratorAccess" {
		t.Errorf("SSORoleName = %q", profile.SSORoleName)
	}
}

func TestWriteProfile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	t.Setenv("AWS_CONFIG_FILE", configPath)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))

	profile := Profile{
		Name:         "test",
		SSOSession:   "test-sso",
		SSOStartURL:  "https://example.awsapps.com/start",
		SSORegion:    "us-east-1",
		SSOAccountID: "111122223333",
		SSORoleName:  "ReadOnly",
		Region:       "us-west-2",
	}

	if err := WriteProfile(profile); err != nil {
		t.Fatalf("WriteProfile: %v", err)
	}

	loaded, _, err := LoadProfile("test")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}

	if loaded.SSOAccountID != profile.SSOAccountID {
		t.Errorf("SSOAccountID = %q, want %q", loaded.SSOAccountID, profile.SSOAccountID)
	}
	if loaded.SSORoleName != profile.SSORoleName {
		t.Errorf("SSORoleName = %q, want %q", loaded.SSORoleName, profile.SSORoleName)
	}
}

func TestListProfiles(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	config := `[profile alpha]
sso_session = s1

[profile beta]
sso_start_url = https://legacy.example.com/start
sso_region = us-east-1

[sso-session s1]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
`
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_CONFIG_FILE", configPath)

	profiles, err := ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}

	if len(profiles) != 2 {
		t.Fatalf("profiles = %v, want 2 entries", profiles)
	}
}
