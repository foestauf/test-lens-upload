# test-lens-upload

CLI to upload coverage reports to [test-lens](https://github.com/foestauf/test-lens).

## GitHub Action

The easiest way to use this in CI:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write
    steps:
      - uses: actions/checkout@v4

      # ... run your tests with coverage ...

      - name: Upload coverage
        uses: foestauf/test-lens-upload@v1
        with:
          file: coverage/lcov.info
```

### Inputs

| Input | Description | Required |
|-------|-------------|----------|
| `file` | Path to coverage file | No (auto-detected) |
| `api-url` | test-lens API base URL | No (defaults to `https://test-lens-api.foestauf.dev`) |
| `no-oidc` | Skip OIDC token auto-detection | No (defaults to `false`) |

## CLI Usage

### Install

Download a binary from [releases](https://github.com/foestauf/test-lens-upload/releases), or:

```bash
go install github.com/foestauf/test-lens-upload@latest
```

### Commands

```bash
test-lens-upload upload [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--file` | Path to coverage file | Auto-detected |
| `--repo-url` | Git repository URL | Auto-detected from git remote |
| `--commit-sha` | Commit SHA | Auto-detected from git |
| `--branch` | Branch name | Auto-detected from git |
| `--api-url` | test-lens API base URL | `TEST_LENS_API_URL` env var or `https://test-lens-api.foestauf.dev` |
| `--no-oidc` | Skip OIDC token auto-detection | `false` |

### Authentication

In GitHub Actions, the CLI automatically requests an OIDC token. Your workflow needs `id-token: write` permission. Use `--no-oidc` to skip.
