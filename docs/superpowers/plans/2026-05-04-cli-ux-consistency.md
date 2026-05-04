# CLI UX Consistency Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the remaining Grounds CLI commands up to the UX standard established by `grounds doctor` and `grounds version --check`.

**Architecture:** Add shared rendering primitives in `internal/render` for status rows, detail rows, command references, and empty states. Then migrate command output incrementally without changing API behavior or machine-readable output.

**Tech Stack:** Go, Cobra, existing `internal/render` package, existing command tests with focused additions.

---

## File Structure

- Create `internal/render/message.go`: shared UX helpers for `[✓]`, `[!]`, `[✗]`, detail lines, and command references.
- Create `internal/render/message_test.go`: unit tests for helper output with colors disabled.
- Modify `cmd/grounds/commands/doctor.go`: replace local badge/detail helpers with `internal/render` helpers.
- Modify `cmd/grounds/commands/init.go` and `cmd/grounds/commands/init_test.go`: status-style scaffold output and next-step formatting.
- Modify `cmd/grounds/commands/login.go`: browser/code/auth output formatting.
- Modify `cmd/grounds/commands/logout.go`: status-style logout output.
- Modify `cmd/grounds/commands/cluster/status.go`, `cluster/up.go`, `cluster/down.go`, `cluster/delete.go`, and `internal/render/status.go`: workspace action and empty-state formatting.
- Modify `cmd/grounds/commands/push/push.go`, `push/retry.go`, and `push/list.go`: actionable error suggestions, retry success, pagination note.
- Modify `cmd/grounds/commands/preview/preview.go` and `preview_test.go`: empty state, show layout, pin/unpin output.
- Modify `cmd/grounds/commands/devspace/generate.go` and `generate_test.go`: success output without byte-count noise.
- Modify `cmd/grounds/commands/bundle/list.go` and `bundle/show.go`: empty state and command references.
- Modify `cmd/grounds/commands/root.go`, `version.go`, and subtree root files: improved `Short`, `Long`, and `Example` copy.

---

### Task 1: Shared Render Helpers

**Files:**
- Create: `internal/render/message.go`
- Create: `internal/render/message_test.go`
- Modify: `cmd/grounds/commands/doctor.go`

- [ ] **Step 1: Write tests for status rows, details, and command references**

Create `internal/render/message_test.go`:

```go
package render

import (
	"bytes"
	"testing"

	"github.com/fatih/color"
)

func TestStatusBadgeNoColor(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	if got := StatusBadge(StatusOK); got != "[✓]" {
		t.Fatalf("StatusBadge(StatusOK) = %q", got)
	}
	if got := StatusBadge(StatusWarn); got != "[!]" {
		t.Fatalf("StatusBadge(StatusWarn) = %q", got)
	}
	if got := StatusBadge(StatusError); got != "[✗]" {
		t.Fatalf("StatusBadge(StatusError) = %q", got)
	}
}

func TestStatusLine(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	StatusLine(&buf, StatusOK, "Init", "Wrote grounds.yaml")

	want := "[✓] Init - Wrote grounds.yaml\n"
	if got := buf.String(); got != want {
		t.Fatalf("StatusLine output = %q, want %q", got, want)
	}
}

func TestDetailLine(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	DetailLine(&buf, StatusWarn, "Run "+Command("grounds push")+" to create one.")

	want := "    ! Run `grounds push` to create one.\n"
	if got := buf.String(); got != want {
		t.Fatalf("DetailLine output = %q, want %q", got, want)
	}
}

func TestCommand(t *testing.T) {
	if got := Command("grounds version --check"); got != "`grounds version --check`" {
		t.Fatalf("Command() = %q", got)
	}
}
```

- [ ] **Step 2: Run the new render tests and verify they fail**

Run: `go test ./internal/render`

Expected: FAIL because `StatusBadge`, `StatusLine`, `DetailLine`, `StatusOK`, `StatusWarn`, `StatusError`, and `Command` do not exist.

- [ ] **Step 3: Implement render helpers**

Create `internal/render/message.go`:

```go
package render

import (
	"fmt"
	"io"
)

type StatusKind string

const (
	StatusOK    StatusKind = "ok"
	StatusWarn  StatusKind = "warn"
	StatusError StatusKind = "error"
)

func StatusBadge(status StatusKind) string {
	switch status {
	case StatusWarn:
		return Yellow("[!]")
	case StatusError:
		return Red("[✗]")
	default:
		return Green("[✓]")
	}
}

func DetailIcon(status StatusKind) string {
	switch status {
	case StatusError:
		return Red("✗")
	case StatusWarn:
		return Yellow("!")
	default:
		return "•"
	}
}

func StatusLine(w io.Writer, status StatusKind, subject, summary string) {
	fmt.Fprintf(w, "%s %s - %s\n", StatusBadge(status), subject, summary)
}

func DetailLine(w io.Writer, status StatusKind, detail string) {
	fmt.Fprintf(w, "    %s %s\n", DetailIcon(status), detail)
}

func Command(command string) string {
	return "`" + command + "`"
}
```

- [ ] **Step 4: Replace doctor-local badge helpers**

Modify `cmd/grounds/commands/doctor.go`:

```go
func printCheckResult(out io.Writer, r checkResult) {
	render.StatusLine(out, renderStatus(r.status), r.name, r.summary)
	for _, detail := range r.details {
		render.DetailLine(out, renderStatus(r.status), detail)
	}
}

func renderStatus(status checkStatus) render.StatusKind {
	switch status {
	case statusWarn:
		return render.StatusWarn
	case statusError:
		return render.StatusError
	default:
		return render.StatusOK
	}
}
```

Delete the local `statusBadge` and `detailIcon` functions from `doctor.go`.

- [ ] **Step 5: Run tests**

Run: `go test ./cmd/grounds/commands ./internal/render`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/render/message.go internal/render/message_test.go cmd/grounds/commands/doctor.go
git commit -m "refactor: share cli status rendering"
```

---

### Task 2: Init, Login, And Logout Output

**Files:**
- Modify: `cmd/grounds/commands/init.go`
- Modify: `cmd/grounds/commands/init_test.go`
- Modify: `cmd/grounds/commands/login.go`
- Modify: `cmd/grounds/commands/logout.go`

- [ ] **Step 1: Add init output assertion**

Update `TestInit_NonInteractive` in `cmd/grounds/commands/init_test.go`:

```go
	if got := buf.String(); got != "[✓] Init - Wrote grounds.yaml\n    • Next: run `grounds push`.\n" {
		t.Fatalf("output = %q", got)
	}
```

- [ ] **Step 2: Run init test and verify it fails**

Run: `go test ./cmd/grounds/commands -run TestInit_NonInteractive`

Expected: FAIL because output still uses `→ Wrote grounds.yaml`.

- [ ] **Step 3: Update init output**

Modify `writeGroundsYaml` in `cmd/grounds/commands/init.go`:

```go
	render.StatusLine(out, render.StatusOK, "Init", "Wrote grounds.yaml")
	render.DetailLine(out, render.StatusOK, "Next: run "+render.Command("grounds push")+".")
```

Add import:

```go
"github.com/groundsgg/grounds-cli/internal/render"
```

- [ ] **Step 4: Update login output**

Modify `cmd/grounds/commands/login.go`:

```go
render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Browser", "Opened device login page")
render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "URL: "+dc.VerificationURI)
render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "Code: "+dc.UserCode)
```

Replace the final login success line with:

```go
subject := preferred
if subject == "" {
	subject = email
}
if subject == "" {
	subject = "current user"
}
render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Auth", "Logged in as "+subject)
```

Add import:

```go
"github.com/groundsgg/grounds-cli/internal/render"
```

- [ ] **Step 5: Update logout output**

Modify `cmd/grounds/commands/logout.go`:

```go
render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Auth", "Logged out")
```

Add import:

```go
"github.com/groundsgg/grounds-cli/internal/render"
```

- [ ] **Step 6: Run tests**

Run: `go test ./cmd/grounds/commands`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/grounds/commands/init.go cmd/grounds/commands/init_test.go cmd/grounds/commands/login.go cmd/grounds/commands/logout.go
git commit -m "style: standardize auth and init output"
```

---

### Task 3: Cluster Output And Workspace Empty State

**Files:**
- Modify: `cmd/grounds/commands/cluster/status.go`
- Modify: `cmd/grounds/commands/cluster/up.go`
- Modify: `cmd/grounds/commands/cluster/down.go`
- Modify: `cmd/grounds/commands/cluster/delete.go`
- Modify: `internal/render/status.go`

- [ ] **Step 1: Update no-workspace status output**

Modify the 404 branch in `cmd/grounds/commands/cluster/status.go`:

```go
render.StatusLine(cmd.OutOrStdout(), render.StatusWarn, "Workspace", "No workspace found")
render.DetailLine(cmd.OutOrStdout(), render.StatusWarn, "Run "+render.Command("grounds push")+" to create one.")
return nil
```

- [ ] **Step 2: Update cluster up/down action lines**

In `cmd/grounds/commands/cluster/up.go`, replace:

```go
fmt.Fprintln(cmd.OutOrStdout(), "✔ Active.")
```

with:

```go
render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Workspace", "Active")
```

In `cmd/grounds/commands/cluster/down.go`, replace:

```go
fmt.Fprintln(cmd.OutOrStdout(), "✔ Paused.")
```

with:

```go
render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Workspace", "Paused")
```

- [ ] **Step 3: Update bundle result output**

Modify `renderBundleResult` in `cmd/grounds/commands/cluster/up.go`:

```go
status := render.StatusOK
summary := fmt.Sprintf("%s with bundle %s in namespace %s", res.State, res.BundleVersion, res.Namespace)
if len(res.Components.Failed) > 0 {
	status = render.StatusWarn
}
render.StatusLine(w, status, "Workspace", summary)
render.DetailLine(w, status, fmt.Sprintf("Components: %d resolved, %d succeeded, %d failed",
	res.Components.Resolved, len(res.Components.Succeeded), len(res.Components.Failed)))
for _, f := range res.Components.Failed {
	render.DetailLine(w, render.StatusError, fmt.Sprintf("%s: %s", f.Name, f.Error))
}
```

- [ ] **Step 4: Update delete warning and result output**

Modify `cmd/grounds/commands/cluster/delete.go`:

```go
render.StatusLine(cmd.OutOrStdout(), render.StatusWarn, "Workspace", "This will permanently delete "+s.Namespace+" and all its data")
```

Replace result output:

```go
case "deleted":
	render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Workspace", "Deleted "+s.Namespace)
case "deleting":
	render.StatusLine(cmd.OutOrStdout(), render.StatusWarn, "Workspace", "Delete is still in progress")
	render.DetailLine(cmd.OutOrStdout(), render.StatusWarn, "Cleanup will continue automatically.")
```

Add import:

```go
"github.com/groundsgg/grounds-cli/internal/render"
```

- [ ] **Step 5: Update paused status warning**

Modify `internal/render/status.go`:

```go
if s.State == "paused" {
	fmt.Fprintln(w)
	StatusLine(w, StatusWarn, "Workspace", "Paused")
	DetailLine(w, StatusWarn, "Next push or "+Command("grounds cluster up")+" resumes it.")
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./cmd/grounds/commands/cluster ./internal/render`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/grounds/commands/cluster/status.go cmd/grounds/commands/cluster/up.go cmd/grounds/commands/cluster/down.go cmd/grounds/commands/cluster/delete.go internal/render/status.go
git commit -m "style: standardize workspace command output"
```

---

### Task 4: Push Output And Error Suggestions

**Files:**
- Modify: `cmd/grounds/commands/push/push.go`
- Modify: `cmd/grounds/commands/push/retry.go`
- Modify: `cmd/grounds/commands/push/list.go`
- Modify: `cmd/grounds/commands/push/push_test.go`

- [ ] **Step 1: Add focused error-copy test for missing Gradle wrapper**

Add to `cmd/grounds/commands/push/push_test.go`:

```go
func TestPushMissingGradleWrapperSuggestsCommand(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(dir)

	cmd := newPush()
	cmd.SetArgs([]string{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing Gradle wrapper error")
	}
	got := err.Error()
	if !strings.Contains(got, "Run `grounds init`") {
		t.Fatalf("error = %q, want command suggestion", got)
	}
	if strings.Contains(got, "→") || strings.Contains(got, "'grounds init'") {
		t.Fatalf("error = %q, should not use arrows or single-quoted commands", got)
	}
}
```

Ensure imports include:

```go
"os"
"strings"
```

- [ ] **Step 2: Run push test and verify it fails**

Run: `go test ./cmd/grounds/commands/push -run TestPushMissingGradleWrapperSuggestsCommand`

Expected: FAIL because current error uses `→` and `'grounds init'`.

- [ ] **Step 3: Update push error suggestions**

Modify `cmd/grounds/commands/push/push.go`:

```go
return fmt.Errorf("%w\n    ! Not a Gradle project? Run %s to scaffold, or cd to your project root.", err, render.Command("grounds init"))
```

and:

```go
return fmt.Errorf("auth refresh failed: %w\n    ! Run %s to re-authenticate.", err, render.Command("grounds login"))
```

Add import:

```go
"github.com/groundsgg/grounds-cli/internal/render"
```

- [ ] **Step 4: Update retry output**

Modify `cmd/grounds/commands/push/retry.go`:

```go
render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Push", "Retry triggered for "+p.ID)
render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "Status: "+p.Status)
```

Add import:

```go
"github.com/groundsgg/grounds-cli/internal/render"
```

- [ ] **Step 5: Update pagination note**

Modify `cmd/grounds/commands/push/list.go`:

```go
if list.NextCursor != "" {
	render.StatusLine(cmd.OutOrStdout(), render.StatusWarn, "Push", "More results are available")
	render.DetailLine(cmd.OutOrStdout(), render.StatusWarn, "Pagination is not available in this CLI version.")
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./cmd/grounds/commands/push`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/grounds/commands/push/push.go cmd/grounds/commands/push/retry.go cmd/grounds/commands/push/list.go cmd/grounds/commands/push/push_test.go
git commit -m "style: improve push command messages"
```

---

### Task 5: Preview Output And Empty States

**Files:**
- Modify: `cmd/grounds/commands/preview/preview.go`
- Modify: `cmd/grounds/commands/preview/preview_test.go`

- [ ] **Step 1: Add unit tests for preview formatting helpers**

Add to `cmd/grounds/commands/preview/preview_test.go`:

```go
func TestPreviewPinSummary(t *testing.T) {
	if got := previewPinSummary(true, "abcdef1234", "plugin-social"); got != "Pinned abcdef12 (plugin-social)" {
		t.Fatalf("previewPinSummary(pin) = %q", got)
	}
	if got := previewPinSummary(false, "abcdef1234", "plugin-social"); got != "Unpinned abcdef12 (plugin-social)" {
		t.Fatalf("previewPinSummary(unpin) = %q", got)
	}
}
```

- [ ] **Step 2: Run preview tests and verify they fail**

Run: `go test ./cmd/grounds/commands/preview`

Expected: FAIL because `previewPinSummary` does not exist.

- [ ] **Step 3: Add preview pin summary helper**

Add to `cmd/grounds/commands/preview/preview.go`:

```go
func previewPinSummary(pin bool, id, manifestName string) string {
	verb := "Pinned"
	if !pin {
		verb = "Unpinned"
	}
	return fmt.Sprintf("%s %s (%s)", verb, shortID(id), manifestName)
}
```

- [ ] **Step 4: Update preview empty state**

Replace:

```go
fmt.Fprintln(cmd.OutOrStdout(), "no preview environments")
```

with:

```go
render.StatusLine(cmd.OutOrStdout(), render.StatusWarn, "Preview", "No preview environments found")
render.DetailLine(cmd.OutOrStdout(), render.StatusWarn, "Run "+render.Command("grounds push --target=staging")+" to create one.")
```

- [ ] **Step 5: Update preview show human output**

Replace the current `fmt.Fprintf` block with:

```go
render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Preview", p.Push.ManifestName+" ("+p.Push.Status+")")
render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "ID: "+p.ID)
render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "Push: "+p.PushID)
render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "Namespace: "+p.Namespace)
render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "Type: "+p.Push.ManifestType)
render.DetailLine(cmd.OutOrStdout(), render.StatusOK, fmt.Sprintf("Pinned: %t", p.Pinned))
render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "Expires: "+formatTime(p.ExpiresAt))
render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "URL: "+p.PublicURL)
```

- [ ] **Step 6: Update preview pin/unpin output**

Replace:

```go
fmt.Fprintf(cmd.OutOrStdout(), "%s %s (%s)\n", verb, shortID(p.ID), p.Push.ManifestName)
```

with:

```go
render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Preview", previewPinSummary(pin, p.ID, p.Push.ManifestName))
```

- [ ] **Step 7: Run tests**

Run: `go test ./cmd/grounds/commands/preview`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add cmd/grounds/commands/preview/preview.go cmd/grounds/commands/preview/preview_test.go
git commit -m "style: improve preview command output"
```

---

### Task 6: DevSpace And Bundle Output

**Files:**
- Modify: `cmd/grounds/commands/devspace/generate.go`
- Modify: `cmd/grounds/commands/devspace/generate_test.go`
- Modify: `cmd/grounds/commands/bundle/list.go`
- Modify: `cmd/grounds/commands/bundle/show.go`

- [ ] **Step 1: Add DevSpace success summary helper test**

Add to `cmd/grounds/commands/devspace/generate_test.go`:

```go
func TestGenerateSuccessSummary(t *testing.T) {
	if got := generateSuccessSummary("./devspace.yaml"); got != "Wrote ./devspace.yaml" {
		t.Fatalf("generateSuccessSummary = %q", got)
	}
}
```

- [ ] **Step 2: Run DevSpace tests and verify they fail**

Run: `go test ./cmd/grounds/commands/devspace`

Expected: FAIL because `generateSuccessSummary` does not exist.

- [ ] **Step 3: Add DevSpace summary helper and update output**

Add to `cmd/grounds/commands/devspace/generate.go`:

```go
func generateSuccessSummary(outputPath string) string {
	return "Wrote " + outputPath
}
```

Replace:

```go
fmt.Fprintf(cmd.OutOrStdout(), "✔ Wrote %d bytes to %s\n", len(yaml), outputPath)
```

with:

```go
render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "DevSpace", generateSuccessSummary(outputPath))
```

Add import:

```go
"github.com/groundsgg/grounds-cli/internal/render"
```

- [ ] **Step 4: Update bundle empty state**

Modify `cmd/grounds/commands/bundle/list.go`:

```go
render.StatusLine(cmd.OutOrStdout(), render.StatusWarn, "Bundle", "No released bundles found")
render.DetailLine(cmd.OutOrStdout(), render.StatusWarn, "Try "+render.Command("grounds bundle show main")+" to inspect the current bundle.")
return nil
```

Add import:

```go
"github.com/groundsgg/grounds-cli/internal/render"
```

- [ ] **Step 5: Update bundle command references**

Modify `cmd/grounds/commands/bundle/list.go` long text so command references use backticks:

```go
the same one `grounds cluster up --bundle main` would track today.
```

Modify `cmd/grounds/commands/bundle/show.go` long text:

```go
component table. <ref> accepts the same shapes as `grounds cluster up --bundle`:
```

- [ ] **Step 6: Run tests**

Run: `go test ./cmd/grounds/commands/devspace ./cmd/grounds/commands/bundle`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/grounds/commands/devspace/generate.go cmd/grounds/commands/devspace/generate_test.go cmd/grounds/commands/bundle/list.go cmd/grounds/commands/bundle/show.go
git commit -m "style: improve devspace and bundle messages"
```

---

### Task 7: Help Text And Examples

**Files:**
- Modify: `cmd/grounds/commands/root.go`
- Modify: `cmd/grounds/commands/version.go`
- Modify: `cmd/grounds/commands/push/push.go`
- Modify: `cmd/grounds/commands/cluster/cluster.go`
- Modify: `cmd/grounds/commands/cluster/status.go`
- Modify: `cmd/grounds/commands/cluster/up.go`
- Modify: `cmd/grounds/commands/preview/preview.go`
- Modify: `cmd/grounds/commands/devspace/devspace.go`
- Modify: `cmd/grounds/commands/bundle/bundle.go`
- Modify: `cmd/grounds/commands/logs/logs.go`

- [ ] **Step 1: Update root and version help**

Modify `cmd/grounds/commands/root.go`:

```go
Short: "Grounds developer platform CLI",
Long:  "Build, deploy, inspect, and troubleshoot Grounds projects from the terminal.",
```

Modify `cmd/grounds/commands/version.go`:

```go
Short:   "Print version information and check for updates",
Example: "  grounds version\n  grounds version --check",
```

- [ ] **Step 2: Add push examples**

Modify `cmd/grounds/commands/push/push.go` root and leaf command examples:

```go
cmd := &cobra.Command{
	Use:   "push",
	Short: "Build and deploy the current project",
	Example: "  grounds push\n  grounds push --target=staging\n  grounds push list --mine",
}
```

For `newPush()`:

```go
Example: "  grounds push\n  grounds push --target=staging",
```

- [ ] **Step 3: Add cluster examples**

Add examples to cluster commands:

```go
Example: "  grounds cluster status\n  grounds cluster up\n  grounds cluster down\n  grounds cluster delete",
```

For `cluster up`:

```go
Example: "  grounds cluster up\n  grounds cluster up --profile=platform\n  grounds cluster up --bundle=0.4.0 --override=./overrides/me.yaml",
```

- [ ] **Step 4: Add preview examples and improve root short**

Modify `NewPreviewCommand`:

```go
Short:   "Manage staging preview environments",
Example: "  grounds preview list\n  grounds preview show <id>\n  grounds preview pin <id>\n  grounds preview unpin <id>",
```

- [ ] **Step 5: Add devspace, bundle, and logs examples**

Add examples:

```go
Example: "  grounds devspace generate plugin-social --bundle main\n  grounds devspace generate plugin-social --override ./me.yaml",
```

```go
Example: "  grounds bundle list\n  grounds bundle show main\n  grounds bundle show 0.4.0",
```

```go
Example: "  grounds logs\n  grounds logs --follow\n  grounds logs deployment <name>",
```

- [ ] **Step 6: Run help smoke checks**

Run:

```bash
go run ./cmd/grounds --help
go run ./cmd/grounds version --help
go run ./cmd/grounds push --help
go run ./cmd/grounds cluster up --help
go run ./cmd/grounds preview --help
```

Expected: each command prints help without errors, examples are visible, and old implementation-leaky wording such as `target=staging deploys` is gone.

- [ ] **Step 7: Run tests**

Run: `go test ./cmd/grounds/...`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add cmd/grounds/commands/root.go cmd/grounds/commands/version.go cmd/grounds/commands/push/push.go cmd/grounds/commands/cluster/cluster.go cmd/grounds/commands/cluster/status.go cmd/grounds/commands/cluster/up.go cmd/grounds/commands/preview/preview.go cmd/grounds/commands/devspace/devspace.go cmd/grounds/commands/bundle/bundle.go cmd/grounds/commands/logs/logs.go
git commit -m "docs: improve cli command help"
```

---

### Task 8: Output Format Contract

**Files:**
- Modify: `cmd/grounds/commands/root.go`
- Modify: command docs/help where needed

- [ ] **Step 1: Decide and encode the contract in root help**

Keep the global `--output` flag, but clarify that structured output is for data commands.

Modify `cmd/grounds/commands/root.go`:

```go
cmd.PersistentFlags().String("output", "table", "output format for data commands: table | json | yaml")
```

- [ ] **Step 2: Audit commands for accidental `--output` confusion**

Run:

```bash
rg -n '"output"|GetString\\("output"\\)|BoolVar.*json|render\\.(JSON|YAML|Table)' cmd/grounds internal/render
```

Expected: identify data commands that already render structured output and action commands that only print human status messages.

- [ ] **Step 3: Add a root help regression test**

Add to `cmd/grounds/commands/root_test.go`:

```go
func TestRootOutputFlagMentionsDataCommands(t *testing.T) {
	root := NewRootCommand()
	flag := root.PersistentFlags().Lookup("output")
	if flag == nil {
		t.Fatal("missing output flag")
	}
	if got := flag.Usage; got != "output format for data commands: table | json | yaml" {
		t.Fatalf("output flag usage = %q", got)
	}
}
```

- [ ] **Step 4: Run root tests**

Run: `go test ./cmd/grounds/commands -run TestRoot`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/grounds/commands/root.go cmd/grounds/commands/root_test.go
git commit -m "docs: clarify output flag scope"
```

---

### Task 9: Final Verification

**Files:**
- No new edits expected.

- [ ] **Step 1: Run full Go tests**

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 2: Run build**

Run: `go build ./cmd/grounds`

Expected: PASS and a local `grounds` binary is produced.

- [ ] **Step 3: Run manual UX smoke checks**

Run:

```bash
./grounds --help
./grounds version
./grounds version --check
./grounds doctor
./grounds init --help
./grounds push --help
./grounds cluster status --help
./grounds preview --help
./grounds bundle --help
./grounds devspace --help
```

Expected:
- Help text is concise and example-driven.
- Human output uses `[✓]`, `[!]`, or `[✗]`.
- Command suggestions use backticks.
- No output uses `→`.
- No successful one-line command adds unnecessary detail lines.

- [ ] **Step 4: Scan for old UX patterns**

Run:

```bash
rg -n '→|✔|⚠|'"'"'grounds [^'"'"']+'"'"'' cmd/grounds internal/render
```

Expected: no remaining old arrow/check/warning symbols in user-facing command output. Single-quoted command references should be gone from help and errors.

- [ ] **Step 5: Commit verification-only fixes if needed**

If Step 4 finds old user-facing copy, update the affected file, rerun:

```bash
go test ./...
go build ./cmd/grounds
```

Then commit:

```bash
git add cmd internal
git commit -m "style: polish remaining cli output"
```

---

## Self-Review

**Spec coverage:** The plan covers shared formatting, action command output, empty states, help examples, `--output` clarity, and final scan-based verification.

**Placeholder scan:** The plan does not use unresolved TODO/TBD language. Each task includes concrete files, snippets, commands, and expected results.

**Type consistency:** The shared helpers are named `StatusKind`, `StatusOK`, `StatusWarn`, `StatusError`, `StatusBadge`, `DetailIcon`, `StatusLine`, `DetailLine`, and `Command`, and all later tasks use those exact names.
