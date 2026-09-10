# codex-account

`codex-account` runs multiple Codex CLI logins in isolated `CODEX_HOME`
directories. It does not copy, decode, or print authentication tokens.

## Install

```sh
go install ./cmd/codex-account
```

Make sure Go's bin directory is on `PATH`.

## Usage

Create isolated accounts using Codex's normal login flow:

```sh
codex-account login personal
codex-account login work --device-auth
```

Select and run an account:

```sh
codex-account list
codex-account use work
codex-account run
codex-account run -- exec "run the tests"
```

Run another account without changing the current selection:

```sh
codex-account run --account personal
```

Each invocation starts the real `codex` executable with an account-specific
`CODEX_HOME`. Accounts can therefore run concurrently without sharing auth,
history, SQLite state, or other runtime files.

State is stored under `~/.codex-accounts` by default. Override it with
`CODEX_ACCOUNTS_HOME`. Account directories and marker files are created with
owner-only permissions.

## Safety

Account deletion requires an explicit irreversible flag:

```sh
codex-account delete work --force
```

Do not commit `~/.codex-accounts` or copy its contents into a repository.
