package accounts

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const rootEnv = "CODEX_ACCOUNTS_HOME"

var validName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Store struct {
	Root string
}

type Identity struct {
	LoggedIn bool
	Email    string
}

func New() (Store, error) {
	if root := os.Getenv(rootEnv); root != "" {
		return Store{Root: root}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Store{}, fmt.Errorf("find home directory: %w", err)
	}
	return Store{Root: filepath.Join(home, ".codex-accounts")}, nil
}

func (s Store) Ensure() error {
	if err := os.MkdirAll(s.AccountsDir(), 0o700); err != nil {
		return fmt.Errorf("create accounts directory: %w", err)
	}
	return os.Chmod(s.Root, 0o700)
}

func (s Store) AccountsDir() string { return filepath.Join(s.Root, "accounts") }
func (s Store) CurrentFile() string { return filepath.Join(s.Root, "current") }

func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return errors.New("account name must start with a letter or digit and contain only letters, digits, '.', '_' or '-'")
	}
	return nil
}

func (s Store) AccountHome(name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	return filepath.Join(s.AccountsDir(), name), nil
}

func (s Store) Create(name string) (string, error) {
	home, err := s.AccountHome(name)
	if err != nil {
		return "", err
	}
	if err := s.Ensure(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return "", fmt.Errorf("create account home: %w", err)
	}
	if err := os.Chmod(home, 0o700); err != nil {
		return "", fmt.Errorf("secure account home: %w", err)
	}
	return home, nil
}

func (s Store) Exists(name string) bool {
	home, err := s.AccountHome(name)
	if err != nil {
		return false
	}
	info, err := os.Stat(home)
	return err == nil && info.IsDir()
}

func (s Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.AccountsDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() && ValidateName(entry.Name()) == nil {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func (s Store) Current() (string, error) {
	data, err := os.ReadFile(s.CurrentFile())
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read current account: %w", err)
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return "", nil
	}
	if !s.Exists(name) {
		return "", fmt.Errorf("current account %q does not exist", name)
	}
	return name, nil
}

func (s Store) Use(name string) error {
	if !s.Exists(name) {
		return fmt.Errorf("account %q does not exist", name)
	}
	if err := s.Ensure(); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.Root, ".current-*")
	if err != nil {
		return fmt.Errorf("create current account marker: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := fmt.Fprintln(tmp, name); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, s.CurrentFile()); err != nil {
		return fmt.Errorf("select account: %w", err)
	}
	return nil
}

func (s Store) Identity(name string) (Identity, error) {
	home, err := s.AccountHome(name)
	if err != nil {
		return Identity{}, err
	}
	data, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if errors.Is(err, os.ErrNotExist) {
		return Identity{}, nil
	}
	if err != nil {
		return Identity{}, fmt.Errorf("read account authentication: %w", err)
	}
	var auth struct {
		Tokens struct {
			IDToken string `json:"id_token"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(data, &auth); err != nil {
		return Identity{LoggedIn: true}, nil
	}
	parts := strings.Split(auth.Tokens.IDToken, ".")
	if len(parts) != 3 {
		return Identity{LoggedIn: true}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Identity{LoggedIn: true}, nil
	}
	var claims struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Identity{LoggedIn: true}, nil
	}
	return Identity{LoggedIn: true, Email: claims.Email}, nil
}
