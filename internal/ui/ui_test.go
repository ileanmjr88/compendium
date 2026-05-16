package ui

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureOutput redirects stdout, runs fn, and returns what was printed
func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	return string(out)
}

func TestPrint(t *testing.T) {
	t.Run("with detail", func(t *testing.T) {
		out := captureOutput(func() {
			Print(Success, "checksum ok", "sha256:abc123")
		})

		if !strings.Contains(out, "checksum ok") {
			t.Errorf("expected output to contain 'checksum ok', got '%s'", out)
		}
		if !strings.Contains(out, "sha256:abc123") {
			t.Errorf("expected output to contain 'sha256:abc123', got '%s'", out)
		}
	})

	t.Run("without detail", func(t *testing.T) {
		out := captureOutput(func() {
			Print(Action, "resolving languages", "")
		})

		if !strings.Contains(out, "resolving languages") {
			t.Errorf("expected output to contain 'resolving languages', got '%s'", out)
		}
	})
}

func TestPrintDetail(t *testing.T) {
	t.Run("with detail", func(t *testing.T) {
		out := captureOutput(func() {
			PrintDetail(Success, "go 1.22.0", "already installed")
		})

		if !strings.Contains(out, "go 1.22.0") {
			t.Errorf("expected output to contain 'go 1.22.0', got '%s'", out)
		}
		if !strings.Contains(out, "already installed") {
			t.Errorf("expected output to contain 'already installed', got '%s'", out)
		}
	})

	t.Run("without detail", func(t *testing.T) {
		out := captureOutput(func() {
			PrintDetail(Action, "cmake 3.28.1", "")
		})

		if !strings.Contains(out, "cmake 3.28.1") {
			t.Errorf("expected output to contain 'cmake 3.28.1', got '%s'", out)
		}
	})
}

func TestPrintWithFailSymbol(t *testing.T) {
	out := captureOutput(func() {
		Print(Fail, "checksum mismatch", "go")
	})

	if !strings.Contains(out, "checksum mismatch") {
		t.Errorf("expected output to contain 'checksum mismatch', got '%s'", out)
	}
}

func TestPrintWithWarningSymbol(t *testing.T) {
	out := captureOutput(func() {
		Print(Warning, "timeout not set", "defaulting to 60s")
	})

	if !strings.Contains(out, "timeout not set") {
		t.Errorf("expected output to contain 'timeout not set', got '%s'", out)
	}
}

func TestProgressBar(t *testing.T) {
	bar := ProgressBar(1024, "go")

	if bar == nil {
		t.Fatal("expected progress bar to not be nil")
	}
}

func TestSymbols(t *testing.T) {
	if !strings.Contains(Action, "→") {
		t.Errorf("Action should contain →")
	}
	if !strings.Contains(Success, "✓") {
		t.Errorf("Success should contain ✓")
	}
	if !strings.Contains(Fail, "✗") {
		t.Errorf("Fail should contain ✗")
	}
	if !strings.Contains(Download, "↓") {
		t.Errorf("Download should contain ↓")
	}
	if !strings.Contains(Warning, "!") {
		t.Errorf("Warning should contain !")
	}
}
