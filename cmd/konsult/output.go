package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/term"
)

var (
	out    io.Writer = os.Stdout
	errOut io.Writer = os.Stderr
)

type format int

const (
	fmtTable format = iota
	fmtJSON
	fmtNDJSON
)

// pickFormat: explicit flags win; otherwise a table on a terminal and JSON when piped.
func pickFormat(asJSON, asNDJSON bool) (format, error) {
	switch {
	case asJSON && asNDJSON:
		return 0, fail(exitUsage, "--json and --ndjson are mutually exclusive")
	case asNDJSON:
		return fmtNDJSON, nil
	case asJSON:
		return fmtJSON, nil
	case term.IsTerminal(int(os.Stdout.Fd())):
		return fmtTable, nil
	}
	return fmtJSON, nil
}

// items splits a response into records: a top-level array, the array under envKey (e.g. market
// "results"), or the object itself as one record. Records stay raw so unknown fields survive.
func items(body []byte, envKey string) ([]json.RawMessage, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, nil
	}
	var arr []json.RawMessage
	if body[0] == '[' {
		return arr, json.Unmarshal(body, &arr)
	}
	if envKey != "" {
		var env map[string]json.RawMessage
		if json.Unmarshal(body, &env) == nil {
			if r, ok := env[envKey]; ok && len(r) > 0 && r[0] == '[' {
				return arr, json.Unmarshal(r, &arr)
			}
		}
	}
	return []json.RawMessage{body}, nil
}

// emit prints a successful response. cols drives the table view; envKey names an envelope array.
func emit(f format, body []byte, envKey string, cols []string) error {
	switch f {
	case fmtJSON:
		var b bytes.Buffer
		if json.Indent(&b, body, "", "  ") != nil {
			return fail(exitGeneral, "service returned invalid JSON")
		}
		fmt.Fprintln(out, b.String())
	case fmtNDJSON:
		recs, err := items(body, envKey)
		if err != nil {
			return fail(exitGeneral, "service returned invalid JSON: %v", err)
		}
		for _, r := range recs {
			var b bytes.Buffer
			if err := json.Compact(&b, r); err != nil {
				return fail(exitGeneral, "service returned invalid JSON: %v", err)
			}
			fmt.Fprintln(out, b.String())
		}
	default:
		recs, err := items(body, envKey)
		if err != nil {
			return fail(exitGeneral, "service returned invalid JSON: %v", err)
		}
		table(recs, cols)
	}
	return nil
}

func table(recs []json.RawMessage, cols []string) {
	if len(recs) == 0 {
		fmt.Fprintln(out, "(no results)")
		return
	}
	if cols == nil { // plain values (e.g. suggested tags)
		for _, r := range recs {
			var v any
			_ = json.Unmarshal(r, &v)
			fmt.Fprintln(out, cell(v))
		}
		return
	}
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, strings.ToUpper(strings.Join(cols, "\t")))
	for _, r := range recs {
		var m map[string]any
		_ = json.Unmarshal(r, &m)
		row := make([]string, len(cols))
		for i, c := range cols {
			row[i] = cell(m[c])
		}
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
}

func cell(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		t = strings.Map(func(r rune) rune {
			if r < 32 || r == 127 {
				return ' ' // strip terminal control chars from remote data
			}
			return r
		}, t)
		if len(t) > 60 {
			t = t[:57] + "..."
		}
		return t
	case float64:
		return fmt.Sprintf("%v", t)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

var redactKeys = map[string]bool{"email": true, "mobile": true, "phone": true, "firstName": true, "lastName": true, "picture": true, "data": true, "reply": true, "linkedInUrl": true}

// redact masks personal and bulky fields in a payload shown by --dry-run.
func redact(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			if redactKeys[k] {
				t[k] = "<redacted>"
			} else {
				t[k] = redact(x)
			}
		}
	case []any:
		for i, x := range t {
			t[i] = redact(x)
		}
	}
	return v
}
