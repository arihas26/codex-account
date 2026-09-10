package accounts

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStoreLifecycle(t *testing.T) {
	s := Store{Root: filepath.Join(t.TempDir(), "state")}
	if _, err := s.Create("work"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("personal"); err != nil {
		t.Fatal(err)
	}
	if err := s.Use("work"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Current()
	if err != nil || got != "work" {
		t.Fatalf("Current() = %q, %v", got, err)
	}
	names, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"personal", "work"}; !reflect.DeepEqual(gotNames(names), want) {
		t.Fatalf("List() = %v, want %v", names, want)
	}
	info, err := os.Stat(s.Root)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("root permissions = %o", info.Mode().Perm())
	}
}

func TestValidateName(t *testing.T) {
	for _, name := range []string{"../work", "with space", "", "/tmp/x"} {
		if ValidateName(name) == nil {
			t.Errorf("ValidateName(%q) unexpectedly succeeded", name)
		}
	}
}

func gotNames(names []string) []string { return names }
