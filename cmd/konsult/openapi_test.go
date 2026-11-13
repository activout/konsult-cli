package main

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The unofficial spec must parse, tag every operation with its evidence level, and have resolvable $refs.
func TestOpenAPISpec(t *testing.T) {
	raw, err := os.ReadFile("../../openapi/unofficial-v2.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	valid := map[string]bool{"documented": true, "sandbox-verified": true, "inferred": true, "unknown": true, "blocked": true}
	n := 0
	for p, v := range doc["paths"].(map[string]any) {
		for m, o := range v.(map[string]any) {
			n++
			if s, _ := o.(map[string]any)["x-verification-status"].(string); !valid[s] {
				t.Errorf("%s %s: missing or invalid x-verification-status %q", m, p, s)
			}
		}
	}
	if n != 22 { // documented inventory: market 2, assignment 6, files 3, tags 1, applications 4, consultants 6
		t.Errorf("expected 22 documented operations, got %d", n)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if i := strings.Index(line, "$ref: '#/"); i >= 0 {
			ref := strings.Split(strings.Split(line[i:], "'")[1], "/")[1:]
			var node any = doc
			for _, k := range ref {
				node = node.(map[string]any)[k]
				if node == nil {
					t.Fatalf("unresolved $ref %v", ref)
				}
			}
		}
	}
}
