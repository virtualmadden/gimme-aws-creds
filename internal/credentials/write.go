package credentials

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// OutputFormat determines how credentials are emitted.
type OutputFormat string

const (
	FormatExport OutputFormat = "export"
	FormatJSON   OutputFormat = "json"
)

// WriteOptions configures credential output.
type WriteOptions struct {
	CredentialsPath string
	ProfileName     string
	Format          OutputFormat
	WriteFile       bool
	Stdout          io.Writer
}

// Write outputs credentials to file and/or stdout based on options.
func Write(creds AWSCredentials, opts WriteOptions) error {
	if opts.WriteFile {
		if err := writeCredentialsFile(opts.CredentialsPath, opts.ProfileName, creds); err != nil {
			return err
		}
		fmt.Fprintf(opts.Stdout, "Wrote credentials to profile %q in %s\n", opts.ProfileName, opts.CredentialsPath)
	}

	if opts.Format != "" {
		switch opts.Format {
		case FormatExport:
			fmt.Fprintf(opts.Stdout, "export AWS_ACCESS_KEY_ID=%s\n", shellEscape(creds.AccessKeyID))
			fmt.Fprintf(opts.Stdout, "export AWS_SECRET_ACCESS_KEY=%s\n", shellEscape(creds.SecretAccessKey))
			fmt.Fprintf(opts.Stdout, "export AWS_SESSION_TOKEN=%s\n", shellEscape(creds.SessionToken))
		case FormatJSON:
			payload := map[string]string{
				"AccessKeyId":     creds.AccessKeyID,
				"SecretAccessKey": creds.SecretAccessKey,
				"SessionToken":    creds.SessionToken,
				"Expiration":      creds.Expiration.Format(time.RFC3339),
				"AccountId":       creds.AccountID,
				"RoleName":        creds.RoleName,
				"ProfileName":     creds.ProfileName,
			}
			enc := json.NewEncoder(opts.Stdout)
			if err := enc.Encode(payload); err != nil {
				return fmt.Errorf("encode JSON credentials: %w", err)
			}
		default:
			return fmt.Errorf("unsupported output format %q", opts.Format)
		}
	}

	return nil
}

func writeCredentialsFile(path, profileName string, creds AWSCredentials) error {
	if err := os.MkdirAll(dirOf(path), 0o700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}

	var cfg *ini.File
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg = ini.Empty()
	} else {
		cfg, err = ini.Load(path)
		if err != nil {
			return fmt.Errorf("parse credentials file: %w", err)
		}
	}

	section, err := cfg.GetSection(profileName)
	if err != nil {
		section, err = cfg.NewSection(profileName)
		if err != nil {
			return fmt.Errorf("create credentials section: %w", err)
		}
	}

	section.Key("aws_access_key_id").SetValue(creds.AccessKeyID)
	section.Key("aws_secret_access_key").SetValue(creds.SecretAccessKey)
	section.Key("aws_session_token").SetValue(creds.SessionToken)
	section.Key("x_security_token_expires").SetValue(creds.Expiration.Format(time.RFC3339))

	return cfg.SaveTo(path)
}

func dirOf(path string) string {
	idx := strings.LastIndex(path, string(os.PathSeparator))
	if idx < 0 {
		return "."
	}
	return path[:idx]
}

func shellEscape(s string) string {
	if strings.ContainsAny(s, " '\"\\$`!") {
		return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
	}
	return s
}
