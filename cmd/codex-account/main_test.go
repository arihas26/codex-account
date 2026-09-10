package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLISelectListAndDelete(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	t.Setenv("CODEX_ACCOUNTS_HOME", root)
	if err := os.MkdirAll(filepath.Join(root, "accounts", "work"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "accounts", "work", "auth.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{"use", "work"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := run([]string{"list"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "* work") || !strings.Contains(got, "logged in") {
		t.Fatalf("unexpected list output: %q", got)
	}

	stdout.Reset()
	if err := run([]string{"delete", "work", "--force"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "current")); !os.IsNotExist(err) {
		t.Fatalf("current marker remains after deletion: %v", err)
	}
}

func TestDeleteRequiresForce(t *testing.T) {
	t.Setenv("CODEX_ACCOUNTS_HOME", filepath.Join(t.TempDir(), "state"))
	if err := run([]string{"delete", "work"}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("delete without --force unexpectedly succeeded")
	}
}

func TestSetEnvReplacesValue(t *testing.T) {
	got := setEnv([]string{"A=1", "CODEX_HOME=old"}, "CODEX_HOME", "new")
	if strings.Join(got, ",") != "A=1,CODEX_HOME=new" {
		t.Fatalf("setEnv() = %v", got)
	}
}
