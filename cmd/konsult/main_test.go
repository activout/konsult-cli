package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// run executes the CLI against a mock server and returns stdout, stderr and the exit code.
func run(t *testing.T, h http.HandlerFunc, args ...string) (string, string, int) {
	t.Helper()
	keyring.MockInit()
	t.Setenv("KONSULT_USER_KEY", "u-key")
	t.Setenv("KONSULT_SENDER_KEY", "s-key")
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	var o, e bytes.Buffer
	out, errOut = &o, &e
	code := execute(append([]string{"--base-url", srv.URL}, args...))
	return o.String(), e.String(), code
}

func TestAuthHeaderAndNDJSON(t *testing.T) {
	var gotAuth string
	o, _, code := run(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`[{"id":1,"extra":{"x":[1,2]}},{"id":2}]`))
	}, "consultants", "list", "--ndjson")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if want := "Bearer " + base64.StdEncoding.EncodeToString([]byte("u-key:s-key")); gotAuth != want {
		t.Fatalf("auth header %q", gotAuth)
	}
	lines := strings.Split(strings.TrimSpace(o), "\n")
	if len(lines) != 2 || lines[0] != `{"id":1,"extra":{"x":[1,2]}}` { // unknown fields preserved
		t.Fatalf("ndjson: %q", o)
	}
}

func TestMarketEnvelopeNDJSON(t *testing.T) {
	o, e, code := run(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/market/search" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`{"results":[{"id":7},{"id":8}],"exportedInPeriod":90,"volume":100,"period":"30d"}`))
	}, "market", "search", "--text", "go", "--ndjson")
	if code != 0 || strings.Count(o, "\n") != 2 || !strings.Contains(e, "WARNING") {
		t.Fatalf("code=%d out=%q err=%q", code, o, e)
	}
}

func TestErrorsMapToExitCodes(t *testing.T) {
	for status, want := range map[int]int{401: exitAuth, 403: exitAuth, 404: exitNotFound, 400: exitInvalid, 429: exitRate, 503: exitServer} {
		o, _, code := run(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(`{"errors":[{"errorCode":3,"message":"bad","field":"title"}]}`))
		}, "assignments", "list", "--json")
		if code != want {
			t.Errorf("status %d: exit %d want %d", status, code, want)
		}
		if !strings.Contains(o, `"errorCode":3`) { // machine-readable error preserved in --json mode
			t.Errorf("status %d: error body missing from stdout: %q", status, o)
		}
	}
}

func TestMutationNeedsYesAndDryRunRedacts(t *testing.T) {
	hit := false
	h := func(w http.ResponseWriter, r *http.Request) { hit = true; w.Write([]byte(`{}`)) }
	f := filepath.Join(t.TempDir(), "c.json")
	os.WriteFile(f, []byte(`{"email":"a@b.se","firstName":"Ann","title":"x"}`), 0o600)

	_, _, code := run(t, h, "consultants", "create", "--file", f)
	if code != exitAborted || hit {
		t.Fatalf("expected refusal without --yes, code=%d hit=%v", code, hit)
	}
	o, _, code := run(t, h, "consultants", "create", "--file", f, "--dry-run", "--json")
	if code != 0 || hit || strings.Contains(o, "a@b.se") || strings.Contains(o, "Ann") || !strings.Contains(o, `"title": "x"`) {
		t.Fatalf("dry-run: code=%d hit=%v out=%s", code, hit, o)
	}
	if _, _, code = run(t, h, "consultants", "create", "--file", f, "--yes"); code != 0 || !hit {
		t.Fatalf("--yes should send, code=%d", code)
	}
}

func TestBoostNeedsAcceptCharge(t *testing.T) {
	hit := false
	h := func(w http.ResponseWriter, r *http.Request) { hit = true }
	if _, _, code := run(t, h, "consultants", "boost", "5", "--yes"); code != exitAborted || hit {
		t.Fatalf("boost without --accept-charge must not send, code=%d", code)
	}
}

func TestBaseURLAndAPIRequestGuards(t *testing.T) {
	keyring.MockInit()
	t.Setenv("KONSULT_USER_KEY", "u")
	t.Setenv("KONSULT_SENDER_KEY", "s")
	var o bytes.Buffer
	out, errOut = &o, &o
	if code := execute([]string{"--base-url", "http://evil.example/v2", "config", "show"}); code != exitUsage {
		t.Errorf("plain http non-loopback must be rejected, code=%d", code)
	}
	for _, p := range []string{"https://evil.example/v2/x", "/v1/x", "/v2/../x", "//evil/v2/x"} {
		if code := execute([]string{"--base-url", "https://127.0.0.1:1", "api", "request", "GET", p}); code != exitUsage {
			t.Errorf("path %q must be rejected, code=%d", p, code)
		}
	}
	if code := execute([]string{"api", "request", "POST", "/v2/consultant"}); code != exitUsage {
		t.Errorf("write methods must be rejected, code=%d", code)
	}
}

func TestDownloadNoOverwriteAndNoPartialOnError(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "cv.pdf")
	ok := func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("%PDF-1.7")) }
	if _, _, code := run(t, ok, "applications", "resume", "1", "-o", dst); code != 0 {
		t.Fatalf("download failed: %d", code)
	}
	if _, _, code := run(t, ok, "applications", "resume", "1", "-o", dst); code != exitUsage {
		t.Fatalf("must refuse to overwrite, code=%d", code)
	}
	bad := filepath.Join(t.TempDir(), "x.pdf")
	run(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }, "applications", "resume", "1", "-o", bad)
	if _, err := os.Stat(bad); err == nil {
		t.Fatal("partial file left behind after error")
	}
}

func TestKeychainRoundTripAndEnvPrecedence(t *testing.T) {
	keyring.MockInit()
	t.Setenv("KONSULT_USER_KEY", "")
	t.Setenv("KONSULT_SENDER_KEY", "")
	if err := saveCreds("sandbox", creds{"ku", "ks"}); err != nil {
		t.Fatal(err)
	}
	c, src, err := loadCreds("sandbox")
	if err != nil || c.User != "ku" || src != "os keychain" {
		t.Fatalf("%v %v %v", c, src, err)
	}
	t.Setenv("KONSULT_USER_KEY", "eu")
	t.Setenv("KONSULT_SENDER_KEY", "es")
	if c, src, _ = loadCreds("sandbox"); c.User != "eu" || src != "environment" {
		t.Fatalf("env must win: %v %v", c, src)
	}
	deleteCreds("sandbox")
	t.Setenv("KONSULT_USER_KEY", "")
	if _, _, err = loadCreds("sandbox"); err == nil {
		t.Fatal("expected not configured after logout")
	}
}

func TestStatusNeverPrintsSecrets(t *testing.T) {
	o, e, _ := run(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`[]`)) }, "auth", "status", "--verify", "--debug", "--json")
	if strings.Contains(o+e, "u-key") || strings.Contains(o+e, "s-key") || strings.Contains(o+e, base64.StdEncoding.EncodeToString([]byte("u-key:s-key"))) {
		t.Fatalf("secret leaked: %s %s", o, e)
	}
	var m map[string]string
	if json.Unmarshal([]byte(o), &m) != nil || m["verified"] != "true" {
		t.Fatalf("bad status output: %s", o)
	}
}

// Sandbox-verified: for these POST/DELETE actions the API reads id (and active) from the QUERY only;
// a body-only id gives 404/400/405 and a body `active` is ignored.
func TestIDActionsUseQueryOnly(t *testing.T) {
	cases := []struct {
		args         []string
		method, path string
		query        string
	}{
		{[]string{"assignments", "close", "7"}, "POST", "/assignment/close", "id=7"},
		{[]string{"assignments", "unpublish", "7"}, "POST", "/assignment/unpublish", "id=7"},
		{[]string{"assignments", "delete", "7"}, "DELETE", "/assignment/delete", "id=7"},
		{[]string{"consultants", "delete", "7"}, "DELETE", "/consultant", "id=7"},
		{[]string{"consultants", "share", "7", "--active=false"}, "POST", "/consultant/share", "active=false&id=7"},
		{[]string{"consultants", "boost", "7", "--active=false"}, "POST", "/consultant/boost", "active=false&id=7"},
	}
	for _, c := range cases {
		var gotQ, gotBody string
		_, _, code := run(t, func(w http.ResponseWriter, r *http.Request) {
			b := new(bytes.Buffer)
			b.ReadFrom(r.Body)
			gotQ, gotBody = r.URL.RawQuery, b.String()
			if r.Method != c.method || r.URL.Path != c.path {
				t.Errorf("%v: got %s %s", c.args, r.Method, r.URL.Path)
			}
		}, append(c.args, "--yes")...)
		if code != 0 || gotQ != c.query || gotBody != "" {
			t.Errorf("%v: code=%d query=%q body=%q", c.args, code, gotQ, gotBody)
		}
	}
}
