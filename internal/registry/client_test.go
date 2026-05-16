package registry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// ---------- fixtures ----------
//
// JSON is kept as raw strings so tests exercise the exact wire format the
// real registry serves, including the depends-as-sibling-of-platforms shape
// that VersionEntry.UnmarshalJSON splits apart.

const indexJSON = `{
  "version": "2",
  "updated_at": "2026-04-11",
  "languages": {
    "go":    {"latest":"1.24.1","versions":1,"file":"languages/go.json"},
    "clang": {"latest":"22.1.3","versions":1,"file":"languages/clang.json"}
  },
  "tools": {
    "cmake": {"latest":"3.28.0","versions":1,"file":"tools/cmake.json"}
  }
}`

const goToolJSON = `{
  "1.24.1": {
    "linux": {
      "amd64": {"url":"https://example.com/go-1.24.1-linux-amd64.tar.gz","checksum":"sha256:goamd64","size":73635665,"strip":1},
      "arm64": {"url":"https://example.com/go-1.24.1-linux-arm64.tar.gz","checksum":"sha256:goarm64","size":71045848,"strip":1}
    }
  }
}`

const clangToolJSON = `{
  "22.1.3": {
    "depends": {"python":"3.11"},
    "linux": {
      "amd64": {"url":"https://example.com/clang-22.1.3-linux-amd64.tar.xz","checksum":"sha256:clang","size":1939973900,"strip":1}
    }
  }
}`

const cmakeToolJSON = `{
  "3.28.0": {
    "linux": {
      "amd64": {"url":"https://example.com/cmake-3.28.0-linux-amd64.tar.gz","checksum":"sha256:cmake","size":52345678,"strip":1}
    }
  }
}`

// testServer wraps httptest.Server with a per-path call counter so tests
// can assert on cache hits vs lazy fetches.
type testServer struct {
	*httptest.Server
	counts map[string]*atomic.Int32
}

func (s *testServer) hits(path string) int32 {
	if c, ok := s.counts[path]; ok {
		return c.Load()
	}
	return 0
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	counts := map[string]*atomic.Int32{
		"/index.json":           {},
		"/languages/go.json":    {},
		"/languages/clang.json": {},
		"/tools/cmake.json":     {},
	}

	mux := http.NewServeMux()
	register := func(path, body string) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			counts[path].Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		})
	}
	register("/index.json", indexJSON)
	register("/languages/go.json", goToolJSON)
	register("/languages/clang.json", clangToolJSON)
	register("/tools/cmake.json", cmakeToolJSON)

	return &testServer{
		Server: httptest.NewServer(mux),
		counts: counts,
	}
}

// ---------- ResolveSource ----------

func TestResolveSource(t *testing.T) {
	got := ResolveSource("public")
	if got != PublicIndexURL {
		t.Errorf(`ResolveSource("public") = %q, want %q`, got, PublicIndexURL)
	}
}

// ---------- FetchIndex ----------

func TestFetchIndex(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	idx, err := FetchIndex(s.URL + "/index.json")
	if err != nil {
		t.Fatalf("FetchIndex error: %v", err)
	}
	if idx.Version != "2" {
		t.Errorf("Version = %q, want %q", idx.Version, "2")
	}
	if got, want := len(idx.Languages), 2; got != want {
		t.Errorf("Languages count = %d, want %d", got, want)
	}
	if got, want := idx.Languages["clang"].File, "languages/clang.json"; got != want {
		t.Errorf("Languages[clang].File = %q, want %q", got, want)
	}
	if got, want := idx.Tools["cmake"].Latest, "3.28.0"; got != want {
		t.Errorf("Tools[cmake].Latest = %q, want %q", got, want)
	}
}

func TestFetchIndexBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	if _, err := FetchIndex(server.URL); err == nil {
		t.Fatal("expected error for 404 status, got nil")
	}
}

func TestFetchIndexBadURL(t *testing.T) {
	if _, err := FetchIndex("http://localhost:0/nope"); err == nil {
		t.Fatal("expected error for unreachable URL, got nil")
	}
}

// ---------- Client.Lookup: happy path + lazy fetch + cache ----------

func TestClientLookupLazyFetch(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	c, err := NewClient(s.URL + "/index.json")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	// NewClient fetched the index but should NOT have touched any sub-file.
	if got := s.hits("/languages/go.json"); got != 0 {
		t.Fatalf("go.json fetched before any Lookup (count=%d)", got)
	}

	// First Lookup triggers a sub-file fetch.
	art, deps, err := c.Lookup("languages", "go", "1.24.1", "linux", "amd64")
	if err != nil {
		t.Fatalf("first Lookup error: %v", err)
	}
	if want := "https://example.com/go-1.24.1-linux-amd64.tar.gz"; art.URL != want {
		t.Errorf("Artifact.URL = %q, want %q", art.URL, want)
	}
	if deps != nil {
		t.Errorf("expected nil Depends for go, got %v", deps)
	}
	if got := s.hits("/languages/go.json"); got != 1 {
		t.Errorf("after first Lookup, /languages/go.json hits = %d, want 1", got)
	}

	// Second Lookup of same tool reuses the cached ToolFile — no new fetch.
	if _, _, err := c.Lookup("languages", "go", "1.24.1", "linux", "arm64"); err != nil {
		t.Fatalf("second Lookup error: %v", err)
	}
	if got := s.hits("/languages/go.json"); got != 1 {
		t.Errorf("after second Lookup, /languages/go.json hits = %d, want 1 (cache miss)", got)
	}
}

func TestClientLookupAcrossTools(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	c, err := NewClient(s.URL + "/index.json")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if _, _, err := c.Lookup("languages", "go", "1.24.1", "linux", "amd64"); err != nil {
		t.Fatalf("Lookup go: %v", err)
	}
	if _, _, err := c.Lookup("languages", "clang", "22.1.3", "linux", "amd64"); err != nil {
		t.Fatalf("Lookup clang: %v", err)
	}
	if _, _, err := c.Lookup("tools", "cmake", "3.28.0", "linux", "amd64"); err != nil {
		t.Fatalf("Lookup cmake: %v", err)
	}

	for _, p := range []string{"/languages/go.json", "/languages/clang.json", "/tools/cmake.json"} {
		if got := s.hits(p); got != 1 {
			t.Errorf("%s hits = %d, want 1", p, got)
		}
	}
}

func TestClientLookupReturnsDepends(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	c, err := NewClient(s.URL + "/index.json")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, deps, err := c.Lookup("languages", "clang", "22.1.3", "linux", "amd64")
	if err != nil {
		t.Fatalf("Lookup error: %v", err)
	}
	if got, want := deps["python"], "3.11"; got != want {
		t.Errorf("Depends[python] = %q, want %q", got, want)
	}
}

// ---------- Client.Lookup: error paths ----------

func TestClientLookupUnknownInIndex(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	c, err := NewClient(s.URL + "/index.json")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if _, _, err := c.Lookup("tools", "does-not-exist", "1.0.0", "linux", "amd64"); err == nil {
		t.Fatal("expected error for unknown tool, got nil")
	}

	// Unknown name must short-circuit before any sub-file fetch.
	for _, p := range []string{"/languages/go.json", "/languages/clang.json", "/tools/cmake.json"} {
		if got := s.hits(p); got != 0 {
			t.Errorf("unexpected sub-file fetch on unknown lookup: %s hits = %d", p, got)
		}
	}
}

func TestClientLookupUnknownKind(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	c, err := NewClient(s.URL + "/index.json")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if _, _, err := c.Lookup("frameworks", "go", "1.24.1", "linux", "amd64"); err == nil {
		t.Fatal("expected error for unknown kind, got nil")
	}
}

func TestClientLookupUnknownVersion(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	c, err := NewClient(s.URL + "/index.json")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if _, _, err := c.Lookup("languages", "go", "99.0.0", "linux", "amd64"); err == nil {
		t.Fatal("expected error for unknown version, got nil")
	}
}

func TestClientLookupUnknownPlatform(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	c, err := NewClient(s.URL + "/index.json")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if _, _, err := c.Lookup("languages", "go", "1.24.1", "windows", "amd64"); err == nil {
		t.Fatal("expected error for unknown platform, got nil")
	}
}

func TestClientLookupUnknownArch(t *testing.T) {
	s := newTestServer(t)
	defer s.Close()

	c, err := NewClient(s.URL + "/index.json")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	if _, _, err := c.Lookup("languages", "clang", "22.1.3", "linux", "arm64"); err == nil {
		t.Fatal("expected error for unknown arch, got nil")
	}
}

func TestClientLookupMissingSubFile(t *testing.T) {
	// Index references languages/go.json but the server returns 404 for it.
	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(indexJSON))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c, err := NewClient(server.URL + "/index.json")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, _, err = c.Lookup("languages", "go", "1.24.1", "linux", "amd64")
	if err == nil {
		t.Fatal("expected error when sub-file is missing, got nil")
	}
	if !strings.Contains(err.Error(), "languages/go") {
		t.Errorf("error %q should name the failing kind/name", err.Error())
	}
}

// ---------- VersionEntry custom unmarshaller ----------

func TestVersionEntryDependsParsed(t *testing.T) {
	body := []byte(`{
		"depends": {"python": "3.11"},
		"linux":   {"amd64": {"url":"u","checksum":"c","size":1,"strip":1}},
		"darwin":  {"arm64": {"url":"u2","checksum":"c2","size":2,"strip":1}}
	}`)

	var v VersionEntry
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, want := v.Depends["python"], "3.11"; got != want {
		t.Errorf("Depends[python] = %q, want %q", got, want)
	}
	if got := len(v.Platforms); got != 2 {
		t.Errorf("Platforms count = %d, want 2", got)
	}
	if v.Platforms["linux"]["amd64"].URL != "u" {
		t.Errorf("Platforms.linux.amd64.URL = %q, want %q", v.Platforms["linux"]["amd64"].URL, "u")
	}
	if _, ok := v.Platforms["depends"]; ok {
		t.Errorf("'depends' must not appear as a platform key")
	}
}

func TestVersionEntryNoDepends(t *testing.T) {
	body := []byte(`{
		"linux": {"amd64": {"url":"u","checksum":"c","size":1,"strip":1}}
	}`)

	var v VersionEntry
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v.Depends != nil {
		t.Errorf("expected nil Depends, got %v", v.Depends)
	}
	if got := len(v.Platforms); got != 1 {
		t.Errorf("Platforms count = %d, want 1", got)
	}
}

// ---------- subFileURL ----------

func TestSubFileURLResolution(t *testing.T) {
	cases := []struct {
		name      string
		indexURL  string
		relFile   string
		want      string
		wantError bool
	}{
		{
			name:     "github raw",
			indexURL: "https://raw.githubusercontent.com/x/y/main/index.json",
			relFile:  "languages/clang.json",
			want:     "https://raw.githubusercontent.com/x/y/main/languages/clang.json",
		},
		{
			name:     "localhost testserver",
			indexURL: "http://localhost:7495/index.json",
			relFile:  "tools/cmake.json",
			want:     "http://localhost:7495/tools/cmake.json",
		},
		{
			name:     "strips query and fragment",
			indexURL: "https://x/y/index.json?ref=main#section",
			relFile:  "tools/ninja.json",
			want:     "https://x/y/tools/ninja.json",
		},
		{
			name:      "rejects empty relFile",
			indexURL:  "https://x/y/index.json",
			relFile:   "",
			wantError: true,
		},
		{
			name:      "rejects absolute relFile",
			indexURL:  "https://x/y/index.json",
			relFile:   "/etc/passwd",
			wantError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := subFileURL(tc.indexURL, tc.relFile)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------- compareVersions ----------

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want int // -1 = a<b, 0 = equal, +1 = a>b (sign only, not exact value)
	}{
		{"equal", "1.2.3", "1.2.3", 0},
		{"plain semver ascending", "1.2.2", "1.2.3", -1},
		{"plain semver descending", "1.2.3", "1.2.2", +1},

		// The case that breaks lexicographic sort.
		{"multi-digit numeric compare", "13.2.0", "13.10.0", -1},
		{"multi-digit numeric compare reversed", "1.11.10", "1.11.2", +1},

		// Missing trailing components default to 0; string tiebreak handles equality.
		{"missing component vs longer", "1.10", "1.10.1", -1},
		{"numerically equal — string tiebreak", "1.10", "1.10.0", -1},

		// Hyphen-suffixed builds (gcc, openocd style).
		{"hyphen build patch", "12.2.0-1", "12.2.0-2", -1},
		{"base release vs patched", "12.2.0", "12.2.0-1", -1},

		// Multi-hyphen (arm-none-eabi-gcc style: "12.3.1-1.1").
		{"multi-hyphen ascending", "12.2.1-1.1", "12.2.1-1.2", -1},
		{"multi-hyphen across base", "12.2.1-1.9", "12.3.0-1.1", -1},

		// Date-format versions (vcpkg).
		{"date year boundary", "2024-12-09", "2025-01-11", -1},
		{"date same year", "2025-01-29", "2025-03-13", -1},

		// Non-numeric components Atoi-fail → 0 → tiebreak on string compare.
		{"alpha tag falls back to string compare", "1.0.0-beta", "1.0.0-alpha", +1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compareVersions(tc.a, tc.b)
			if sign(got) != tc.want {
				t.Errorf("compareVersions(%q, %q) = %d, want sign %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return +1
	default:
		return 0
	}
}

// ---------- GITHUB_TOKEN propagation ----------

func TestFetchUsesGitHubToken(t *testing.T) {
	var indexAuth, subAuth string

	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		indexAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(indexJSON))
	})
	mux.HandleFunc("/languages/go.json", func(w http.ResponseWriter, r *http.Request) {
		subAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(goToolJSON))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "shhhh-secret")

	c, err := NewClient(server.URL + "/index.json")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	if _, _, err := c.Lookup("languages", "go", "1.24.1", "linux", "amd64"); err != nil {
		t.Fatalf("Lookup error: %v", err)
	}

	const want = "Bearer shhhh-secret"
	if indexAuth != want {
		t.Errorf("index Authorization = %q, want %q", indexAuth, want)
	}
	if subAuth != want {
		t.Errorf("sub-file Authorization = %q, want %q", subAuth, want)
	}
}
