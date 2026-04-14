# s3browser

An interactive terminal browser for S3-compatible object storage. Navigate buckets and prefixes with arrow keys, view file metadata, open files with your default application, and upload or delete objects.

## Installation

```bash
git clone https://github.com/lofsgaard/s3browser
cd s3browser
go build -o s3browser.exe .
```

## Usage

```bash
s3browser --bucket <bucket-name> [options]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--bucket` | — | Bucket name (required). Can also be set via `S3_BUCKET` env var |
| `--region` | `us-east-1` | AWS region. Can also be set via `AWS_DEFAULT_REGION` |
| `--profile` | — | AWS credentials profile from `~/.aws/credentials` |
| `--endpoint` | — | Custom endpoint URL for S3-compatible services (e.g. MinIO) |

## Authentication

The app uses the [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2) default credential chain. Credentials are resolved automatically in this order:

### 1. Environment variables

```bash
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_SESSION_TOKEN=your-session-token  # optional, for temporary credentials
```

### 2. AWS credentials file

Located at `~/.aws/credentials` on Linux/macOS or `%USERPROFILE%\.aws\credentials` on Windows:

```ini
[default]
aws_access_key_id = your-access-key
aws_secret_access_key = your-secret-key

[staging]
aws_access_key_id = staging-access-key
aws_secret_access_key = staging-secret-key
```

Use a named profile with `--profile`:

```bash
s3browser --bucket my-bucket --profile staging
```

### 3. AWS config file

Located at `~/.aws/config`. Useful for SSO or assume-role setups:

```ini
[profile my-sso-profile]
sso_start_url = https://my-org.awsapps.com/start
sso_region = eu-west-1
sso_account_id = 123456789012
sso_role_name = ReadOnly
region = eu-west-1
```

Log in first, then run:

```bash
aws sso login --profile my-sso-profile
s3browser --bucket my-bucket --profile my-sso-profile
```

### 4. EC2 / ECS instance metadata

When running on AWS infrastructure (EC2, ECS, Lambda), credentials are picked up automatically from the instance metadata service — no configuration needed.

## S3-compatible services

Point `--endpoint` at any S3-compatible service. The app uses path-style URLs automatically when an endpoint is set.

```bash
# MinIO
s3browser --bucket my-bucket --endpoint http://localhost:9000

# Intility or other hosted S3-compatible services
AWS_ACCESS_KEY_ID=key AWS_SECRET_ACCESS_KEY=secret \
  s3browser --bucket my-bucket --endpoint https://s3.intility.com

# LocalStack
s3browser --bucket my-bucket --endpoint http://localhost:4566 --region us-east-1
```

## Key bindings

| Key | Action |
|-----|--------|
| `↑` / `↓` or `k` / `j` | Move cursor |
| `Enter` / `→` | Open folder or open file with default app |
| `←` / `Backspace` | Go up one level |
| `n` | Next page (buckets with >1000 objects) |
| `p` | Previous page |
| `d` | Delete selected file (asks for confirmation) |
| `u` | Upload a local file to the current prefix |
| `q` / `Ctrl+C` | Quit |

## How file opening works

Pressing `Enter` on a file downloads it to your system temp directory and opens it with the OS default application for that file type. The file persists in temp after the app closes so you don't need to wait for it to finish loading.
