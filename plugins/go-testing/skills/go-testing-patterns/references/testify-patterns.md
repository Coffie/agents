# Advanced testify Patterns

This guide covers advanced usage of testify for Go testing, including when to use each subpackage and common patterns.

## testify Subpackages

| Package | Purpose | When to Use |
|---------|---------|-------------|
| `assert` | Non-fatal assertions | Multiple checks per test, see all failures |
| `require` | Fatal assertions | Preconditions, setup validation |
| `mock` | Mock objects | Complex dependencies with call verification |
| `suite` | Test suites | Shared setup/teardown across tests |

## assert vs require Decision Tree

```
Is this a precondition for the rest of the test?
├── YES → Use require (stops test on failure)
│         Example: require.NoError(t, err)
│
└── NO → Is this one of multiple assertions?
         ├── YES → Use assert (continues on failure)
         │         Example: assert.Equal(t, want, got)
         │
         └── NO → Either works, prefer assert for consistency
```

## Common Assertion Patterns

### Equality

```go
import "github.com/stretchr/testify/assert"

// Basic equality
assert.Equal(t, expected, actual)           // DeepEqual
assert.NotEqual(t, unexpected, actual)

// Exact same object (pointer equality)
assert.Same(t, expectedPtr, actualPtr)
assert.NotSame(t, ptr1, ptr2)

// Approximately equal (floats)
assert.InDelta(t, 3.14, actual, 0.01)       // Within delta
assert.InEpsilon(t, 100, actual, 0.1)       // Within 10%
```

### Nil and Empty

```go
// Nil checks
assert.Nil(t, ptr)
assert.NotNil(t, ptr)

// Empty checks (works with strings, slices, maps, channels)
assert.Empty(t, slice)       // len == 0 or nil
assert.NotEmpty(t, slice)

// Zero value
assert.Zero(t, value)        // zero value for type
assert.NotZero(t, value)
```

### Collections

```go
// Length
assert.Len(t, slice, 5)

// Contains
assert.Contains(t, slice, element)          // slice/array/map/string
assert.NotContains(t, slice, element)

// Subset
assert.Subset(t, superset, subset)

// Element conditions
assert.ElementsMatch(t, expected, actual)   // Same elements, any order
```

### Strings

```go
assert.Equal(t, "hello", s)                 // Exact match
assert.Contains(t, s, "ell")                // Substring
assert.Regexp(t, `^hello.*`, s)             // Regex match
assert.JSONEq(t, expectedJSON, actualJSON)  // JSON equality
```

### Errors

```go
// Error existence
assert.NoError(t, err)
assert.Error(t, err)

// Error matching
assert.ErrorIs(t, err, ErrNotFound)         // errors.Is
assert.ErrorAs(t, err, &target)             // errors.As
assert.ErrorContains(t, err, "not found")   // Message substring

// Specific error type (older pattern)
assert.EqualError(t, err, "exact message")
```

### Panics

```go
assert.Panics(t, func() {
    functionThatPanics()
})

assert.PanicsWithValue(t, "panic message", func() {
    panic("panic message")
})

assert.PanicsWithError(t, "error message", func() {
    panic(errors.New("error message"))
})

assert.NotPanics(t, func() {
    safeFun()
})
```

### Type Assertions

```go
assert.IsType(t, &User{}, result)           // Exact type
assert.Implements(t, (*io.Reader)(nil), r)  // Interface implementation
```

### Conditions

```go
assert.True(t, condition)
assert.False(t, condition)

assert.Greater(t, 5, 3)
assert.GreaterOrEqual(t, 5, 5)
assert.Less(t, 3, 5)
assert.LessOrEqual(t, 5, 5)

assert.Positive(t, 5)
assert.Negative(t, -5)
```

### Time

```go
assert.WithinDuration(t, expected, actual, time.Second)
```

## Mock Patterns

### Basic Mock

```go
import "github.com/stretchr/testify/mock"

type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) Get(id int) (*User, error) {
    args := m.Called(id)
    // Handle nil return
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*User), args.Error(1)
}

func (m *MockRepository) Save(user *User) error {
    args := m.Called(user)
    return args.Error(0)
}
```

### Setting Expectations

```go
func TestService(t *testing.T) {
    repo := new(MockRepository)

    // Return specific values
    repo.On("Get", 1).Return(&User{ID: 1, Name: "John"}, nil)

    // Return error
    repo.On("Get", 999).Return(nil, ErrNotFound)

    // Match any argument
    repo.On("Save", mock.Anything).Return(nil)

    // Match argument type
    repo.On("Save", mock.AnythingOfType("*User")).Return(nil)

    // Match with function
    repo.On("Save", mock.MatchedBy(func(u *User) bool {
        return u.Name != ""
    })).Return(nil)

    // ... test code ...

    // Verify all expectations were met
    repo.AssertExpectations(t)
}
```

### Call Counting

```go
// Expect exactly N calls
repo.On("Get", 1).Return(&User{}, nil).Times(3)

// Expect at least once
repo.On("Get", 1).Return(&User{}, nil).Once()
repo.On("Get", 1).Return(&User{}, nil).Twice()

// Verify specific calls
repo.AssertCalled(t, "Get", 1)
repo.AssertNotCalled(t, "Delete", mock.Anything)
repo.AssertNumberOfCalls(t, "Get", 2)
```

### Run Functions on Call

```go
repo.On("Save", mock.Anything).Run(func(args mock.Arguments) {
    user := args.Get(0).(*User)
    user.ID = 123 // Simulate database assigning ID
}).Return(nil)
```

### Different Returns on Successive Calls

```go
repo.On("Get", 1).Return(nil, ErrNotFound).Once()
repo.On("Get", 1).Return(&User{ID: 1}, nil).Once()

// First call returns error, second returns user
```

## Suite Patterns

### Basic Suite

```go
import (
    "testing"
    "github.com/stretchr/testify/suite"
)

type ServiceTestSuite struct {
    suite.Suite
    service *MyService
    mockDep *MockDependency
}

// Run once before all tests
func (s *ServiceTestSuite) SetupSuite() {
    // One-time setup (e.g., database connection)
}

// Run once after all tests
func (s *ServiceTestSuite) TearDownSuite() {
    // Cleanup
}

// Run before each test
func (s *ServiceTestSuite) SetupTest() {
    s.mockDep = new(MockDependency)
    s.service = NewService(s.mockDep)
}

// Run after each test
func (s *ServiceTestSuite) TearDownTest() {
    // Per-test cleanup
}

// Test methods must start with "Test"
func (s *ServiceTestSuite) TestCreate() {
    s.mockDep.On("Save", mock.Anything).Return(nil)

    err := s.service.Create("test")

    s.NoError(err)
    s.mockDep.AssertExpectations(s.T())
}

func (s *ServiceTestSuite) TestCreateError() {
    s.mockDep.On("Save", mock.Anything).Return(errors.New("db error"))

    err := s.service.Create("test")

    s.Error(err)
    s.Contains(err.Error(), "db error")
}

// Entry point - runs the suite
func TestServiceTestSuite(t *testing.T) {
    suite.Run(t, new(ServiceTestSuite))
}
```

### Suite with Database

```go
type DatabaseTestSuite struct {
    suite.Suite
    db *sql.DB
    tx *sql.Tx
}

func (s *DatabaseTestSuite) SetupSuite() {
    var err error
    s.db, err = sql.Open("postgres", os.Getenv("TEST_DATABASE_URL"))
    s.Require().NoError(err)
}

func (s *DatabaseTestSuite) TearDownSuite() {
    s.db.Close()
}

func (s *DatabaseTestSuite) SetupTest() {
    var err error
    s.tx, err = s.db.Begin()
    s.Require().NoError(err)
}

func (s *DatabaseTestSuite) TearDownTest() {
    s.tx.Rollback() // Rollback to clean state
}

func (s *DatabaseTestSuite) TestInsert() {
    _, err := s.tx.Exec("INSERT INTO users (name) VALUES ($1)", "John")
    s.NoError(err)

    var count int
    s.tx.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
    s.Equal(1, count)
}
```

## Custom Message Pattern

Add context to failures:

```go
// Add message to any assertion
assert.Equal(t, expected, actual, "user ID should match after save")

// Format string
assert.Equal(t, expected, actual, "expected %d items in cart", expectedCount)

// With require
require.NoError(t, err, "setup failed: could not connect to database")
```

## Converting from Standard Library

| Standard Library | testify/assert |
|-----------------|----------------|
| `if got != want { t.Errorf(...) }` | `assert.Equal(t, want, got)` |
| `if err != nil { t.Fatal(err) }` | `require.NoError(t, err)` |
| `if err == nil { t.Fatal("expected error") }` | `assert.Error(t, err)` |
| `if !reflect.DeepEqual(got, want) {...}` | `assert.Equal(t, want, got)` |
| `if len(slice) != 5 {...}` | `assert.Len(t, slice, 5)` |
| `if s == "" {...}` | `assert.Empty(t, s)` |

## When to Avoid testify

1. **Simple comparisons**: `if got != want` is clearer than `assert.Equal`
2. **Custom error messages**: Standard library gives full control
3. **Performance-critical tests**: Reflection has overhead
4. **Minimal dependencies**: Standard library has zero deps
5. **Teaching Go**: Standard patterns are more educational
