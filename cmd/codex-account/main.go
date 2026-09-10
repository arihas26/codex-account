package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/arihas26/codex-account/internal/accounts"
	"golang.org/x/term"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		store, err := accounts.New()
		if err != nil {
			return err
		}
		return pickAccount(store, stdout)
	}
	store, err := accounts.New()
	if err != nil {
		return err
	}

	switch args[0] {
	case "help", "-h", "--help":
		usage(stdout)
		return nil
	case "version", "--version":
		fmt.Fprintln(stdout, version)
		return nil
	case "login":
		return login(store, args[1:], stdout, stderr)
	case "list":
		return list(store, stdout)
	case "use":
		if len(args) != 2 {
			return errors.New("usage: codex-account use <name>")
		}
		if err := store.Use(args[1]); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Current account: %s\n", args[1])
		return nil
	case "current":
		name, err := store.Current()
		if err != nil {
			return err
		}
		if name == "" {
			return errors.New("no current account; run 'codex-account login <name>'")
		}
		fmt.Fprintln(stdout, name)
		return nil
	case "run":
		return runCodex(store, args[1:], stdout, stderr)
	case "delete":
		return deleteAccount(store, args[1:], stdout)
	case "doctor":
		return doctor(store, stdout)
	default:
		return fmt.Errorf("unknown command %q; run 'codex-account help'", args[0])
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `codex-account manages isolated Codex CLI accounts.

Usage:
  codex-account                       Interactively select an account
  codex-account login <name> [--device-auth]
  codex-account list
  codex-account use <name>
  codex-account current
  codex-account run [--account <name>] [-- <codex arguments...>]
  codex-account delete <name> --force
  codex-account doctor
  codex-account version

Environment:
  CODEX_ACCOUNTS_HOME  State directory (default: ~/.codex-accounts)
  CODEX_BINARY         Codex executable (default: codex)
`)
}

func pickAccount(store accounts.Store, stdout io.Writer) error {
	names, err := store.List()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return errors.New("no accounts; run 'codex-account login <name>'")
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("interactive selection requires a terminal; use 'codex-account use <name>'")
	}

	selected := 0
	if current, err := store.Current(); err == nil {
		for i, name := range names {
			if name == current {
				selected = i
				break
			}
		}
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("enable interactive input: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	defer fmt.Fprint(stdout, "\x1b[?25h")
	fmt.Fprint(stdout, "\x1b[?25l")

	lines := len(names) + 3
	firstRender := true
	for {
		if !firstRender {
			fmt.Fprintf(stdout, "\x1b[%dA", lines)
		}
		firstRender = false
		fmt.Fprint(stdout, "\r\x1b[2KSelect a Codex account\r\n")
		for i, name := range names {
			prefix := "  "
			if i == selected {
				prefix = "❯ "
			}
			fmt.Fprintf(stdout, "\r\x1b[2K%s%s\r\n", prefix, name)
		}
		fmt.Fprint(stdout, "\r\x1b[2K\r\n\r\x1b[2K↑/↓ or j/k: move  Enter: select  q: cancel\r\n")

		key, err := readKey(os.Stdin)
		if err != nil {
			return fmt.Errorf("read selection: %w", err)
		}
		switch key {
		case "up", "k":
			selected = moveSelection(selected, -1, len(names))
		case "down", "j":
			selected = moveSelection(selected, 1, len(names))
		case "enter":
			if err := store.Use(names[selected]); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "\r\nCurrent account: %s\r\n", names[selected])
			return nil
		case "q", "ctrl-c":
			fmt.Fprint(stdout, "\r\nCancelled.\r\n")
			return nil
		}
	}
}

func readKey(r io.Reader) (string, error) {
	buf := make([]byte, 1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	switch buf[0] {
	case '\r', '\n':
		return "enter", nil
	case 3:
		return "ctrl-c", nil
	case 27:
		sequence := make([]byte, 2)
		if _, err := io.ReadFull(r, sequence); err != nil {
			return "escape", nil
		}
		if sequence[0] == '[' && sequence[1] == 'A' {
			return "up", nil
		}
		if sequence[0] == '[' && sequence[1] == 'B' {
			return "down", nil
		}
		return "escape", nil
	default:
		return string(buf), nil
	}
}

func moveSelection(current, delta, count int) int {
	return (current + delta + count) % count
}

func login(store accounts.Store, args []string, stdout, stderr io.Writer) error {
	if len(args) < 1 {
		return errors.New("usage: codex-account login <name> [--device-auth]")
	}
	name := args[0]
	loginArgs := []string{"login"}
	for _, arg := range args[1:] {
		if arg != "--device-auth" {
			return fmt.Errorf("unsupported login option %q", arg)
		}
		loginArgs = append(loginArgs, arg)
	}
	home, err := store.Create(name)
	if err != nil {
		return err
	}
	if err := executeCodex(home, loginArgs, stdout, stderr); err != nil {
		return err
	}
	if err := store.Use(name); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Logged in and selected account: %s\n", name)
	return nil
}

func list(store accounts.Store, stdout io.Writer) error {
	names, err := store.List()
	if err != nil {
		return err
	}
	current, currentErr := store.Current()
	for _, name := range names {
		marker := " "
		if currentErr == nil && name == current {
			marker = "*"
		}
		auth := "not logged in"
		if _, err := os.Stat(filepath.Join(store.AccountsDir(), name, "auth.json")); err == nil {
			auth = "logged in"
		}
		fmt.Fprintf(stdout, "%s %-20s %s\n", marker, name, auth)
	}
	if len(names) == 0 {
		fmt.Fprintln(stdout, "No accounts. Run 'codex-account login <name>'.")
	}
	return currentErr
}

func runCodex(store accounts.Store, args []string, stdout, stderr io.Writer) error {
	name := ""
	if len(args) >= 2 && args[0] == "--account" {
		name = args[1]
		args = args[2:]
	}
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if name == "" {
		var err error
		name, err = store.Current()
		if err != nil {
			return err
		}
	}
	if name == "" {
		return errors.New("no account selected; run 'codex-account use <name>' or pass --account")
	}
	home, err := store.AccountHome(name)
	if err != nil {
		return err
	}
	if !store.Exists(name) {
		return fmt.Errorf("account %q does not exist", name)
	}
	return executeCodex(home, args, stdout, stderr)
}

func executeCodex(home string, args []string, stdout, stderr io.Writer) error {
	binary := os.Getenv("CODEX_BINARY")
	if binary == "" {
		binary = "codex"
	}
	cmd := exec.Command(binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = setEnv(os.Environ(), "CODEX_HOME", home)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("codex exited with status %d", exitErr.ExitCode())
		}
		return fmt.Errorf("run codex: %w", err)
	}
	return nil
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, item := range env {
		if !strings.HasPrefix(item, prefix) {
			out = append(out, item)
		}
	}
	return append(out, prefix+value)
}

func deleteAccount(store accounts.Store, args []string, stdout io.Writer) error {
	if len(args) != 2 || args[1] != "--force" {
		return errors.New("deletion is irreversible; use: codex-account delete <name> --force")
	}
	home, err := store.AccountHome(args[0])
	if err != nil {
		return err
	}
	if !store.Exists(args[0]) {
		return fmt.Errorf("account %q does not exist", args[0])
	}
	current, _ := store.Current()
	if err := os.RemoveAll(home); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	if current == args[0] {
		_ = os.Remove(store.CurrentFile())
	}
	fmt.Fprintf(stdout, "Deleted account: %s\n", args[0])
	return nil
}

func doctor(store accounts.Store, stdout io.Writer) error {
	binary := os.Getenv("CODEX_BINARY")
	if binary == "" {
		binary = "codex"
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return fmt.Errorf("codex executable not found: %w", err)
	}
	fmt.Fprintf(stdout, "OK codex: %s\n", path)
	fmt.Fprintf(stdout, "OK platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(stdout, "OK state: %s\n", store.Root)
	names, err := store.List()
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "OK accounts: %d\n", len(names))
	return nil
}
