package version

import "testing"

func TestDetectInstallMethod(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		home        string
		wantMethod  InstallMethod
		wantCommand string
	}{
		{
			name:        "homebrew arm",
			path:        "/opt/homebrew/Cellar/grounds/0.1.13/bin/grounds",
			wantMethod:  InstallHomebrew,
			wantCommand: "brew upgrade groundsgg/tap/grounds",
		},
		{
			name:        "scoop",
			path:        `C:\Users\Lukas\scoop\apps\grounds\current\grounds.exe`,
			wantMethod:  InstallScoop,
			wantCommand: "scoop update grounds",
		},
		{
			name:        "raw local bin",
			path:        "/home/lukas/.local/bin/grounds",
			home:        "/home/lukas",
			wantMethod:  InstallRaw,
			wantCommand: `curl -sSL https://github.com/groundsgg/grounds-cli/releases/latest/download/install.sh | bash`,
		},
		{
			name:        "raw custom install dir under home",
			path:        "/home/lukas/tools/grounds/bin/grounds",
			home:        "/home/lukas",
			wantMethod:  InstallRaw,
			wantCommand: `INSTALL_DIR=/home/lukas/tools/grounds/bin curl -sSL https://github.com/groundsgg/grounds-cli/releases/latest/download/install.sh | bash`,
		},
		{
			name:       "usr local system path remains package managed",
			path:       "/usr/local/bin/grounds",
			home:       "/home/lukas",
			wantMethod: InstallSystem,
		},
		{
			name:       "usr system path remains package managed",
			path:       "/usr/bin/grounds",
			home:       "/home/lukas",
			wantMethod: InstallSystem,
		},
		{
			name:       "unknown",
			path:       "/tmp/grounds",
			wantMethod: InstallUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectInstallMethod(tt.path, tt.home)
			if got.Method != tt.wantMethod {
				t.Fatalf("method = %q, want %q", got.Method, tt.wantMethod)
			}
			if got.UpdateCommand != tt.wantCommand {
				t.Fatalf("update command = %q, want %q", got.UpdateCommand, tt.wantCommand)
			}
		})
	}
}
