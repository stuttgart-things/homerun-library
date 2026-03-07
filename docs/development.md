# Development

## Prerequisites

- Go 1.26.0+
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

### Integration tests via Dagger (starts Redis automatically)

```bash
task run-go-tests
```

### All tests with report

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

```bash
task run-lint-stage
```

## CI/CD

The project uses two GitHub Actions workflows:

| Workflow | Trigger | Purpose |
|---|---|---|
| `dagger-static-checks.yaml` | push, PR | Dagger-based static analysis (lint + unit tests) |
| `dagger-tests.yaml` | push, PR | Dagger-based integration tests with Redis |

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
