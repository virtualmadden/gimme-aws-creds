package credentials

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/virtualmadden/gimme-aws-creds/internal/awsconfig"
	"github.com/virtualmadden/gimme-aws-creds/internal/ssologin"
	"github.com/virtualmadden/gimme-aws-creds/internal/ui"
)

// AWSCredentials holds temporary AWS credentials from SSO.
type AWSCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
	AccountID       string
	RoleName        string
	ProfileName     string
}

// FetchOptions configures credential retrieval.
type FetchOptions struct {
	Profile      awsconfig.Profile
	Paths        awsconfig.Paths
	AccountID    string
	RoleName     string
	Interactive  bool
	SaveSelection bool
}

// Fetch retrieves temporary AWS credentials for the given profile.
func Fetch(ctx context.Context, opts FetchOptions) (AWSCredentials, error) {
	token, err := ssologin.LoadToken(opts.Profile.SSOStartURL)
	if err != nil {
		return AWSCredentials{}, err
	}
	if token == nil {
		return AWSCredentials{}, fmt.Errorf("no valid SSO session; run `gimme-aws-creds login --profile %s`", opts.Profile.Name)
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.Profile.SSORegion))
	if err != nil {
		return AWSCredentials{}, fmt.Errorf("load AWS config: %w", err)
	}

	client := sso.NewFromConfig(cfg)

	accountID := opts.AccountID
	if accountID == "" {
		accountID = opts.Profile.SSOAccountID
	}
	roleName := opts.RoleName
	if roleName == "" {
		roleName = opts.Profile.SSORoleName
	}

	if accountID == "" || roleName == "" {
		if !opts.Interactive {
			return AWSCredentials{}, fmt.Errorf("profile %q is missing sso_account_id or sso_role_name", opts.Profile.Name)
		}

		accountID, roleName, err = SelectAccountAndRole(ctx, opts.Profile, SelectionOptions{})
		if err != nil {
			return AWSCredentials{}, err
		}

		if opts.SaveSelection {
			if err := awsconfig.UpdateProfileAccountRole(opts.Profile.Name, accountID, roleName); err != nil {
				return AWSCredentials{}, fmt.Errorf("save account/role to config: %w", err)
			}
		}
	}

	output, err := client.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{
		AccessToken: aws.String(token.AccessToken),
		AccountId:   aws.String(accountID),
		RoleName:    aws.String(roleName),
	})
	if err != nil {
		return AWSCredentials{}, fmt.Errorf("get role credentials: %w", err)
	}

	expiration := time.Unix(0, output.RoleCredentials.Expiration*int64(time.Millisecond)).UTC()

	return AWSCredentials{
		AccessKeyID:     aws.ToString(output.RoleCredentials.AccessKeyId),
		SecretAccessKey: aws.ToString(output.RoleCredentials.SecretAccessKey),
		SessionToken:    aws.ToString(output.RoleCredentials.SessionToken),
		Expiration:      expiration,
		AccountID:       accountID,
		RoleName:        roleName,
		ProfileName:     opts.Profile.Name,
	}, nil
}

// SelectionOptions optionally pre-selects an account or role to skip prompts.
type SelectionOptions struct {
	AccountID string
	RoleName  string
}

// SelectAccountAndRole lists SSO accounts and roles and prompts for any missing values.
func SelectAccountAndRole(ctx context.Context, profile awsconfig.Profile, opts SelectionOptions) (string, string, error) {
	token, err := ssologin.LoadToken(profile.SSOStartURL)
	if err != nil {
		return "", "", err
	}
	if token == nil {
		return "", "", fmt.Errorf("no valid SSO session; authenticate first")
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(profile.SSORegion))
	if err != nil {
		return "", "", fmt.Errorf("load AWS config: %w", err)
	}

	client := sso.NewFromConfig(cfg)
	return pickAccountAndRole(ctx, client, token.AccessToken, opts)
}

func pickAccountAndRole(ctx context.Context, client *sso.Client, accessToken string, opts SelectionOptions) (string, string, error) {
	accountID := opts.AccountID
	roleName := opts.RoleName

	if accountID == "" {
		accounts, err := listAccounts(ctx, client, accessToken)
		if err != nil {
			return "", "", err
		}
		if len(accounts) == 0 {
			return "", "", fmt.Errorf("no AWS accounts available for this SSO user")
		}

		accountIdx, err := ui.Pick("Select AWS account:", accountLabels(accounts))
		if err != nil {
			return "", "", err
		}
		accountID = aws.ToString(accounts[accountIdx].AccountId)
	}

	if roleName == "" {
		roles, err := listRoles(ctx, client, accessToken, accountID)
		if err != nil {
			return "", "", err
		}
		if len(roles) == 0 {
			return "", "", fmt.Errorf("no roles available for account %s", accountID)
		}

		roleIdx, err := ui.Pick("Select IAM role:", roleLabels(roles))
		if err != nil {
			return "", "", err
		}
		roleName = aws.ToString(roles[roleIdx].RoleName)
	}

	return accountID, roleName, nil
}

func listAccounts(ctx context.Context, client *sso.Client, accessToken string) ([]types.AccountInfo, error) {
	var accounts []types.AccountInfo
	paginator := sso.NewListAccountsPaginator(client, &sso.ListAccountsInput{
		AccessToken: aws.String(accessToken),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list accounts: %w", err)
		}
		accounts = append(accounts, page.AccountList...)
	}
	return accounts, nil
}

func listRoles(ctx context.Context, client *sso.Client, accessToken, accountID string) ([]types.RoleInfo, error) {
	var roles []types.RoleInfo
	paginator := sso.NewListAccountRolesPaginator(client, &sso.ListAccountRolesInput{
		AccessToken: aws.String(accessToken),
		AccountId:   aws.String(accountID),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list roles: %w", err)
		}
		roles = append(roles, page.RoleList...)
	}
	return roles, nil
}

func accountLabels(accounts []types.AccountInfo) []string {
	labels := make([]string, len(accounts))
	for i, a := range accounts {
		name := aws.ToString(a.AccountName)
		if name == "" {
			name = aws.ToString(a.AccountId)
		} else {
			name = fmt.Sprintf("%s (%s)", name, aws.ToString(a.AccountId))
		}
		labels[i] = name
	}
	return labels
}

func roleLabels(roles []types.RoleInfo) []string {
	labels := make([]string, len(roles))
	for i, r := range roles {
		labels[i] = aws.ToString(r.RoleName)
	}
	return labels
}
