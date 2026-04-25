# Changelog

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
