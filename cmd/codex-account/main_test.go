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

func TestMoveSelectionWraps(t *testing.T) {
	tests := []struct {
		current int
		delta   int
		want    int
	}{
		{current: 0, delta: -1, want: 2},
		{current: 2, delta: 1, want: 0},
		{current: 1, delta: 1, want: 2},
	}
	for _, test := range tests {
		if got := moveSelection(test.current, test.delta, 3); got != test.want {
			t.Errorf("moveSelection(%d, %d, 3) = %d, want %d", test.current, test.delta, got, test.want)
		}
	}
}

func TestReadKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"\x1b[A", "up"},
		{"\x1b[B", "down"},
		{"j", "j"},
		{"k", "k"},
		{"\r", "enter"},
		{"q", "q"},
	}
	for _, test := range tests {
		got, err := readKey(strings.NewReader(test.input))
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Errorf("readKey(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}
