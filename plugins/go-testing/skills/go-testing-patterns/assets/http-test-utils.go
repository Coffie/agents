// Package httptest provides HTTP testing utilities for Go projects.
// These utilities simplify testing HTTP handlers, middleware, and clients.
package httptest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// RequestBuilder helps construct test HTTP requests.
type RequestBuilder struct {
	t       *testing.T
	method  string
	path    string
	body    io.Reader
	headers map[string]string
	ctx     context.Context
}

// NewRequest creates a new RequestBuilder.
func NewRequest(t *testing.T, method, path string) *RequestBuilder {
	t.Helper()
	return &RequestBuilder{
		t:       t,
		method:  method,
		path:    path,
		headers: make(map[string]string),
		ctx:     context.Background(),
	}
}

// GET creates a GET request builder.
func GET(t *testing.T, path string) *RequestBuilder {
	return NewRequest(t, http.MethodGet, path)
}

// POST creates a POST request builder.
func POST(t *testing.T, path string) *RequestBuilder {
	return NewRequest(t, http.MethodPost, path)
}

// PUT creates a PUT request builder.
func PUT(t *testing.T, path string) *RequestBuilder {
	return NewRequest(t, http.MethodPut, path)
}

// DELETE creates a DELETE request builder.
func DELETE(t *testing.T, path string) *RequestBuilder {
	return NewRequest(t, http.MethodDelete, path)
}

// WithBody sets the request body as a string.
func (rb *RequestBuilder) WithBody(body string) *RequestBuilder {
	rb.body = strings.NewReader(body)
	return rb
}

// WithJSON sets the request body as JSON.
func (rb *RequestBuilder) WithJSON(v any) *RequestBuilder {
	rb.t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		rb.t.Fatalf("failed to marshal JSON body: %v", err)
	}
	rb.body = bytes.NewReader(data)
	rb.headers["Content-Type"] = "application/json"
	return rb
}

// WithHeader adds a header to the request.
func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	rb.headers[key] = value
	return rb
}

// WithAuth adds an Authorization header.
func (rb *RequestBuilder) WithAuth(token string) *RequestBuilder {
	rb.headers["Authorization"] = "Bearer " + token
	return rb
}

// WithBasicAuth adds Basic authentication.
func (rb *RequestBuilder) WithBasicAuth(username, password string) *RequestBuilder {
	rb.t.Helper()
	req := httptest.NewRequest(rb.method, rb.path, nil)
	req.SetBasicAuth(username, password)
	rb.headers["Authorization"] = req.Header.Get("Authorization")
	return rb
}

// WithContext sets the request context.
func (rb *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	rb.ctx = ctx
	return rb
}

// Build creates the http.Request.
func (rb *RequestBuilder) Build() *http.Request {
	rb.t.Helper()
	req := httptest.NewRequest(rb.method, rb.path, rb.body)
	req = req.WithContext(rb.ctx)
	for k, v := range rb.headers {
		req.Header.Set(k, v)
	}
	return req
}

// ResponseRecorder wraps httptest.ResponseRecorder with helper methods.
type ResponseRecorder struct {
	*httptest.ResponseRecorder
	t *testing.T
}

// NewRecorder creates a new ResponseRecorder.
func NewRecorder(t *testing.T) *ResponseRecorder {
	return &ResponseRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		t:                t,
	}
}

// AssertStatus checks the response status code.
func (rr *ResponseRecorder) AssertStatus(want int) *ResponseRecorder {
	rr.t.Helper()
	if rr.Code != want {
		rr.t.Errorf("status = %d, want %d", rr.Code, want)
	}
	return rr
}

// AssertOK asserts status 200 OK.
func (rr *ResponseRecorder) AssertOK() *ResponseRecorder {
	return rr.AssertStatus(http.StatusOK)
}

// AssertCreated asserts status 201 Created.
func (rr *ResponseRecorder) AssertCreated() *ResponseRecorder {
	return rr.AssertStatus(http.StatusCreated)
}

// AssertBadRequest asserts status 400 Bad Request.
func (rr *ResponseRecorder) AssertBadRequest() *ResponseRecorder {
	return rr.AssertStatus(http.StatusBadRequest)
}

// AssertUnauthorized asserts status 401 Unauthorized.
func (rr *ResponseRecorder) AssertUnauthorized() *ResponseRecorder {
	return rr.AssertStatus(http.StatusUnauthorized)
}

// AssertForbidden asserts status 403 Forbidden.
func (rr *ResponseRecorder) AssertForbidden() *ResponseRecorder {
	return rr.AssertStatus(http.StatusForbidden)
}

// AssertNotFound asserts status 404 Not Found.
func (rr *ResponseRecorder) AssertNotFound() *ResponseRecorder {
	return rr.AssertStatus(http.StatusNotFound)
}

// AssertInternalError asserts status 500 Internal Server Error.
func (rr *ResponseRecorder) AssertInternalError() *ResponseRecorder {
	return rr.AssertStatus(http.StatusInternalServerError)
}

// AssertHeader checks a response header value.
func (rr *ResponseRecorder) AssertHeader(key, want string) *ResponseRecorder {
	rr.t.Helper()
	got := rr.Header().Get(key)
	if got != want {
		rr.t.Errorf("header %q = %q, want %q", key, got, want)
	}
	return rr
}

// AssertContentType checks the Content-Type header.
func (rr *ResponseRecorder) AssertContentType(want string) *ResponseRecorder {
	return rr.AssertHeader("Content-Type", want)
}

// AssertJSON asserts Content-Type is application/json.
func (rr *ResponseRecorder) AssertJSON() *ResponseRecorder {
	rr.t.Helper()
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		rr.t.Errorf("Content-Type = %q, want application/json", ct)
	}
	return rr
}

// AssertBodyContains checks if body contains a substring.
func (rr *ResponseRecorder) AssertBodyContains(want string) *ResponseRecorder {
	rr.t.Helper()
	body := rr.Body.String()
	if !strings.Contains(body, want) {
		rr.t.Errorf("body %q does not contain %q", body, want)
	}
	return rr
}

// AssertBodyEquals checks if body equals expected string.
func (rr *ResponseRecorder) AssertBodyEquals(want string) *ResponseRecorder {
	rr.t.Helper()
	body := rr.Body.String()
	if body != want {
		rr.t.Errorf("body = %q, want %q", body, want)
	}
	return rr
}

// AssertBodyJSON unmarshals body and compares to expected value.
func (rr *ResponseRecorder) AssertBodyJSON(want any) *ResponseRecorder {
	rr.t.Helper()

	wantJSON, err := json.Marshal(want)
	if err != nil {
		rr.t.Fatalf("failed to marshal want: %v", err)
	}

	// Normalize both by unmarshaling and remarshaling
	var gotObj, wantObj any
	if err := json.Unmarshal(rr.Body.Bytes(), &gotObj); err != nil {
		rr.t.Fatalf("failed to unmarshal response body: %v", err)
	}
	if err := json.Unmarshal(wantJSON, &wantObj); err != nil {
		rr.t.Fatalf("failed to unmarshal want: %v", err)
	}

	gotNorm, _ := json.Marshal(gotObj)
	wantNorm, _ := json.Marshal(wantObj)

	if string(gotNorm) != string(wantNorm) {
		rr.t.Errorf("JSON body mismatch:\ngot:  %s\nwant: %s", gotNorm, wantNorm)
	}
	return rr
}

// DecodeJSON unmarshals response body into v.
func (rr *ResponseRecorder) DecodeJSON(v any) *ResponseRecorder {
	rr.t.Helper()
	if err := json.Unmarshal(rr.Body.Bytes(), v); err != nil {
		rr.t.Fatalf("failed to decode JSON response: %v", err)
	}
	return rr
}

// HandlerTest simplifies testing http.Handler implementations.
type HandlerTest struct {
	t       *testing.T
	handler http.Handler
}

// NewHandlerTest creates a handler test helper.
func NewHandlerTest(t *testing.T, handler http.Handler) *HandlerTest {
	return &HandlerTest{t: t, handler: handler}
}

// Do executes a request and returns the response recorder.
func (ht *HandlerTest) Do(req *http.Request) *ResponseRecorder {
	ht.t.Helper()
	rr := NewRecorder(ht.t)
	ht.handler.ServeHTTP(rr.ResponseRecorder, req)
	return rr
}

// Get performs a GET request.
func (ht *HandlerTest) Get(path string) *ResponseRecorder {
	return ht.Do(GET(ht.t, path).Build())
}

// PostJSON performs a POST request with JSON body.
func (ht *HandlerTest) PostJSON(path string, body any) *ResponseRecorder {
	return ht.Do(POST(ht.t, path).WithJSON(body).Build())
}

// PutJSON performs a PUT request with JSON body.
func (ht *HandlerTest) PutJSON(path string, body any) *ResponseRecorder {
	return ht.Do(PUT(ht.t, path).WithJSON(body).Build())
}

// Delete performs a DELETE request.
func (ht *HandlerTest) Delete(path string) *ResponseRecorder {
	return ht.Do(DELETE(ht.t, path).Build())
}

// TestServer wraps httptest.Server with convenience methods.
type TestServer struct {
	*httptest.Server
	t *testing.T
}

// NewTestServer creates a test server for the handler.
func NewTestServer(t *testing.T, handler http.Handler) *TestServer {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &TestServer{Server: server, t: t}
}

// NewTLSTestServer creates a TLS test server.
func NewTLSTestServer(t *testing.T, handler http.Handler) *TestServer {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	return &TestServer{Server: server, t: t}
}

// Client returns an HTTP client configured for this test server.
func (ts *TestServer) Client() *http.Client {
	return ts.Server.Client()
}

// URL returns the base URL of the test server.
func (ts *TestServer) URL() string {
	return ts.Server.URL
}

// Example usage in tests:
//
//	func TestUserHandler(t *testing.T) {
//	    handler := NewUserHandler(mockRepo)
//	    ht := NewHandlerTest(t, handler)
//
//	    // Test GET
//	    ht.Get("/users/123").
//	        AssertOK().
//	        AssertJSON().
//	        AssertBodyContains("John")
//
//	    // Test POST
//	    ht.PostJSON("/users", User{Name: "Jane"}).
//	        AssertCreated().
//	        AssertBodyContains("Jane")
//
//	    // Test with custom request
//	    req := POST(t, "/users").
//	        WithJSON(User{Name: "Bob"}).
//	        WithAuth("valid-token").
//	        Build()
//
//	    ht.Do(req).AssertCreated()
//	}
