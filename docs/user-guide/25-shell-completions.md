# Shell Completions

reflect ships completion scripts for **bash**, **zsh**, **fish**, and **PowerShell**,
plus a machine-readable **JSON** spec for IDE integration.

## Quick Install

```bash
# Auto-install to the standard location for your shell and OS:
reflect completion install bash
reflect completion install zsh
reflect completion install fish
```

## Manual Setup

### Bash

```bash
# Load for current session:
source <(reflect completion bash)

# Persist (Linux):
reflect completion bash > ~/.local/share/bash-completion/completions/reflect

# Persist (macOS with Homebrew):
reflect completion bash > /opt/homebrew/etc/bash_completion.d/reflect
```

### Zsh

```bash
# Load for current session:
source <(reflect completion zsh)

# Persist:
reflect completion zsh > "${fpath[1]}/_reflect"
```

### Fish

```bash
# Load for current session:
reflect completion fish | source

# Persist:
reflect completion fish > ~/.config/fish/completions/reflect.fish
```

### PowerShell

```powershell
# Load for current session:
reflect completion powershell | Out-String | Invoke-Expression

# Persist: add to your $PROFILE
reflect completion powershell > reflect.ps1
. ./reflect.ps1
```

## JSON Spec (IDE Integration)

```bash
reflect completion json
```

Prints a machine-readable command/flag spec that IDEs and editor plugins can
consume for inline completions without shell integration.

## Install Paths

`reflect completion install` resolves the correct path automatically:

| Shell | Linux | macOS (Homebrew) |
|-------|-------|-------------------|
| bash | `~/.local/share/bash-completion/completions/reflect` | `/opt/homebrew/etc/bash_completion.d/reflect` |
| zsh | First `$fpath` entry (e.g. `/usr/local/share/zsh/site-functions/_reflect`) | Same |
| fish | `~/.config/fish/completions/reflect.fish` | Same |
