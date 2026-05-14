# grounds CLI

Drives the Grounds Developer Platform from your terminal.

## Install

### macOS / Linux (Homebrew)

```bash
brew install --cask groundsgg/tap/grounds
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
grounds workspace scan ../ --yes       # discover sibling plugin repos
grounds push                           # first push
grounds cluster status                 # observe namespace
grounds logs <pushId> --follow         # tail logs
```

`grounds init` loads base-image choices from Forge's runtime catalog and falls back to built-in defaults when the API is unavailable.

## Commands

| Command                                     | What                                         |
|---------------------------------------------|----------------------------------------------|
| `grounds login / logout`                    | Auth via Keycloak device flow                |
| `grounds version [--check]`                 | Build info and optional release update check |
| `grounds completion <shell>`                | Shell completions                            |
| `grounds doctor`                            | Diagnose env and warn about CLI updates      |
| `grounds init`                              | Scaffold a grounds.yaml                      |
| `grounds workspace scan/add/list/enable`    | Manage local plugin workspace overrides      |
| `grounds cluster up/down/delete/status`     | Workspace lifecycle                          |
| `grounds push [--target=dev]`               | Build + deploy via Gradle plugin             |
| `grounds push retry/list`                   | Re-run / list pushes                         |
| `grounds logs <pushId> [--follow]`          | Stream logs                                  |
| `grounds logs deployment <name> [--follow]` | Stream deployment logs                       |

## Configuration

`~/.config/grounds/config.yaml` (XDG-aware). Overridable via flags or env vars (`GROUNDS_API_URL`, `GROUNDS_TOKEN`, `GROUNDS_CONFIG_DIR`).

Local plugin workspace overrides are stored in `~/.config/grounds/workspace.yaml`.
Default `grounds push` still uses the plugin sources pinned in committed `grounds.yaml`.
Use local plugin artifacts only when requested:

```bash
grounds workspace scan ../              # preview sibling plugin repos, then confirm
grounds workspace scan ../ --yes        # write discovered mappings without prompting
grounds workspace add plugin-chat ../plugin-chat --variant paper
grounds push --local plugin-chat        # override one plugin from the workspace
grounds push --local plugin-chat,plugin-permissions
grounds push --with-local               # override every enabled workspace entry in grounds.yaml
```

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
