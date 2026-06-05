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

var version = "dev"

type globalFlags struct {
	profile      string
	output       string
	noWrite      bool
	noSaveConfig bool
	region       string
	debug        bool
	noBrowser    bool
}

var flags globalFlags

var rootCmd = &cobra.Command{
	Use:   "gimme-aws-creds",
	Short: "Acquire temporary AWS credentials via AWS SSO",
	Long: `gimme-aws-creds is a CLI that uses AWS IAM Identity Center (SSO)
to acquire short-lived AWS credentials and write them to ~/.aws/credentials
or export them to your shell.`,
	RunE: runDefault,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flags.profile, "profile", "default", "AWS config profile name")
	rootCmd.PersistentFlags().StringVarP(&flags.output, "output", "o", "", "Output format: export or json")
	rootCmd.PersistentFlags().BoolVar(&flags.noWrite, "no-write", false, "Do not write to ~/.aws/credentials")
	rootCmd.PersistentFlags().BoolVar(&flags.noSaveConfig, "no-save-config", false, "Do not save account/role selection to ~/.aws/config")
	rootCmd.PersistentFlags().StringVar(&flags.region, "region", "", "Override AWS region for the profile")
	rootCmd.PersistentFlags().BoolVar(&flags.debug, "debug", false, "Enable debug output")
	rootCmd.PersistentFlags().BoolVar(&flags.noBrowser, "no-browser", false, "Do not open browser during SSO login")

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(configureCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gimme-aws-creds %s\n", version)
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func runDefault(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	profile, paths, err := awsconfig.LoadProfile(flags.profile)
	if err != nil {
		return err
	}

	if flags.region != "" {
		profile.Region = flags.region
	}

	if err := ssologin.EnsureLoggedIn(ctx, profile.SSOStartURL, profile.SSORegion, profile.RegistrationScopes, os.Stdout); err != nil {
		return err
	}

	saveSelection := !flags.noSaveConfig && (profile.SSOAccountID == "" || profile.SSORoleName == "")

	creds, err := credentials.Fetch(ctx, credentials.FetchOptions{
		Profile:       profile,
		Paths:         paths,
		Interactive:   true,
		SaveSelection: saveSelection,
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
