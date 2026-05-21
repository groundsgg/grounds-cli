# Changelog

## [0.3.0](https://github.com/groundsgg/grounds-cli/compare/v0.2.0...v0.3.0) (2026-05-21)


### Features

* **init:** scaffold app flavor manifests ([#54](https://github.com/groundsgg/grounds-cli/issues/54)) ([4a657e2](https://github.com/groundsgg/grounds-cli/commit/4a657e2854f3891449801e9aa4666db8b75a3a68))

## [0.2.0](https://github.com/groundsgg/grounds-cli/compare/v0.1.19...v0.2.0) (2026-05-20)


### Features

* **push:** pass app flavor selection ([#51](https://github.com/groundsgg/grounds-cli/issues/51)) ([2138009](https://github.com/groundsgg/grounds-cli/commit/21380097ffa714640cc3aeca7b2efdf2bf6ede52))

## [0.1.19](https://github.com/groundsgg/grounds-cli/compare/v0.1.18...v0.1.19) (2026-05-14)


### Features

* add local plugin override workspace ([#49](https://github.com/groundsgg/grounds-cli/issues/49)) ([50149b0](https://github.com/groundsgg/grounds-cli/commit/50149b0fddccccbe84502db220116f737724526d))

## [0.1.18](https://github.com/groundsgg/grounds-cli/compare/v0.1.17...v0.1.18) (2026-05-09)


### Features

* **init:** select base images from catalog ([#47](https://github.com/groundsgg/grounds-cli/issues/47)) ([b2cc06c](https://github.com/groundsgg/grounds-cli/commit/b2cc06cb34f80bfe518b3f0bd0cfd006d60a7149))

## [0.1.17](https://github.com/groundsgg/grounds-cli/compare/v0.1.16...v0.1.17) (2026-05-08)


### Bug Fixes

* **auth:** dual-write credentials to keyring AND file ([#44](https://github.com/groundsgg/grounds-cli/issues/44)) ([c0142ea](https://github.com/groundsgg/grounds-cli/commit/c0142ea02d0c2951e9d35610fb067afb05d2503d))

## [0.1.16](https://github.com/groundsgg/grounds-cli/compare/v0.1.15...v0.1.16) (2026-05-06)


### Bug Fixes

* detect homebrew cask installs ([#42](https://github.com/groundsgg/grounds-cli/issues/42)) ([a2ec0a1](https://github.com/groundsgg/grounds-cli/commit/a2ec0a1a76da77d002c9408dab49ae72245e4acc))

## [0.1.15](https://github.com/groundsgg/grounds-cli/compare/v0.1.14...v0.1.15) (2026-05-06)


### Bug Fixes

* **auth:** treat refresh_expires_in=0 as no-expiry for offline tokens ([#40](https://github.com/groundsgg/grounds-cli/issues/40)) ([81355c7](https://github.com/groundsgg/grounds-cli/commit/81355c7f32c77ab0ed6202bb7db87127415e86f3))

## [0.1.14](https://github.com/groundsgg/grounds-cli/compare/v0.1.13...v0.1.14) (2026-05-05)


### Features

* **push:** --force flag to skip forge contentHash dedup ([#38](https://github.com/groundsgg/grounds-cli/issues/38)) ([712863d](https://github.com/groundsgg/grounds-cli/commit/712863d058b3e502a8d41236a257a6e871db4af6))

## [0.1.13](https://github.com/groundsgg/grounds-cli/compare/v0.1.12...v0.1.13) (2026-05-04)


### Features

* implement update checking ([#33](https://github.com/groundsgg/grounds-cli/issues/33)) ([0c8367d](https://github.com/groundsgg/grounds-cli/commit/0c8367dd51aa6553a2c217b2931f61fc27e49e77))
* improve CLI UX consistency ([#36](https://github.com/groundsgg/grounds-cli/issues/36)) ([45a0855](https://github.com/groundsgg/grounds-cli/commit/45a0855fb5881b32865dc51a3e4c66bb88993aaa))


### Bug Fixes

* update github actions deps ([#35](https://github.com/groundsgg/grounds-cli/issues/35)) ([a646fa9](https://github.com/groundsgg/grounds-cli/commit/a646fa99b60ec749d53e48b86a12babcb7ebc2d4))

## [0.1.12](https://github.com/groundsgg/grounds-cli/compare/v0.1.11...v0.1.12) (2026-05-04)


### Features

* **bundle:** grounds bundle list / show ([#29](https://github.com/groundsgg/grounds-cli/issues/29)) ([ad42a5f](https://github.com/groundsgg/grounds-cli/commit/ad42a5fb428ac2d458c7abedeafeacdd9b0deb7a))
* **cluster:** add --bundle / --override to drive PlatformBundle deploys ([#27](https://github.com/groundsgg/grounds-cli/issues/27)) ([b9b07f1](https://github.com/groundsgg/grounds-cli/commit/b9b07f181cc6b518fcd12ede1f34b056735b8e0f))
* **devspace:** add `grounds devspace generate <component>` subcommand ([#28](https://github.com/groundsgg/grounds-cli/issues/28)) ([82ba13a](https://github.com/groundsgg/grounds-cli/commit/82ba13ac7d98792097272a04defc70d6c224dc73))

## [0.1.11](https://github.com/groundsgg/grounds-cli/compare/v0.1.10...v0.1.11) (2026-04-30)


### Bug Fixes

* **release:** publish Homebrew formula instead of cask ([#24](https://github.com/groundsgg/grounds-cli/issues/24)) ([03fd213](https://github.com/groundsgg/grounds-cli/commit/03fd213f604001bfc32d033f9dc16d82ad6e6b1d))

## [0.1.10](https://github.com/groundsgg/grounds-cli/compare/v0.1.9...v0.1.10) (2026-04-30)


### Bug Fixes

* **release:** drop `environment` — GlitchTip lacks Sentry deploys API ([c3728cf](https://github.com/groundsgg/grounds-cli/commit/c3728cf98937a4528cf9090fe784d96e54093f5c))

## [0.1.9](https://github.com/groundsgg/grounds-cli/compare/v0.1.8...v0.1.9) (2026-04-30)


### Features

* **auth:** request offline_access scope in device flow ([#21](https://github.com/groundsgg/grounds-cli/issues/21)) ([304931b](https://github.com/groundsgg/grounds-cli/commit/304931b0525798c4e7cab344788871ddde4acc9a))


### Bug Fixes

* **release:** drop unsupported version_prefix; expose release-please version output ([cf4702e](https://github.com/groundsgg/grounds-cli/commit/cf4702e4aefdf17958f7cab44175127b10d4b8bd))

## [0.1.8](https://github.com/groundsgg/grounds-cli/compare/v0.1.7...v0.1.8) (2026-04-30)


### Features

* **observability:** wire Sentry/GlitchTip error tracking ([cb9c0eb](https://github.com/groundsgg/grounds-cli/commit/cb9c0eb05d1ef3bf661a1e3842360a5513672c40))


### Bug Fixes

* avoid powershell browser opener on wsl ([#19](https://github.com/groundsgg/grounds-cli/issues/19)) ([93bc212](https://github.com/groundsgg/grounds-cli/commit/93bc212bdc5093b7353ccbe13f039b92431c93db))

## [0.1.7](https://github.com/groundsgg/grounds-cli/compare/v0.1.6...v0.1.7) (2026-04-29)


### Bug Fixes

* **auth:** write version field in credentials.json for grounds-push compat ([#16](https://github.com/groundsgg/grounds-cli/issues/16)) ([78839a1](https://github.com/groundsgg/grounds-cli/commit/78839a16f072a49a102b40263e97c13fd701a80a))
* **push:** pre-refresh access token before invoking Gradle plugin ([#18](https://github.com/groundsgg/grounds-cli/issues/18)) ([109da76](https://github.com/groundsgg/grounds-cli/commit/109da769af0dddf87ca4771cfaab54a031fbd28e))

## [0.1.6](https://github.com/groundsgg/grounds-cli/compare/v0.1.5...v0.1.6) (2026-04-28)


### Features

* **cli:** add --profile flag to grounds cluster up ([#14](https://github.com/groundsgg/grounds-cli/issues/14)) ([fa5159d](https://github.com/groundsgg/grounds-cli/commit/fa5159ddeb01332fcccfaa937beda565ebe3517f))

## [0.1.5](https://github.com/groundsgg/grounds-cli/compare/v0.1.4...v0.1.5) (2026-04-28)


### Bug Fixes

* **doctor:** refresh expired access tokens instead of reporting failure ([#12](https://github.com/groundsgg/grounds-cli/issues/12)) ([96e969b](https://github.com/groundsgg/grounds-cli/commit/96e969b1192b49820fc9114bb39685de1768e5b9))

## [0.1.4](https://github.com/groundsgg/grounds-cli/compare/v0.1.3...v0.1.4) (2026-04-28)


### Bug Fixes

* **config:** default APIURL to platform.grnds.io (forge subdomain doesn't exist) ([#10](https://github.com/groundsgg/grounds-cli/issues/10)) ([d813b0a](https://github.com/groundsgg/grounds-cli/commit/d813b0a1d2450444836e7bcb80f1d65a4d63a22c))

## [0.1.3](https://github.com/groundsgg/grounds-cli/compare/v0.1.2...v0.1.3) (2026-04-28)


### Bug Fixes

* **auth:** add PKCE to device flow (Keycloak now enforces) ([#8](https://github.com/groundsgg/grounds-cli/issues/8)) ([cae6df9](https://github.com/groundsgg/grounds-cli/commit/cae6df9ec7682fc89de7d06d74bb21e9e517d190))

## [0.1.2](https://github.com/groundsgg/grounds-cli/compare/v0.1.1...v0.1.2) (2026-04-28)


### Features

* **cli:** T8 — global --project flag ([#4](https://github.com/groundsgg/grounds-cli/issues/4)) ([dfd42ad](https://github.com/groundsgg/grounds-cli/commit/dfd42ad85497ccdb5fc60494c0eb81455e52be82))
* **preview:** grounds preview list/show/pin/unpin subcommands ([#7](https://github.com/groundsgg/grounds-cli/issues/7)) ([af9e31b](https://github.com/groundsgg/grounds-cli/commit/af9e31bca10e32ba36500dc8b6af415ca7ba5f12))
* **push:** support --target=staging for preview envs ([#6](https://github.com/groundsgg/grounds-cli/issues/6)) ([224cded](https://github.com/groundsgg/grounds-cli/commit/224cded7c2991418e4504a9c5900ca1906dc7cf1))

## [0.1.1](https://github.com/groundsgg/grounds-cli/compare/v0.1.0...v0.1.1) (2026-04-25)


### Features

* **api:** /v1/cluster GET/up/down/delete bindings ([e7827eb](https://github.com/groundsgg/grounds-cli/commit/e7827eb393b86d035c086a58eb80170304f9f5ae))
* **api:** /v1/pushes + /v1/deployments bindings ([33df5c0](https://github.com/groundsgg/grounds-cli/commit/33df5c0de4027d3ba173e1fc898aca52c0e1a694))
* **api:** http client + typed error mapping ([4ce0d5a](https://github.com/groundsgg/grounds-cli/commit/4ce0d5a6d803eeb95050ce727aed1a04da1b843a))
* **auth:** credentials store with keyring + 0600 file fallback ([b39892a](https://github.com/groundsgg/grounds-cli/commit/b39892a281b40035fb8cf3009f3f7c1332528637))
* **auth:** OAuth device flow + token refresh ([0c545b2](https://github.com/groundsgg/grounds-cli/commit/0c545b2bd0c7bb8feb9f8a4b098d5ce106cb8d31))
* bootstrap grounds CLI with cobra root + version command ([95e2a3a](https://github.com/groundsgg/grounds-cli/commit/95e2a3a6c11db037c617a3b098cd190f01586d13))
* **cli:** add completion subcommand for bash/zsh/fish/pwsh ([8594624](https://github.com/groundsgg/grounds-cli/commit/8594624ef86b8b5f2d9d72d1d5f6a103cd9dee56))
* **cli:** grounds cluster up/down/delete/status ([3034096](https://github.com/groundsgg/grounds-cli/commit/303409682a58435371d57c3d7533d71137e03550))
* **cli:** grounds doctor diagnoses config/auth/api/gradle/java ([16dd0bc](https://github.com/groundsgg/grounds-cli/commit/16dd0bcfc31f2debbbe9697ceacbbac43c0f9507))
* **cli:** grounds init scaffolds grounds.yaml ([945930d](https://github.com/groundsgg/grounds-cli/commit/945930df937f627996f574130a70f89c26500a6e))
* **cli:** grounds login/logout via OAuth device flow ([935fd64](https://github.com/groundsgg/grounds-cli/commit/935fd64f662c030aa1be3c695cd0e4cd34a50365))
* **cli:** grounds logs &lt;pushId&gt; + grounds logs deployment ([3c14994](https://github.com/groundsgg/grounds-cli/commit/3c14994b961029b991076e1e51490712ed7bff01))
* **cli:** grounds push retry + list with SSE streaming ([76de9ab](https://github.com/groundsgg/grounds-cli/commit/76de9abf10d9a79f7b43a329d2f76f0cf304d37b))
* **cli:** grounds push wraps ./gradlew groundsPush ([bff4ce6](https://github.com/groundsgg/grounds-cli/commit/bff4ce66c47c193a0c0578a05b1161d1f848d4d6))
* **config:** viper-backed config with env override + XDG dirs ([6600c59](https://github.com/groundsgg/grounds-cli/commit/6600c593d6da89afb597cac880bb2e81c666dd8d))
* **render:** table + cluster status renderer + json/yaml ([e9e4357](https://github.com/groundsgg/grounds-cli/commit/e9e4357345a146e226d1749c368365440545e82b))
* **sse:** SSE stream subscriber with bounded backoff + renderer ([cae62db](https://github.com/groundsgg/grounds-cli/commit/cae62db970355ac4a0cec5c601d7bcbbaf9f6afe))


### Bug Fixes

* **tests:** make credentials + gradle tests Windows-aware ([ec13cf3](https://github.com/groundsgg/grounds-cli/commit/ec13cf30b7d8cba3fd5a303ef406fec415d2da15))

## 0.1.0 (2026-04-25)


### Features

* **api:** /v1/cluster GET/up/down/delete bindings ([e7827eb](https://github.com/groundsgg/grounds-cli/commit/e7827eb393b86d035c086a58eb80170304f9f5ae))
* **api:** /v1/pushes + /v1/deployments bindings ([33df5c0](https://github.com/groundsgg/grounds-cli/commit/33df5c0de4027d3ba173e1fc898aca52c0e1a694))
* **api:** http client + typed error mapping ([4ce0d5a](https://github.com/groundsgg/grounds-cli/commit/4ce0d5a6d803eeb95050ce727aed1a04da1b843a))
* **auth:** credentials store with keyring + 0600 file fallback ([b39892a](https://github.com/groundsgg/grounds-cli/commit/b39892a281b40035fb8cf3009f3f7c1332528637))
* **auth:** OAuth device flow + token refresh ([0c545b2](https://github.com/groundsgg/grounds-cli/commit/0c545b2bd0c7bb8feb9f8a4b098d5ce106cb8d31))
* bootstrap grounds CLI with cobra root + version command ([95e2a3a](https://github.com/groundsgg/grounds-cli/commit/95e2a3a6c11db037c617a3b098cd190f01586d13))
* **cli:** add completion subcommand for bash/zsh/fish/pwsh ([8594624](https://github.com/groundsgg/grounds-cli/commit/8594624ef86b8b5f2d9d72d1d5f6a103cd9dee56))
* **cli:** grounds cluster up/down/delete/status ([3034096](https://github.com/groundsgg/grounds-cli/commit/303409682a58435371d57c3d7533d71137e03550))
* **cli:** grounds doctor diagnoses config/auth/api/gradle/java ([16dd0bc](https://github.com/groundsgg/grounds-cli/commit/16dd0bcfc31f2debbbe9697ceacbbac43c0f9507))
* **cli:** grounds init scaffolds grounds.yaml ([945930d](https://github.com/groundsgg/grounds-cli/commit/945930df937f627996f574130a70f89c26500a6e))
* **cli:** grounds login/logout via OAuth device flow ([935fd64](https://github.com/groundsgg/grounds-cli/commit/935fd64f662c030aa1be3c695cd0e4cd34a50365))
* **cli:** grounds logs &lt;pushId&gt; + grounds logs deployment ([3c14994](https://github.com/groundsgg/grounds-cli/commit/3c14994b961029b991076e1e51490712ed7bff01))
* **cli:** grounds push retry + list with SSE streaming ([76de9ab](https://github.com/groundsgg/grounds-cli/commit/76de9abf10d9a79f7b43a329d2f76f0cf304d37b))
* **cli:** grounds push wraps ./gradlew groundsPush ([bff4ce6](https://github.com/groundsgg/grounds-cli/commit/bff4ce66c47c193a0c0578a05b1161d1f848d4d6))
* **config:** viper-backed config with env override + XDG dirs ([6600c59](https://github.com/groundsgg/grounds-cli/commit/6600c593d6da89afb597cac880bb2e81c666dd8d))
* **render:** table + cluster status renderer + json/yaml ([e9e4357](https://github.com/groundsgg/grounds-cli/commit/e9e4357345a146e226d1749c368365440545e82b))
* **sse:** SSE stream subscriber with bounded backoff + renderer ([cae62db](https://github.com/groundsgg/grounds-cli/commit/cae62db970355ac4a0cec5c601d7bcbbaf9f6afe))


### Bug Fixes

* **tests:** make credentials + gradle tests Windows-aware ([ec13cf3](https://github.com/groundsgg/grounds-cli/commit/ec13cf30b7d8cba3fd5a303ef406fec415d2da15))

## [0.1.0](https://github.com/groundsgg/grounds-cli/compare/grounds-cli-v0.0.1...grounds-cli-v0.1.0) (2026-04-25)


### Features

* **api:** /v1/cluster GET/up/down/delete bindings ([e7827eb](https://github.com/groundsgg/grounds-cli/commit/e7827eb393b86d035c086a58eb80170304f9f5ae))
* **api:** /v1/pushes + /v1/deployments bindings ([33df5c0](https://github.com/groundsgg/grounds-cli/commit/33df5c0de4027d3ba173e1fc898aca52c0e1a694))
* **api:** http client + typed error mapping ([4ce0d5a](https://github.com/groundsgg/grounds-cli/commit/4ce0d5a6d803eeb95050ce727aed1a04da1b843a))
* **auth:** credentials store with keyring + 0600 file fallback ([b39892a](https://github.com/groundsgg/grounds-cli/commit/b39892a281b40035fb8cf3009f3f7c1332528637))
* **auth:** OAuth device flow + token refresh ([0c545b2](https://github.com/groundsgg/grounds-cli/commit/0c545b2bd0c7bb8feb9f8a4b098d5ce106cb8d31))
* bootstrap grounds CLI with cobra root + version command ([95e2a3a](https://github.com/groundsgg/grounds-cli/commit/95e2a3a6c11db037c617a3b098cd190f01586d13))
* **cli:** add completion subcommand for bash/zsh/fish/pwsh ([8594624](https://github.com/groundsgg/grounds-cli/commit/8594624ef86b8b5f2d9d72d1d5f6a103cd9dee56))
* **cli:** grounds cluster up/down/delete/status ([3034096](https://github.com/groundsgg/grounds-cli/commit/303409682a58435371d57c3d7533d71137e03550))
* **cli:** grounds doctor diagnoses config/auth/api/gradle/java ([16dd0bc](https://github.com/groundsgg/grounds-cli/commit/16dd0bcfc31f2debbbe9697ceacbbac43c0f9507))
* **cli:** grounds init scaffolds grounds.yaml ([945930d](https://github.com/groundsgg/grounds-cli/commit/945930df937f627996f574130a70f89c26500a6e))
* **cli:** grounds login/logout via OAuth device flow ([935fd64](https://github.com/groundsgg/grounds-cli/commit/935fd64f662c030aa1be3c695cd0e4cd34a50365))
* **cli:** grounds logs &lt;pushId&gt; + grounds logs deployment ([3c14994](https://github.com/groundsgg/grounds-cli/commit/3c14994b961029b991076e1e51490712ed7bff01))
* **cli:** grounds push retry + list with SSE streaming ([76de9ab](https://github.com/groundsgg/grounds-cli/commit/76de9abf10d9a79f7b43a329d2f76f0cf304d37b))
* **cli:** grounds push wraps ./gradlew groundsPush ([bff4ce6](https://github.com/groundsgg/grounds-cli/commit/bff4ce66c47c193a0c0578a05b1161d1f848d4d6))
* **config:** viper-backed config with env override + XDG dirs ([6600c59](https://github.com/groundsgg/grounds-cli/commit/6600c593d6da89afb597cac880bb2e81c666dd8d))
* **render:** table + cluster status renderer + json/yaml ([e9e4357](https://github.com/groundsgg/grounds-cli/commit/e9e4357345a146e226d1749c368365440545e82b))
* **sse:** SSE stream subscriber with bounded backoff + renderer ([cae62db](https://github.com/groundsgg/grounds-cli/commit/cae62db970355ac4a0cec5c601d7bcbbaf9f6afe))


### Bug Fixes

* **tests:** make credentials + gradle tests Windows-aware ([ec13cf3](https://github.com/groundsgg/grounds-cli/commit/ec13cf30b7d8cba3fd5a303ef406fec415d2da15))
