# Development

## Prerequisites

- Go 1.26.6+
- [Task](https://taskfile.dev/) (task runner)
- [Dagger](https://dagger.io/) (CI/CD engine)
- [pre-commit](https://pre-commit.com/)
- Docker (for Redis integration tests)

## Setup

```bash
git clone https://github.com/stuttgart-things/homerun-library.git
cd homerun-library
go mod download
pre-commit install
```

## Running Tests

### Unit tests (no Redis required)

```bash
go test ./...
```

### Integration tests (need a running Redis)

The integration suite is behind a build tag so `go test ./...` stays hermetic.
See "Manual Redis for local integration testing" below for a server to point it at.

```bash
go test -tags=integration ./...
```

### Full Dagger suite with report (starts Redis automatically)

```bash
task test-all
```

### Manual Redis for local integration testing

```bash
docker run -d --name redis-stack-server \
  -p 6379:6379 \
  -e REDIS_ARGS="--requirepass mypassword" \
  redis/redis-stack-server:7.2.0-v18

export REDIS_PASSWORD=mypassword
go run tests/pitcher/pitch_message.go
```

## Linting

The linter set is pinned in `.golangci.yml` so local runs and CI agree.

```bash
task lint                  # golangci-lint directly
task ci-run-static-checks  # the same stage CI runs, via Dagger
```

## Vulnerability scanning

`govulncheck` reports only vulnerabilities that are actually reachable from this
module's code. It is a hard gate in CI: this is a library, so every finding
propagates into all seven homerun2 services that import it.

```bash
task govulncheck
```

## CI/CD

The project uses two GitHub Actions workflows:

| Workflow | Trigger | Purpose |
|---|---|---|
| `dagger-static-checks.yaml` | push to `main`, PR (incl. every push to it) | Lint, unit tests, govulncheck |
| `dagger-tests.yaml` | push to `main`, PR (incl. every push to it) | Integration tests with Redis |

Both carry a concurrency group keyed on the PR, so a new push cancels the run it
supersedes.

## Release

Releases follow [semantic-release](https://semantic-release.gitbook.io/) with the Angular commit convention:

- `feat:` - minor version bump
- `fix:` - patch version bump
- `feat!:` or `BREAKING CHANGE:` - major version bump

```bash
task release
```

## Branch Strategy

- `main` - production branch
- `feature/**` - new features
- `fix/**` - bug fixes
