# Minestom Distribution Push Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `grounds push --flavor=minestom --with-local` path that builds a Gradle application distribution with local module overrides and uploads it to Forge as a Minestom server image input.

**Architecture:** This is a coordinated vertical slice across Forge and the CLI. Forge receives first-class `minestom-server` support so it can accept tar/gzip distribution uploads and deploy them as Minecraft workloads. The CLI detects `type: minestom-server` flavors, generates a temporary Gradle composite init script from `workspace.yaml`, builds the configured distribution artifact, and uploads it directly to Forge's existing multipart `/v1/pushes` endpoint.

**Tech Stack:** Go 1.26, Cobra, Gradle wrapper invocation, YAML parsing, multipart HTTP uploads, TypeScript, Fastify, Zod, Vitest, Prisma mocks.

---

## File Structure

- `grounds-forge/src/buildrunner/manifest.ts`: add `minestom-server` as an internal normalized manifest type while keeping public `minestom`.
- `grounds-forge/src/buildrunner/baseImageCatalog.ts`: seed the `minestom` base image catalog entry.
- `grounds-forge/src/buildrunner/baseImages.ts`: keep fallback/static base image resolution aligned with the catalog.
- `grounds-forge/src/buildrunner/templates.ts`: add Dockerfile templates for Minestom distributions.
- `grounds-forge/src/workloads/types.ts`: make `minestom-server` a Minecraft workload with port 25565 defaults.
- `grounds-forge/src/workloads/renderer.ts`: allow the new workload type through Minecraft routing branches.
- `grounds-forge/src/routes/pushes.ts`: allow tar/gzip uploads for `minestom-server` and keep rejecting generic `service` bundles.
- `grounds-forge/tests/*.test.ts`: cover manifest normalization, base image seed, templates, push upload, and renderer behavior.
- `grounds-cli/internal/minestom/manifest.go`: parse the selected Minestom flavor from `grounds.yaml`.
- `grounds-cli/internal/minestom/composition.go`: resolve local module overrides, generate Gradle init scripts, and resolve distribution artifacts.
- `grounds-cli/internal/minestom/manifest_test.go`: parser tests.
- `grounds-cli/internal/minestom/composition_test.go`: local override, init script, and artifact tests.
- `grounds-cli/internal/api/push.go`: add multipart push creation.
- `grounds-cli/internal/api/push_test.go`: verify multipart request shape.
- `grounds-cli/cmd/grounds/commands/push/push.go`: dispatch Minestom flavors to the new build/upload path while preserving existing Gradle `groundsPush` behavior.
- `grounds-cli/cmd/grounds/commands/push/push_test.go`: verify Minestom build args/upload dispatch and non-Minestom behavior.

## Scope Check

The spec spans two repos, but the work is one deployable vertical slice: Forge must understand the artifact before the CLI can verify a real push. Keep this as one PR stack or two linked PRs opened together. Do not add Portal UI, pod sync, or version-composition UI in this plan.

### Task 1: Forge Manifest Type

**Files:**
- Modify: `grounds-forge/tests/manifest.test.ts`
- Modify: `grounds-forge/src/buildrunner/manifest.ts`

- [ ] **Step 1: Write failing manifest normalization tests**

In `grounds-forge/tests/manifest.test.ts`, update the `normalizes new public single-runtime aliases` Minestom case:

```ts
{
  inputType: "minestom",
  runtimeType: "minestom-server",
  publicType: "minestom",
  flavorKey: "minestom",
  baseImage: "minestom",
},
```

Add a direct internal type test:

```ts
it("accepts minestom-server as the internal Minestom runtime type", () => {
  const manifest = parseManifest({
    name: "minestom-demo",
    type: "minestom-server",
    baseImage: "minestom",
  });

  expect(selectManifestFlavor(manifest, undefined)).toMatchObject({
    appName: "minestom-demo",
    flavorKey: "minestom",
    runtime: {
      name: "minestom-demo",
      type: "minestom-server",
      publicType: "minestom",
      baseImage: "minestom",
    },
  });
});
```

- [ ] **Step 2: Run test to verify failure**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test -- tests/manifest.test.ts
```

Expected: FAIL because `minestom-server` is not in the public type schema or normalized type union.

- [ ] **Step 3: Implement manifest type support**

In `grounds-forge/src/buildrunner/manifest.ts`, add `minestom-server` to `PublicManifestType`, `publicManifestTypeSchema`, and `NormalizedManifestType`:

```ts
export type PublicManifestType =
  | "paper"
  | "velocity"
  | "gamemode"
  | "minestom"
  | "minestom-server"
  | "service"
  | "plugin-paper"
  | "plugin-velocity";

export type NormalizedManifestType =
  | "plugin-paper"
  | "plugin-velocity"
  | "gamemode"
  | "minestom-server"
  | "service";
```

Add `"minestom-server"` to the `z.enum([...])` list.

Update `normalizeManifestType`:

```ts
case "minestom":
case "minestom-server":
  return { publicType: "minestom", internalType: "minestom-server" };
```

- [ ] **Step 4: Run test to verify pass**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test -- tests/manifest.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit Forge manifest type**

```bash
cd /home/lukas/grounds/grounds-forge
git add src/buildrunner/manifest.ts tests/manifest.test.ts
git commit -S -m "feat: add minestom server manifest type"
```

### Task 2: Forge Minestom Base Image, Template, and Workload Defaults

**Files:**
- Modify: `grounds-forge/tests/templates.test.ts`
- Modify: `grounds-forge/tests/baseImages.test.ts`
- Modify: `grounds-forge/tests/baseImageCatalogSeed.test.ts`
- Modify: `grounds-forge/tests/renderer.test.ts`
- Modify: `grounds-forge/src/buildrunner/templates.ts`
- Modify: `grounds-forge/src/buildrunner/baseImages.ts`
- Modify: `grounds-forge/src/buildrunner/baseImageCatalog.ts`
- Modify: `grounds-forge/src/workloads/types.ts`
- Modify: `grounds-forge/src/workloads/renderer.ts`

- [ ] **Step 1: Write failing template tests**

In `grounds-forge/tests/templates.test.ts`, add:

```ts
it("renders minestom-server as a Gradle application distribution", () => {
  const result = renderDockerfile("minestom-server", "eclipse-temurin:21-jre-alpine");
  expect(result).toBe(
    'FROM eclipse-temurin:21-jre-alpine\nCOPY app/ /app/\nENTRYPOINT ["/app/bin/app"]\n',
  );
});

it("renders minestom-server bundle as an extracted Gradle application distribution", () => {
  const result = renderDockerfile("minestom-server", "eclipse-temurin:21-jre-alpine", {
    bundle: true,
  });
  expect(result).toBe(
    'FROM eclipse-temurin:21-jre-alpine\nCOPY app/ /app/\nENTRYPOINT ["/app/bin/app"]\n',
  );
});
```

Add `"minestom-server"` to the no-placeholder leakage loop.

- [ ] **Step 2: Write failing base image tests**

In `grounds-forge/tests/baseImages.test.ts`, add:

```ts
it("minestom base image resolves only for type=minestom-server", () => {
  expect(resolveBaseImage("minestom", "minestom-server").type).toBe("minestom-server");
  expect(() => resolveBaseImage("minestom", "service")).toThrow(/minestom-server/);
});
```

In `grounds-forge/tests/baseImageCatalogSeed.test.ts`, add an assertion in the seed test that the source keys include `minestom`, and assert:

```ts
expect(upsertsByKey.minestom.create).toMatchObject({
  key: "minestom",
  displayName: "Minestom",
  manifestType: "minestom-server",
  registryHost: "docker.io",
  repository: "library/eclipse-temurin",
  defaultChannel: "stable",
});
```

- [ ] **Step 3: Write failing renderer test**

In `grounds-forge/tests/renderer.test.ts`, add:

```ts
it("renders minestom-server as a Minecraft workload", () => {
  const out = render({
    ...baseInput,
    manifestType: "minestom-server",
    manifestBaseImage: "minestom",
  });

  expect(out.publicUrl).toBe("minecraft://sample-hendrik.mc.grnds.io");
  expect(out.service.spec?.ports?.[0]?.port).toBe(25565);
  expect(out.deployment.spec?.template.spec?.containers?.[0]?.ports?.[0]?.containerPort).toBe(25565);
});
```

- [ ] **Step 4: Run tests to verify failure**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test -- tests/templates.test.ts tests/baseImages.test.ts tests/baseImageCatalogSeed.test.ts tests/renderer.test.ts
```

Expected: FAIL on unknown `minestom-server` template/type.

- [ ] **Step 5: Implement Forge runtime support**

In `grounds-forge/src/buildrunner/templates.ts`, add `minestom-server` to both template maps:

```ts
"minestom-server":
  'FROM ${FULL_IMAGE}\nCOPY app/ /app/\nENTRYPOINT ["/app/bin/app"]\n',
```

In `grounds-forge/src/buildrunner/baseImages.ts`, extend the type union and add:

```ts
minestom: {
  image: "eclipse-temurin",
  tag: "21-jre-alpine",
  type: "minestom-server",
},
```

In `grounds-forge/src/buildrunner/baseImageCatalog.ts`, add a `DEFAULT_BASE_IMAGE_SOURCES` entry:

```ts
{
  key: "minestom",
  displayName: "Minestom",
  manifestType: "minestom-server",
  registryHost: "docker.io",
  repository: "library/eclipse-temurin",
  defaultChannel: "stable",
  discoveryKind: "manual",
  versions: [{ version: "21-jre-alpine", tag: "21-jre-alpine", state: "stable" }],
},
```

In `grounds-forge/src/workloads/types.ts`, extend `ManifestType` and defaults:

```ts
export type ManifestType =
  | "plugin-paper"
  | "plugin-velocity"
  | "gamemode"
  | "minestom-server"
  | "service";
```

```ts
"minestom-server": { requests: { cpu: "500m", memory: "1Gi" }, limits: { cpu: "4", memory: "2Gi" }, containerPort: 25565, portProtocol: "TCP", probe: { initialDelaySeconds: 20, periodSeconds: 10 } },
```

Update `isMinecraftType`:

```ts
return t === "plugin-paper" || t === "plugin-velocity" || t === "gamemode" || t === "minestom-server";
```

In `grounds-forge/src/workloads/renderer.ts`, update gamemode-only branches only where Agones Fleet behavior is intended. Do not place `minestom-server` into Fleet/Agones logic unless the spec later changes. It should get Minecraft service/routing via `isMinecraftType`, but normal Deployment semantics like `plugin-paper`.

- [ ] **Step 6: Run tests to verify pass**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test -- tests/templates.test.ts tests/baseImages.test.ts tests/baseImageCatalogSeed.test.ts tests/renderer.test.ts
```

Expected: PASS.

- [ ] **Step 7: Commit Forge runtime support**

```bash
cd /home/lukas/grounds/grounds-forge
git add src/buildrunner/templates.ts src/buildrunner/baseImages.ts src/buildrunner/baseImageCatalog.ts src/workloads/types.ts src/workloads/renderer.ts tests/templates.test.ts tests/baseImages.test.ts tests/baseImageCatalogSeed.test.ts tests/renderer.test.ts
git commit -S -m "feat: support minestom server runtime"
```

### Task 3: Forge Minestom Distribution Upload

**Files:**
- Modify: `grounds-forge/tests/pushes.post.test.ts`
- Modify: `grounds-forge/src/routes/pushes.ts`

- [ ] **Step 1: Write failing Minestom upload test**

In `grounds-forge/tests/pushes.post.test.ts`, add under the artifact kind tests:

```ts
it("accepts tar.gz upload for minestom-server", async () => {
  const prismaMock = makePrismaMock();
  const blobStoreMock = makeBlobStoreMock();
  const kubeConfigMock = makeKubeConfigMock();

  const app = await buildApp({
    prisma: prismaMock as never,
    blobStore: blobStoreMock as never,
    kubeConfig: kubeConfigMock as never,
    authRequired: makeAuthRequired(makeUser()),
    events: new EventEmitter(),
    metrics: makeFakeMetrics(),
  });

  const manifest = {
    name: "minestom-demo",
    type: "minestom-server",
    baseImage: "minestom",
  };

  const { body, headers } = buildMultipart(
    { manifest: JSON.stringify(manifest), jar: GZIP_BYTES },
    "app.tar.gz",
  );

  const res = await app.inject({
    method: "POST",
    url: "/v1/pushes",
    headers,
    payload: body,
  });

  expect(res.statusCode).toBe(202);
  const createData = (prismaMock.push.create as ReturnType<typeof vi.fn>).mock.calls[0]![0]!.data;
  expect(createData.artifactKind).toBe("bundle");
  expect(createData.manifest).toMatchObject({
    name: "minestom-demo",
    type: "minestom-server",
    publicType: "minestom",
    baseImage: "minestom",
  });
  const job = (kubeConfigMock._createNamespacedJob as ReturnType<typeof vi.fn>).mock.calls[0]![0]!.body;
  const dockerfile = JSON.stringify(job);
  expect(dockerfile).toContain("/app/bin/app");
});
```

- [ ] **Step 2: Run test to verify failure**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test -- tests/pushes.post.test.ts
```

Expected: FAIL until Task 1 and Task 2 are complete, then PASS without weakening service bundle rejection. If it fails because the route rejects `service` only, inspect the failure before editing.

- [ ] **Step 3: Keep upload guard scoped**

In `grounds-forge/src/routes/pushes.ts`, the guard should remain:

```ts
if (isBundle && runtimeManifest.type === "service") {
  metrics.pushTotal.inc({ target, result: "rejected" });
  await cleanupTemp();
  return reply.code(400).send({
    error: "bundle_unsupported_type",
    message: "type 'service' does not support multi-plugin bundles",
  });
}
```

Do not change it to reject all non-plugin types. With `minestom-server` normalized separately, this guard allows Minestom distributions and still blocks generic services.

- [ ] **Step 4: Run push route test to verify pass**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test -- tests/pushes.post.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit Forge upload test**

```bash
cd /home/lukas/grounds/grounds-forge
git add src/routes/pushes.ts tests/pushes.post.test.ts
git commit -S -m "test: cover minestom distribution uploads"
```

### Task 4: CLI Minestom Manifest Parser

**Files:**
- Create: `grounds-cli/internal/minestom/manifest.go`
- Create: `grounds-cli/internal/minestom/manifest_test.go`

- [ ] **Step 1: Write failing parser tests**

Create `grounds-cli/internal/minestom/manifest_test.go`:

```go
package minestom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPushManifestSelectsMinestomFlavor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grounds.yaml")
	writeTestFile(t, path, `
name: minigame-bedwars
flavors:
  minestom:
    type: minestom-server
    baseImage: minestom
    build:
      task: :examples:minigame-agones:distTar
      artifact: examples/minigame-agones/build/distributions/*.tar
    modules:
      - id: plugin-config
        variant: minestom
        source: github:groundsgg/plugin-config@v0.4.0:plugin-config-minestom.jar
      - id: plugin-agones
        variant: minestom
        source: github:groundsgg/plugin-agones@v0.2.0:plugin-agones-minestom.jar
`)

	manifest, err := LoadPushManifest(path, "minestom")
	if err != nil {
		t.Fatalf("LoadPushManifest() error = %v", err)
	}

	if manifest.Name != "minigame-bedwars" {
		t.Fatalf("Name = %q", manifest.Name)
	}
	if manifest.FlavorKey != "minestom" {
		t.Fatalf("FlavorKey = %q", manifest.FlavorKey)
	}
	if manifest.Runtime.Type != "minestom-server" {
		t.Fatalf("Runtime.Type = %q", manifest.Runtime.Type)
	}
	if manifest.Runtime.Build.Task != ":examples:minigame-agones:distTar" {
		t.Fatalf("Build.Task = %q", manifest.Runtime.Build.Task)
	}
	if len(manifest.Runtime.Modules) != 2 {
		t.Fatalf("Modules len = %d", len(manifest.Runtime.Modules))
	}
}

func TestLoadPushManifestRejectsMissingMinestomBuild(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grounds.yaml")
	writeTestFile(t, path, `
name: minigame-bedwars
flavors:
  minestom:
    type: minestom-server
    baseImage: minestom
`)

	_, err := LoadPushManifest(path, "minestom")
	if err == nil || !strings.Contains(err.Error(), "build.task") || !strings.Contains(err.Error(), "build.artifact") {
		t.Fatalf("LoadPushManifest() error = %v, want missing build fields", err)
	}
}

func TestLoadPushManifestReturnsNonMinestomForPaperFlavor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "grounds.yaml")
	writeTestFile(t, path, `
name: plugin-config
flavors:
  paper:
    type: paper
    baseImage: paper
`)

	manifest, err := LoadPushManifest(path, "paper")
	if err != nil {
		t.Fatalf("LoadPushManifest() error = %v", err)
	}
	if manifest.IsMinestomServer() {
		t.Fatalf("paper flavor should not be a Minestom server")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
GOMODCACHE=/tmp/grounds-go/pkg/mod GOCACHE=/tmp/grounds-go/cache GOPATH=/tmp/grounds-go go test ./internal/minestom
```

Expected: FAIL because package `internal/minestom` does not exist.

- [ ] **Step 3: Implement parser**

Create `grounds-cli/internal/minestom/manifest.go`:

```go
package minestom

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type PushManifest struct {
	Name      string
	FlavorKey string
	Runtime   Runtime
	Full      map[string]any
}

type Runtime struct {
	Type      string   `yaml:"type" json:"type"`
	PublicType string  `yaml:"-" json:"publicType,omitempty"`
	BaseImage string   `yaml:"baseImage" json:"baseImage"`
	Build     Build    `yaml:"build" json:"-"`
	Modules   []Module `yaml:"modules" json:"-"`
}

type Build struct {
	Task     string `yaml:"task"`
	Artifact string `yaml:"artifact"`
}

type Module struct {
	ID      string `yaml:"id"`
	Variant string `yaml:"variant"`
	Source  string `yaml:"source"`
}

func (m PushManifest) IsMinestomServer() bool {
	return m.Runtime.Type == "minestom-server"
}

func LoadPushManifest(path, flavor string) (*PushManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Name    string             `yaml:"name"`
		Type    string             `yaml:"type"`
		BaseImage string            `yaml:"baseImage"`
		Build   Build              `yaml:"build"`
		Modules []Module           `yaml:"modules"`
		Flavors map[string]Runtime `yaml:"flavors"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	var full map[string]any
	if err := yaml.Unmarshal(raw, &full); err != nil {
		return nil, err
	}
	if len(doc.Flavors) > 0 {
		key := strings.TrimSpace(flavor)
		if key == "" {
			return nil, fmt.Errorf("grounds.yaml: flavor selection required (available=%s)", availableFlavorKeys(doc.Flavors))
		}
		selected, ok := doc.Flavors[key]
		if !ok {
			return nil, fmt.Errorf("grounds.yaml: unknown flavor %q (available=%s)", key, availableFlavorKeys(doc.Flavors))
		}
		if selected.Type == "minestom" {
			selected.Type = "minestom-server"
		}
		selected.PublicType = publicTypeFor(selected.Type)
		if err := validateRuntime(key, selected); err != nil {
			return nil, err
		}
		return &PushManifest{Name: strings.TrimSpace(doc.Name), FlavorKey: key, Runtime: selected, Full: full}, nil
	}
	runtime := Runtime{
		Type:      strings.TrimSpace(doc.Type),
		BaseImage: strings.TrimSpace(doc.BaseImage),
		Build:     doc.Build,
		Modules:   doc.Modules,
	}
	if runtime.Type == "minestom" {
		runtime.Type = "minestom-server"
	}
	runtime.PublicType = publicTypeFor(runtime.Type)
	if err := validateRuntime(runtime.PublicType, runtime); err != nil {
		return nil, err
	}
	return &PushManifest{Name: strings.TrimSpace(doc.Name), FlavorKey: runtime.PublicType, Runtime: runtime, Full: full}, nil
}

func validateRuntime(flavor string, runtime Runtime) error {
	if runtime.Type != "minestom-server" {
		return nil
	}
	var missing []string
	if strings.TrimSpace(runtime.Build.Task) == "" {
		missing = append(missing, "build.task")
	}
	if strings.TrimSpace(runtime.Build.Artifact) == "" {
		missing = append(missing, "build.artifact")
	}
	if len(runtime.Modules) == 0 {
		missing = append(missing, "modules")
	}
	if len(missing) > 0 {
		return fmt.Errorf("grounds.yaml: minestom flavor %q missing %s", flavor, strings.Join(missing, ", "))
	}
	for i, module := range runtime.Modules {
		if strings.TrimSpace(module.ID) == "" {
			return fmt.Errorf("grounds.yaml: minestom module at index %d missing id", i)
		}
		if strings.TrimSpace(module.Source) == "" {
			return fmt.Errorf("grounds.yaml: minestom module %q missing source", module.ID)
		}
	}
	return nil
}

func publicTypeFor(runtimeType string) string {
	if runtimeType == "minestom-server" {
		return "minestom"
	}
	return runtimeType
}

func availableFlavorKeys(flavors map[string]Runtime) string {
	keys := make([]string, 0, len(flavors))
	for key := range flavors {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
```

- [ ] **Step 4: Run parser tests to verify pass**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
GOMODCACHE=/tmp/grounds-go/pkg/mod GOCACHE=/tmp/grounds-go/cache GOPATH=/tmp/grounds-go go test ./internal/minestom
```

Expected: PASS.

- [ ] **Step 5: Commit CLI parser**

```bash
cd /home/lukas/grounds/grounds-cli
git add internal/minestom/manifest.go internal/minestom/manifest_test.go
git commit -S -m "feat: parse minestom push manifests"
```

### Task 5: CLI Local Module Composition

**Files:**
- Create: `grounds-cli/internal/minestom/composition.go`
- Create/modify: `grounds-cli/internal/minestom/composition_test.go`
- Modify: `grounds-cli/internal/workspace/config.go`
- Modify: `grounds-cli/internal/workspace/config_test.go`

- [ ] **Step 1: Write failing workspace metadata test**

In `grounds-cli/internal/workspace/config_test.go`, add:

```go
func TestEntryForVariantIncludesCompositeMetadata(t *testing.T) {
	cfg := &Config{Repos: map[string]Repo{
		"plugin-agones": {
			Path: "/repos/plugin-agones",
			Variants: map[string]Variant{
				"minestom": {
					Artifact: "minestom/build/libs/*.jar",
					Build: "./gradlew :minestom:build",
					Enabled: true,
					Module: "gg.grounds:plugin-agones-minestom",
					Project: ":minestom",
				},
			},
		},
	}}

	entry, ok := cfg.EntryForVariant("plugin-agones", "minestom")
	if !ok {
		t.Fatal("EntryForVariant() ok = false")
	}
	if entry.Module != "gg.grounds:plugin-agones-minestom" {
		t.Fatalf("Module = %q", entry.Module)
	}
	if entry.Project != ":minestom" {
		t.Fatalf("Project = %q", entry.Project)
	}
}
```

- [ ] **Step 2: Implement workspace metadata fields**

In `grounds-cli/internal/workspace/config.go`, extend `Variant` and `ResolvedEntry`:

```go
type Variant struct {
	Artifact string `yaml:"artifact,omitempty"`
	Build    string `yaml:"build,omitempty"`
	Enabled  bool   `yaml:"enabled"`
	Module   string `yaml:"module,omitempty"`
	Project  string `yaml:"project,omitempty"`
}

type ResolvedEntry struct {
	Path     string
	Artifact string
	Build    string
	Enabled  bool
	Module   string
	Project  string
}
```

When returning a variant entry, set `Module: v.Module` and `Project: v.Project`.

- [ ] **Step 3: Write failing composition tests**

Create `grounds-cli/internal/minestom/composition_test.go`:

```go
package minestom

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	internalworkspace "github.com/groundsgg/grounds-cli/internal/workspace"
)

func TestResolveLocalModulesSelectsEnabledMinestomEntries(t *testing.T) {
	repoA := t.TempDir()
	repoB := t.TempDir()
	manifest := PushManifest{Runtime: Runtime{Modules: []Module{
		{ID: "plugin-agones", Variant: "minestom", Source: "github:groundsgg/plugin-agones@v0.2.0:plugin-agones-minestom.jar"},
		{ID: "plugin-config", Variant: "minestom", Source: "github:groundsgg/plugin-config@v0.4.0:plugin-config-minestom.jar"},
	}}}
	cfg := &internalworkspace.Config{Repos: map[string]internalworkspace.Repo{
		"plugin-agones": {Path: repoA, Variants: map[string]internalworkspace.Variant{
			"minestom": {Enabled: true, Module: "gg.grounds:plugin-agones-minestom", Project: ":minestom"},
		}},
		"plugin-config": {Path: repoB, Variants: map[string]internalworkspace.Variant{
			"minestom": {Enabled: false, Module: "gg.grounds:plugin-config-minestom", Project: ":minestom"},
		}},
	}}

	plan, err := ResolveLocalModules(context.Background(), manifest, cfg, ResolveOptions{WithLocal: true})
	if err != nil {
		t.Fatalf("ResolveLocalModules() error = %v", err)
	}
	if len(plan.LocalModules) != 1 {
		t.Fatalf("LocalModules len = %d", len(plan.LocalModules))
	}
	if plan.LocalModules[0].ID != "plugin-agones" {
		t.Fatalf("local module = %#v", plan.LocalModules[0])
	}
	if len(plan.EffectivePluginSources) != 2 {
		t.Fatalf("EffectivePluginSources len = %d", len(plan.EffectivePluginSources))
	}
}

func TestResolveLocalModulesHonorsExplicitLocalIDs(t *testing.T) {
	repo := t.TempDir()
	manifest := PushManifest{Runtime: Runtime{Modules: []Module{
		{ID: "plugin-agones", Variant: "minestom", Source: "github:groundsgg/plugin-agones@v0.2.0:plugin-agones-minestom.jar"},
	}}}
	cfg := &internalworkspace.Config{Repos: map[string]internalworkspace.Repo{
		"plugin-agones": {Path: repo, Variants: map[string]internalworkspace.Variant{
			"minestom": {Enabled: false, Module: "gg.grounds:plugin-agones-minestom", Project: ":minestom"},
		}},
	}}

	plan, err := ResolveLocalModules(context.Background(), manifest, cfg, ResolveOptions{LocalIDs: []string{"plugin-agones"}})
	if err != nil {
		t.Fatalf("ResolveLocalModules() error = %v", err)
	}
	if len(plan.LocalModules) != 1 {
		t.Fatalf("LocalModules len = %d", len(plan.LocalModules))
	}
}

func TestWriteCompositeInitScriptIsDeterministicAndUnique(t *testing.T) {
	dir := t.TempDir()
	plan := &LocalPlan{LocalModules: []LocalModule{
		{ID: "plugin-b", Path: filepath.Join(dir, "b")},
		{ID: "plugin-a", Path: filepath.Join(dir, "a")},
		{ID: "plugin-a-duplicate", Path: filepath.Join(dir, "a")},
	}}

	path, err := WriteCompositeInitScript(plan)
	if err != nil {
		t.Fatalf("WriteCompositeInitScript() error = %v", err)
	}
	defer os.Remove(path)

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	got := string(raw)
	firstA := strings.Index(got, "includeBuild("+quoteForTest(filepath.Join(dir, "a"))+")")
	firstB := strings.Index(got, "includeBuild("+quoteForTest(filepath.Join(dir, "b"))+")")
	if firstA < 0 || firstB < 0 || firstA > firstB {
		t.Fatalf("init script = %s", got)
	}
	if strings.Count(got, filepath.ToSlash(filepath.Join(dir, "a"))) != 1 {
		t.Fatalf("init script should include path a once: %s", got)
	}
}

func TestResolveDistributionArtifactPicksNewestTar(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "build", "distributions", "old.tar")
	newPath := filepath.Join(dir, "build", "distributions", "new.tar")
	mkdirTestDir(t, filepath.Dir(oldPath))
	writeTestFile(t, oldPath, "old")
	writeTestFile(t, newPath, "new")

	got, err := ResolveDistributionArtifact(dir, "build/distributions/*.tar")
	if err != nil {
		t.Fatalf("ResolveDistributionArtifact() error = %v", err)
	}
	if got != newPath {
		t.Fatalf("artifact = %q, want %q", got, newPath)
	}
}

func mkdirTestDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func quoteForTest(value string) string {
	return "\"" + strings.ReplaceAll(filepath.ToSlash(value), "\"", "\\\"") + "\""
}
```

- [ ] **Step 4: Run tests to verify failure**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
GOMODCACHE=/tmp/grounds-go/pkg/mod GOCACHE=/tmp/grounds-go/cache GOPATH=/tmp/grounds-go go test ./internal/workspace ./internal/minestom
```

Expected: FAIL because composition functions do not exist.

- [ ] **Step 5: Implement composition**

Create `grounds-cli/internal/minestom/composition.go` with:

```go
package minestom

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	internalworkspace "github.com/groundsgg/grounds-cli/internal/workspace"
)

type ResolveOptions struct {
	LocalIDs  []string
	WithLocal bool
}

type LocalPlan struct {
	LocalModules           []LocalModule
	EffectivePluginSources []internalworkspace.EffectiveSource
}

type LocalModule struct {
	ID      string
	Variant string
	Path    string
	Module  string
	Project string
}

func ResolveLocalModules(ctx context.Context, manifest PushManifest, cfg *internalworkspace.Config, opts ResolveOptions) (*LocalPlan, error) {
	_ = ctx
	explicitIDs := internalworkspace.NormalizeLocalIDs(opts.LocalIDs)
	explicit := map[string]bool{}
	for _, id := range explicitIDs {
		explicit[id] = true
		if !manifestHasModule(manifest.Runtime.Modules, id) {
			return nil, fmt.Errorf("--local module %q not found in grounds.yaml", id)
		}
	}

	plan := &LocalPlan{}
	for _, module := range manifest.Runtime.Modules {
		entry, ok := cfg.EntryForVariant(module.ID, module.Variant)
		selected := explicit[module.ID] || (opts.WithLocal && ok && entry.Enabled)
		if selected {
			if !ok {
				return nil, fmt.Errorf("local workspace entry for %q variant %q not found", module.ID, module.Variant)
			}
			if strings.TrimSpace(entry.Path) == "" {
				return nil, fmt.Errorf("local workspace entry for %q has no path", module.ID)
			}
			plan.LocalModules = append(plan.LocalModules, LocalModule{
				ID:      module.ID,
				Variant: module.Variant,
				Path:    entry.Path,
				Module:  entry.Module,
				Project: entry.Project,
			})
			plan.EffectivePluginSources = append(plan.EffectivePluginSources, internalworkspace.EffectiveSource{
				ID:            module.ID,
				Variant:       module.Variant,
				Effective:     "local",
				DefaultSource: module.Source,
			})
			continue
		}
		plan.EffectivePluginSources = append(plan.EffectivePluginSources, internalworkspace.EffectiveSource{
			ID:        module.ID,
			Variant:   module.Variant,
			Effective: "release",
			Source:    module.Source,
		})
	}
	return plan, nil
}

func WriteCompositeInitScript(plan *LocalPlan) (string, error) {
	paths := map[string]bool{}
	for _, module := range plan.LocalModules {
		if module.Path != "" {
			paths[module.Path] = true
		}
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		ordered = append(ordered, filepath.ToSlash(abs))
	}
	sort.Strings(ordered)

	var b strings.Builder
	b.WriteString("settingsEvaluated {\n")
	for _, path := range ordered {
		b.WriteString("    includeBuild(")
		b.WriteString(kotlinString(path))
		b.WriteString(")\n")
	}
	b.WriteString("}\n")

	file, err := os.CreateTemp("", "grounds-minestom-composites-*.gradle.kts")
	if err != nil {
		return "", err
	}
	if _, err := file.WriteString(b.String()); err != nil {
		file.Close()
		os.Remove(file.Name())
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func ResolveDistributionArtifact(projectRoot, pattern string) (string, error) {
	if strings.TrimSpace(pattern) == "" {
		return "", fmt.Errorf("artifact glob is empty")
	}
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(projectRoot, filepath.FromSlash(pattern))
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	candidates := matches[:0]
	for _, match := range matches {
		ext := strings.ToLower(filepath.Ext(match))
		if ext == ".tar" || ext == ".gz" || strings.HasSuffix(strings.ToLower(match), ".tar.gz") {
			candidates = append(candidates, match)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("expected at least one distribution artifact for %s, found 0", pattern)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, leftErr := os.Stat(candidates[i])
		right, rightErr := os.Stat(candidates[j])
		if leftErr == nil && rightErr == nil && !left.ModTime().Equal(right.ModTime()) {
			return left.ModTime().After(right.ModTime())
		}
		return candidates[i] < candidates[j]
	})
	return candidates[0], nil
}

func ArtifactSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func manifestHasModule(modules []Module, id string) bool {
	for _, module := range modules {
		if module.ID == id {
			return true
		}
	}
	return false
}

func kotlinString(value string) string {
	return "\"" + strings.ReplaceAll(value, "\"", "\\\"") + "\""
}
```

- [ ] **Step 6: Run tests to verify pass**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
GOMODCACHE=/tmp/grounds-go/pkg/mod GOCACHE=/tmp/grounds-go/cache GOPATH=/tmp/grounds-go go test ./internal/workspace ./internal/minestom
```

Expected: PASS.

- [ ] **Step 7: Commit CLI composition**

```bash
cd /home/lukas/grounds/grounds-cli
git add internal/workspace/config.go internal/workspace/config_test.go internal/minestom/composition.go internal/minestom/composition_test.go
git commit -S -m "feat: resolve minestom local modules"
```

### Task 6: CLI Multipart Push API

**Files:**
- Modify: `grounds-cli/internal/api/push.go`
- Modify: `grounds-cli/internal/api/push_test.go`

- [ ] **Step 1: Write failing multipart API test**

In `grounds-cli/internal/api/push_test.go`, add:

```go
func TestCreatePushSendsMultipartDistribution(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "app.tar")
	if err := os.WriteFile(artifact, []byte{0x1f, 0x8b, 0x08, 0x00}, 0o600); err != nil {
		t.Fatalf("WriteFile(artifact) error = %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/pushes" {
			t.Fatalf("got %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("projectId"); got != "project-1" {
			t.Fatalf("projectId = %q", got)
		}
		if got := r.URL.Query().Get("force"); got != "true" {
			t.Fatalf("force = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		if got := r.FormValue("target"); got != "staging" {
			t.Fatalf("target = %q", got)
		}
		if got := r.FormValue("flavor"); got != "minestom" {
			t.Fatalf("flavor = %q", got)
		}
		if !strings.Contains(r.FormValue("manifest"), `"type":"minestom-server"`) {
			t.Fatalf("manifest = %s", r.FormValue("manifest"))
		}
		file, header, err := r.FormFile("jar")
		if err != nil {
			t.Fatalf("FormFile(jar) error = %v", err)
		}
		defer file.Close()
		if header.Filename != "app.tar" {
			t.Fatalf("filename = %q", header.Filename)
		}
		json.NewEncoder(w).Encode(map[string]any{"pushId": "push-1", "status": "building", "logsUrl": "/v1/pushes/push-1/logs"})
	}))
	defer srv.Close()

	c := New(srv.URL, staticToken("test-token"))
	c.ProjectID = "project-1"
	res, err := c.CreatePush(context.Background(), CreatePushRequest{
		Target: "staging",
		Flavor: "minestom",
		Force: true,
		Manifest: map[string]any{
			"name": "minestom-demo",
			"type": "minestom-server",
			"baseImage": "minestom",
		},
		ArtifactPath: artifact,
	})
	if err != nil {
		t.Fatalf("CreatePush() error = %v", err)
	}
	if res.PushID != "push-1" || res.Status != "building" {
		t.Fatalf("response = %#v", res)
	}
}

type staticToken string

func (s staticToken) Token(context.Context) (string, error) { return string(s), nil }
```

Add missing imports: `os`, `path/filepath`, and `strings`.

- [ ] **Step 2: Run test to verify failure**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
GOMODCACHE=/tmp/grounds-go/pkg/mod GOCACHE=/tmp/grounds-go/cache GOPATH=/tmp/grounds-go go test ./internal/api
```

Expected: FAIL because `CreatePush` does not exist.

- [ ] **Step 3: Implement multipart upload**

In `grounds-cli/internal/api/push.go`, add:

```go
type CreatePushRequest struct {
	Target                 string
	Flavor                 string
	Force                  bool
	Manifest               any
	EffectivePluginSources any
	ArtifactPath           string
}

type CreatePushResponse struct {
	PushID   string `json:"pushId"`
	Status   string `json:"status"`
	Reused   bool   `json:"reused,omitempty"`
	FlavorKey string `json:"flavorKey,omitempty"`
	LogsURL  string `json:"logsUrl,omitempty"`
}
```

Implement:

```go
func (c *Client) CreatePush(ctx context.Context, req CreatePushRequest) (*CreatePushResponse, error) {
	if req.ArtifactPath == "" {
		return nil, fmt.Errorf("artifact path is required")
	}
	file, err := os.Open(req.ArtifactPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("target", req.Target); err != nil {
		return nil, err
	}
	if req.Flavor != "" {
		if err := writer.WriteField("flavor", req.Flavor); err != nil {
			return nil, err
		}
	}
	manifestRaw, err := json.Marshal(req.Manifest)
	if err != nil {
		return nil, err
	}
	if err := writer.WriteField("manifest", string(manifestRaw)); err != nil {
		return nil, err
	}
	if req.EffectivePluginSources != nil {
		raw, err := json.Marshal(req.EffectivePluginSources)
		if err != nil {
			return nil, err
		}
		if err := writer.WriteField("effectivePluginSources", string(raw)); err != nil {
			return nil, err
		}
	}
	part, err := writer.CreateFormFile("jar", filepath.Base(req.ArtifactPath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	path := "/v1/pushes"
	if req.Force {
		path += "?force=true"
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+c.scopedPath(path), &body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	if c.Tokens != nil {
		tok, err := c.Tokens.Token(ctx)
		if err != nil {
			return nil, fmt.Errorf("auth: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, parseError(resp)
	}
	out := &CreatePushResponse{}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, err
	}
	return out, nil
}
```

Add imports: `mime/multipart`, `os`, and `path/filepath`.

- [ ] **Step 4: Run test to verify pass**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
GOMODCACHE=/tmp/grounds-go/pkg/mod GOCACHE=/tmp/grounds-go/cache GOPATH=/tmp/grounds-go go test ./internal/api
```

Expected: PASS.

- [ ] **Step 5: Commit CLI API upload**

```bash
cd /home/lukas/grounds/grounds-cli
git add internal/api/push.go internal/api/push_test.go
git commit -S -m "feat: add push upload api client"
```

### Task 7: CLI Push Command Dispatch

**Files:**
- Modify: `grounds-cli/cmd/grounds/commands/push/push.go`
- Modify: `grounds-cli/cmd/grounds/commands/push/push_test.go`

- [ ] **Step 1: Add command seams for testing**

In `grounds-cli/cmd/grounds/commands/push/push.go`, add package variables near imports:

```go
var (
	findGradleWrapper = gradle.FindWrapper
	runGradleWrapper  = gradle.Run
	newAPIClient      = api.New
)
```

Replace direct calls to `gradle.FindWrapper`, `gradle.Run`, and `api.New` in this file with the variables. Existing tests must keep passing.

- [ ] **Step 2: Write failing Minestom command test**

In `grounds-cli/cmd/grounds/commands/push/push_test.go`, add:

```go
func TestPushMinestomFlavorBuildsDistributionAndUploads(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses POSIX shell gradle wrapper")
	}
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("Chdir(%q) error = %v", cwd, err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}
	writePushTestFile(t, "grounds.yaml", `
name: minestom-demo
flavors:
  minestom:
    type: minestom-server
    baseImage: minestom
    build:
      task: :server:distTar
      artifact: server/build/distributions/*.tar
    modules:
      - id: plugin-agones
        variant: minestom
        source: github:groundsgg/plugin-agones@v0.2.0:plugin-agones-minestom.jar
`)
	if err := os.MkdirAll("server/build/distributions", 0o755); err != nil {
		t.Fatalf("MkdirAll(distributions) error = %v", err)
	}
	writePushTestFile(t, "server/build/distributions/minestom-demo.tar", "\x1f\x8b\x08\x00")
	writePushTestFile(t, "gradlew", "#!/bin/sh\nprintf '%s\n' \"$@\" > args.txt\n")
	if err := os.Chmod("gradlew", 0o755); err != nil {
		t.Fatalf("Chmod(gradlew) error = %v", err)
	}

	workspaceDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(workspaceDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(config) error = %v", err)
	}
	t.Setenv("GROUNDS_CONFIG_DIR", workspaceDir)
	t.Setenv("GROUNDS_TOKEN", "test-token")
	writePushTestFile(t, filepath.Join(workspaceDir, "workspace.yaml"), `
repos:
  plugin-agones:
    path: `+filepath.ToSlash(filepath.Join(dir, "plugin-agones"))+`
    variants:
      minestom:
        enabled: true
`)

	var uploaded bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploaded = true
		if r.Method != "POST" || r.URL.Path != "/v1/pushes" {
			t.Fatalf("got %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		if got := r.FormValue("flavor"); got != "minestom" {
			t.Fatalf("flavor = %q", got)
		}
		if !strings.Contains(r.FormValue("manifest"), `"type":"minestom-server"`) {
			t.Fatalf("manifest = %s", r.FormValue("manifest"))
		}
		json.NewEncoder(w).Encode(map[string]any{"pushId": "push-1", "status": "building"})
	}))
	defer srv.Close()
	t.Setenv("GROUNDS_API_URL", srv.URL)

	cmd := newPush()
	cmd.SetArgs([]string{"--flavor=minestom", "--with-local"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !uploaded {
		t.Fatal("expected upload request")
	}
	raw, err := os.ReadFile("args.txt")
	if err != nil {
		t.Fatalf("ReadFile(args.txt) error = %v", err)
	}
	got := string(raw)
	if !strings.Contains(got, ":server:distTar\n") || !strings.Contains(got, "-I\n") {
		t.Fatalf("gradle args = %q, want distTar and -I", got)
	}
}

func writePushTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
```

Add imports: `encoding/json`, `net/http`, `net/http/httptest`.

- [ ] **Step 3: Run command tests to verify failure**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
GOMODCACHE=/tmp/grounds-go/pkg/mod GOCACHE=/tmp/grounds-go/cache GOPATH=/tmp/grounds-go go test ./cmd/grounds/commands/push
```

Expected: FAIL because the command still delegates all pushes to `groundsPush`.

- [ ] **Step 4: Implement Minestom dispatch**

In `grounds-cli/cmd/grounds/commands/push/push.go`, after locating the wrapper and refreshing auth, load the Minestom manifest:

```go
manifestPath := filepath.Join(filepath.Dir(wrapper), "grounds.yaml")
pushManifest, err := minestom.LoadPushManifest(manifestPath, flavor)
if err != nil {
	return err
}
if pushManifest.IsMinestomServer() {
	return runMinestomPush(ctx, cmd, wrapper, pushManifest, target, flavor, force, local, withLocal)
}
```

Move the existing Gradle `groundsPush` behavior into `runGradlePush(...)` or leave it inline after the Minestom branch. Implement `runMinestomPush`:

```go
func runMinestomPush(ctx context.Context, cmd *cobra.Command, wrapper string, pushManifest *minestom.PushManifest, target, flavor string, force bool, local []string, withLocal bool) error {
	projectRoot := filepath.Dir(wrapper)
	workspaceConfig, err := internalworkspace.Load("")
	if err != nil {
		return err
	}
	localPlan, err := minestom.ResolveLocalModules(ctx, *pushManifest, workspaceConfig, minestom.ResolveOptions{
		LocalIDs: local,
		WithLocal: withLocal,
	})
	if err != nil {
		return err
	}
	if len(localPlan.EffectivePluginSources) > 0 {
		renderBundleSources(cmd.OutOrStdout(), &internalworkspace.Plan{EffectivePluginSources: localPlan.EffectivePluginSources})
	}

	args := []string{pushManifest.Runtime.Build.Task}
	var initScript string
	if len(localPlan.LocalModules) > 0 {
		initScript, err = minestom.WriteCompositeInitScript(localPlan)
		if err != nil {
			return err
		}
		defer os.Remove(initScript)
		args = append(args, "-I", initScript)
	}
	if err := runGradleWrapper(ctx, wrapper, args, cmd.OutOrStdout(), cmd.ErrOrStderr(), 0); err != nil {
		return err
	}
	artifact, err := minestom.ResolveDistributionArtifact(projectRoot, pushManifest.Runtime.Build.Artifact)
	if err != nil {
		return err
	}
	cfg, err := config.Load("")
	if err != nil {
		return err
	}
	ts := api.NewEnvTokenSource()
	if ts == nil {
		ts = &auth.FileTokenSource{Store: auth.NewStore(cfg.Dir), Device: defaultDevice()}
	}
	client := newAPIClient(cfg.APIURL, ts)
	client.ProjectID = projectIDFrom(cmd)
	response, err := client.CreatePush(ctx, api.CreatePushRequest{
		Target: target,
		Flavor: flavor,
		Force: force,
		Manifest: map[string]any{
			"name": pushManifest.Name,
			"type": pushManifest.Runtime.Type,
			"publicType": "minestom",
			"baseImage": pushManifest.Runtime.BaseImage,
		},
		EffectivePluginSources: localPlan.EffectivePluginSources,
		ArtifactPath: artifact,
	})
	if err != nil {
		return err
	}
	render.StatusLine(cmd.OutOrStdout(), render.StatusOK, "Push", "Submitted "+response.PushID)
	render.DetailLine(cmd.OutOrStdout(), render.StatusOK, "Status: "+response.Status)
	return nil
}
```

Ensure existing auth refresh is shared by both paths, and ensure `--force` maps to `CreatePushRequest.Force`.

- [ ] **Step 5: Run command tests to verify pass**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
GOMODCACHE=/tmp/grounds-go/pkg/mod GOCACHE=/tmp/grounds-go/cache GOPATH=/tmp/grounds-go go test ./cmd/grounds/commands/push
```

Expected: PASS.

- [ ] **Step 6: Run all CLI tests**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
GOMODCACHE=/tmp/grounds-go/pkg/mod GOCACHE=/tmp/grounds-go/cache GOPATH=/tmp/grounds-go go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit CLI push dispatch**

```bash
cd /home/lukas/grounds/grounds-cli
git add cmd/grounds/commands/push/push.go cmd/grounds/commands/push/push_test.go
git commit -S -m "feat: push minestom distributions"
```

### Task 8: Real Local Push Verification

**Files:**
- Modify only if needed: `grounds-minestom-runtime/grounds.yaml`
- Modify only if needed: `grounds-minestom-runtime/examples/minigame-agones/build.gradle.kts`
- Do not commit uncommitted marker changes used only for verification.

- [ ] **Step 1: Prepare verification manifest**

In `/home/lukas/grounds/grounds-minestom-runtime`, create or update `grounds.yaml` only if no equivalent exists:

```yaml
name: minestom-runtime-example
flavors:
  minestom:
    type: minestom-server
    baseImage: minestom
    build:
      task: :examples:minigame-agones:distTar
      artifact: examples/minigame-agones/build/distributions/*.tar
    modules:
      - id: plugin-agones
        variant: minestom
        source: github:groundsgg/plugin-agones@v0.2.0:plugin-agones-minestom.jar
      - id: plugin-config
        variant: minestom
        source: github:groundsgg/plugin-config@v0.4.0:plugin-config-minestom.jar
```

- [ ] **Step 2: Configure workspace mappings**

Run:

```bash
grounds workspace add plugin-agones /home/lukas/grounds/plugin-agones --variant=minestom
grounds workspace add plugin-config /home/lukas/grounds/plugin-config --variant=minestom
grounds workspace list
```

Expected: `plugin-agones` and `plugin-config` show `minestom` variants and enabled local paths. If `plugin-config` does not exist locally, create a small temporary reference module in `/tmp/grounds-minestom-reference-module` and map that instead, then update `grounds.yaml` modules to use its id.

- [ ] **Step 3: Make an observable local change**

In one mapped local module, add an uncommitted marker resource:

```bash
mkdir -p /home/lukas/grounds/plugin-agones/minestom/src/main/resources
printf 'grounds-local-minestom-verification\n' > /home/lukas/grounds/plugin-agones/minestom/src/main/resources/grounds-local-marker.txt
```

Do not commit this marker.

- [ ] **Step 4: Run local Gradle build through CLI**

Run from `/home/lukas/grounds/grounds-minestom-runtime` with the locally built CLI binary:

```bash
cd /home/lukas/grounds/grounds-cli
go build -o /tmp/grounds-cli-minestom ./cmd/grounds
cd /home/lukas/grounds/grounds-minestom-runtime
/tmp/grounds-cli-minestom push --flavor=minestom --with-local --force
```

Expected:

- CLI prints bundle/local source information.
- Gradle args include `:examples:minigame-agones:distTar` and `-I /tmp/grounds-minestom-composites-*.gradle.kts`.
- Forge returns a push id or a clear live-environment error.

- [ ] **Step 5: Verify distribution contains local output**

Run:

```bash
tar -tf examples/minigame-agones/build/distributions/*.tar | rg 'plugin-agones|grounds-local-marker'
```

If the marker is inside a nested JAR, extract the distribution to `/tmp/minestom-dist-check` and inspect the JAR:

```bash
mkdir -p /tmp/minestom-dist-check
tar -xf examples/minigame-agones/build/distributions/*.tar -C /tmp/minestom-dist-check
find /tmp/minestom-dist-check -name '*.jar' -print
```

Expected: the distribution resolves the local module output, not only the released dependency.

- [ ] **Step 6: Verify Forge accepted or document blocker**

If live Forge accepts the upload, run:

```bash
/tmp/grounds-cli-minestom push list --mine --limit=5
```

Expected: newest push shows the Minestom app in `building`, `deploying`, or `ready`.

If live Forge is not yet deployed with Task 1-3 changes, document the exact response code and error in the PR body. Then run the CLI upload test against a local Forge dev server after applying Forge changes:

```bash
cd /home/lukas/grounds/grounds-forge
npm run dev
```

In another terminal:

```bash
GROUNDS_API_URL=http://127.0.0.1:3000 /tmp/grounds-cli-minestom push --flavor=minestom --with-local --force
```

Expected: local Forge receives the multipart Minestom distribution upload.

- [ ] **Step 7: Revert verification marker**

Remove only the uncommitted marker file from the local module:

```bash
rm /home/lukas/grounds/plugin-agones/minestom/src/main/resources/grounds-local-marker.txt
```

Do not revert unrelated user changes.

### Task 9: Final Verification and PRs

**Files:**
- Modify: PR body files under `/tmp` if needed.

- [ ] **Step 1: Run Forge verification**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test -- tests/manifest.test.ts tests/templates.test.ts tests/baseImages.test.ts tests/baseImageCatalogSeed.test.ts tests/renderer.test.ts tests/pushes.post.test.ts
npm run typecheck
```

Expected: PASS.

- [ ] **Step 2: Run CLI verification**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
GOMODCACHE=/tmp/grounds-go/pkg/mod GOCACHE=/tmp/grounds-go/cache GOPATH=/tmp/grounds-go go test ./...
```

Expected: PASS.

- [ ] **Step 3: Open linked PRs using the org template**

If the org `.github` PR template is unavailable locally, fetch it before creating bodies:

```bash
gh repo view groundsgg/.github --json nameWithOwner
gh api repos/groundsgg/.github/contents/.github/pull_request_template.md --jq '.content' | base64 -d > /tmp/grounds-pr-template.md
```

Create the Forge PR:

```bash
cd /home/lukas/grounds/grounds-forge
git status --short
git push -u origin HEAD
gh pr create --draft --title "feat: support minestom server distributions" --body-file /tmp/grounds-forge-minestom-pr.md
```

Create the CLI PR:

```bash
cd /home/lukas/grounds/grounds-cli
git status --short
git push -u origin HEAD
gh pr create --draft --title "feat: push minestom distributions" --body-file /tmp/grounds-cli-minestom-pr.md
```

Expected: both PRs are draft PRs, linked to each other, and include:

- Summary of Forge runtime/upload support.
- Summary of CLI build/upload/local override support.
- Unit test commands and results.
- Real `grounds push --flavor=minestom --with-local` verification result.
- Any live Forge limitation if deployment was not available at verification time.

- [ ] **Step 4: Final cleanup check**

Run:

```bash
cd /home/lukas/grounds/grounds-cli && git status --short
cd /home/lukas/grounds/grounds-forge && git status --short
cd /home/lukas/grounds/grounds-minestom-runtime && git status --short
cd /home/lukas/grounds/plugin-agones && git status --short
```

Expected: only intentional committed changes or explicitly documented verification files remain. No marker file remains in `plugin-agones`.
