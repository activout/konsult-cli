package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var version = "dev" // set with -ldflags "-X main.version=..."

const unofficial = "Unofficial command-line interface for the Brainville API v2. Not affiliated with or endorsed by Brainville AB."

type globalOpts struct {
	env, baseURL     string
	timeout          time.Duration
	debug            bool
	asJSON, asNDJSON bool
}

var (
	g     globalOpts
	stdin = bufio.NewReader(os.Stdin)
)

func main() { os.Exit(execute(os.Args[1:])) }

// execute runs the CLI and returns the process exit code (separate from main so tests can call it).
func execute(args []string) int {
	root := newRoot()
	root.SetArgs(args)
	root.SetOut(out)
	root.SetErr(errOut)
	err := root.Execute()
	if err == nil {
		return 0
	}
	var ee *exitError
	if errors.As(err, &ee) {
		if len(ee.body) > 0 && (g.asJSON || g.asNDJSON) {
			var b bytes.Buffer
			if json.Compact(&b, ee.body) == nil {
				fmt.Fprintln(out, b.String())
			}
		}
		fmt.Fprintln(errOut, "konsult:", ee.err)
		return ee.code
	}
	fmt.Fprintln(errOut, "konsult:", err)
	return exitUsage
}

func newRoot() *cobra.Command {
	g = globalOpts{}
	env := os.Getenv("KONSULT_ENV")
	if env == "" {
		env = "production"
	}
	root := &cobra.Command{
		Use:           "konsult",
		Short:         unofficial,
		Long:          unofficial + "\n\n`market` = marketplace assignments (needs Premium Sales, export quota applies).\n`assignments` = your own organisation's assignments.\nDefault environment is production; use --env sandbox for Brainville's test environment.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	f := root.PersistentFlags()
	f.StringVar(&g.env, "env", env, "environment: production (default) or sandbox (test) [KONSULT_ENV]")
	f.StringVar(&g.baseURL, "base-url", "", "developer override of the API base URL (https only)")
	f.DurationVar(&g.timeout, "timeout", 30*time.Second, "request timeout")
	f.BoolVar(&g.debug, "debug", false, "log method, path and status to stderr (never secrets or bodies)")
	f.BoolVar(&g.asJSON, "json", false, "output the service JSON (default when stdout is not a terminal)")
	f.BoolVar(&g.asNDJSON, "ndjson", false, "output one compact JSON record per line")

	root.AddCommand(
		&cobra.Command{Use: "version", Short: "Print version", Run: func(c *cobra.Command, _ []string) { fmt.Fprintln(out, "konsult", version) }},
		authCmd(), configCmd(), marketCmd(), assignmentsCmd(), consultantsCmd(), applicationsCmd(), apiCmd(),
	)
	return root
}

func getClient() (*Client, error) {
	base, err := resolveBase(g.env, g.baseURL)
	if err != nil {
		return nil, err
	}
	c, _, err := loadCreds(strings.ToLower(g.env))
	if err != nil {
		return nil, err
	}
	return newClient(base, c, g.timeout, g.debug), nil
}

// read performs a non-mutating request and prints the result.
func read(method, path string, q url.Values, body any, envKey string, cols []string) ([]byte, error) {
	f, err := pickFormat(g.asJSON, g.asNDJSON)
	if err != nil {
		return nil, err
	}
	c, err := getClient()
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(method, path, q, body)
	if err != nil {
		return nil, err
	}
	b, err := readBody(resp)
	if err != nil {
		return nil, err
	}
	return b, emit(f, b, envKey, cols)
}

type mutFlags struct{ dryRun, yes bool }

func (m *mutFlags) bind(c *cobra.Command) {
	c.Flags().BoolVar(&m.dryRun, "dry-run", false, "show the sanitised request and exit without sending")
	c.Flags().BoolVar(&m.yes, "yes", false, "confirm without prompting (required when not interactive)")
}

type mutation struct {
	method, path string
	q            url.Values
	body         any
	warn         string // consequence shown before confirming
}

func (m *mutFlags) run(mu mutation) error {
	f, err := pickFormat(g.asJSON, g.asNDJSON)
	if err != nil {
		return err
	}
	if m.dryRun {
		var rb any
		if mu.body != nil { // deep copy so redaction never touches the real payload
			j, _ := json.Marshal(mu.body)
			_ = json.Unmarshal(j, &rb)
			rb = redact(rb)
		}
		base, err := resolveBase(g.env, g.baseURL)
		if err != nil {
			return err
		}
		q := url.Values{}
		for k, v := range mu.q {
			for _, s := range v {
				if strings.Contains(s, "@") {
					s = "<redacted>"
				}
				q.Add(k, s)
			}
		}
		u := base + mu.path
		if len(q) > 0 {
			u += "?" + q.Encode()
		}
		var jb bytes.Buffer
		enc := json.NewEncoder(&jb)
		enc.SetEscapeHTML(false) // keep "<redacted>" readable
		_ = enc.Encode(map[string]any{"dryRun": true, "method": mu.method, "url": u, "body": rb, "warning": mu.warn})
		return emit(f, jb.Bytes(), "", []string{"dryRun", "method", "url"})
	}
	if !m.yes {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fail(exitAborted, "refusing to run %s %s non-interactively without --yes. %s", mu.method, mu.path, mu.warn)
		}
		fmt.Fprintf(errOut, "%s %s\n%s\nProceed? [y/N] ", mu.method, mu.path, mu.warn)
		line, _ := stdin.ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
			return fail(exitAborted, "aborted")
		}
	}
	c, err := getClient()
	if err != nil {
		return err
	}
	resp, err := c.Do(mu.method, mu.path, mu.q, mu.body)
	if err != nil {
		return err
	}
	b, err := readBody(resp)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(b)) == 0 || !json.Valid(b) {
		b = []byte(fmt.Sprintf(`{"ok":true,"http":%d}`, resp.StatusCode))
	}
	return emit(f, b, "", []string{"ok", "http"})
}

func idVal(s string) any {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	return s
}

// readJSONFile loads a JSON object from a file ("-" = stdin).
func readJSONFile(path string) (map[string]any, error) {
	var r io.Reader = stdin
	if path != "-" {
		fh, err := os.Open(path)
		if err != nil {
			return nil, fail(exitUsage, "%v", err)
		}
		defer fh.Close()
		r = fh
	}
	var m map[string]any
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return nil, fail(exitUsage, "invalid JSON object in %s: %v", path, err)
	}
	return m, nil
}

// download streams a binary response to a file (created exclusively, 0600) or stdout ("-").
func download(path string, q url.Values, output string, force bool) error {
	if output == "" {
		return fail(exitUsage, "--output is required (use - for stdout)")
	}
	c, err := getClient()
	if err != nil {
		return err
	}
	var w io.Writer = out
	var fh *os.File
	if output != "-" {
		flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
		if force {
			flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		}
		if fh, err = os.OpenFile(output, flags, 0o600); err != nil {
			if errors.Is(err, os.ErrExist) {
				return fail(exitUsage, "%s exists; use --force to overwrite", output)
			}
			return fail(exitUsage, "%v", err)
		}
		w = fh
	}
	resp, err := c.Do("GET", path, q, nil)
	if err != nil {
		if fh != nil {
			fh.Close()
			os.Remove(output)
		}
		return err
	}
	defer resp.Body.Close()
	n, err := io.Copy(w, resp.Body)
	if fh != nil {
		fh.Close()
		if err != nil {
			os.Remove(output)
		} else {
			fmt.Fprintf(errOut, "saved %d bytes to %s (may contain personal data)\n", n, output)
		}
	}
	if err != nil {
		return fail(exitGeneral, "download failed: %v", err)
	}
	return nil
}
