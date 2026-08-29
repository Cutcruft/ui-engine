package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadState(t *testing.T) {
	dir := t.TempDir()
	doc := `
state:
  counter:
    type: object
    fields:
      count:
        type: int
        default: 0
  ui:
    type: object
    fields:
      theme:
        type: string
        default: light
`
	if err := os.WriteFile(filepath.Join(dir, "state.yaml"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(dir)
	s, err := l.LoadState("state.yaml")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil State")
	}
	counter, ok := s.Vars["counter"]
	if !ok {
		t.Fatalf("missing var counter, got %v", s.Vars)
	}
	if counter.Type != "object" {
		t.Fatalf("counter.type = %q, want object", counter.Type)
	}
	count, ok := counter.Fields["count"]
	if !ok {
		t.Fatalf("missing field count")
	}
	if count.Type != "int" {
		t.Fatalf("count.type = %q, want int", count.Type)
	}
	if count.Default != 0 {
		t.Fatalf("count.default = %v, want 0", count.Default)
	}
	if s.Vars["ui"].Fields["theme"].Default != "light" {
		t.Fatalf("theme.default = %v, want light", s.Vars["ui"].Fields["theme"].Default)
	}
}

func TestLoadStateOptional(t *testing.T) {
	dir := t.TempDir()
	l := NewLoader(dir)
	s, err := l.LoadState("") // пустой путь -> nil мимо
	if err != nil {
		t.Fatalf("LoadState empty: %v", err)
	}
	if s != nil {
		t.Fatalf("expected nil for empty path, got %+v", s)
	}
	// отсутствующий файл -> nil
	s, err = l.LoadState("nope.yaml")
	if err != nil {
		t.Fatalf("LoadState missing: %v", err)
	}
	if s != nil {
		t.Fatalf("expected nil for missing file, got %+v", s)
	}
}