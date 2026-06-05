package credentials

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/ini.v1"
)

func TestWriteCredentialsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")

	creds := AWSCredentials{
		AccessKeyID:     "AKIAEXAMPLE",
		SecretAccessKey: "secret",
		SessionToken:    "token",
		Expiration:      time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC),
		ProfileName:     "dev",
	}

	if err := Write(creds, WriteOptions{
		CredentialsPath: path,
		ProfileName:     "dev",
		WriteFile:       true,
		Stdout:          ioDiscard{},
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	cfg, err := ini.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	section := cfg.Section("dev")
	if section.Key("aws_access_key_id").String() != creds.AccessKeyID {
		t.Errorf("access key = %q", section.Key("aws_access_key_id").String())
	}
	if section.Key("x_security_token_expires").String() != "2026-06-04T12:00:00Z" {
		t.Errorf("expires = %q", section.Key("x_security_token_expires").String())
	}
}

func TestWriteExportFormat(t *testing.T) {
	var buf bytes.Buffer
	creds := AWSCredentials{
		AccessKeyID:     "AKIAEXAMPLE",
		SecretAccessKey: "s!ecret",
		SessionToken:    "token",
	}

	if err := Write(creds, WriteOptions{
		Format:    FormatExport,
		WriteFile: false,
		Stdout:    &buf,
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "export AWS_ACCESS_KEY_ID=AKIAEXAMPLE") {
		t.Errorf("unexpected export output: %s", out)
	}
	if !strings.Contains(out, "export AWS_SECRET_ACCESS_KEY='s!ecret'") {
		t.Errorf("expected quoted secret, got: %s", out)
	}
}

func TestWriteJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	creds := AWSCredentials{
		AccessKeyID:  "AKIAEXAMPLE",
		AccountID:    "123456789012",
		RoleName:     "Admin",
		ProfileName:  "dev",
		Expiration:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := Write(creds, WriteOptions{
		Format:    FormatJSON,
		WriteFile: false,
		Stdout:    &buf,
	}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if !strings.Contains(buf.String(), `"AccessKeyId":"AKIAEXAMPLE"`) &&
		!strings.Contains(buf.String(), `"AccessKeyId": "AKIAEXAMPLE"`) {
		t.Errorf("unexpected JSON: %s", buf.String())
	}
}

func TestShellEscape(t *testing.T) {
	if shellEscape("plain") != "plain" {
		t.Error("plain string should not be quoted")
	}
	if shellEscape("has!bang") != "'has!bang'" {
		t.Errorf("got %q", shellEscape("has!bang"))
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func TestCredentialsFileCreated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "credentials")

	creds := AWSCredentials{
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		SessionToken:    "token",
		Expiration:      time.Now().Add(time.Hour),
	}

	if err := writeCredentialsFile(path, "test", creds); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("credentials file not created: %v", err)
	}

	parentInfo, err := os.Stat(filepath.Join(dir, "nested"))
	if err != nil {
		t.Fatal(err)
	}
	if parentInfo.Mode().Perm() != 0o700 {
		t.Errorf("created parent dir mode = %o, want 0700", parentInfo.Mode().Perm())
	}
}
