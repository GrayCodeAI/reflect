# Shell Completions

decoder ships completion scripts for **bash**, **zsh**, **fish**, and **PowerShell**,
plus a machine-readable **JSON** spec for IDE integration.

## Quick Install

```bash
# Auto-install to the standard location for your shell and OS:
decoder completion install bash
decoder completion install zsh
decoder completion install fish
```

## Manual Setup

### Bash

```bash
# Load for current session:
source <(decoder completion bash)

# Persist (Linux):
decoder completion bash > ~/.local/share/bash-completion/completions/decoder

# Persist (macOS with Homebrew):
decoder completion bash > /opt/homebrew/etc/bash_completion.d/decoder
```

### Zsh

```bash
# Load for current session:
source <(decoder completion zsh)

# Persist:
decoder completion zsh > "${fpath[1]}/_decoder"
```

### Fish

```bash
# Load for current session:
decoder completion fish | source

# Persist:
decoder completion fish > ~/.config/fish/completions/decoder.fish
```

### PowerShell

```powershell
# Load for current session:
decoder completion powershell | Out-String | Invoke-Expression

# Persist: add to your $PROFILE
decoder completion powershell > decoder.ps1
. ./decoder.ps1
```

## JSON Spec (IDE Integration)

```bash
decoder completion json
```

Prints a machine-readable command/flag spec that IDEs and editor plugins can
consume for inline completions without shell integration.

## Install Paths

`decoder completion install` resolves the correct path automatically:

| Shell | Linux | macOS (Homebrew) |
|-------|-------|-------------------|
| bash | `~/.local/share/bash-completion/completions/decoder` | `/opt/homebrew/etc/bash_completion.d/decoder` |
| zsh | First `$fpath` entry (e.g. `/usr/local/share/zsh/site-functions/_decoder`) | Same |
| fish | `~/.config/fish/completions/decoder.fish` | Same |
