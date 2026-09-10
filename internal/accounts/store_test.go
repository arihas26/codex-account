package accounts

import (
	"encoding/base64"
	"fmt"
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

func TestIdentity(t *testing.T) {
	s := Store{Root: filepath.Join(t.TempDir(), "state")}
	home, err := s.Create("work")
	if err != nil {
		t.Fatal(err)
	}
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"email":"dev@example.com"}`))
	auth := fmt.Sprintf(`{"tokens":{"id_token":"header.%s.signature"}}`, payload)
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(auth), 0o600); err != nil {
		t.Fatal(err)
	}
	identity, err := s.Identity("work")
	if err != nil {
		t.Fatal(err)
	}
	if !identity.LoggedIn || identity.Email != "dev@example.com" {
		t.Fatalf("Identity() = %+v", identity)
	}
}

func TestIdentityWithoutAuth(t *testing.T) {
	s := Store{Root: filepath.Join(t.TempDir(), "state")}
	if _, err := s.Create("work"); err != nil {
		t.Fatal(err)
	}
	identity, err := s.Identity("work")
	if err != nil {
		t.Fatal(err)
	}
	if identity.LoggedIn {
		t.Fatalf("Identity() = %+v, want logged out", identity)
	}
}

func gotNames(names []string) []string { return names }
