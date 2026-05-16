package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

const PublicIndexURL = "https://raw.githubusercontent.com/ileanmjr88/compendium-registry/main/index.json"

type Client struct {
	index    *Index
	indexURL string
	cache    map[string]ToolFile
}

type ToolVersions struct {
	Name     string
	Kind     string
	Latest   string
	Versions []string
}

type IndexRow struct {
	Kind   string
	Name   string
	Latest string
	Count  int
}

func httpGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("status %d from %s", resp.StatusCode, url)
	}

	return resp, nil
}

func subFileURL(indexURL, relFile string) (string, error) {
	if relFile == "" || strings.HasPrefix(relFile, "/") {
		return "", fmt.Errorf("invalid file path: %q", relFile)
	}

	u, err := url.Parse(indexURL)
	if err != nil {
		return "", fmt.Errorf("parsing index URL: %w", err)
	}
	u.Path = path.Join(path.Dir(u.Path), relFile)
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func FetchIndex(url string) (*Index, error) {
	resp, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var idx Index
	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		return nil, fmt.Errorf("parsing index: %w", err)
	}

	return &idx, nil
}

func NewClient(indexURL string) (*Client, error) {
	idx, err := FetchIndex(indexURL)
	if err != nil {
		return nil, fmt.Errorf("fetching index: %w", err)
	}

	return &Client{
		indexURL: indexURL,
		index:    idx,
		cache:    map[string]ToolFile{},
	}, nil
}

func (c *Client) Lookup(kind, name, version, platform, arch string) (*Artifact, map[string]string, error) {
	// 1. Pick the right IndexEntry map
	var entries map[string]IndexEntry

	switch kind {
	case "languages":
		entries = c.index.Languages
	case "tools":
		entries = c.index.Tools
	default:
		return nil, nil, fmt.Errorf("unknown kind: %s", kind)
	}

	// 2. Look up the IndexEntry. Errors before any HTTP.
	indexEntry, ok := entries[name]
	if !ok {
		return nil, nil, fmt.Errorf("unknown %s: %s", kind, name)
	}

	// 3 & 4. Cache check: lazy fetch on miss.
	cacheKey := kind + "/" + name
	toolFile, cached := c.cache[cacheKey]
	if !cached {
		subURL, err := subFileURL(c.indexURL, indexEntry.File)
		if err != nil {
			return nil, nil, fmt.Errorf("resolving sub-file URL: %w", err)
		}
		resp, err := httpGet(subURL)
		if err != nil {
			return nil, nil, fmt.Errorf("fetching %s/%s: %w", kind, name, err)
		}
		defer func() { _ = resp.Body.Close() }()

		toolFile = ToolFile{}
		if err := json.NewDecoder(resp.Body).Decode(&toolFile); err != nil {
			return nil, nil, fmt.Errorf("parsing %s/%s: %w", kind, name, err)
		}
		c.cache[cacheKey] = toolFile
	}

	// 5. Drill in
	versionEntry, ok := toolFile[version]
	if !ok {
		return nil, nil, fmt.Errorf("unknown version: %s %s", name, version)
	}
	archMap, ok := versionEntry.Platforms[platform]
	if !ok {
		return nil, nil, fmt.Errorf("unknown platform: %s for %s %s", platform, name, version)
	}
	artifact, ok := archMap[arch]
	if !ok {
		return nil, nil, fmt.Errorf("unknown arch: %s for %s %s on %s", arch, name, version, platform)
	}

	return &artifact, versionEntry.Depends, nil
}

func ResolveSource(source string) string {
	return PublicIndexURL
}

func (c *Client) findEntry(name string) (kind string, entry IndexEntry, ok bool) {
	if e, found := c.index.Languages[name]; found {
		return "languages", e, true
	}
	if e, found := c.index.Tools[name]; found {
		return "tools", e, true
	}
	return "", IndexEntry{}, false
}

func compareVersions(a, b string) int {
	splitter := func(r rune) bool { return r == '.' || r == '-' }
	ap := strings.FieldsFunc(a, splitter)
	bp := strings.FieldsFunc(b, splitter)
	n := max(len(ap), len(bp))

	for i := 0; i < n; i++ {
		var an, bn int
		if i < len(ap) {
			an, _ = strconv.Atoi(ap[i])
		}
		if i < len(bp) {
			bn, _ = strconv.Atoi(bp[i])
		}
		if an != bn {
			return an - bn
		}
	}
	return strings.Compare(a, b)
}

func (c *Client) ListVersions(name string) (*ToolVersions, error) {
	kind, entry, ok := c.findEntry(name)
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	cacheKey := kind + "/" + name
	toolFile, cached := c.cache[cacheKey]
	if !cached {
		subURL, err := subFileURL(c.indexURL, entry.File)
		if err != nil {
			return nil, fmt.Errorf("resolving sub-file URL: %w", err)
		}
		resp, err := httpGet(subURL)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", name, err)
		}
		defer func() { _ = resp.Body.Close() }()

		toolFile = ToolFile{}
		if err := json.NewDecoder(resp.Body).Decode(&toolFile); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}
		c.cache[cacheKey] = toolFile
	}

	versions := make([]string, 0, len(toolFile))
	for v := range toolFile {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) > 0
	})

	return &ToolVersions{
		Name:     name,
		Kind:     kind,
		Latest:   entry.Latest,
		Versions: versions,
	}, nil
}

func (c *Client) ListAll() []IndexRow {
	rows := make([]IndexRow, 0, len(c.index.Languages)+len(c.index.Tools))

	for name, entry := range c.index.Languages {
		rows = append(rows, IndexRow{
			Kind:   "languages",
			Name:   name,
			Latest: entry.Latest,
			Count:  entry.Versions,
		})
	}
	for name, entry := range c.index.Tools {
		rows = append(rows, IndexRow{
			Kind:   "tools",
			Name:   name,
			Latest: entry.Latest,
			Count:  entry.Versions,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Kind != rows[j].Kind {
			return rows[i].Kind < rows[j].Kind
		}
		return rows[i].Name < rows[j].Name
	})

	return rows
}
