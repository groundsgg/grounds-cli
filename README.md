# grounds CLI

Drives the Grounds Internal Developer Platform from your terminal.

## Install

### macOS / Linux (Homebrew)

```bash
brew install groundsgg/tap/grounds
```

### Windows (Scoop)

```powershell
scoop bucket add groundsgg https://github.com/groundsgg/scoop-bucket
scoop install grounds
```

### Linux (.deb)

Download the latest .deb from [releases](https://github.com/groundsgg/grounds-cli/releases/latest)
and run `sudo apt install ./grounds_*.deb`.

### Raw

```bash
curl -sSL https://github.com/groundsgg/grounds-cli/releases/latest/download/install.sh | bash
```

## Quickstart

```bash
grounds login                          # OAuth device flow
grounds init                           # scaffold grounds.yaml
grounds push                           # first push
grounds cluster status                 # observe namespace
grounds logs <pushId> --follow         # tail logs
```

## Commands

| Command | What |
|---|---|
| `grounds login / logout` | Auth via Keycloak device flow |
| `grounds version` | Build info |
| `grounds completion <shell>` | Shell completions |
| `grounds doctor` | Diagnose env (auth, API, gradle, java) |
| `grounds init` | Scaffold a grounds.yaml |
| `grounds cluster up/down/delete/status` | Workspace lifecycle |
| `grounds push [--target=dev]` | Build + deploy via Gradle plugin |
| `grounds push retry/list` | Re-run / list pushes |
| `grounds logs <pushId> [--follow]` | Stream logs |
| `grounds logs deployment <name> [--follow]` | Stream deployment logs |

## Configuration

`~/.config/grounds/config.yaml` (XDG-aware). Overridable via flags or env vars (`GROUNDS_API_URL`, `GROUNDS_TOKEN`, `GROUNDS_CONFIG_DIR`).

## Troubleshooting

Run `grounds doctor`. If something looks off, the report tells you which check failed and how to fix it.

Shell completions work in all four shells (bash/zsh/fish/powershell):
```bash
grounds completion bash | sudo tee /etc/bash_completion.d/grounds
grounds completion zsh  | sudo tee /usr/share/zsh/site-functions/_grounds
grounds completion fish | sudo tee /usr/share/fish/vendor_completions.d/grounds.fish
grounds completion powershell | Out-File $PROFILE\..\grounds.ps1
```

## License

Apache-2.0.
