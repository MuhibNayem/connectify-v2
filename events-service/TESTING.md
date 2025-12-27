# Testing Guide - Events Service

## Overview
This service follows **Go testing best practices** with tests colocated alongside source code.

## Test Structure

### Unit Tests
```
internal/service/
├── event_service.go          # Source code
├── event_service_test.go     # Unit tests (same package)
├── mocks/                    # Mock implementations
│   ├── event_repository_mock.go
│   └── event_broadcaster_mock.go
└── testutil/                 # Shared test utilities
    └── builders.go           # Test data builders
```

### Integration Tests
```
integration_tests/
├── repository_test.go        # MongoDB integration tests
├── cache_test.go             # Redis integration tests
└── docker-compose.test.yml   # Test infrastructure
```

## Running Tests

### Unit Tests (Fast)
```bash
# Run all unit tests
go test ./internal/service/... -v

# Run with coverage
go test ./internal/service/... -cover -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out
```

### Benchmark Tests
```bash
# Run benchmarks
go test ./internal/service/... -bench=. -benchmem

# Specific benchmark
go test ./internal/service/... -bench=BenchmarkEventService_CreateEvent
```

### Integration Tests (Require Docker)
```bash
# Start test dependencies
docker-compose -f integration_tests/docker-compose.test.yml up -d

# Run integration tests
go test ./integration_tests/... -tags=integration

# Cleanup
docker-compose -f integration_tests/docker-compose.test.yml down
```

### All Tests
```bash
# Run everything
make test

# Or manually
go test ./... -v
```

## Test Coverage Goals

- **Unit Tests**: 80%+ coverage
- **Critical Paths**: 95%+ coverage (RSVP, Create, Update)
- **Integration Tests**: Happy path + error scenarios

## Test Patterns

### Table-Driven Tests
```go
tests := []struct {
    name      string
    input     Input
    mockSetup func(*mocks.Repository)
    wantErr   bool
}{
    {
        name: "success case",
        // ...
    },
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test implementation
    })
}
```

### Test Builders (Fluent API)
```go
event := testutil.NewEventBuilder().
    WithTitle("My Event").
    WithPrivacy(models.EventPrivacyPublic).
    WithAttendee(userID, models.RSVPStatusGoing).
    Build()
```

### Mock Usage
```go
mockRepo := &mocks.MockEventRepository{
    GetByIDFunc: func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
        return testEvent, nil
    },
}
```

## Why Tests are Colocated

This follows **Go idioms**:

1. ✅ `go test` command expects this structure
2. ✅ Tests can access package-private functions
3. ✅ Easy to find relevant tests
4. ✅ Standard across Go ecosystem (stdlib, major projects)
5. ✅ Refactoring tools understand this pattern

**Reference**: [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)

## CI/CD Integration

Tests run automatically on:
- Pull requests
- Commits to main branch
- Nightly builds (integration tests)

## Writing New Tests

1. Create `*_test.go` file next to source
2. Use table-driven tests for multiple scenarios
3. Mock external dependencies
4. Use test builders for complex fixtures
5. Add benchmarks for performance-critical code
