package protoshellmanifest

import (
	"strings"
	"testing"
)

func TestSlugFromSpecName(t *testing.T) {
	if got := SlugFromSpecName("FEAT-spec-prototype-shell-deck"); got != "feat-spec-prototype-shell-deck" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildFeatureRoute(t *testing.T) {
	y := "prototype:\n  file: .vibex/prototypes/a.html\n"
	r := BuildFeatureRoute("FEAT-x", "specs/L4-feature/FEAT-x.yaml", "X", y, nil)
	if r.EntryHTML != ".vibex/prototypes/a.html" {
		t.Fatalf("entry %q", r.EntryHTML)
	}
	r2 := BuildFeatureRoute("FEAT-x", "specs/L4-feature/FEAT-x.yaml", "X", "spec:\n  name: X\n", nil)
	if !strings.HasSuffix(r2.EntryHTML, ".vibex/prototypes/FEAT-x.html") {
		t.Fatalf("entry %q", r2.EntryHTML)
	}
}

