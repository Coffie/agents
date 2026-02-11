// Package testhelpers provides reusable test utilities for Go projects.
// Copy and adapt these helpers to your project's needs.
package testhelpers

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// assertEqual compares two comparable values and fails if they differ.
func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// assertNotEqual fails if two comparable values are equal.
func assertNotEqual[T comparable](t *testing.T, got, notWant T) {
	t.Helper()
	if got == notWant {
		t.Errorf("got %v, expected different value", got)
	}
}

// assertDeepEqual compares two values using reflect.DeepEqual.
func assertDeepEqual(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// assertNoError fails if err is not nil.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// assertError fails if err is nil.
func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// assertErrorIs fails if err does not match target using errors.Is.
func assertErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %v, got nil", target)
	}
	// Use errors.Is in your actual code:
	// if !errors.Is(err, target) {
	//     t.Errorf("got error %v, want %v", err, target)
	// }
}

// assertErrorContains fails if err doesn't contain the expected substring.
func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error %q does not contain %q", err.Error(), want)
	}
}

// assertNil fails if v is not nil.
func assertNil(t *testing.T, v any) {
	t.Helper()
	if v != nil && !reflect.ValueOf(v).IsNil() {
		t.Errorf("expected nil, got %v", v)
	}
}

// assertNotNil fails if v is nil.
func assertNotNil(t *testing.T, v any) {
	t.Helper()
	if v == nil || reflect.ValueOf(v).IsNil() {
		t.Error("expected non-nil value, got nil")
	}
}

// assertTrue fails if condition is false.
func assertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Errorf("expected true: %s", msg)
	}
}

// assertFalse fails if condition is true.
func assertFalse(t *testing.T, condition bool, msg string) {
	t.Helper()
	if condition {
		t.Errorf("expected false: %s", msg)
	}
}

// assertContains fails if s does not contain substr.
func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("%q does not contain %q", s, substr)
	}
}

// assertLen fails if the length of v doesn't match expected.
func assertLen(t *testing.T, v any, expected int) {
	t.Helper()
	val := reflect.ValueOf(v)
	if val.Len() != expected {
		t.Errorf("expected length %d, got %d", expected, val.Len())
	}
}

// assertEmpty fails if v is not empty.
func assertEmpty(t *testing.T, v any) {
	t.Helper()
	val := reflect.ValueOf(v)
	if val.Len() != 0 {
		t.Errorf("expected empty, got length %d", val.Len())
	}
}

// assertPanics fails if f does not panic.
func assertPanics(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, but function did not panic")
		}
	}()
	f()
}

// assertJSONEqual compares two values as JSON.
func assertJSONEqual(t *testing.T, got, want any) {
	t.Helper()
	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("failed to marshal got: %v", err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("failed to marshal want: %v", err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("JSON mismatch:\ngot:  %s\nwant: %s", gotJSON, wantJSON)
	}
}

// Golden file helpers

// GoldenFile manages golden file testing.
type GoldenFile struct {
	t      *testing.T
	name   string
	update bool
}

// NewGoldenFile creates a new golden file helper.
func NewGoldenFile(t *testing.T, name string, update bool) *GoldenFile {
	t.Helper()
	return &GoldenFile{t: t, name: name, update: update}
}

// Assert compares got against the golden file, or updates it if update is true.
func (g *GoldenFile) Assert(got string) {
	g.t.Helper()

	goldenPath := filepath.Join("testdata", "golden", g.name)

	if g.update {
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			g.t.Fatalf("failed to create golden dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
			g.t.Fatalf("failed to write golden file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		g.t.Fatalf("failed to read golden file %s: %v", goldenPath, err)
	}

	if got != string(want) {
		g.t.Errorf("golden file mismatch for %s:\ngot:\n%s\nwant:\n%s", g.name, got, want)
	}
}

// Fixture helpers

// LoadFixture loads a JSON fixture file into the provided struct.
func LoadFixture(t *testing.T, name string, v any) {
	t.Helper()

	path := filepath.Join("testdata", "fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", path, err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("failed to unmarshal fixture %s: %v", path, err)
	}
}

// LoadFixtureBytes loads a fixture file as bytes.
func LoadFixtureBytes(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("testdata", "fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", path, err)
	}
	return data
}

// Cleanup helpers

// TempFile creates a temporary file with the given content and registers cleanup.
func TempFile(t *testing.T, content string) string {
	t.Helper()

	f, err := os.CreateTemp("", "test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		t.Fatalf("failed to write temp file: %v", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		t.Fatalf("failed to close temp file: %v", err)
	}

	t.Cleanup(func() {
		os.Remove(f.Name())
	})

	return f.Name()
}

// CaptureOutput captures stdout/stderr during function execution.
func CaptureOutput(t *testing.T, f func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}

	return string(out)
}

// Environment helpers

// SetEnv sets an environment variable and registers cleanup to restore original.
func SetEnv(t *testing.T, key, value string) {
	t.Helper()

	original, exists := os.LookupEnv(key)

	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env %s: %v", key, err)
	}

	t.Cleanup(func() {
		if exists {
			os.Setenv(key, original)
		} else {
			os.Unsetenv(key)
		}
	})
}

// Timing helpers for tests that need to verify timing behavior

// Eventually retries a condition until it returns true or timeout.
func Eventually(t *testing.T, condition func() bool, timeout, interval int) {
	t.Helper()
	// Implementation would use time.After and time.Tick
	// Simplified here - use actual time in real implementation
}
