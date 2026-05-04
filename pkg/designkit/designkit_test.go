package designkit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceRelSafe(t *testing.T) {
	root := t.TempDir()
	aDir := filepath.Join(root, "a")
	if err := os.MkdirAll(aDir, 0755); err != nil {
		t.Fatal(err)
	}
	bFile := filepath.Join(aDir, "b.txt")
	if err := os.WriteFile(bFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	relGood := filepath.Join("a", "b.txt")
	abs, err := WorkspaceRelSafe(root, relGood)
	if err != nil {
		t.Fatalf("good rel: %v", err)
	}
	if abs != bFile {
		if filepath.Clean(abs) != filepath.Clean(bFile) {
			t.Fatalf("abs %q != %q", abs, bFile)
		}
	}

	if _, err := WorkspaceRelSafe(root, ".."+string(filepath.Separator)+".."+string(filepath.Separator)+"etc"+string(filepath.Separator)+"passwd"); err == nil {
		t.Fatal("expected error for traversal")
	}
}
