---
name: go-testing-patterns
description: Master Go testing with table-driven tests, benchmarks, fuzzing, httptest, and mocking. Includes testify guidance with clear recommendations on when to use assertions vs standard library patterns. Use when writing, reviewing, or improving Go tests.
---

# Go Testing Patterns

Comprehensive guide to testing in Go using idiomatic patterns, the standard library, and popular testing tools.

## When to Use This Skill

- Writing unit tests for Go code
- Setting up table-driven tests
- Benchmarking Go functions
- Implementing fuzz tests
- Testing HTTP handlers and middleware
- Deciding whether to use testify or standard library
- Mocking dependencies with interfaces, testify/mock, or gomock
- Organizing test files and fixtures

## Core Philosophy

Go's testing philosophy differs from other languages:

| Principle | Go Approach | Other Languages |
|-----------|-------------|-----------------|
| Assertions | `if` + `t.Error` | `assert.Equal()` |
| Test runner | `go test` (built-in) | External frameworks |
| Mocking | Interfaces | Framework-specific |
| Setup/Teardown | `t.Cleanup()`, subtests | `@Before`, `@After` |
| Test discovery | `*_test.go` files | Annotations, conventions |

## Table-Driven Tests

The cornerstone of Go testing. Define test cases as data, iterate over them.

### Basic Pattern

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name string
        a, b int
        want int
    }{
        {"positive numbers", 2, 3, 5},
        {"negative numbers", -1, -2, -3},
        {"zero", 0, 0, 0},
        {"mixed", -1, 5, 4},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Add(tt.a, tt.b)
            if got != tt.want {
                t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
            }
        })
    }
}
```

### With Expected Errors

```go
func TestDivide(t *testing.T) {
    tests := []struct {
        name    string
        a, b    float64
        want    float64
        wantErr bool
    }{
        {"normal division", 10, 2, 5, false},
        {"divide by zero", 10, 0, 0, true},
        {"negative result", -10, 2, -5, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Divide(tt.a, tt.b)

            if (err != nil) != tt.wantErr {
                t.Errorf("Divide(%v, %v) error = %v, wantErr %v", tt.a, tt.b, err, tt.wantErr)
                return
            }

            if !tt.wantErr && got != tt.want {
                t.Errorf("Divide(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
            }
        })
    }
}
```

### Parallel Table Tests

```go
func TestProcess(t *testing.T) {
    tests := []struct {
        name  string
        input string
        want  string
    }{
        {"empty", "", ""},
        {"simple", "hello", "HELLO"},
        {"with spaces", "hello world", "HELLO WORLD"},
    }

    for _, tt := range tests {
        tt := tt // Capture range variable (not needed in Go 1.22+)
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // Run subtests in parallel
            got := Process(tt.input)
            if got != tt.want {
                t.Errorf("Process(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}
```

### Map-Based Tests (Unordered)

Use maps when test order doesn't matter and names are unique:

```go
func TestStatusCodes(t *testing.T) {
    tests := map[string]struct {
        code int
        want string
    }{
        "OK":        {200, "success"},
        "NotFound":  {404, "not found"},
        "ServerErr": {500, "server error"},
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            got := StatusMessage(tt.code)
            if got != tt.want {
                t.Errorf("StatusMessage(%d) = %q, want %q", tt.code, got, tt.want)
            }
        })
    }
}
```

## Test Helpers

Use `t.Helper()` for clean stack traces:

```go
func assertEqual[T comparable](t *testing.T, got, want T) {
    t.Helper() // Marks this as a helper - errors point to caller
    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
}

func assertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func assertError(t *testing.T, err error) {
    t.Helper()
    if err == nil {
        t.Fatal("expected error, got nil")
    }
}

// Usage
func TestSomething(t *testing.T) {
    result, err := DoSomething()
    assertNoError(t, err)
    assertEqual(t, result, "expected")
}
```

## testify/assert - When to Use It

### Installation

```bash
go get github.com/stretchr/testify
```

### When to USE testify/assert

**1. Large teams with mixed language backgrounds**

Developers from Python/Java/JS find assertions familiar:

```go
import "github.com/stretchr/testify/assert"

func TestUser(t *testing.T) {
    user, err := GetUser(123)

    assert.NoError(t, err)
    assert.Equal(t, "John", user.Name)
    assert.Equal(t, 30, user.Age)
    assert.NotNil(t, user.CreatedAt)
}
```

**2. Complex struct comparisons**

`assert.Equal` handles deep comparison with readable diffs:

```go
func TestComplexStruct(t *testing.T) {
    got := BuildConfig()
    want := Config{
        Name: "app",
        Settings: map[string]string{
            "timeout": "30s",
            "retries": "3",
        },
        Enabled: true,
    }

    // Shows detailed diff on failure
    assert.Equal(t, want, got)
}
```

**3. Multiple non-fatal assertions**

`assert` continues after failure (unlike `require`):

```go
func TestMultipleFields(t *testing.T) {
    resp := CallAPI()

    // All assertions run, see all failures at once
    assert.Equal(t, 200, resp.StatusCode)
    assert.Equal(t, "application/json", resp.ContentType)
    assert.NotEmpty(t, resp.Body)
    assert.True(t, resp.Success)
}
```

**4. Readable test output**

testify provides formatted diffs:

```
Error:      Not equal:
            expected: "hello world"
            actual  : "hello word"

            Diff:
            --- Expected
            +++ Actual
            @@ -1 +1 @@
            -hello world
            +hello word
```

### When NOT to USE testify

**1. Small projects or single developers**

Standard library is sufficient:

```go
// This is perfectly fine - no dependency needed
func TestAdd(t *testing.T) {
    got := Add(2, 3)
    if got != 5 {
        t.Errorf("Add(2, 3) = %d, want 5", got)
    }
}
```

**2. Zero-dependency requirements**

Some projects mandate minimal dependencies:

```go
// Standard library only - no external packages
func TestParse(t *testing.T) {
    got, err := Parse("input")
    if err != nil {
        t.Fatalf("Parse() error: %v", err)
    }
    if got != "expected" {
        t.Errorf("Parse() = %q, want %q", got, "expected")
    }
}
```

**3. When you need custom error messages**

Standard library gives full control:

```go
func TestCalculation(t *testing.T) {
    input := ComplexInput{...}
    got := Calculate(input)

    if got != expected {
        t.Errorf("Calculate() with input %+v\ngot:  %v\nwant: %v\nThis might be caused by...",
            input, got, expected)
    }
}
```

**4. Performance-critical test suites**

testify has overhead from reflection:

```go
// Faster - direct comparison
if got != want {
    t.Errorf(...)
}

// Slower - uses reflection
assert.Equal(t, want, got)
```

### testify/require vs assert

- `assert` - Logs failure, test continues
- `require` - Logs failure, test stops immediately

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWithRequire(t *testing.T) {
    user, err := GetUser(123)

    // Use require for preconditions - stop if this fails
    require.NoError(t, err)
    require.NotNil(t, user)

    // Use assert for actual assertions - see all failures
    assert.Equal(t, "John", user.Name)
    assert.Equal(t, 30, user.Age)
}
```

### testify/suite for Complex Setup

Use when tests share significant setup:

```go
import (
    "testing"
    "github.com/stretchr/testify/suite"
)

type UserServiceSuite struct {
    suite.Suite
    db      *sql.DB
    service *UserService
}

func (s *UserServiceSuite) SetupSuite() {
    // Run once before all tests
    s.db = setupTestDB()
}

func (s *UserServiceSuite) TearDownSuite() {
    s.db.Close()
}

func (s *UserServiceSuite) SetupTest() {
    // Run before each test
    s.service = NewUserService(s.db)
    s.db.Exec("DELETE FROM users")
}

func (s *UserServiceSuite) TestCreateUser() {
    user, err := s.service.Create("John", "john@example.com")

    s.NoError(err)
    s.Equal("John", user.Name)
}

func (s *UserServiceSuite) TestGetUser() {
    // Uses clean database from SetupTest
    created, _ := s.service.Create("Jane", "jane@example.com")
    got, err := s.service.Get(created.ID)

    s.NoError(err)
    s.Equal(created.ID, got.ID)
}

func TestUserServiceSuite(t *testing.T) {
    suite.Run(t, new(UserServiceSuite))
}
```

## Mocking

### Interface-Based Mocking (Preferred)

Define interfaces at the consumer, not the provider:

```go
// In your code - define the interface where it's used
type UserRepository interface {
    Get(id int) (*User, error)
    Save(user *User) error
}

type UserService struct {
    repo UserRepository
}

// In tests - implement the interface
type mockUserRepo struct {
    users map[int]*User
    err   error
}

func (m *mockUserRepo) Get(id int) (*User, error) {
    if m.err != nil {
        return nil, m.err
    }
    return m.users[id], nil
}

func (m *mockUserRepo) Save(user *User) error {
    if m.err != nil {
        return m.err
    }
    m.users[user.ID] = user
    return nil
}

func TestUserService_GetUser(t *testing.T) {
    repo := &mockUserRepo{
        users: map[int]*User{
            1: {ID: 1, Name: "John"},
        },
    }
    service := &UserService{repo: repo}

    user, err := service.GetUser(1)

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if user.Name != "John" {
        t.Errorf("got name %q, want %q", user.Name, "John")
    }
}
```

### testify/mock

For complex mocking with call verification:

```go
import "github.com/stretchr/testify/mock"

type MockUserRepo struct {
    mock.Mock
}

func (m *MockUserRepo) Get(id int) (*User, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*User), args.Error(1)
}

func (m *MockUserRepo) Save(user *User) error {
    args := m.Called(user)
    return args.Error(0)
}

func TestWithTestifyMock(t *testing.T) {
    repo := new(MockUserRepo)

    // Set expectations
    repo.On("Get", 1).Return(&User{ID: 1, Name: "John"}, nil)
    repo.On("Save", mock.AnythingOfType("*User")).Return(nil)

    service := &UserService{repo: repo}
    user, _ := service.GetUser(1)

    assert.Equal(t, "John", user.Name)

    // Verify expectations were met
    repo.AssertExpectations(t)
}
```

### gomock (Generated Mocks)

For large interfaces or strict contract verification:

```bash
go install go.uber.org/mock/mockgen@latest
mockgen -source=repository.go -destination=mock_repository_test.go -package=mypackage_test
```

```go
import (
    "go.uber.org/mock/gomock"
)

func TestWithGomock(t *testing.T) {
    ctrl := gomock.NewController(t)

    repo := NewMockUserRepository(ctrl)

    // Strict expectations with order
    gomock.InOrder(
        repo.EXPECT().Get(1).Return(&User{ID: 1, Name: "John"}, nil),
        repo.EXPECT().Save(gomock.Any()).Return(nil),
    )

    service := &UserService{repo: repo}
    // ... test code
}
```

## HTTP Testing

### Testing Handlers with httptest.ResponseRecorder

```go
import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestGetUserHandler(t *testing.T) {
    // Create request
    req := httptest.NewRequest(http.MethodGet, "/users/123", nil)

    // Create response recorder
    w := httptest.NewRecorder()

    // Call handler
    GetUserHandler(w, req)

    // Check response
    resp := w.Result()
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
    }

    body, _ := io.ReadAll(resp.Body)
    if !strings.Contains(string(body), "John") {
        t.Errorf("body = %q, want to contain %q", body, "John")
    }
}
```

### Testing with httptest.Server

For integration tests with real HTTP:

```go
func TestAPIClient(t *testing.T) {
    // Create test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/api/users/1" {
            t.Errorf("unexpected path: %s", r.URL.Path)
        }
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"id": 1, "name": "John"}`))
    }))
    defer server.Close()

    // Use test server URL
    client := NewAPIClient(server.URL)
    user, err := client.GetUser(1)

    if err != nil {
        t.Fatalf("GetUser() error: %v", err)
    }
    if user.Name != "John" {
        t.Errorf("Name = %q, want %q", user.Name, "John")
    }
}
```

### Testing Middleware

```go
func TestAuthMiddleware(t *testing.T) {
    tests := []struct {
        name       string
        authHeader string
        wantStatus int
    }{
        {"valid token", "Bearer valid-token", http.StatusOK},
        {"missing header", "", http.StatusUnauthorized},
        {"invalid token", "Bearer invalid", http.StatusUnauthorized},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create a simple handler wrapped with middleware
            handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
            }))

            req := httptest.NewRequest(http.MethodGet, "/", nil)
            if tt.authHeader != "" {
                req.Header.Set("Authorization", tt.authHeader)
            }

            w := httptest.NewRecorder()
            handler.ServeHTTP(w, req)

            if w.Code != tt.wantStatus {
                t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
            }
        })
    }
}
```

## Benchmarking

### Basic Benchmark

```go
func BenchmarkFibonacci(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Fibonacci(20)
    }
}
```

Run with: `go test -bench=. -benchmem`

### Benchmark with Setup

```go
func BenchmarkProcess(b *testing.B) {
    // Setup - not timed
    data := generateTestData(1000)

    b.ResetTimer() // Start timing here

    for i := 0; i < b.N; i++ {
        Process(data)
    }
}
```

### Sub-Benchmarks

```go
func BenchmarkSort(b *testing.B) {
    sizes := []int{100, 1000, 10000}

    for _, size := range sizes {
        b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
            data := generateData(size)
            b.ResetTimer()

            for i := 0; i < b.N; i++ {
                Sort(data)
            }
        })
    }
}
```

### Comparing Benchmarks

```bash
# Run benchmarks and save
go test -bench=. -count=10 > old.txt

# Make changes, run again
go test -bench=. -count=10 > new.txt

# Compare with benchstat
go install golang.org/x/perf/cmd/benchstat@latest
benchstat old.txt new.txt
```

## Fuzzing (Go 1.18+)

### Basic Fuzz Test

```go
func FuzzParse(f *testing.F) {
    // Add seed corpus
    f.Add("hello")
    f.Add("hello world")
    f.Add("")
    f.Add("special: !@#$%")

    f.Fuzz(func(t *testing.T, input string) {
        result, err := Parse(input)

        // Test invariants
        if err == nil && result == nil {
            t.Error("Parse returned nil result without error")
        }

        // Round-trip test
        if err == nil {
            serialized := result.String()
            reparsed, err2 := Parse(serialized)
            if err2 != nil {
                t.Errorf("round-trip failed: %v", err2)
            }
            if !reflect.DeepEqual(result, reparsed) {
                t.Error("round-trip produced different result")
            }
        }
    })
}
```

Run with: `go test -fuzz=FuzzParse -fuzztime=30s`

### Fuzz with Multiple Inputs

```go
func FuzzJSONRoundTrip(f *testing.F) {
    f.Add("name", 25, true)
    f.Add("", 0, false)

    f.Fuzz(func(t *testing.T, name string, age int, active bool) {
        user := User{Name: name, Age: age, Active: active}

        data, err := json.Marshal(user)
        if err != nil {
            return // Invalid input, skip
        }

        var decoded User
        if err := json.Unmarshal(data, &decoded); err != nil {
            t.Errorf("unmarshal failed: %v", err)
        }

        if decoded != user {
            t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, user)
        }
    })
}
```

## Test Organization

### File Structure

```
mypackage/
├── user.go
├── user_test.go          # Unit tests (package mypackage)
├── user_integration_test.go  # Integration tests with build tag
├── export_test.go        # Export internals for testing
└── testdata/
    ├── golden/
    │   └── user.json
    └── fixtures/
        └── test_users.json
```

### Black-Box Testing

Test from external perspective:

```go
// user_test.go
package mypackage_test  // Note: _test suffix

import (
    "testing"
    "mymodule/mypackage"
)

func TestCreateUser(t *testing.T) {
    // Can only access exported symbols
    user := mypackage.NewUser("John")
    if user.Name() != "John" {
        t.Error("wrong name")
    }
}
```

### Export Internals for Testing

```go
// export_test.go
package mypackage

// Export internal functions for testing
var (
    ValidateEmail = validateEmail
    HashPassword  = hashPassword
)
```

### Build Tags for Integration Tests

```go
//go:build integration

package mypackage_test

func TestDatabaseIntegration(t *testing.T) {
    // Requires real database
    db := connectToTestDB()
    defer db.Close()
    // ...
}
```

Run with: `go test -tags=integration`

### Golden Files

```go
func TestGenerateReport(t *testing.T) {
    got := GenerateReport(testData)

    goldenFile := filepath.Join("testdata", "golden", "report.txt")

    if *update {
        os.WriteFile(goldenFile, []byte(got), 0644)
        return
    }

    want, err := os.ReadFile(goldenFile)
    if err != nil {
        t.Fatalf("failed to read golden file: %v", err)
    }

    if got != string(want) {
        t.Errorf("output mismatch.\ngot:\n%s\nwant:\n%s", got, want)
    }
}

var update = flag.Bool("update", false, "update golden files")
```

Update with: `go test -update`

## Resources

- **references/testify-patterns.md**: Advanced testify patterns
- **references/mocking-strategies.md**: When to use which mocking approach
- **assets/test-helpers.go**: Reusable test helper functions
- **assets/http-test-utils.go**: HTTP testing utilities

## Best Practices Summary

1. **Prefer table-driven tests** for comprehensive coverage
2. **Use `t.Helper()`** in all helper functions
3. **Write clear error messages** with got/want format
4. **Use `t.Parallel()`** when tests are independent
5. **Mock at interface boundaries** not everywhere
6. **Use testify when it improves readability** for your team
7. **Avoid testify for simple comparisons** where `if` is clearer
8. **Keep tests fast** - slow tests don't get run
9. **Test behavior, not implementation** - focus on outcomes
10. **Use `t.Cleanup()`** instead of defer for test cleanup
