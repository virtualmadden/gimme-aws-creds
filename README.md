# gimme-aws-creds

![PR Test](https://github.com/virtualmadden/gimme-aws-creds/actions/workflows/pr-test.yml/badge.svg)

A Go CLI that uses **AWS IAM Identity Center (SSO)** to acquire short-lived AWS credentials and write them to `~/.aws/credentials` or export them to your shell.

Inspired by [Nike-Inc/gimme-aws-creds](https://github.com/Nike-Inc/gimme-aws-creds), but replaces Okta/SAML authentication with native AWS SSO.

## Quick Start

```bash
# 1. Install
go install github.com/virtualmadden/gimme-aws-creds/cmd/gimme-aws-creds@latest

# Ensure Go's bin directory is on your PATH (required once per shell profile)
export PATH="$PATH:$(go env GOPATH)/bin"

# 2. Configure a profile (writes to ~/.aws/config)
gimme-aws-creds configure

# 3. Get credentials
gimme-aws-creds --profile <profile-name>
```

## Prerequisites

- Go 1.22+ (for building from source)
- An AWS IAM Identity Center (SSO) portal configured in your organization
- AWS config profiles using the [sso-session format](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sso.html)

## Installation

### From source

```bash
git clone https://github.com/virtualmadden/gimme-aws-creds.git
cd gimme-aws-creds
make install
```

`make install` places the binary in `$(go env GOPATH)/bin` (usually `~/go/bin`). Add it to your PATH:

```bash
# zsh (~/.zshrc) or bash (~/.bashrc)
export PATH="$PATH:$HOME/go/bin"
```

Then reload your shell (`source ~/.zshrc`) or open a new terminal.

**Alternative:** install directly to `/usr/local/bin` (already on most macOS PATHs):

```bash
make install-path
```

Or run without installing:

```bash
make build
./bin/gimme-aws-creds version
```

### Pre-built releases

Download the latest binary from [GitHub Releases](https://github.com/virtualmadden/gimme-aws-creds/releases).

## Configuration

Run the interactive wizard:

```bash
gimme-aws-creds configure
```

Or add profiles manually to `~/.aws/config`:

```ini
[profile dev]
sso_session = corp
sso_account_id = 123456789012
sso_role_name = AdministratorAccess
region = us-west-2

[sso-session corp]
sso_start_url = https://my-org.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access
```

`sso_account_id` and `sso_role_name` are optional. If omitted, you will be prompted to pick an account and role interactively.

## Usage

```bash
# Login and fetch credentials (default)
gimme-aws-creds --profile dev

# SSO login with interactive account/role selection
gimme-aws-creds login --profile dev

# Skip account picker when account is known
gimme-aws-creds login --profile dev --account-id 123456789012

# Export credentials for shell eval
gimme-aws-creds --profile dev -o export
eval "$(gimme-aws-creds --profile dev -o export)"

# JSON output
gimme-aws-creds --profile dev -o json

# Skip writing to ~/.aws/credentials
gimme-aws-creds --profile dev --no-write
```

### Flags

| Flag | Description |
|------|-------------|
| `--profile` | AWS config profile name (default: `default`) |
| `-o`, `--output` | Output format: `export` or `json` |
| `--no-write` | Do not write to `~/.aws/credentials` |
| `--no-save-config` | Do not save account/role selection to `~/.aws/config` |
| `--region` | Override the profile's AWS region |
| `--no-browser` | Do not open browser during SSO login |
| `--debug` | Enable debug output |

### Environment variables

| Variable | Description |
|----------|-------------|
| `AWS_CONFIG_FILE` | Override path to AWS config (default: `~/.aws/config`) |
| `AWS_SHARED_CREDENTIALS_FILE` | Override credentials file (default: `~/.aws/credentials`) |

## How it works

1. **SSO login** — Uses the AWS OIDC device authorization flow (same as `aws sso login`) and caches the token in `~/.aws/sso/cache/` in AWS CLI-compatible format.
2. **Role selection** — Lists available accounts and roles via the SSO API, or uses values from your profile.
3. **Credential retrieval** — Calls `GetRoleCredentials` and writes temporary STS credentials with `x_security_token_expires`.

## Development

```bash
make build    # build to bin/gimme-aws-creds
make test     # run unit tests
make lint     # go vet
```

## License

Apache License 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
