package lockfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGolden(t *testing.T) {
	lf, err := Load(filepath.Join("testdata", "golden.toml"))
	if err != nil {
		t.Fatalf("Load() golden: %v", err)
	}
	if lf.Meta.Project != "wordNebula" {
		t.Errorf("project = %q, want wordNebula", lf.Meta.Project)
	}
}

func TestLoadErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
		if err == nil {
			t.Fatalf("expected error for missing file")
		}
	})

	t.Run("malformed toml", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.toml")
		if err := os.WriteFile(path, []byte("= = not toml"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("schema too new", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "future.toml")
		if err := os.WriteFile(path, []byte("[meta]\nschema = 99\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		var sve *SchemaVersionError
		if _, err := Load(path); !errors.As(err, &sve) {
			t.Fatalf("want *SchemaVersionError, got %T: %v", err, err)
		}
	})

	t.Run("fails validation", func(t *testing.T) {
		// Parses fine, but schema 0 != CurrentSchema, so Validate rejects it
		// and Load surfaces the wrapped *ValidationError.
		path := filepath.Join(t.TempDir(), "invalid.toml")
		if err := os.WriteFile(path, []byte("[meta]\nproject = \"x\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		var ve *ValidationError
		if _, err := Load(path); !errors.As(err, &ve) {
			t.Fatalf("want *ValidationError, got %T: %v", err, err)
		}
	})
}

func TestSchemaVersionError(t *testing.T) {
	err := &SchemaVersionError{Path: "x.lock", Found: 2, Max: 1}
	want := "x.lock: lockfile schema 2 in newer supported schema 1; upgrade compendium"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
