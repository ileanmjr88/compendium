package installer

import (
	"testing"

	"github.com/ileanmjr88/compendium/internal/config"
)

func TestResolve(t *testing.T) {
	t.Run("language only", func(t *testing.T) {
		cfg := config.Config{
			Languages: map[string]string{"go": "1.22.0"},
		}
		items := Resolve(cfg)

		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}

		if items[0].Name != "go" {
			t.Errorf("expected name 'go', got '%s'", items[0].Name)
		}

		if items[0].Version != "1.22.0" {
			t.Errorf("expected version '1.22.0', got '%s'", items[0].Version)
		}

		if items[0].Kind != "languages" {
			t.Errorf("expected kind 'languages', got '%s'", items[0].Kind)
		}
	})

	t.Run("tools only", func(t *testing.T) {
		cfg := config.Config{
			Tools: map[string]string{"cmake": "4.2.3"},
		}
		items := Resolve(cfg)

		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}

		if items[0].Name != "cmake" {
			t.Errorf("expected name 'cmake', got '%s'", items[0].Name)
		}

		if items[0].Version != "4.2.3" {
			t.Errorf("expected version '4.2.3', got '%s'", items[0].Version)
		}

		if items[0].Kind != "tools" {
			t.Errorf("expected kind 'tools', got '%s'", items[0].Kind)
		}
	})

	t.Run("at syntax", func(t *testing.T) {
		cfg := config.Config{
			Languages: map[string]string{"c": "gcc@13.2.0"},
		}
		items := Resolve(cfg)

		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}

		if items[0].Name != "gcc" {
			t.Errorf("expected name 'gcc', got '%s'", items[0].Name)
		}

		if items[0].Version != "13.2.0" {
			t.Errorf("expected version '13.2.0', got '%s'", items[0].Version)
		}

		if items[0].Kind != "languages" {
			t.Errorf("expected kind 'languages', got '%s'", items[0].Kind)
		}
	})

	t.Run("empty config", func(t *testing.T) {
		cfg := config.Config{}
		items := Resolve(cfg)

		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}
	})

	t.Run("mixed languages and tools", func(t *testing.T) {
		cfg := config.Config{
			Languages: map[string]string{"go": "1.22.0"},
			Tools:     map[string]string{"cmake": "3.28.1"},
		}
		items := Resolve(cfg)

		if len(items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(items))
		}

		goItem := findItem(items, "go")
		if goItem == nil {
			t.Fatalf("expected to find 'go' item")
		}
		if goItem.Kind != "languages" {
			t.Errorf("expected kind 'languages', got '%s'", goItem.Kind)
		}

		cmakeItem := findItem(items, "cmake")
		if cmakeItem == nil {
			t.Fatalf("expected to find 'cmake' item")
		}
		if cmakeItem.Kind != "tools" {
			t.Errorf("expected kind 'tools', got '%s'", cmakeItem.Kind)
		}
	})

	t.Run("multiple at symbols", func(t *testing.T) {
		cfg := config.Config{
			Languages: map[string]string{"c": "gcc@12.2.1@extra"},
		}
		items := Resolve(cfg)

		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}

		if items[0].Name != "gcc" {
			t.Errorf("expected name 'gcc', got '%s'", items[0].Name)
		}
		if items[0].Version != "12.2.1@extra" {
			t.Errorf("expected version '12.2.1@extra', got '%s'", items[0].Version)
		}
	})

	t.Run("c and cpp both set skips cpp", func(t *testing.T) {
		cfg := config.Config{
			Languages: map[string]string{
				"c":   "gcc@13.2.0",
				"cpp": "gcc@13.2.0",
			},
		}
		items := Resolve(cfg)

		if len(items) != 1 {
			t.Fatalf("expected 1 item (cpp deduped), got %d", len(items))
		}
		if items[0].Name != "gcc" || items[0].Version != "13.2.0" {
			t.Errorf("expected gcc@13.2.0, got %s@%s", items[0].Name, items[0].Version)
		}
	})

	t.Run("only cpp set still resolves", func(t *testing.T) {
		cfg := config.Config{
			Languages: map[string]string{"cpp": "clang@22.1.2"},
		}
		items := Resolve(cfg)

		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}
		if items[0].Name != "clang" {
			t.Errorf("expected clang, got %s", items[0].Name)
		}
	})

	t.Run("empty version", func(t *testing.T) {
		cfg := config.Config{
			Languages: map[string]string{"go": ""},
		}
		items := Resolve(cfg)

		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}

		if items[0].Name != "go" {
			t.Errorf("expected name 'go', got '%s'", items[0].Name)
		}
		if items[0].Version != "" {
			t.Errorf("expected empty version, got '%s'", items[0].Version)
		}
	})
}
