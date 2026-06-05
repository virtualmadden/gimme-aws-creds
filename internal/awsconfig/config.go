package awsconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

const (
	defaultConfigPath      = ".aws/config"
	defaultCredentialsPath = ".aws/credentials"
)

// Profile holds resolved SSO profile settings from ~/.aws/config.
type Profile struct {
	Name            string
	SSOSession      string
	SSOStartURL     string
	SSORegion       string
	SSOAccountID    string
	SSORoleName     string
	Region          string
	RegistrationScopes string
}

// Paths resolves AWS config and credentials file paths from environment.
type Paths struct {
	Config      string
	Credentials string
}

// DefaultPaths returns config file paths honoring AWS_* env vars.
func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve home directory: %w", err)
	}

	configPath := os.Getenv("AWS_CONFIG_FILE")
	if configPath == "" {
		configPath = filepath.Join(home, defaultConfigPath)
	}

	credsPath := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	if credsPath == "" {
		credsPath = filepath.Join(home, defaultCredentialsPath)
	}

	return Paths{Config: configPath, Credentials: credsPath}, nil
}

// LoadProfile reads and resolves an AWS config profile with its sso-session.
func LoadProfile(profileName string) (Profile, Paths, error) {
	paths, err := DefaultPaths()
	if err != nil {
		return Profile{}, Paths{}, err
	}

	profile, err := loadProfileFromFile(paths.Config, profileName)
	if err != nil {
		return Profile{}, paths, err
	}

	profile.Name = profileName
	return profile, paths, nil
}

func loadProfileFromFile(configPath, profileName string) (Profile, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return Profile{}, fmt.Errorf("AWS config file not found at %s; run `gimme-aws-creds configure` first", configPath)
	}

	cfg, err := ini.Load(configPath)
	if err != nil {
		return Profile{}, fmt.Errorf("parse AWS config: %w", err)
	}

	sectionName := profileSectionName(profileName)
	section, err := cfg.GetSection(sectionName)
	if err != nil {
		return Profile{}, fmt.Errorf("profile %q not found in %s; run `gimme-aws-creds configure`", profileName, configPath)
	}

	profile := Profile{
		Name:         profileName,
		SSOSession:   section.Key("sso_session").String(),
		SSOAccountID: section.Key("sso_account_id").String(),
		SSORoleName:  section.Key("sso_role_name").String(),
		Region:       section.Key("region").String(),
	}

	if profile.SSOSession == "" {
		// Legacy inline SSO profile format.
		profile.SSOStartURL = section.Key("sso_start_url").String()
		profile.SSORegion = section.Key("sso_region").String()
		if profile.SSOStartURL == "" || profile.SSORegion == "" {
			return Profile{}, fmt.Errorf("profile %q is missing sso_session or sso_start_url/sso_region", profileName)
		}
		return profile, nil
	}

	sessionSection, err := cfg.GetSection("sso-session " + profile.SSOSession)
	if err != nil {
		return Profile{}, fmt.Errorf("sso-session %q referenced by profile %q not found", profile.SSOSession, profileName)
	}

	profile.SSOStartURL = sessionSection.Key("sso_start_url").String()
	profile.SSORegion = sessionSection.Key("sso_region").String()
	profile.RegistrationScopes = sessionSection.Key("sso_registration_scopes").String()

	if profile.SSOStartURL == "" || profile.SSORegion == "" {
		return Profile{}, fmt.Errorf("sso-session %q is missing sso_start_url or sso_region", profile.SSOSession)
	}

	if profile.RegistrationScopes == "" {
		profile.RegistrationScopes = "sso:account:access"
	}

	return profile, nil
}

func profileSectionName(name string) string {
	if name == "default" {
		return "default"
	}
	return "profile " + name
}

// WriteProfile appends or updates an sso-session and profile in ~/.aws/config.
func WriteProfile(profile Profile) error {
	paths, err := DefaultPaths()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(paths.Config), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	var cfg *ini.File
	if _, err := os.Stat(paths.Config); os.IsNotExist(err) {
		cfg = ini.Empty()
	} else {
		cfg, err = ini.Load(paths.Config)
		if err != nil {
			return fmt.Errorf("parse AWS config: %w", err)
		}
	}

	sessionName := profile.SSOSession
	if sessionName == "" {
		sessionName = sanitizeName(profile.Name) + "-sso"
		profile.SSOSession = sessionName
	}

	sessionSection, err := cfg.GetSection("sso-session " + sessionName)
	if err != nil {
		sessionSection, err = cfg.NewSection("sso-session " + sessionName)
		if err != nil {
			return fmt.Errorf("create sso-session section: %w", err)
		}
	}

	sessionSection.Key("sso_start_url").SetValue(profile.SSOStartURL)
	sessionSection.Key("sso_region").SetValue(profile.SSORegion)
	scopes := profile.RegistrationScopes
	if scopes == "" {
		scopes = "sso:account:access"
	}
	sessionSection.Key("sso_registration_scopes").SetValue(scopes)

	profileSection, err := cfg.GetSection(profileSectionName(profile.Name))
	if err != nil {
		profileSection, err = cfg.NewSection(profileSectionName(profile.Name))
		if err != nil {
			return fmt.Errorf("create profile section: %w", err)
		}
	}

	profileSection.Key("sso_session").SetValue(sessionName)
	if profile.SSOAccountID != "" {
		profileSection.Key("sso_account_id").SetValue(profile.SSOAccountID)
	}
	if profile.SSORoleName != "" {
		profileSection.Key("sso_role_name").SetValue(profile.SSORoleName)
	}
	if profile.Region != "" {
		profileSection.Key("region").SetValue(profile.Region)
	}

	return cfg.SaveTo(paths.Config)
}

// UpdateProfileAccountRole updates sso_account_id and sso_role_name for a profile.
func UpdateProfileAccountRole(profileName, accountID, roleName string) error {
	paths, err := DefaultPaths()
	if err != nil {
		return err
	}

	cfg, err := ini.Load(paths.Config)
	if err != nil {
		return fmt.Errorf("parse AWS config: %w", err)
	}

	section, err := cfg.GetSection(profileSectionName(profileName))
	if err != nil {
		return fmt.Errorf("profile %q not found", profileName)
	}

	section.Key("sso_account_id").SetValue(accountID)
	section.Key("sso_role_name").SetValue(roleName)

	return cfg.SaveTo(paths.Config)
}

// ListProfiles returns all profile names that reference SSO.
func ListProfiles() ([]string, error) {
	paths, err := DefaultPaths()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(paths.Config); os.IsNotExist(err) {
		return nil, nil
	}

	cfg, err := ini.Load(paths.Config)
	if err != nil {
		return nil, fmt.Errorf("parse AWS config: %w", err)
	}

	var profiles []string
	for _, section := range cfg.Sections() {
		name := section.Name()
		switch {
		case name == "default" && (section.HasKey("sso_session") || section.HasKey("sso_start_url")):
			profiles = append(profiles, "default")
		case strings.HasPrefix(name, "profile ") && (section.HasKey("sso_session") || section.HasKey("sso_start_url")):
			profiles = append(profiles, strings.TrimPrefix(name, "profile "))
		}
	}

	return profiles, nil
}

func sanitizeName(name string) string {
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, name)
	return strings.Trim(name, "-")
}
