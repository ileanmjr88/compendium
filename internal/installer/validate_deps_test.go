package installer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/registry"
)

// ---------- fixtures ----------

const validateIndexJSON = `{
	"version": "2",
	"updated_at": "2026-04-11",
	"languages": {
		"clang":  {"latest":"22.1.3","versions":1,"file":"languages/clang.json"},
		"python": {"latest":"3.11.4","versions":1,"file":"languages/python.json"},
		"gcc":    {"latest":"13.2.0","versions":1,"file":"languages/gcc.json"}
	},
	"tools": {}
}`

// clang declares depends on python 3.11
const clangDependsJSON = `{
	"22.1.3": {
		"depends": {"python": "3.11"},
		"linux":  {"amd64": {"url":"u","checksum":"c","size":1,"strip":1}},
		"darwin": {"arm64": {"url":"u","checksum":"c","size":1,"strip":1}}
	}
}`

const pythonNoDepsJSON = `{
	"3.11":   {"linux": {"amd64": {"url":"u","checksum":"c","size":1,"strip":1}}},
	"3.11.4": {"linux": {"amd64": {"url":"u","checksum":"c","size":1,"strip":1}}},
	"3.10":   {"linux": {"amd64": {"url":"u","checksum":"c","size":1,"strip":1}}},
	"3.110":  {"linux": {"amd64": {"url":"u","checksum":"c","size":1,"strip":1}}}
}`

const gccNoDepsJSON = `{
	"13.2.0": {
		"linux":  {"amd64": {"url":"u","checksum":"c","size":1,"strip":1}},
		"darwin": {"arm64": {"url":"u","checksum":"c","size":1,"strip":1}}
	}
}`

// newValidateClient returns a *registry.Client pointed at a test server that
// serves the index plus per-tool files for clang/python/gcc. Clang declares
// depends on python; python and gcc have no depends.
func newValidateClient(t *testing.T) *registry.Client {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(validateIndexJSON))
	})
	mux.HandleFunc("/languages/clang.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(clangDependsJSON))
	})
	mux.HandleFunc("/languages/python.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(pythonNoDepsJSON))
	})
	mux.HandleFunc("/languages/gcc.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(gccNoDepsJSON))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	c, err := registry.NewClient(server.URL + "/index.json")
	if err != nil {
		t.Fatalf("registry.NewClient: %v", err)
	}
	return c
}

// item is a tiny constructor to keep test cases readable.
func item(kind, name, version string) InstallItem {
	return InstallItem{
		Kind:     kind,
		Name:     name,
		Version:  version,
		Platform: "linux",
		Arch:     "amd64",
	}
}

// ---------- ValidateDeps: happy paths ----------

func TestValidateDepsHappyPrefixMatch(t *testing.T) {
	// clang requires python 3.11; user pins python 3.11.4 — satisfies via prefix.
	client := newValidateClient(t)
	items := []InstallItem{
		item("languages", "clang", "22.1.3"),
		item("languages", "python", "3.11.4"),
	}
	cfg := config.Config{Languages: config.Languages{"clang": "22.1.3", "python": "3.11.4"}}

	if err := ValidateDeps(items, client, cfg); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateDepsHappyExactMatch(t *testing.T) {
	// Required 3.11, pinned 3.11 — exact equality.
	client := newValidateClient(t)
	items := []InstallItem{
		item("languages", "clang", "22.1.3"),
		item("languages", "python", "3.11"),
	}
	cfg := config.Config{Languages: config.Languages{"clang": "22.1.3", "python": "3.11"}}

	if err := ValidateDeps(items, client, cfg); err != nil {
		t.Errorf("expected nil for exact match, got %v", err)
	}
}

func TestValidateDepsNoDepends(t *testing.T) {
	// gcc has no depends in the registry — should pass through cleanly.
	client := newValidateClient(t)
	items := []InstallItem{item("languages", "gcc", "13.2.0")}
	cfg := config.Config{Languages: config.Languages{"gcc": "13.2.0"}}

	if err := ValidateDeps(items, client, cfg); err != nil {
		t.Errorf("expected nil for item with no depends, got %v", err)
	}
}

func TestValidateDepsEmptyItems(t *testing.T) {
	client := newValidateClient(t)
	if err := ValidateDeps(nil, client, config.Config{}); err != nil {
		t.Errorf("expected nil for empty items, got %v", err)
	}
}

// ---------- ValidateDeps: error paths ----------

func TestValidateDepsMissing(t *testing.T) {
	// clang requires python; user has no python pinned at all.
	client := newValidateClient(t)
	items := []InstallItem{item("languages", "clang", "22.1.3")}
	cfg := config.Config{Languages: config.Languages{"clang": "22.1.3"}}

	err := ValidateDeps(items, client, cfg)
	if err == nil {
		t.Fatal("expected error when python is not pinned")
	}
	msg := err.Error()
	if !strings.Contains(msg, "python") {
		t.Errorf("error %q should mention 'python'", msg)
	}
	if !strings.Contains(msg, "compendium.toml") {
		t.Errorf("error %q should reference 'compendium.toml'", msg)
	}
}

func TestValidateDepsMismatch(t *testing.T) {
	// clang requires python 3.11, user pins python 3.10.
	client := newValidateClient(t)
	items := []InstallItem{
		item("languages", "clang", "22.1.3"),
		item("languages", "python", "3.10"),
	}
	cfg := config.Config{Languages: config.Languages{"clang": "22.1.3", "python": "3.10"}}

	err := ValidateDeps(items, client, cfg)
	if err == nil {
		t.Fatal("expected error for mismatched python version")
	}
	msg := err.Error()
	if !strings.Contains(msg, "3.11") {
		t.Errorf("error %q should mention required version 3.11", msg)
	}
	if !strings.Contains(msg, "3.10") {
		t.Errorf("error %q should mention pinned version 3.10", msg)
	}
}

func TestValidateDepsPrefixBoundary(t *testing.T) {
	// Required 3.11, pinned 3.110 — must NOT satisfy. This is the trailing-dot
	// trick: without it, naive HasPrefix("3.110", "3.11") would return true.
	client := newValidateClient(t)
	items := []InstallItem{
		item("languages", "clang", "22.1.3"),
		item("languages", "python", "3.110"),
	}
	cfg := config.Config{Languages: config.Languages{"clang": "22.1.3", "python": "3.110"}}

	if err := ValidateDeps(items, client, cfg); err == nil {
		t.Fatal("expected error: 3.110 must not satisfy 3.11")
	}
}

func TestValidateDepsValidatesAllItems(t *testing.T) {
	// gcc (no depends) and clang (depends on python). Even though gcc validates
	// fine, clang's dependency check must still fire.
	client := newValidateClient(t)
	items := []InstallItem{
		item("languages", "gcc", "13.2.0"),
		item("languages", "clang", "22.1.3"),
	}
	cfg := config.Config{Languages: config.Languages{"gcc": "13.2.0", "clang": "22.1.3"}}

	if err := ValidateDeps(items, client, cfg); err == nil {
		t.Fatal("expected error: clang requires python, which isn't pinned")
	}
}

// ---------- satisfies: pure-function table test ----------

func TestSatisfies(t *testing.T) {
	cases := []struct {
		name             string
		pinned, required string
		want             bool
	}{
		{"exact match", "3.11", "3.11", true},
		{"prefix match", "3.11.4", "3.11", true},
		{"deeper prefix match", "3.11.4.1", "3.11", true},
		{"different minor", "3.10", "3.11", false},
		{"prefix boundary - 3.110 must not satisfy 3.11", "3.110", "3.11", false},
		{"required coarser than pinned", "3.11.4", "3", true},
		{"required more specific than pinned", "3.11", "3.11.4", false},
		{"empty pinned", "", "3.11", false},
		{"empty required matched by anything starting with dot - degenerate", "3.11", "", false},
		{"both empty", "", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := satisfies(tc.pinned, tc.required); got != tc.want {
				t.Errorf("satisfies(%q, %q) = %v, want %v", tc.pinned, tc.required, got, tc.want)
			}
		})
	}
}
