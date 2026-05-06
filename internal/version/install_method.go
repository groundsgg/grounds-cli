package version

import (
	"path/filepath"
	"strings"
)

type InstallMethod string

const (
	InstallHomebrew InstallMethod = "homebrew"
	InstallScoop    InstallMethod = "scoop"
	InstallRaw      InstallMethod = "raw"
	InstallSystem   InstallMethod = "system-package"
	InstallUnknown  InstallMethod = "unknown"
)

type InstallInfo struct {
	Method        InstallMethod
	UpdateCommand string
}

const rawInstallCommand = "curl -sSL https://github.com/groundsgg/grounds-cli/releases/latest/download/install.sh | bash"
const homebrewFormulaUpdateCommand = "brew upgrade groundsgg/tap/grounds"
const homebrewCaskUpdateCommand = "brew upgrade --cask groundsgg/tap/grounds"

func DetectInstallMethod(executablePath, homeDir string) InstallInfo {
	normalized := normalizePath(executablePath)
	if resolved, err := filepath.EvalSymlinks(executablePath); err == nil {
		normalized = normalizePath(resolved)
	}
	home := normalizePath(homeDir)

	if strings.Contains(normalized, "/homebrew/cellar/") || strings.Contains(normalized, "/cellar/grounds/") {
		return InstallInfo{Method: InstallHomebrew, UpdateCommand: homebrewFormulaUpdateCommand}
	}

	if strings.Contains(normalized, "/caskroom/grounds/") {
		return InstallInfo{Method: InstallHomebrew, UpdateCommand: homebrewCaskUpdateCommand}
	}

	if strings.Contains(normalized, "/scoop/apps/grounds/") {
		return InstallInfo{Method: InstallScoop, UpdateCommand: "scoop update grounds"}
	}

	if home != "" && normalized == home+"/.local/bin/grounds" {
		return InstallInfo{
			Method:        InstallRaw,
			UpdateCommand: rawInstallCommand,
		}
	}

	if normalized == "/usr/bin/grounds" || normalized == "/usr/local/bin/grounds" {
		return InstallInfo{Method: InstallSystem}
	}

	if home != "" && strings.HasPrefix(normalized, home+"/") && strings.HasSuffix(normalized, "/grounds") {
		return InstallInfo{
			Method:        InstallRaw,
			UpdateCommand: "INSTALL_DIR=" + strings.TrimSuffix(normalized, "/grounds") + " " + rawInstallCommand,
		}
	}

	return InstallInfo{Method: InstallUnknown}
}

func normalizePath(path string) string {
	path = filepath.ToSlash(path)
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.ToLower(path)
	return strings.TrimRight(path, "/")
}
