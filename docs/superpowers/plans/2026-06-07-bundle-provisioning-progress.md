# Bundle Provisioning Progress Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show live, dynamic bundle provisioning progress for `grounds cluster up --bundle=<ref>` by persisting progress in Forge and rendering it in the CLI.

**Architecture:** Forge writes a best-effort `bundleProgress` JSON object to `DevCluster` during `PlatformBundleProfileReconciler.ensure()` and includes it in `GET /v1/cluster`. The CLI treats the field as optional, renders a single live spinner line in interactive terminals, and emits sparse progress lines for non-TTY output.

**Tech Stack:** TypeScript, Prisma, Fastify, Vitest in `/home/lukas/grounds/grounds-forge`; Go, Cobra, `golang.org/x/term`, and `go test` in `/home/lukas/grounds/grounds-cli`.

---

## File Structure

Forge files:

- Create `/home/lukas/grounds/grounds-forge/prisma/migrations/<timestamp>_devcluster_bundle_progress/migration.sql`: adds nullable `bundleProgress JSONB`.
- Modify `/home/lukas/grounds/grounds-forge/prisma/schema.prisma`: adds `bundleProgress Json?` to `DevCluster`.
- Create `/home/lukas/grounds/grounds-forge/src/devcluster/bundleProgress.ts`: owns progress types, phase constants, builders, and the best-effort DB writer.
- Modify `/home/lukas/grounds/grounds-forge/src/devcluster/PlatformBundleProfileReconciler.ts`: calls progress writer at infrastructure and component steps.
- Modify `/home/lukas/grounds/grounds-forge/src/routes/cluster.ts`: includes `bundleProgress` in `DevClusterRow` and `serializeStatus()`.
- Modify `/home/lukas/grounds/grounds-forge/src/main.ts`: ensures `findDevCluster` selects or returns `bundleProgress`.
- Test `/home/lukas/grounds/grounds-forge/tests/routes/cluster.test.ts`: status serialization includes progress.
- Test `/home/lukas/grounds/grounds-forge/tests/devcluster/bundleProgress.test.ts`: progress writer shape and dynamic component counts.

CLI files:

- Modify `/home/lukas/grounds/grounds-cli/internal/api/cluster.go`: adds `BundleProgress` to `ClusterStatus`.
- Modify `/home/lukas/grounds/grounds-cli/internal/api/cluster_test.go`: verifies decoding.
- Create `/home/lukas/grounds/grounds-cli/internal/render/progress.go`: reusable spinner, TTY-safe line clearing, and bundle progress summary formatting.
- Create `/home/lukas/grounds/grounds-cli/internal/render/progress_test.go`: summary and non-ANSI tests.
- Modify `/home/lukas/grounds/grounds-cli/cmd/grounds/commands/cluster/up.go`: uses progress renderer in `waitForBundle`.
- Modify `/home/lukas/grounds/grounds-cli/cmd/grounds/commands/cluster/up_test.go`: polling output and fallback tests.

## Execution Notes

- Work on `feat/bundle-provisioning-progress`.
- Keep Forge and CLI commits separate because they are separate repos.
- Forge log messages must follow the repository logging guidelines: outcome-first, context-rich, English, no secrets.
- Gradle checks do not apply to these repos. Verify Forge with `npm test` and `npm run build`; verify CLI with `go test ./...` and `go build ./cmd/grounds`.

### Task 1: Add Forge Bundle Progress Persistence

**Files:**
- Create: `/home/lukas/grounds/grounds-forge/prisma/migrations/<timestamp>_devcluster_bundle_progress/migration.sql`
- Modify: `/home/lukas/grounds/grounds-forge/prisma/schema.prisma`
- Create: `/home/lukas/grounds/grounds-forge/src/devcluster/bundleProgress.ts`
- Test: `/home/lukas/grounds/grounds-forge/tests/devcluster/bundleProgress.test.ts`

- [ ] **Step 1: Write the failing progress helper test**

Create `/home/lukas/grounds/grounds-forge/tests/devcluster/bundleProgress.test.ts`:

```ts
import { describe, expect, it, vi } from "vitest";
import {
  bundleProgress,
  persistBundleProgress,
  type BundleProgress,
} from "../../src/devcluster/bundleProgress.js";

describe("bundleProgress", () => {
  it("builds dynamic component progress from the resolved bundle position", () => {
    const progress = bundleProgress({
      bundleRef: "main",
      phase: "deploying_components",
      message: "Deploying bundle components",
      currentComponent: {
        name: "plugin-config",
        type: "plugin-velocity",
        mode: "gradle-local",
      },
      componentsTotal: 3,
      componentsDone: 1,
      componentsSucceeded: 1,
      componentsFailed: 0,
      now: new Date("2026-06-07T20:45:12.000Z"),
    });

    expect(progress).toEqual({
      bundleRef: "main",
      phase: "deploying_components",
      message: "Deploying bundle components",
      currentComponent: "plugin-config",
      currentComponentType: "plugin-velocity",
      currentComponentMode: "gradle-local",
      componentsTotal: 3,
      componentsDone: 1,
      componentsSucceeded: 1,
      componentsFailed: 0,
      updatedAt: "2026-06-07T20:45:12.000Z",
    });
  });

  it("persists progress without throwing when the database update succeeds", async () => {
    const prisma = {
      devCluster: {
        update: vi.fn().mockResolvedValue({ id: "dc-1" }),
      },
    };
    const logger = { warn: vi.fn() };
    const progress: BundleProgress = bundleProgress({
      bundleRef: "0.5.0",
      phase: "installing_vcluster",
      message: "Installing vCluster",
      now: new Date("2026-06-07T20:45:12.000Z"),
    });

    await persistBundleProgress({
      prisma,
      logger,
      devClusterId: "dc-1",
      namespace: "vcluster-lukas",
      progress,
    });

    expect(prisma.devCluster.update).toHaveBeenCalledWith({
      where: { id: "dc-1" },
      data: { bundleProgress: progress },
    });
    expect(logger.warn).not.toHaveBeenCalled();
  });

  it("logs a structured warning and does not throw when progress persistence fails", async () => {
    const prisma = {
      devCluster: {
        update: vi.fn().mockRejectedValue(new Error("db timeout")),
      },
    };
    const logger = { warn: vi.fn() };
    const progress: BundleProgress = bundleProgress({
      bundleRef: "main",
      phase: "deploying_components",
      message: "Deploying bundle components",
      now: new Date("2026-06-07T20:45:12.000Z"),
    });

    await persistBundleProgress({
      prisma,
      logger,
      devClusterId: "dc-1",
      namespace: "vcluster-lukas",
      progress,
    });

    expect(logger.warn).toHaveBeenCalledWith(
      {
        devClusterId: "dc-1",
        namespace: "vcluster-lukas",
        phase: "deploying_components",
        reason: "db timeout",
      },
      "Failed to persist bundle progress",
    );
  });
});
```

- [ ] **Step 2: Run the failing Forge helper test**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test -- tests/devcluster/bundleProgress.test.ts
```

Expected: FAIL because `src/devcluster/bundleProgress.ts` does not exist.

- [ ] **Step 3: Add the Prisma schema field and migration**

Modify `/home/lukas/grounds/grounds-forge/prisma/schema.prisma` in `model DevCluster` after `lastBundleResult Json?`:

```prisma
  // In-flight platform-bundle progress for the current or most recent
  // reconcile. Shape is documented in grounds-cli's bundle progress design.
  bundleProgress Json?
```

Create `/home/lukas/grounds/grounds-forge/prisma/migrations/<timestamp>_devcluster_bundle_progress/migration.sql`:

```sql
ALTER TABLE "DevCluster" ADD COLUMN "bundleProgress" JSONB;
```

- [ ] **Step 4: Add the progress helper implementation**

Create `/home/lukas/grounds/grounds-forge/src/devcluster/bundleProgress.ts`:

```ts
import type { Prisma } from "@prisma/client";

export type BundleProgressPhase =
  | "initializing"
  | "ensuring_namespace"
  | "installing_vcluster"
  | "waiting_for_vcluster"
  | "provisioning_pull_secret"
  | "provisioning_forwarding_secret"
  | "installing_nats"
  | "installing_postgres"
  | "loading_bundle"
  | "deploying_components"
  | "finalizing"
  | "active"
  | "failed";

export interface BundleProgress {
  readonly bundleRef: string;
  readonly phase: BundleProgressPhase;
  readonly message: string;
  readonly currentComponent?: string;
  readonly currentComponentType?: string;
  readonly currentComponentMode?: "image" | "gradle-local";
  readonly componentsTotal?: number;
  readonly componentsDone?: number;
  readonly componentsSucceeded?: number;
  readonly componentsFailed?: number;
  readonly updatedAt: string;
}

export interface BundleProgressInput {
  readonly bundleRef: string;
  readonly phase: BundleProgressPhase;
  readonly message: string;
  readonly currentComponent?: {
    readonly name: string;
    readonly type: string;
    readonly mode: "image" | "gradle-local";
  };
  readonly componentsTotal?: number;
  readonly componentsDone?: number;
  readonly componentsSucceeded?: number;
  readonly componentsFailed?: number;
  readonly now?: Date;
}

export interface BundleProgressPersistDeps {
  readonly prisma: {
    readonly devCluster: {
      update(args: {
        where: { id: string };
        data: { bundleProgress: Prisma.InputJsonValue };
      }): Promise<unknown>;
    };
  };
  readonly logger: {
    warn(obj: Record<string, unknown>, msg: string): void;
  };
  readonly devClusterId: string;
  readonly namespace: string;
  readonly progress: BundleProgress;
}

export function bundleProgress(input: BundleProgressInput): BundleProgress {
  return {
    bundleRef: input.bundleRef,
    phase: input.phase,
    message: input.message,
    ...(input.currentComponent
      ? {
          currentComponent: input.currentComponent.name,
          currentComponentType: input.currentComponent.type,
          currentComponentMode: input.currentComponent.mode,
        }
      : {}),
    ...(input.componentsTotal !== undefined ? { componentsTotal: input.componentsTotal } : {}),
    ...(input.componentsDone !== undefined ? { componentsDone: input.componentsDone } : {}),
    ...(input.componentsSucceeded !== undefined ? { componentsSucceeded: input.componentsSucceeded } : {}),
    ...(input.componentsFailed !== undefined ? { componentsFailed: input.componentsFailed } : {}),
    updatedAt: (input.now ?? new Date()).toISOString(),
  };
}

export async function persistBundleProgress(deps: BundleProgressPersistDeps): Promise<void> {
  try {
    await deps.prisma.devCluster.update({
      where: { id: deps.devClusterId },
      data: { bundleProgress: deps.progress as unknown as Prisma.InputJsonValue },
    });
  } catch (err) {
    deps.logger.warn(
      {
        devClusterId: deps.devClusterId,
        namespace: deps.namespace,
        phase: deps.progress.phase,
        reason: err instanceof Error ? err.message : String(err),
      },
      "Failed to persist bundle progress",
    );
  }
}
```

- [ ] **Step 5: Run the helper test**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test -- tests/devcluster/bundleProgress.test.ts
```

Expected: PASS.

- [ ] **Step 6: Commit Forge progress persistence foundation**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
git add prisma/schema.prisma prisma/migrations src/devcluster/bundleProgress.ts tests/devcluster/bundleProgress.test.ts
git commit -m "feat(cluster): add bundle progress persistence"
```

### Task 2: Expose Bundle Progress From Forge Status

**Files:**
- Modify: `/home/lukas/grounds/grounds-forge/src/routes/cluster.ts`
- Modify: `/home/lukas/grounds/grounds-forge/src/main.ts`
- Test: `/home/lukas/grounds/grounds-forge/tests/routes/cluster.test.ts`

- [ ] **Step 1: Write the failing route serialization test**

Modify `/home/lukas/grounds/grounds-forge/tests/routes/cluster.test.ts` by adding this test under `describe("GET /v1/cluster", ...)`:

```ts
  it("includes bundleProgress when the workspace has in-flight bundle progress", async () => {
    const { app } = buildApp({
      user: { id: "u-1", sub: "sub-1" },
      devCluster: {
        id: "dc-1",
        namespace: "vcluster-test",
        state: "creating",
        profile: "platform-bundle",
        createdAt: new Date("2026-06-07T20:00:00.000Z"),
        lastActivityAt: new Date("2026-06-07T20:00:00.000Z"),
        pausedAt: null,
        pauseScheduledAt: null,
        warningAt: null,
        quota: { cpu: "4", memory: "8Gi", storage: "20Gi" },
        pauseReason: null,
        lastBundleResult: null,
        bundleProgress: {
          bundleRef: "main",
          phase: "deploying_components",
          message: "Deploying bundle components",
          currentComponent: "plugin-config",
          componentsTotal: 14,
          componentsDone: 7,
          componentsSucceeded: 6,
          componentsFailed: 1,
          updatedAt: "2026-06-07T20:45:12.000Z",
        },
      },
    });

    const r = await app.inject({ method: "GET", url: "/v1/cluster" });

    expect(r.statusCode).toBe(200);
    expect(r.json()).toMatchObject({
      namespace: "vcluster-test",
      state: "creating",
      profile: "platform-bundle",
      bundleProgress: {
        bundleRef: "main",
        phase: "deploying_components",
        currentComponent: "plugin-config",
        componentsTotal: 14,
        componentsDone: 7,
      },
    });
  });
```

- [ ] **Step 2: Run the failing route test**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test -- tests/routes/cluster.test.ts
```

Expected: FAIL because `bundleProgress` is not part of `DevClusterRow` or serialized output.

- [ ] **Step 3: Add `bundleProgress` to `DevClusterRow` and `serializeStatus()`**

Modify `/home/lukas/grounds/grounds-forge/src/routes/cluster.ts`:

```ts
export interface DevClusterRow {
  id: string;
  namespace: string;
  state: string;
  profile: string;
  createdAt: Date;
  lastActivityAt: Date;
  pausedAt: Date | null;
  pauseScheduledAt: Date | null;
  warningAt: Date | null;
  quota: unknown;
  pauseReason: string | null;
  lastBundleResult: unknown;
  bundleProgress: unknown;
}
```

Then add this field to the object returned by `serializeStatus()`:

```ts
    bundleProgress: dc.bundleProgress ?? null,
```

The surrounding final fields should read:

```ts
    deploymentsReady,
    bundleProgress: dc.bundleProgress ?? null,
    bundleResult: dc.lastBundleResult ?? null,
    failureReason: dc.state === "failed" ? dc.pauseReason : null,
```

- [ ] **Step 4: Ensure production `findDevCluster` returns `bundleProgress`**

Inspect `/home/lukas/grounds/grounds-forge/src/main.ts`. If any `findDevCluster` implementation uses an explicit `select`, add `bundleProgress: true` to that selection. For implementations returning full Prisma rows through `findFirst()` or `findUnique()`, no change is needed after Prisma schema generation.

Expected explicit select shape if present:

```ts
select: {
  id: true,
  namespace: true,
  state: true,
  profile: true,
  createdAt: true,
  lastActivityAt: true,
  pausedAt: true,
  pauseScheduledAt: true,
  warningAt: true,
  quota: true,
  pauseReason: true,
  lastBundleResult: true,
  bundleProgress: true,
}
```

- [ ] **Step 5: Run route tests**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test -- tests/routes/cluster.test.ts
```

Expected: PASS.

- [ ] **Step 6: Commit Forge status exposure**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
git add src/routes/cluster.ts src/main.ts tests/routes/cluster.test.ts
git commit -m "feat(cluster): expose bundle progress in status"
```

### Task 3: Write Bundle Progress From the Forge Reconciler

**Files:**
- Modify: `/home/lukas/grounds/grounds-forge/src/devcluster/PlatformBundleProfileReconciler.ts`
- Test: `/home/lukas/grounds/grounds-forge/tests/devcluster/bundleProgress.test.ts`

- [ ] **Step 1: Extend helper tests for finished component progress**

Add this test to `/home/lukas/grounds/grounds-forge/tests/devcluster/bundleProgress.test.ts`:

```ts
  it("builds final active progress with component totals", () => {
    const progress = bundleProgress({
      bundleRef: "0.5.0",
      phase: "active",
      message: "Bundle provisioning completed",
      componentsTotal: 2,
      componentsDone: 2,
      componentsSucceeded: 1,
      componentsFailed: 1,
      now: new Date("2026-06-07T20:50:00.000Z"),
    });

    expect(progress).toEqual({
      bundleRef: "0.5.0",
      phase: "active",
      message: "Bundle provisioning completed",
      componentsTotal: 2,
      componentsDone: 2,
      componentsSucceeded: 1,
      componentsFailed: 1,
      updatedAt: "2026-06-07T20:50:00.000Z",
    });
  });
```

- [ ] **Step 2: Run the helper tests**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test -- tests/devcluster/bundleProgress.test.ts
```

Expected: PASS after Task 1; this locks the final progress shape before wiring the reconciler.

- [ ] **Step 3: Import the progress helpers**

Modify `/home/lukas/grounds/grounds-forge/src/devcluster/PlatformBundleProfileReconciler.ts` imports:

```ts
import {
  bundleProgress,
  persistBundleProgress,
  type BundleProgressPhase,
} from "./bundleProgress.js";
```

- [ ] **Step 4: Add a private progress method**

Inside `PlatformBundleProfileReconciler`, add:

```ts
  private async writeProgress(
    devClusterId: string,
    namespace: string,
    bundleRef: string,
    phase: BundleProgressPhase,
    message: string,
    extra: Omit<Parameters<typeof bundleProgress>[0], "bundleRef" | "phase" | "message"> = {},
  ): Promise<void> {
    await persistBundleProgress({
      prisma: this.deps.prisma,
      logger: this.deps.logger,
      devClusterId,
      namespace,
      progress: bundleProgress({
        bundleRef,
        phase,
        message,
        ...extra,
      }),
    });
  }
```

- [ ] **Step 5: Write infrastructure phase progress in `ensure()`**

Modify `ensure()` so the major steps are preceded by progress writes:

```ts
    const dc = await this.upsertDevCluster(userId, ns, ephemeral);
    try {
      await this.writeProgress(dc.id, ns, opts.bundleRef, "initializing", "Preparing bundle workspace");

      await this.writeProgress(dc.id, ns, opts.bundleRef, "ensuring_namespace", "Ensuring host namespace");
      await this.ensureNamespace(ns, ephemeral);

      await this.writeProgress(dc.id, ns, opts.bundleRef, "installing_vcluster", "Installing vCluster");
      await this.helmInstallVCluster(release, ns);

      await this.writeProgress(dc.id, ns, opts.bundleRef, "waiting_for_vcluster", "Waiting for vCluster API");
      const vClusterKubeconfig = await this.waitForVClusterReady(ns, release);

      await this.writeProgress(dc.id, ns, opts.bundleRef, "provisioning_pull_secret", "Provisioning vCluster pull secret");
      await this.ensureVclusterPullSecret(vClusterKubeconfig, handle);

      await this.writeProgress(dc.id, ns, opts.bundleRef, "provisioning_forwarding_secret", "Provisioning Velocity forwarding secret");
      await this.ensureForwardingSecret(vClusterKubeconfig, handle);

      await this.writeProgress(dc.id, ns, opts.bundleRef, "installing_nats", "Installing shared NATS");
      const natsBrokerInstalled = await this.installNatsChart(vClusterKubeconfig, handle);

      await this.writeProgress(dc.id, ns, opts.bundleRef, "installing_postgres", "Installing shared Postgres");
      const postgresInstalled = await this.installPostgresChart(vClusterKubeconfig, handle);

      await this.writeProgress(dc.id, ns, opts.bundleRef, "loading_bundle", "Loading bundle");
      const bundle = await loadBundle(opts.bundleRef, {
        githubToken: this.deps.githubToken,
      });
```

- [ ] **Step 6: Write dynamic component progress in `deployComponents()`**

Change `deployComponents()` signature:

```ts
  private async deployComponents(
    bundle: ResolvedBundle,
    vClusterKubeconfig: string,
    handle: string,
    installed: { nats: boolean; postgres: boolean },
    progress: { devClusterId: string; namespace: string; bundleRef: string },
  ): Promise<{ succeeded: string[]; failed: { name: string; error: string }[] }> {
```

Update the caller:

```ts
      const { succeeded, failed } = await this.deployComponents(
        resolved,
        vClusterKubeconfig,
        handle,
        { nats: natsBrokerInstalled, postgres: postgresInstalled },
        { devClusterId: dc.id, namespace: ns, bundleRef: opts.bundleRef },
      );
```

Inside the component loop, write progress before and after each component:

```ts
      for (const component of bundle.components) {
        await this.writeProgress(
          progress.devClusterId,
          progress.namespace,
          progress.bundleRef,
          "deploying_components",
          "Deploying bundle components",
          {
            currentComponent: {
              name: component.name,
              type: component.type,
              mode: component.mode,
            },
            componentsTotal: bundle.components.length,
            componentsDone: succeeded.length + failed.length,
            componentsSucceeded: succeeded.length,
            componentsFailed: failed.length,
          },
        );

        const result = await this.deployOne(component, kubeconfigPath, ctx);
        if (result.ok) {
          succeeded.push(component.name);
        } else {
          failed.push({ name: component.name, error: result.error });
        }

        await this.writeProgress(
          progress.devClusterId,
          progress.namespace,
          progress.bundleRef,
          "deploying_components",
          "Deploying bundle components",
          {
            currentComponent: {
              name: component.name,
              type: component.type,
              mode: component.mode,
            },
            componentsTotal: bundle.components.length,
            componentsDone: succeeded.length + failed.length,
            componentsSucceeded: succeeded.length,
            componentsFailed: failed.length,
          },
        );
      }
```

- [ ] **Step 7: Write final active and failed progress**

Before `markActive()`:

```ts
      await this.writeProgress(dc.id, ns, opts.bundleRef, "finalizing", "Finalizing bundle workspace", {
        componentsTotal: resolved.components.length,
        componentsDone: succeeded.length + failed.length,
        componentsSucceeded: succeeded.length,
        componentsFailed: failed.length,
      });
```

After `markActive()`:

```ts
      await this.writeProgress(dc.id, ns, opts.bundleRef, "active", "Bundle provisioning completed", {
        componentsTotal: resolved.components.length,
        componentsDone: succeeded.length + failed.length,
        componentsSucceeded: succeeded.length,
        componentsFailed: failed.length,
      });
```

In the `catch` block, write failed progress before rethrowing:

```ts
    } catch (err) {
      await this.writeProgress(dc.id, ns, opts.bundleRef, "failed", "Bundle provisioning failed");
      await this.markFailed(dc.id, err);
      throw err;
    }
```

- [ ] **Step 8: Run Forge build and tests**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test
npm run build
```

Expected: both pass.

- [ ] **Step 9: Commit Forge reconciler progress**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
git add src/devcluster/PlatformBundleProfileReconciler.ts tests/devcluster/bundleProgress.test.ts
git commit -m "feat(cluster): report bundle provisioning phases"
```

### Task 4: Decode Bundle Progress in the CLI API Model

**Files:**
- Modify: `/home/lukas/grounds/grounds-cli/internal/api/cluster.go`
- Modify: `/home/lukas/grounds/grounds-cli/internal/api/cluster_test.go`

- [ ] **Step 1: Write the failing decode test**

Modify `TestGetCluster` in `/home/lukas/grounds/grounds-cli/internal/api/cluster_test.go` so the server response includes:

```go
"bundleProgress": map[string]any{
	"bundleRef":            "main",
	"phase":                "deploying_components",
	"message":              "Deploying bundle components",
	"currentComponent":     "plugin-config",
	"currentComponentType": "grpc-service",
	"currentComponentMode": "gradle-local",
	"componentsTotal":      14,
	"componentsDone":       7,
	"componentsSucceeded":  6,
	"componentsFailed":     1,
	"updatedAt":            "2026-06-07T20:45:12.000Z",
},
```

Add this assertion after the existing status assertion:

```go
if s.BundleProgress == nil {
	t.Fatal("BundleProgress is nil")
}
if s.BundleProgress.Phase != "deploying_components" ||
	s.BundleProgress.CurrentComponent != "plugin-config" ||
	s.BundleProgress.ComponentsTotal != 14 ||
	s.BundleProgress.ComponentsDone != 7 {
	t.Errorf("BundleProgress = %+v", s.BundleProgress)
}
```

- [ ] **Step 2: Run the failing CLI API test**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
go test ./internal/api
```

Expected: FAIL because `ClusterStatus.BundleProgress` does not exist.

- [ ] **Step 3: Add the API structs**

Modify `/home/lukas/grounds/grounds-cli/internal/api/cluster.go`:

```go
type ClusterStatus struct {
	Namespace        string             `json:"namespace"`
	State            string             `json:"state"`
	Profile          string             `json:"profile"`
	CreatedAt        time.Time          `json:"createdAt"`
	LastActivityAt   time.Time          `json:"lastActivityAt"`
	PausedAt         *time.Time         `json:"pausedAt"`
	PauseScheduledAt *time.Time         `json:"pauseScheduledAt"`
	WarningAt        *time.Time         `json:"warningAt"`
	AutoPauseAt      *time.Time         `json:"autoPauseAt"`
	AutoDeleteAt     *time.Time         `json:"autoDeleteAt"`
	Quota            map[string]string  `json:"quota"`
	DeploymentsReady int                `json:"deploymentsReady"`
	BundleProgress   *BundleProgress    `json:"bundleProgress"`
	BundleResult     *BundleResult      `json:"bundleResult"`
	FailureReason    string             `json:"failureReason"`
}

type BundleProgress struct {
	BundleRef            string `json:"bundleRef"`
	Phase                string `json:"phase"`
	Message              string `json:"message"`
	CurrentComponent     string `json:"currentComponent"`
	CurrentComponentType string `json:"currentComponentType"`
	CurrentComponentMode string `json:"currentComponentMode"`
	ComponentsTotal      int    `json:"componentsTotal"`
	ComponentsDone       int    `json:"componentsDone"`
	ComponentsSucceeded  int    `json:"componentsSucceeded"`
	ComponentsFailed     int    `json:"componentsFailed"`
	UpdatedAt            string `json:"updatedAt"`
}
```

- [ ] **Step 4: Run the API tests**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
go test ./internal/api
```

Expected: PASS.

- [ ] **Step 5: Commit CLI API model**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
git add internal/api/cluster.go internal/api/cluster_test.go
git commit -m "feat(cluster): decode bundle progress"
```

### Task 5: Add CLI Progress Summary and Spinner Renderer

**Files:**
- Create: `/home/lukas/grounds/grounds-cli/internal/render/progress.go`
- Create: `/home/lukas/grounds/grounds-cli/internal/render/progress_test.go`

- [ ] **Step 1: Write progress summary tests**

Create `/home/lukas/grounds/grounds-cli/internal/render/progress_test.go`:

```go
package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"

	"github.com/groundsgg/grounds-cli/internal/api"
)

func TestBundleProgressSummaryWithComponent(t *testing.T) {
	got := BundleProgressSummary(&api.BundleProgress{
		Phase:                "deploying_components",
		CurrentComponent:     "plugin-config",
		CurrentComponentType: "grpc-service",
		CurrentComponentMode: "gradle-local",
		ComponentsTotal:      14,
		ComponentsDone:       7,
	})

	want := "deploying components 7/14: plugin-config (grpc-service, gradle-local)"
	if got != want {
		t.Fatalf("BundleProgressSummary() = %q, want %q", got, want)
	}
}

func TestBundleProgressSummaryUnknownPhase(t *testing.T) {
	got := BundleProgressSummary(&api.BundleProgress{Phase: "warming_cache"})
	if got != "warming cache" {
		t.Fatalf("BundleProgressSummary() = %q, want %q", got, "warming cache")
	}
}

func TestBundleProgressSummaryFallbackState(t *testing.T) {
	got := BundleProgressSummary(nil)
	if got != "" {
		t.Fatalf("BundleProgressSummary(nil) = %q, want empty", got)
	}
}

func TestSpinnerLineClearsWithoutLeavingText(t *testing.T) {
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = false })

	var buf bytes.Buffer
	line := NewSpinnerLine(&buf)
	line.Update("deploying components 7/14: plugin-config", 90*time.Second, 5*time.Second)
	line.Clear()

	got := buf.String()
	if !strings.Contains(got, "\033[1A\r\033[K") {
		t.Fatalf("spinner output did not clear line: %q", got)
	}
}
```

- [ ] **Step 2: Run the failing render tests**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
go test ./internal/render
```

Expected: FAIL because `BundleProgressSummary` and `NewSpinnerLine` do not exist.

- [ ] **Step 3: Implement progress formatting and spinner**

Create `/home/lukas/grounds/grounds-cli/internal/render/progress.go`:

```go
package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/groundsgg/grounds-cli/internal/api"
)

var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type SpinnerLine struct {
	w     io.Writer
	frame int
	shown bool
}

func NewSpinnerLine(w io.Writer) *SpinnerLine {
	return &SpinnerLine{w: w}
}

func (s *SpinnerLine) Update(summary string, elapsed, nextPoll time.Duration) {
	if s.shown {
		s.Clear()
	}
	frame := SpinnerFrames[s.frame%len(SpinnerFrames)]
	s.frame++
	fmt.Fprintf(
		s.w,
		"    %s %s (elapsed %s, next check in %s)\n",
		Yellow(frame),
		summary,
		formatClock(elapsed),
		formatSeconds(nextPoll),
	)
	s.shown = true
}

func (s *SpinnerLine) Clear() {
	if !s.shown {
		return
	}
	fmt.Fprint(s.w, "\033[1A\r\033[K")
	s.shown = false
}

func BundleProgressSummary(progress *api.BundleProgress) string {
	if progress == nil {
		return ""
	}
	label := phaseLabel(progress.Phase)
	if progress.ComponentsTotal > 0 {
		label = fmt.Sprintf("%s %d/%d", label, progress.ComponentsDone, progress.ComponentsTotal)
	}
	if progress.CurrentComponent != "" {
		label += ": " + progress.CurrentComponent
	}
	parts := make([]string, 0, 2)
	if progress.CurrentComponentType != "" {
		parts = append(parts, progress.CurrentComponentType)
	}
	if progress.CurrentComponentMode != "" {
		parts = append(parts, progress.CurrentComponentMode)
	}
	if len(parts) > 0 {
		label += " (" + strings.Join(parts, ", ") + ")"
	}
	return label
}

func phaseLabel(phase string) string {
	switch phase {
	case "initializing":
		return "preparing bundle workspace"
	case "ensuring_namespace":
		return "ensuring namespace"
	case "installing_vcluster":
		return "installing vCluster"
	case "waiting_for_vcluster":
		return "waiting for vCluster API"
	case "provisioning_pull_secret":
		return "provisioning pull secret"
	case "provisioning_forwarding_secret":
		return "provisioning forwarding secret"
	case "installing_nats":
		return "installing shared NATS"
	case "installing_postgres":
		return "installing shared Postgres"
	case "loading_bundle":
		return "loading bundle"
	case "deploying_components":
		return "deploying components"
	case "finalizing":
		return "finalizing"
	case "active":
		return "bundle provisioning completed"
	case "failed":
		return "bundle provisioning failed"
	default:
		return strings.ReplaceAll(phase, "_", " ")
	}
}

func formatClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int(d.Round(time.Second).Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func formatSeconds(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	return fmt.Sprintf("%ds", int(d.Round(time.Second).Seconds()))
}
```

- [ ] **Step 4: Run render tests**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
go test ./internal/render
```

Expected: PASS.

- [ ] **Step 5: Commit CLI renderer**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
git add internal/render/progress.go internal/render/progress_test.go
git commit -m "feat(cluster): render bundle progress summaries"
```

### Task 6: Wire CLI Bundle Polling to Progress Renderer

**Files:**
- Modify: `/home/lukas/grounds/grounds-cli/cmd/grounds/commands/cluster/up.go`
- Modify: `/home/lukas/grounds/grounds-cli/cmd/grounds/commands/cluster/up_test.go`

- [ ] **Step 1: Add non-TTY progress output test**

Add this test to `/home/lukas/grounds/grounds-cli/cmd/grounds/commands/cluster/up_test.go`:

```go
func TestRenderBundleProgressLine(t *testing.T) {
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = false })

	var buf bytes.Buffer
	state := bundleWaitRenderState{}
	status := &api.ClusterStatus{
		State: "creating",
		BundleProgress: &api.BundleProgress{
			Phase:            "deploying_components",
			CurrentComponent: "plugin-config",
			ComponentsTotal:  14,
			ComponentsDone:   7,
		},
	}

	renderBundleWaitProgress(&buf, false, nil, &state, status, 90*time.Second, 5*time.Second)
	renderBundleWaitProgress(&buf, false, nil, &state, status, 95*time.Second, 5*time.Second)

	want := "    • phase: deploying components 7/14: plugin-config\n"
	if got := buf.String(); got != want {
		t.Fatalf("progress output = %q, want %q", got, want)
	}
}
```

Add imports if missing:

```go
import "time"
```

- [ ] **Step 2: Run the failing cluster tests**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
go test ./cmd/grounds/commands/cluster
```

Expected: FAIL because `bundleWaitRenderState` and `renderBundleWaitProgress` do not exist.

- [ ] **Step 3: Add render state and helper in `up.go`**

Modify `/home/lukas/grounds/grounds-cli/cmd/grounds/commands/cluster/up.go`:

```go
type bundleWaitRenderState struct {
	lastState   string
	lastSummary string
}

func renderBundleWaitProgress(
	w io.Writer,
	interactive bool,
	spinner *render.SpinnerLine,
	state *bundleWaitRenderState,
	s *api.ClusterStatus,
	elapsed time.Duration,
	nextPoll time.Duration,
) {
	summary := render.BundleProgressSummary(s.BundleProgress)
	if summary == "" {
		summary = "state: " + s.State
	}
	if interactive && spinner != nil {
		spinner.Update(summary, elapsed, nextPoll)
		return
	}
	if summary != state.lastSummary {
		render.DetailLine(w, render.StatusOK, "phase: "+summary)
		state.lastSummary = summary
	}
}
```

- [ ] **Step 4: Make `waitForBundle` TTY-aware**

Change `waitForBundle(ctx, c, w)` to create interactive rendering:

```go
func waitForBundle(ctx context.Context, c *api.Client, w io.Writer) (*api.ClusterStatus, error) {
	const (
		interval = 5 * time.Second
		overall  = 20 * time.Minute
		rowGrace = 30 * time.Second
	)
	startedAt := time.Now()
	deadline := startedAt.Add(overall)
	graceUntil := startedAt.Add(rowGrace)
	lastState := ""
	lastSummary := ""
	interactive := isOutputTerminal(w)
	var spinner *render.SpinnerLine
	if interactive {
		spinner = render.NewSpinnerLine(w)
		defer spinner.Clear()
	}
	progressState := &bundleWaitRenderState{}
	for {
		s, err := c.GetCluster(ctx)
		if err != nil {
			var apiErr *api.Error
			if errors.As(err, &apiErr) && apiErr.StatusCode == 404 && time.Now().Before(graceUntil) {
			} else {
				return nil, err
			}
		} else {
			elapsed := time.Since(startedAt)
			renderBundleWaitProgress(w, interactive, spinner, progressState, s, elapsed, interval)
			summary := render.BundleProgressSummary(s.BundleProgress)
			if summary != "" {
				lastSummary = summary
			}
			if s.State != lastState {
				if !interactive && summary == "" {
					render.DetailLine(w, render.StatusOK, "state: "+s.State)
				}
				lastState = s.State
			}
			switch s.State {
			case "active", "failed":
				if spinner != nil {
					spinner.Clear()
				}
				return s, nil
			}
		}
		if time.Now().After(deadline) {
			if lastSummary != "" {
				return nil, fmt.Errorf("timed out after %s waiting for the bundle (still %q: %s); check `grounds cluster status`", overall, lastState, lastSummary)
			}
			return nil, fmt.Errorf("timed out after %s waiting for the bundle (still %q); check `grounds cluster status`", overall, lastState)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}
```

Add helper near the bottom of `up.go`:

```go
func isOutputTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}
```

Add import:

```go
"golang.org/x/term"
```

- [ ] **Step 5: Run cluster tests**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
go test ./cmd/grounds/commands/cluster
```

Expected: PASS.

- [ ] **Step 6: Run full CLI verification**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
go test ./...
go build ./cmd/grounds
```

Expected: both pass.

- [ ] **Step 7: Commit CLI polling integration**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
git add cmd/grounds/commands/cluster/up.go cmd/grounds/commands/cluster/up_test.go
git commit -m "feat(cluster): show live bundle provisioning progress"
```

### Task 7: Final Cross-Repo Verification

**Files:**
- Read-only verification across `/home/lukas/grounds/grounds-forge`
- Read-only verification across `/home/lukas/grounds/grounds-cli`

- [ ] **Step 1: Verify Forge**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
npm test
npm run build
git status --short
```

Expected:

- `npm test` passes.
- `npm run build` passes.
- `git status --short` is empty.

- [ ] **Step 2: Verify CLI**

Run:

```bash
cd /home/lukas/grounds/grounds-cli
go test ./...
go build ./cmd/grounds
git status --short
```

Expected:

- `go test ./...` passes.
- `go build ./cmd/grounds` passes.
- `git status --short` is empty.

- [ ] **Step 3: Inspect commit history**

Run:

```bash
cd /home/lukas/grounds/grounds-forge
git log --oneline --decorate -5
cd /home/lukas/grounds/grounds-cli
git log --oneline --decorate -5
```

Expected:

- Forge has three conventional feature commits for persistence, status exposure, and reconciler progress.
- CLI has three conventional feature commits for API decode, renderer, and polling integration.
- Branch names use Conventional Commit-style naming and do not mention the agent.

### Task 8: Optional Portal Follow-Up Ticket

**Files:**
- No code changes in this plan.

- [ ] **Step 1: Record follow-up scope**

If Portal should show the same progress later, create a follow-up issue or note with this exact scope:

```md
Portal follow-up: render `ClusterStatus.bundleProgress` on bundle apply screens.

Backend support:
- `GET /v1/cluster` returns nullable `bundleProgress`.
- Progress is dynamic per resolved bundle.

Portal work:
- Add `bundleProgress` to `src/lib/api/types.ts`.
- Surface current phase, component count, and current component while cluster state is `creating`.
- Keep final `lastApplyResult` behavior unchanged.
```

- [ ] **Step 2: Do not implement Portal in this branch**

Expected: No files under `/home/lukas/grounds/grounds-portal` are modified by this implementation plan.

## Self-Review

Spec coverage:

- Live progress during `grounds cluster up --bundle=<ref>`: Tasks 1, 3, 5, 6.
- Dynamic component counts from resolved bundle: Task 3.
- `GET /v1/cluster` remains polling source: Task 2 and Task 6.
- TTY spinner and non-TTY sparse lines: Task 5 and Task 6.
- Existing final success/failure output preserved: Task 6 only changes waiting output and leaves `renderBundleStatus()` intact.
- Failure handling and timeout summary: Task 6.
- Logging guidelines: Task 1 uses structured WARN logging without sensitive fields.
- Tests and rollout compatibility: Tasks 1-7.

Placeholder scan:

- No `TBD`, `TODO`, or open-ended implementation instructions remain.

Type consistency:

- Forge field name is consistently `bundleProgress`.
- CLI struct name is consistently `BundleProgress`.
- Phase field remains a string in the CLI for forward compatibility.
