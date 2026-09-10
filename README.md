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
codex-account                    # interactive picker
codex-account status             # selected account and login email
codex-account use work
codex-account run
codex-account run -- exec "run the tests"
```

The interactive picker supports arrow keys or `j`/`k`. Press Enter to select
the highlighted account, or `q` to cancel.

## Use the selected account with plain `codex`

Install the shell integration once by adding this line to `~/.zshrc`:

```sh
eval "$(codex-account shell-init zsh)"
```

Then reload the shell:

```sh
source ~/.zshrc
```

After that, the normal `codex` command uses the account selected by the picker:

```sh
codex-account
codex
```

Each launch prints the selected profile before Codex starts:

```text
Codex account: work (developer@example.com)
```

The wrapper also configures the Codex status line to show the model, current
directory, thread name, five-hour usage limit, and weekly usage limit.

The shell wrapper sets `CODEX_HOME` dynamically for each invocation. If no
account is selected, it falls back to the normal Codex home.

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
