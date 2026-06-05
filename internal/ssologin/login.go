package ssologin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssooidctypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/pkg/browser"
)

const (
	deviceCodeGrant = "urn:ietf:params:oauth:grant-type:device_code"
	clientName      = "gimme-aws-creds"
)

// LoginOptions configures the SSO login flow.
type LoginOptions struct {
	StartURL string
	Region   string
	Scopes   []string
	Stdout   io.Writer
	NoBrowser bool
}

// Login performs the AWS SSO OIDC device authorization flow and caches the token.
func Login(ctx context.Context, opts LoginOptions) error {
	if opts.StartURL == "" {
		return fmt.Errorf("sso_start_url is required")
	}
	if opts.Region == "" {
		return fmt.Errorf("sso_region is required")
	}

	scopes := opts.Scopes
	if len(scopes) == 0 {
		scopes = []string{"sso:account:access"}
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(opts.Region))
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	client := ssooidc.NewFromConfig(cfg)

	register, err := client.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String(clientName),
		ClientType: aws.String("public"),
		Scopes:     scopes,
		GrantTypes: []string{
			"urn:ietf:params:oauth:grant-type:device_code",
			"refresh_token",
		},
	})
	if err != nil {
		return fmt.Errorf("register SSO client: %w", err)
	}

	deviceAuth, err := client.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		StartUrl:     aws.String(opts.StartURL),
	})
	if err != nil {
		return fmt.Errorf("start device authorization: %w", err)
	}

	verificationURL := aws.ToString(deviceAuth.VerificationUriComplete)
	if verificationURL == "" {
		verificationURL = aws.ToString(deviceAuth.VerificationUri)
	}

	fmt.Fprintf(opts.Stdout, "Attempting to open the SSO authorization page in your default browser.\n")
	fmt.Fprintf(opts.Stdout, "If the browser does not open, visit:\n%s\n", verificationURL)
	fmt.Fprintf(opts.Stdout, "User code: %s\n", aws.ToString(deviceAuth.UserCode))

	if !opts.NoBrowser && verificationURL != "" {
		if err := browser.OpenURL(verificationURL); err != nil {
			fmt.Fprintf(opts.Stdout, "Could not open browser: %v\n", err)
		}
	}

	interval := time.Duration(deviceAuth.Interval) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}

	deadline := time.Now().Add(time.Duration(deviceAuth.ExpiresIn) * time.Second)
	pollCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	for {
		select {
		case <-pollCtx.Done():
			return fmt.Errorf("SSO login timed out")
		case <-time.After(interval):
		}

		token, err := client.CreateToken(pollCtx, &ssooidc.CreateTokenInput{
			ClientId:     register.ClientId,
			ClientSecret: register.ClientSecret,
			GrantType:    aws.String(deviceCodeGrant),
			DeviceCode:   deviceAuth.DeviceCode,
		})
		if err != nil {
			var pending *ssooidctypes.AuthorizationPendingException
			var slowDown *ssooidctypes.SlowDownException
			if errors.As(err, &pending) {
				continue
			}
			if errors.As(err, &slowDown) {
				interval += 5 * time.Second
				continue
			}
			return fmt.Errorf("create SSO token: %w", err)
		}

		expiresAt := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).UTC().Format(time.RFC3339)

		cache := TokenCache{
			AccessToken:  aws.ToString(token.AccessToken),
			RefreshToken: aws.ToString(token.RefreshToken),
			ClientID:     aws.ToString(register.ClientId),
			ClientSecret: aws.ToString(register.ClientSecret),
			ExpiresAt:    expiresAt,
		}

		if register.ClientSecretExpiresAt != 0 {
			cache.RegistrationExpiresAt = time.Unix(register.ClientSecretExpiresAt, 0).UTC().Format(time.RFC3339)
		}

		if err := StoreToken(opts.StartURL, cache); err != nil {
			return err
		}

		fmt.Fprintln(opts.Stdout, "Successfully logged into AWS SSO.")
		return nil
	}
}

// EnsureLoggedIn triggers login if no valid cached token exists.
func EnsureLoggedIn(ctx context.Context, startURL, region, scopes string, stdout io.Writer) error {
	if IsValid(startURL) {
		return nil
	}

	var scopeList []string
	if scopes != "" {
		scopeList = []string{scopes}
	}

	return Login(ctx, LoginOptions{
		StartURL: startURL,
		Region:   region,
		Scopes:   scopeList,
		Stdout:   stdout,
	})
}
