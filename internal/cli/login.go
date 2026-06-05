package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/virtualmadden/gimme-aws-creds/internal/awsconfig"
	"github.com/virtualmadden/gimme-aws-creds/internal/credentials"
	"github.com/virtualmadden/gimme-aws-creds/internal/ssologin"
)

var loginFlags struct {
	accountID string
	roleName  string
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with AWS SSO and select an account",
	Long: `Perform the AWS SSO browser login flow, then interactively select
an AWS account and IAM role from your SSO portal. The selection can be
saved to ~/.aws/config for future runs.`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().StringVar(&loginFlags.accountID, "account-id", "", "AWS account ID (skips account picker)")
	loginCmd.Flags().StringVar(&loginFlags.roleName, "role-name", "", "IAM role name (skips role picker)")
}

func runLogin(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	profile, paths, err := awsconfig.LoadProfile(flags.profile)
	if err != nil {
		return err
	}

	if !ssologin.IsValid(profile.SSOStartURL) {
		var scopes []string
		if profile.RegistrationScopes != "" {
			scopes = []string{profile.RegistrationScopes}
		}

		if err := ssologin.Login(ctx, ssologin.LoginOptions{
			StartURL:  profile.SSOStartURL,
			Region:    profile.SSORegion,
			Scopes:    scopes,
			Stdout:    os.Stdout,
			NoBrowser: flags.noBrowser,
		}); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(os.Stdout, "SSO session for profile %q is already valid.\n", flags.profile)
	}

	accountID, roleName, err := credentials.SelectAccountAndRole(ctx, profile, credentials.SelectionOptions{
		AccountID: loginFlags.accountID,
		RoleName:  loginFlags.roleName,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\nSelected account: %s\n", accountID)
	fmt.Fprintf(os.Stdout, "Selected role:    %s\n", roleName)

	if !flags.noSaveConfig {
		if err := awsconfig.UpdateProfileAccountRole(flags.profile, accountID, roleName); err != nil {
			return err
		}
	}

	creds, err := credentials.Fetch(ctx, credentials.FetchOptions{
		Profile:     profile,
		Paths:       paths,
		AccountID:   accountID,
		RoleName:    roleName,
		Interactive: false,
	})
	if err != nil {
		return err
	}

	writeFile := !flags.noWrite && flags.output == ""
	outputFormat := credentials.OutputFormat(flags.output)
	if flags.noWrite && flags.output == "" {
		outputFormat = credentials.FormatExport
	}

	return credentials.Write(creds, credentials.WriteOptions{
		CredentialsPath: paths.Credentials,
		ProfileName:     profile.Name,
		Format:          outputFormat,
		WriteFile:       writeFile,
		Stdout:          os.Stdout,
	})
}
