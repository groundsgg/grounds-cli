# Changelog

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
