# Proliferate (pro)

Proliferate is a CLI tool designed to automate and manage multiple pull requests across repositories inspired by [multi-gitter](https://github.com/lindell/multi-gitter). It supports templating, script execution, and status tracking of pull requests.

## Features

- 🔄 Create and manage multiple pull requests from templates
- 📝 Template-based PR generation with dynamic values
- 🚀 Automated script execution in PR context
- 📊 Status tracking and visualization
- 🎨 Beautiful console output with color-coded status indicators
- 🏷️ Automatic PR labeling and assignee management

## Installation

### Binary Installation

Download the latest binary for your platform from the [releases page](https://github.com/nsxbet/proliferate/releases).

Available binaries:
- Linux (amd64, arm64)
- macOS (amd64, arm64)

### From Source

```bash
go install github.com/nsxbet/proliferate@latest
```

## Configuration

Create a `config.yaml` file in your working directory:

```yaml
# The name, including owner of a GitHub repository in the format "ownerName/repoName"
repo:
  - nsxbet/proliferate

# Email of the committer. If not set, the global git config setting will be used
author-email: proliferate@proliferate.dev

# Name of the committer. If not set, the global git config setting will be used
author-name: SRE Team
```

### Authentication

Set your GitHub token using one of these methods:
- Environment variable: `GITHUB_TOKEN` or `GHA_PAT`
- Config file: `github-token: your-token-here`

## Usage

### Basic Commands

```bash
# Show pull request status
pro pr status [namespace]

# Apply pull request templates
pro pr apply [template-file] [--dry-run]
```

### Template Example

```yaml
apiVersion: pullrequest.pro.dev/v1alpha1
kind: PullRequest
metadata:
  name: example-pull-request
  namespace: my-namespace
spec:
  repo: github.com/myorg/sample-app
  branch: feature/new-feature
  commitMessage: "feat: add new feature"
  prTitle: "Add New Feature"
  prBody: |
    This PR adds a new feature.
    
    ## Changes
    - Feature implementation
    - Tests
  prLabels:
    - enhancement
    - automated
  prAssignees:
    - username1
    - username2
  scriptsContext:
    environment: staging
    region: us-east-1
  scripts:
    - path/to/script.sh
```

### Value Templating

Proliferate supports Go templating in PR templates. Example:

```yaml
{{- range $app, $values := .Values }}
apiVersion: pullrequest.pro.dev/v1alpha1
kind: PullRequest
metadata:
  name: {{ $app }}
spec:
  repo: {{ $values.repo }}
  branch: "feature/update-{{ $app }}"
  # ... other fields
{{- end }}
```

## Development

### Prerequisites

- Go 1.21+
- Git
- Docker (optional, for containerized development)

### Local Development

```bash
# Build the binary
go build -o pro ./cmd/pro

# Run tests
go test ./...

# Run with Docker
docker-compose run --rm sh
```

## License

This project is licensed under the MIT License
