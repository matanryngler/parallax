# Local Testing

This directory contains comprehensive integration tests that require external services (API server, PostgreSQL). These tests are not run in CI but are available for local development and validation.

## Running Comprehensive Tests Locally

### Prerequisites

- Docker and docker-compose
- Kind cluster
- kubectl configured

### Setup

1. **Start test infrastructure:**
   ```bash
   cd test/local
   docker-compose -f testdata/docker-compose.yml up -d
   ```

2. **Create Kind cluster:**
   ```bash
   kind create cluster --name comprehensive-test
   ```

3. **Deploy operator:**
   ```bash
   make install
   make deploy IMG=example.com/parallax:v0.0.1
   ```

### Run Tests

```bash
# Run all comprehensive tests
go test ./test/local/ -v -timeout=30m

# Run specific test patterns
go test ./test/local/ --ginkgo.focus "API ListSource Tests" -v
go test ./test/local/ --ginkgo.focus "PostgreSQL ListSource Tests" -v
go test ./test/local/ --ginkgo.focus "Integration Tests" -v
```

### Cleanup

```bash
# Stop infrastructure
docker-compose -f testdata/docker-compose.yml down -v

# Delete cluster
kind delete cluster --name comprehensive-test
```

## Test Coverage

- **API ListSource Tests**: Tests REST API integration with various authentication methods
- **PostgreSQL ListSource Tests**: Tests database integration with connection pooling and queries
- **Integration Tests**: Tests full end-to-end workflows with ListJobs and ListCronJobs

These tests validate the advanced functionality that requires external dependencies and are designed for thorough local validation before contributing changes.