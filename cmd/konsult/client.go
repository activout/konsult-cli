package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var bases = map[string]string{
	"sandbox":    "https://apisandbox.brainville.com/v2",
	"production": "https://api.brainville.com/v2",
}

// Exit codes (documented in README).
const (
	exitGeneral  = 1
	exitUsage    = 2
	exitAuth     = 3
	exitNotFound = 4
	exitInvalid  = 5
	exitRate     = 6
	exitServer   = 7
	exitAborted  = 8
)

type exitError struct {
	code int
	err  error
	body []byte // raw API error body, printed to stdout in --json/--ndjson mode
}

func (e *exitError) Error() string { return e.err.Error() }

func fail(code int, format string, a ...any) *exitError {
	return &exitError{code: code, err: fmt.Errorf(format, a...)}
}

type Client struct {
	base  string
	auth  string
	hc    *http.Client
	debug bool
}

func newClient(base string, c creds, timeout time.Duration, debug bool) *Client {
	return &Client{
		base:  strings.TrimRight(base, "/"),
		auth:  "Bearer " + base64.StdEncoding.EncodeToString([]byte(c.User+":"+c.Sender)),
		debug: debug,
		hc: &http.Client{
			Timeout: timeout,
			// Never follow redirects: the Authorization header must not leave the configured host.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

// Do performs one request. On status >= 400 the body is consumed and an *exitError is returned.
// On success the caller must close resp.Body. No automatic retries: mutations and market calls
// may be non-idempotent or quota-consuming.
func (c *Client) Do(method, path string, q url.Values, body any) (*http.Response, error) {
	u := c.base + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fail(exitUsage, "encode body: %v", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, u, rdr)
	if err != nil {
		return nil, fail(exitGeneral, "%v", err)
	}
	req.Header.Set("Authorization", c.auth)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if c.debug {
		// Only method, path (no query: may hold names/emails) and status are ever logged.
		st := "error"
		if resp != nil {
			st = resp.Status
		}
		fmt.Fprintf(errOut, "debug: %s %s -> %s\n", method, path, st)
	}
	if err != nil {
		return nil, fail(exitGeneral, "request failed: %v", redactURLErr(err))
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return nil, apiError(resp, b)
	}
	return resp, nil
}

func redactURLErr(err error) error {
	if ue, ok := err.(*url.Error); ok {
		return fmt.Errorf("%s: %v", ue.Op, ue.Err)
	}
	return err
}

type apiErrBody struct {
	Errors []struct {
		ErrorCode int    `json:"errorCode"`
		Message   string `json:"message"`
		Field     string `json:"field"`
	} `json:"errors"`
}

var errorCodeNames = map[int]string{0: "unknown", 1: "not found", 2: "missing value", 3: "invalid value", 4: "duplicate", 5: "not permitted", 6: "incorrect payment details"}

func apiError(resp *http.Response, b []byte) *exitError {
	code := exitGeneral
	switch s := resp.StatusCode; {
	case s == 401 || s == 403:
		code = exitAuth
	case s == 404:
		code = exitNotFound
	case s == 429:
		code = exitRate
	case s >= 500:
		code = exitServer
	case s >= 400:
		code = exitInvalid
	}
	msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
	var eb apiErrBody
	if json.Unmarshal(b, &eb) == nil {
		for _, e := range eb.Errors {
			msg += fmt.Sprintf("; %s (%d)", errorCodeNames[e.ErrorCode], e.ErrorCode)
			if e.Field != "" {
				msg += " field " + e.Field
			}
			if e.Message != "" {
				msg += ": " + e.Message
			}
		}
	}
	if resp.StatusCode == 401 {
		msg += " (bad credentials, or this area needs a licence/role: API Access, Premium Buyer/Sales)"
	}
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		msg += "; Retry-After: " + ra
	}
	e := &exitError{code: code, err: fmt.Errorf("%s", msg)}
	if json.Valid(b) {
		e.body = b
	}
	return e
}

func readBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fail(exitGeneral, "read response: %v", err)
	}
	return b, nil
}
