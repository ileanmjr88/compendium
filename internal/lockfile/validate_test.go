package lockfile

import (
	"errors"
	"path/filepath"
	"testing"
)

// validLockfile returns a minimal Lockfile that passes Validate.
// Each TestValidate case starts from this and mutates exactly one thing,
// so a failure points at the single field under test.
func validLockfile() Lockfile {
	return Lockfile{
		Meta: Meta{
			Schema:    CurrentSchema,
			Project:   "wordNebula",
			Platforms: []string{"darwin-arm64"},
		},
		Languages: []Entry{{
			Name:   "go",
			Source: SourceRegistry,
			Artifacts: []Artifact{{
				Platform: "darwin",
				Arch:     "arm64",
				URL:      "https://example.com/go.tar.gz",
				Checksum: "sha256:abc123",
				Size:     123,
			}},
		}},
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Lockfile)
		wantErr bool
		field   string // expected ValidationError.Field; "" = don't assert on it
	}{
		{"valid", func(*Lockfile) {}, false, ""},
		{"bad schema", func(l *Lockfile) { l.Meta.Schema = CurrentSchema + 1 }, true, "meta.schema"},
		{"no platforms", func(l *Lockfile) { l.Meta.Platforms = nil }, true, "meta.platforms"},
		{"unknown platform", func(l *Lockfile) { l.Meta.Platforms = []string{"plan9-386"} }, true, "meta.platforms"},
		{"dup language", func(l *Lockfile) { l.Languages = append(l.Languages, l.Languages[0]) }, true, "languages.name"},
		{"dup tool", func(l *Lockfile) { l.Tools = []Entry{l.Languages[0], l.Languages[0]} }, true, "tool.name"},
		{"bad source", func(l *Lockfile) { l.Languages[0].Source = "public" }, true, ""},
		{"empty url", func(l *Lockfile) { l.Languages[0].Artifacts[0].URL = "" }, true, ""},
		{"bad checksum", func(l *Lockfile) { l.Languages[0].Artifacts[0].Checksum = "md5:abc" }, true, ""},
		{"checksum missing colon", func(l *Lockfile) { l.Languages[0].Artifacts[0].Checksum = "sha256abc" }, true, ""},
		{"zero size", func(l *Lockfile) { l.Languages[0].Artifacts[0].Size = 0 }, true, ""},
		{"dup platform/arch", func(l *Lockfile) {
			a := l.Languages[0].Artifacts[0]
			l.Languages[0].Artifacts = append(l.Languages[0].Artifacts, a)
		}, true, ""},
		{"dup ecosystem", func(l *Lockfile) {
			l.Packages.Lockfiles = []EcoLockRef{{Ecosystem: "npm"}, {Ecosystem: "npm"}}
		}, true, "packages.lockfile.ecosystem"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lf := validLockfile()
			tt.mutate(&lf)

			err := lf.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				return
			}

			// Validate always reports failures as *ValidationError.
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("want *ValidationError, got %T: %v", err, err)
			}
			if tt.field != "" && ve.Field != tt.field {
				t.Errorf("Field = %q, want %q", ve.Field, tt.field)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		err := &ValidationError{Field: "meta.schema", Value: "2", Message: "schema must equal 1"}
		want := `invalid lockfile: schema must equal 1 (meta.schema="2")`
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without value", func(t *testing.T) {
		err := &ValidationError{Field: "meta.platforms", Message: "must list at least one platform"}
		want := "invalid lockfile: must list at least one platform (meta.platforms)"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

// TestLoadInvalidFixtures loads each fixture in testdata/invalid/ through Load
// and asserts it is rejected with the expected typed error. This exercises the
// real file path (parse + schema check + Validate), complementing TestValidate's
// in-memory coverage.
func TestLoadInvalidFixtures(t *testing.T) {
	tests := []struct {
		file      string
		schemaErr bool // true: want *SchemaVersionError; false: want *ValidationError
	}{
		{"schema-too-new.toml", true},
		{"duplicate-language-name.toml", false},
		{"duplicate-platform-arch.toml", false},
		{"missing-checksum.toml", false},
		{"zero-size.toml", false},
		{"non-sha256-checksum.toml", false},
		{"unknown-source.toml", false},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			_, err := Load(filepath.Join("testdata", "invalid", tt.file))
			if err == nil {
				t.Fatalf("Load(%s): expected error, got nil", tt.file)
			}

			if tt.schemaErr {
				var sve *SchemaVersionError
				if !errors.As(err, &sve) {
					t.Fatalf("want *SchemaVersionError, got %T: %v", err, err)
				}
				return
			}

			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("want *ValidationError, got %T: %v", err, err)
			}
		})
	}
}
