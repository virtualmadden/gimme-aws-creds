package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/virtualmadden/gimme-aws-creds/internal/awsconfig"
	"github.com/virtualmadden/gimme-aws-creds/internal/ui"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure an AWS SSO profile",
	Long:  "Interactive wizard to add an sso-session and profile to ~/.aws/config.",
	RunE:  runConfigure,
}

func runConfigure(cmd *cobra.Command, args []string) error {
	fmt.Fprintln(os.Stdout, "Configure AWS SSO profile")
	fmt.Fprintln(os.Stdout, "Press Enter to accept defaults shown in brackets.")
	fmt.Fprintln(os.Stdout)

	profileName, err := ui.PromptDefault("Profile name", flags.profile)
	if err != nil {
		return err
	}
	if profileName == "" {
		return fmt.Errorf("profile name is required")
	}

	startURL, err := ui.Prompt("SSO start URL (e.g. https://my-org.awsapps.com/start): ")
	if err != nil {
		return err
	}
	if startURL == "" {
		return fmt.Errorf("sso_start_url is required")
	}

	ssoRegion, err := ui.PromptDefault("SSO region", "us-east-1")
	if err != nil {
		return err
	}

	accountID, err := ui.PromptDefault("SSO account ID (optional)", "")
	if err != nil {
		return err
	}

	roleName, err := ui.PromptDefault("SSO role name (optional)", "")
	if err != nil {
		return err
	}

	region, err := ui.PromptDefault("Default AWS region", "us-west-2")
	if err != nil {
		return err
	}

	sessionName, err := ui.PromptDefault("SSO session name", profileName+"-sso")
	if err != nil {
		return err
	}

	profile := awsconfig.Profile{
		Name:            profileName,
		SSOSession:      sessionName,
		SSOStartURL:     startURL,
		SSORegion:       ssoRegion,
		SSOAccountID:    accountID,
		SSORoleName:     roleName,
		Region:          region,
		RegistrationScopes: "sso:account:access",
	}

	if err := awsconfig.WriteProfile(profile); err != nil {
		return err
	}

	paths, err := awsconfig.DefaultPaths()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\nConfiguration written to %s\n", paths.Config)
	fmt.Fprintf(os.Stdout, "Run `gimme-aws-creds --profile %s` to authenticate and fetch credentials.\n", profileName)
	return nil
}
