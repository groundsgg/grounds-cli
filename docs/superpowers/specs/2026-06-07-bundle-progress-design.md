# Bundle Provisioning Progress Design

## Context

`grounds cluster up --bundle=<ref>` starts a long-running Forge reconcile and then polls `GET /v1/cluster` until the workspace reaches `active` or `failed`. Today the CLI prints the initial status and only emits a new detail line when the coarse workspace `state` changes. For the normal long `creating` phase, users see no visible activity for minutes.

The current Forge API already exposes final bundle apply data through `bundleResult` and `failureReason` on `GET /v1/cluster`, but it does not expose incremental progress while the bundle is being reconciled.

## Goals

- Show live progress for `grounds cluster up --bundle=<ref>` during long bundle provisioning.
- Make component progress dynamic per resolved bundle, after overrides and optional-component filtering.
- Keep `GET /v1/cluster` as the polling source of truth.
- Keep terminal output readable in interactive terminals and clean in redirected output, CI logs, and tests.
- Preserve the existing final success and failure output, including per-component succeeded/failed summaries.

## Non-Goals

- Do not introduce SSE or a separate event stream for this first version.
- Do not hardcode bundle component names in the CLI.
- Do not require the CLI to understand bundle schema internals.
- Do not expose secrets, override contents, kubeconfig data, or Helm values in progress output.

## API Shape

Forge adds a persisted `bundleProgress` JSON field to the `DevCluster` row and includes it in `GET /v1/cluster` when present.

Suggested response shape:

```json
{
  "namespace": "vcluster-lukas",
  "state": "creating",
  "profile": "platform-bundle",
  "deploymentsReady": 0,
  "bundleProgress": {
    "bundleRef": "main",
    "phase": "deploying_components",
    "message": "Deploying bundle components",
    "currentComponent": "plugin-config",
    "currentComponentType": "grpc-service",
    "currentComponentMode": "gradle-local",
    "componentsTotal": 14,
    "componentsDone": 7,
    "componentsSucceeded": 6,
    "componentsFailed": 1,
    "updatedAt": "2026-06-07T20:45:12Z"
  }
}
```

All fields except `phase`, `message`, and `updatedAt` are optional. This lets infrastructure phases omit component fields, and lets older/partial progress writers remain forward-compatible.

The fixed phase enum is:

- `initializing`
- `ensuring_namespace`
- `installing_vcluster`
- `waiting_for_vcluster`
- `provisioning_pull_secret`
- `provisioning_forwarding_secret`
- `installing_nats`
- `installing_postgres`
- `loading_bundle`
- `deploying_components`
- `finalizing`
- `active`
- `failed`

The phase enum is intentionally fixed so clients can render stable labels. The component names, counts, type, and mode are dynamic and come from the resolved bundle after `loadBundle()` and `applyOverrides()`.

## Forge Behavior

`PlatformBundleProfileReconciler.ensure()` updates `bundleProgress` at each major step:

1. Before namespace work: `initializing`.
2. Around host namespace creation: `ensuring_namespace`.
3. Around vCluster Helm install: `installing_vcluster`.
4. While waiting for the vCluster API: `waiting_for_vcluster`.
5. Before vCluster pull secret setup: `provisioning_pull_secret`.
6. Before Velocity forwarding secret setup: `provisioning_forwarding_secret`.
7. Around shared NATS install: `installing_nats`.
8. Around shared Postgres install: `installing_postgres`.
9. Before bundle fetch and override resolution: `loading_bundle`.
10. During component deploy loop: `deploying_components`, with dynamic counts and current component metadata from `ResolvedBundle.components`.
11. Before marking active: `finalizing`.
12. On success: `active`, with final counts.
13. On failure: `failed`, with a non-sensitive failure summary.

Progress updates are best-effort. A failed progress write should be logged once at WARN level with context and should not fail the reconcile.

The component loop increments progress after each component finishes, regardless of success or failure. Example for a 14-component resolved bundle:

- Before `plugin-config`: `componentsDone=6`, `currentComponent=plugin-config`.
- After success: `componentsDone=7`, `componentsSucceeded=7`, `componentsFailed=0`.
- After failure: `componentsDone=7`, `componentsSucceeded=6`, `componentsFailed=1`.

When a new bundle apply starts on an existing workspace, Forge clears stale final progress and writes a fresh `initializing` object with the requested `bundleRef`.

## CLI Behavior

The CLI extends `api.ClusterStatus` with an optional `BundleProgress` struct. The polling loop still calls only `GET /v1/cluster`.

Interactive terminal rendering:

```text
[✓] Workspace - provisioning bundle main - this takes a few minutes...
    ⠴ deploying components 7/14: plugin-config (grpc-service, gradle-local, elapsed 03:12, next check in 5s)
```

The spinner line is rewritten in place. It includes:

- human-readable phase label
- dynamic component count and current component when available
- component type and mode when available
- elapsed time
- next poll countdown

Non-interactive rendering:

```text
[✓] Workspace - provisioning bundle main - this takes a few minutes...
    • phase: installing vCluster
    • phase: deploying components 7/14: plugin-config
    • state: active
```

Non-TTY output emits sparse lines only when the rendered progress summary changes. It never writes ANSI cursor controls.

If Forge does not return `bundleProgress`, the CLI falls back to the current `state: creating` behavior plus elapsed/next-check in the spinner line.

## Rendering Labels

CLI labels map phase values to short English text:

- `ensuring_namespace` -> `ensuring namespace`
- `installing_vcluster` -> `installing vCluster`
- `waiting_for_vcluster` -> `waiting for vCluster API`
- `deploying_components` -> `deploying components`

Unknown phase values render as the raw phase with underscores replaced by spaces. This keeps the CLI useful if Forge adds a phase before the CLI updates.

## Error Handling

- If `GET /v1/cluster` returns a brief 404 after the 202, the existing row grace behavior remains.
- If the workspace reaches `failed`, the spinner is cleared, the final failure status renders, and `failureReason` remains the primary error detail.
- If the CLI times out, the error includes the last state and the last progress summary when available.
- If the user cancels with Ctrl-C, the spinner clears before returning the context error.

## Logging

New Forge logs must follow the repository logging guidelines:

- Log progress-write failures at WARN, not INFO.
- Include `devClusterId`, `namespace`, `phase`, and non-sensitive `reason`.
- Do not log override contents, tokens, kubeconfigs, cookies, passwords, or Helm values.

Example:

```text
Failed to persist bundle progress (devClusterId=..., namespace=vcluster-lukas, phase=deploying_components, reason=db_timeout)
```

## Tests

Forge tests:

- Migration adds `DevCluster.bundleProgress`.
- `serializeStatus()` includes `bundleProgress`.
- `POST /v1/cluster/bundle` starts with fresh progress.
- Reconciler writes infrastructure phases.
- Reconciler writes dynamic component progress based on `ResolvedBundle.components`.
- Reconciler marks `failed` progress when the reconcile fails.

CLI tests:

- API model decodes `bundleProgress`.
- Progress summaries render expected phase/component text.
- Non-TTY polling output emits sparse progress lines without ANSI cursor controls.
- TTY progress renderer clears the spinner on success, failure, timeout, and cancellation.
- Missing `bundleProgress` falls back gracefully.

## Rollout

1. Deploy Forge with the additive `bundleProgress` field and status serialization.
2. Release CLI support. Older Forge remains compatible because the CLI treats missing progress as optional.
3. Optionally update Portal later to display the same `bundleProgress` during bundle applies.
