# Minestom Distribution Push Design

## Context

`grounds push` currently builds the local project and delegates deployment to the Gradle `groundsPush` task. The CLI already has a local workspace resolver for plugin-style development:

- `grounds workspace scan <root>` records local repositories in `workspace.yaml`.
- `grounds workspace add <id> <path> --variant=<variant>` adds a local mapping manually.
- `grounds push --local=<id>` and `grounds push --with-local` resolve `grounds.yaml` plugin entries against the local workspace map, build matching local artifacts, and pass a resolved plugin plan to Gradle.

Minestom server development needs the same dynamic local lookup, but the deployable unit is different. A Minestom gamemode or lobby is not a hot-loaded plugin. It is a complete server distribution built from the gamemode plus its selected runtime modules, such as `plugin-config`, `plugin-agones`, and `grounds-vanilla`.

The current Forge upload path already accepts both JAR uploads and gzip bundle uploads. However, current Forge manifest normalization maps public `type: minestom` to internal `type: service`, and service bundle uploads are rejected. This feature therefore needs a small coordinated Forge change: introduce `minestom-server` as a first-class internal runtime type while preserving `publicType: minestom` for user-facing flavor identity.

## Goals

- Keep `grounds push` as the user-facing command for Minestom dev pushes.
- Let developers use `workspace.yaml` local mappings instead of hardcoded relative `../repo` paths.
- Build a complete Minestom server distribution with local module overrides applied through Gradle composite builds.
- Normalize the Gradle distribution into Forge's gzip bundle layout before upload.
- Upload one deployable Minestom distribution bundle to Forge.
- Let Forge build the runtime image from the uploaded distribution and deploy it to the developer workspace.
- Verify the feature with a real `grounds push` flow using local changes and local module mappings.

## Non-Goals

- Do not add a separate `grounds dev sync` command for the first version.
- Do not implement direct pod file sync for Minestom in this feature.
- Do not make Forge understand individual local module paths.
- Do not require gamemode repositories to commit machine-specific paths.
- Do not change `grounds-minestom-runtime` module discovery semantics.

## Command Shape

The preferred command remains:

```bash
grounds push --flavor=minestom --with-local
```

Specific local modules can be selected explicitly:

```bash
grounds push --flavor=minestom --local=plugin-config,plugin-agones
```

The selected flavor in `grounds.yaml` must describe a Minestom server distribution, not a plugin JAR. A representative shape is:

```yaml
name: minigame-bedwars
flavors:
  minestom:
    type: minestom-server
    baseImage: minestom
    build:
      task: :server:distTar
      artifact: server/build/distributions/*.tar
    modules:
      - id: plugin-config
        variant: minestom
        source: github:groundsgg/plugin-config@v0.4.0:plugin-config-minestom.jar
      - id: plugin-agones
        variant: minestom
        source: github:groundsgg/plugin-agones@v0.2.0:plugin-agones-minestom.jar
```

For the first implementation, the CLI should accept this explicit `build.task`, `build.artifact`, and `modules` shape for Minestom flavors. Existing plugin flavors keep their current behavior.

`modules` is CLI composition metadata. It tells `grounds push` which workspace entries can be selected for local override and which released source each module would normally use. Gradle dependencies in the Minestom project still decide the actual compile/runtime classpath.

## Workspace Lookup

The existing workspace map remains the source of truth for local repository paths:

```yaml
repos:
  plugin-config:
    path: /home/lukas/grounds/plugin-config
    variants:
      minestom:
        enabled: true
        module: gg.grounds:plugin-config-minestom
        project: :minestom
  plugin-agones:
    path: /home/lukas/grounds/plugin-agones
    variants:
      minestom:
        enabled: true
        module: gg.grounds:plugin-agones-minestom
        project: :minestom
```

`module` and `project` are optional in the first pass when Gradle's standard composite substitution is enough. They are kept in the design as escape hatches for repositories whose local project path or published coordinates do not follow conventions.

`grounds workspace scan` should continue to discover direct child repositories and known variants. It can later infer Minestom module metadata from Gradle publications, but the first implementation can rely on manually configured metadata when conventions are not enough.

## Composite Build Overlay

For Minestom pushes with local overrides, the CLI generates a temporary Gradle init script from the resolved workspace entries. The script includes the selected local repositories as composite builds, so the gamemode can keep normal released dependencies:

```kotlin
settingsEvaluated {
    includeBuild("/home/lukas/grounds/plugin-config")
    includeBuild("/home/lukas/grounds/plugin-agones")
}
```

The gamemode build keeps normal Maven coordinates:

```kotlin
implementation("gg.grounds:plugin-config-minestom:0.4.0")
implementation("gg.grounds:plugin-agones-minestom:0.2.0")
```

When the local builds publish matching Gradle module identity, Gradle substitutes those dependencies with local projects automatically. If a repository needs explicit substitution later, the `module` and `project` metadata in `workspace.yaml` can drive that without changing the gamemode repository.

The generated init script is:

- created under the OS temp directory;
- passed only to the local Gradle invocation;
- deleted after `grounds push` exits;
- never committed into the gamemode repository.

## Minestom Push Flow

For `grounds push --flavor=minestom --with-local`:

1. CLI locates the project Gradle wrapper.
2. CLI loads `grounds.yaml` and selects the `minestom` flavor.
3. CLI loads `workspace.yaml`.
4. CLI resolves enabled or explicitly requested local workspace entries for modules listed by the Minestom flavor.
5. CLI writes a temporary composite-build init script.
6. CLI runs the Minestom distribution build task, for example:

   ```bash
   ./gradlew :server:distTar -I /tmp/grounds-minestom-composites.gradle.kts
   ```

7. CLI resolves the configured distribution artifact, for example:

   ```text
   server/build/distributions/minigame-bedwars.tar
   ```

8. CLI normalizes the Gradle distribution into a temporary gzip tarball whose root is `app/` and whose launcher is `app/bin/app`.
9. CLI uploads the normalized distribution bundle directly to Forge's existing `/v1/pushes` multipart endpoint.
10. Forge receives one built Minestom distribution bundle.
11. Forge builds an image from that distribution and deploys it to the workspace.

For `grounds push --flavor=minestom` without local overrides, the same flow runs without the generated composite init script.

## Forge Contract

Forge should receive a single resolved deployable artifact plus metadata. It should not receive local repository paths or per-module local state.

Minimum metadata:

```json
{
  "flavor": "minestom",
  "type": "minestom-server",
  "baseImage": "minestom",
  "artifactName": "minigame-bedwars.tar",
  "artifactSha256": "...",
  "localOverrides": [
    {
      "id": "plugin-config",
      "variant": "minestom",
      "path": "/home/lukas/grounds/plugin-config",
      "git": {
        "remote": "git@github.com:groundsgg/plugin-config.git",
        "commit": "...",
        "dirty": true
      }
    }
  ]
}
```

The path is useful for local CLI output and audit logs before upload, but Forge should not rely on it for image builds. The uploaded distribution is the source of truth.

Gradle's `distTar` output usually extracts to a versioned root directory such as `minigame-agones-local-SNAPSHOT/` with a generated launcher such as `bin/minigame-agones`. Forge already treats bundle uploads as gzip tarballs extracted directly into the Kaniko build context. To keep Forge deterministic and avoid requiring every Minestom project to hardcode Gradle distribution names, the CLI owns a normalization step:

- accept `.tar`, `.tar.gz`, or `.tgz` Gradle distribution outputs;
- extract the single Gradle distribution root into a temporary staging directory;
- repackage it as `app/` in a gzip tarball;
- rename the non-Windows generated launcher under `bin/` to `app`;
- preserve executable file modes for launcher scripts;
- upload only the normalized gzip tarball to Forge.

Forge must treat `minestom-server` as a Minecraft workload, not as a generic HTTP service:

- `publicType` remains `minestom`.
- The base image catalog includes a `minestom` entry whose manifest type is `minestom-server`.
- Workload rendering uses the Minecraft port, Minecraft public URL, and Minecraft routing behavior.
- Bundle/distribution uploads are allowed for `minestom-server`.
- Generic `service` bundle uploads remain rejected.
- The Minestom distribution Dockerfile copies the extracted Gradle application distribution into `/app` and starts the generated launch script.

## Portal Behavior

Portal does not need to understand local module composition in the first version. It should be able to show that a workspace component is running from a local `grounds push` artifact and display the resulting image/build status from Forge.

Later Portal can expose a higher-level composition editor for Git refs and released versions. That is separate from local uncommitted-code development, which remains CLI-driven.

## Verification Requirements

Implementation is not complete until it is verified with real push behavior, not only unit tests.

Required local verification:

1. Create or use a Minestom reference project whose `grounds.yaml` has a `minestom-server` flavor.
2. Configure local workspace mappings for at least `plugin-agones:minestom` and one second module such as `plugin-config:minestom` or a small temporary reference module.
3. Make a local uncommitted change in one mapped module that is observable in the built distribution, such as a version marker string or test resource.
4. Run:

   ```bash
   grounds workspace list
   grounds push --flavor=minestom --with-local
   ```

5. Confirm the CLI generated and used the composite-build init script.
6. Confirm the distribution artifact contains the local module output, not the released dependency.
7. Confirm the push path uploads the normalized Minestom distribution bundle to Forge.
8. Confirm Forge accepts the upload and starts the image build/deploy flow.

If the live Forge environment is unavailable, the implementation must still run a local or test-server `grounds push` equivalent that exercises the CLI upload path and must clearly document that live Forge deployment was not verified.

Required automated tests:

- Manifest parsing recognizes a `minestom-server` flavor with `build.task` and `build.artifact`.
- Workspace resolution selects enabled Minestom local entries for `--with-local`.
- Explicit `--local=<id>` selects only requested local Minestom entries.
- Composite init script generation is deterministic and includes each selected local repository once.
- Minestom push invokes Gradle with the configured build task and generated `-I` argument.
- Artifact resolution prefers the configured distribution artifact over plugin JAR heuristics.
- CLI multipart upload sends `manifest`, `target`, optional `flavor`, optional `effectivePluginSources`, and the distribution artifact as field `jar`.
- Forge accepts a `minestom-server` gzip distribution upload.
- Forge still rejects generic `service` gzip bundle uploads.
- Forge renders Minestom workloads with Minecraft routing defaults.
- Non-Minestom push behavior remains unchanged.

## Rollout

1. Implement and test Forge support for `minestom-server` distributions so the upload target can accept the new artifact type.
2. Implement and test the CLI Minestom distribution build/upload path behind `type: minestom-server`.
3. Use `grounds-minestom-runtime` or a small reference fixture to prove local composite substitution.
4. Verify `grounds push --flavor=minestom --with-local` with local module mappings and local changes.
5. Keep non-Minestom push behavior on the existing Gradle `groundsPush` delegation path.
