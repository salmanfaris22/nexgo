// Package testing provides test utilities for NexGo applications.
package testing

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"
)

// Client is a test client for making requests against NexGo handlers.
type Client struct {
	handler http.Handler
	headers map[string]string
	cookies []*http.Cookie
}

// NewClient creates a test client wrapping an http.Handler.
func NewClient(handler http.Handler) *Client {
	return &Client{
		handler: handler,
		headers: make(map[string]string),
	}
}

// SetHeader adds a default header to all requests.
func (c *Client) SetHeader(key, value string) *Client {
	c.headers[key] = value
	return c
}

// SetCookie adds a cookie to all requests.
func (c *Client) SetCookie(cookie *http.Cookie) *Client {
	c.cookies = append(c.cookies, cookie)
	return c
}

// SetAuth sets a Bearer token header.
func (c *Client) SetAuth(token string) *Client {
	c.headers["Authorization"] = "Bearer " + token
	return c
}

// Response wraps httptest.ResponseRecorder with helper methods.
type Response struct {
	*httptest.ResponseRecorder
}

// StatusCode returns the response status code.
func (r *Response) StatusCode() int {
	return r.Code
}

// BodyString returns the response body as a string.
func (r *Response) BodyString() string {
	return r.Body.String()
}

// BodyJSON decodes the response body into v.
func (r *Response) BodyJSON(v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// BodyMap decodes the response body as a map.
func (r *Response) BodyMap() map[string]interface{} {
	var m map[string]interface{}
	json.NewDecoder(bytes.NewReader(r.Body.Bytes())).Decode(&m)
	return m
}

// Header returns a response header value.
func (r *Response) Header(key string) string {
	return r.HeaderMap.Get(key)
}

// HasHeader checks if a response header exists.
func (r *Response) HasHeader(key string) bool {
	return r.HeaderMap.Get(key) != ""
}

// IsOK checks if status is 200.
func (r *Response) IsOK() bool {
	return r.Code == http.StatusOK
}

// IsRedirect checks if status is 3xx.
func (r *Response) IsRedirect() bool {
	return r.Code >= 300 && r.Code < 400
}

// IsJSON checks if Content-Type is application/json.
func (r *Response) IsJSON() bool {
	return strings.Contains(r.HeaderMap.Get("Content-Type"), "application/json")
}

// IsHTML checks if Content-Type is text/html.
func (r *Response) IsHTML() bool {
	return strings.Contains(r.HeaderMap.Get("Content-Type"), "text/html")
}

// ContainsString checks if the body contains a substring.
func (r *Response) ContainsString(s string) bool {
	return strings.Contains(r.Body.String(), s)
}

// --- Request builders ---

// GET sends a GET request.
func (c *Client) GET(path string) *Response {
	return c.do("GET", path, nil)
}

// POST sends a POST request with JSON body.
func (c *Client) POST(path string, body interface{}) *Response {
	var reader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reader = bytes.NewReader(data)
	}
	return c.doWithContentType("POST", path, reader, "application/json")
}

// PostForm sends a POST request with form data.
func (c *Client) PostForm(path string, values url.Values) *Response {
	return c.doWithContentType("POST", path,
		strings.NewReader(values.Encode()),
		"application/x-www-form-urlencoded")
}

// PUT sends a PUT request with JSON body.
func (c *Client) PUT(path string, body interface{}) *Response {
	var reader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reader = bytes.NewReader(data)
	}
	return c.doWithContentType("PUT", path, reader, "application/json")
}

// PATCH sends a PATCH request with JSON body.
func (c *Client) PATCH(path string, body interface{}) *Response {
	var reader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reader = bytes.NewReader(data)
	}
	return c.doWithContentType("PATCH", path, reader, "application/json")
}

// DELETE sends a DELETE request.
func (c *Client) DELETE(path string) *Response {
	return c.do("DELETE", path, nil)
}

func (c *Client) do(method, path string, body io.Reader) *Response {
	return c.doWithContentType(method, path, body, "")
}

func (c *Client) doWithContentType(method, path string, body io.Reader, contentType string) *Response {
	req := httptest.NewRequest(method, path, body)
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	w := httptest.NewRecorder()
	c.handler.ServeHTTP(w, req)

	// Save response cookies
	for _, cookie := range w.Result().Cookies() {
		c.cookies = append(c.cookies, cookie)
	}

	return &Response{ResponseRecorder: w}
}

// --- Route testing ---

// RouteTest helps test individual route handlers.
type RouteTest struct {
	handler http.HandlerFunc
}

// NewRouteTest creates a route test for a handler.
func NewRouteTest(handler http.HandlerFunc) *RouteTest {
	return &RouteTest{handler: handler}
}

// GET sends a GET request to the handler.
func (rt *RouteTest) GET(path string) *Response {
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	rt.handler(w, req)
	return &Response{ResponseRecorder: w}
}

// POST sends a POST with JSON body to the handler.
func (rt *RouteTest) POST(path string, body interface{}) *Response {
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rt.handler(w, req)
	return &Response{ResponseRecorder: w}
}

// --- Assertion helpers ---

// AssertStatus checks the response status code.
func AssertStatus(resp *Response, expected int) error {
	if resp.Code != expected {
		return &AssertionError{
			Expected: expected,
			Actual:   resp.Code,
			Message:  "unexpected status code",
		}
	}
	return nil
}

// AssertBodyContains checks if the body contains a string.
func AssertBodyContains(resp *Response, substr string) error {
	if !resp.ContainsString(substr) {
		return &AssertionError{
			Message: "body does not contain: " + substr,
		}
	}
	return nil
}

// AssertJSON checks that a JSON field has an expected value.
func AssertJSON(resp *Response, field string, expected interface{}) error {
	m := resp.BodyMap()
	actual, ok := m[field]
	if !ok {
		return &AssertionError{Message: "field not found: " + field}
	}
	if actual != expected {
		return &AssertionError{
			Expected: expected,
			Actual:   actual,
			Message:  "JSON field mismatch: " + field,
		}
	}
	return nil
}

// AssertionError is a test assertion failure.
type AssertionError struct {
	Expected interface{}
	Actual   interface{}
	Message  string
}

func (e *AssertionError) Error() string {
	if e.Expected != nil {
		return strings.Join([]string{e.Message, ": expected ", toString(e.Expected), " got ", toString(e.Actual)}, "")
	}
	return e.Message
}

func toString(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	data, _ := json.Marshal(v)
	return string(data)
}

// --- Benchmark helpers ---

// BenchmarkHandler runs n requests against a handler and returns avg duration.
func BenchmarkHandler(handler http.HandlerFunc, method, path string, n int) BenchmarkResult {
	var totalNs int64
	for i := 0; i < n; i++ {
		req := httptest.NewRequest(method, path, nil)
		w := httptest.NewRecorder()
		start := nanotime()
		handler(w, req)
		totalNs += nanotime() - start
	}
	avgNs := totalNs / int64(n)
	return BenchmarkResult{
		Requests:  n,
		TotalNs:   totalNs,
		AvgNs:     avgNs,
		ReqPerSec: float64(n) / (float64(totalNs) / 1e9),
	}
}

// BenchmarkResult holds benchmark stats.
type BenchmarkResult struct {
	Requests  int
	TotalNs   int64
	AvgNs     int64
	ReqPerSec float64
}

func nanotime() int64 {
	return time.Now().UnixNano()
}
