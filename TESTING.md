# Testing Guide

This document describes the testing strategy and practices for registry-mirror.

## Test Coverage

Current test coverage: **85%+** (target achieved)

### Coverage by Package

- `internal/storage`: Database operations and policy management
- `internal/registry`: Docker Hub client and authentication
- `internal/cache`: Cache policy enforcement
- `internal/mirror`: Image syncing logic

## Running Tests

### Quick Test
```bash
go test ./...
```

### With Coverage
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### View HTML Coverage Report
```bash
go tool cover -html=coverage.out
```

### Verbose Output
```bash
go test -v ./...
```

### Race Detection (Linux/Mac)
```bash
go test -race ./...
```

### Run Benchmarks
```bash
go test -bench=. -benchmem ./...
```

### Using Makefile
```bash
make test              # Run all tests
make test-coverage     # Run with coverage
make test-verbose      # Verbose output
make bench             # Run benchmarks
```

## Test Structure

### Unit Tests

Unit tests focus on individual functions and methods in isolation.

**Example:**
```go
func TestSetCachePolicy(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    err := db.SetCachePolicy("nginx:latest", 3600, "test")
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
}
```

### Table-Driven Tests

We use table-driven tests for testing multiple scenarios efficiently.

**Example:**
```go
func TestImageParsing(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"official image", "nginx:latest", "library/nginx", false},
        {"user image", "user/app:v1", "user/app", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic here
        })
    }
}
```

### Mock-Based Tests

External dependencies (like Docker Hub API) are mocked using `httptest`.

**Example:**
```go
func TestGetDockerHubToken(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]string{"token": "test-token"})
    }))
    defer server.Close()
    
    // Test with mock server
}
```

### Benchmark Tests

Performance-critical paths have benchmark tests.

**Example:**
```go
func BenchmarkRecordSync(b *testing.B) {
    db := setupBenchDB(b)
    defer db.Close()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        db.RecordSync("nginx:latest", "completed", 1024000, 5.5)
    }
}
```

## Test Isolation

Each test uses an isolated temporary database to prevent interference:

```go
func setupTestDB(t *testing.T) (*DB, func()) {
    tmpDir, _ := os.MkdirTemp("", "registry-mirror-test-*")
    os.Setenv("HOME", tmpDir)
    
    db, _ := NewDB()
    
    cleanup := func() {
        db.Close()
        os.RemoveAll(tmpDir)
    }
    
    return db, cleanup
}
```

## Concurrency Testing

Tests verify thread-safety of concurrent operations:

```go
func TestDatabaseConcurrency(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    done := make(chan bool)
    
    for i := 0; i < 10; i++ {
        go func(id int) {
            db.RecordSync("test:latest", "completed", int64(id), float64(id))
            done <- true
        }(i)
    }
    
    for i := 0; i < 10; i++ {
        <-done
    }
}
```

## Continuous Integration

Tests run automatically on every push via GitHub Actions:

- Unit tests with coverage
- Race condition detection
- Linting
- Build verification
- Coverage threshold enforcement (70% minimum)

See `.github/workflows/test.yml` for CI configuration.

## Writing New Tests

### Guidelines

1. **Test file naming**: `*_test.go`
2. **Test function naming**: `TestFunctionName` or `TestType_Method`
3. **Use table-driven tests** for multiple scenarios
4. **Mock external dependencies** (HTTP, filesystem, etc.)
5. **Clean up resources** using `defer cleanup()`
6. **Test error cases** as well as success cases
7. **Add benchmarks** for performance-critical code

### Example Test Template

```go
func TestNewFeature(t *testing.T) {
    // Setup
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid case", "input1", "output1", false},
        {"error case", "bad", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := NewFeature(tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Test Coverage Goals

- **Overall**: 85%+ ✅
- **Critical paths**: 95%+
- **Error handling**: 100%
- **Public APIs**: 100%

## Running Specific Tests

```bash
# Test a specific package
go test ./internal/storage

# Test a specific function
go test -run TestSetCachePolicy ./internal/storage

# Test with timeout
go test -timeout 30s ./...
```

## Debugging Tests

```bash
# Print test output
go test -v ./...

# Run with race detector
go test -race ./...

# Run with CPU profiling
go test -cpuprofile=cpu.prof ./...

# Run with memory profiling
go test -memprofile=mem.prof ./...
```

## Best Practices

1. ✅ Write tests before or alongside code (TDD)
2. ✅ Keep tests simple and focused
3. ✅ Use descriptive test names
4. ✅ Test edge cases and error conditions
5. ✅ Avoid test interdependencies
6. ✅ Use setup/teardown functions
7. ✅ Mock external dependencies
8. ✅ Measure and maintain coverage
9. ✅ Run tests before committing
10. ✅ Keep tests fast (< 1s per test)
