## [3.0.4](https://github.com/stuttgart-things/homerun-library/compare/v3.0.3...v3.0.4) (2026-03-10)


### Bug Fixes

* update module path from v2 to v3 to match semver tags ([#78](https://github.com/stuttgart-things/homerun-library/issues/78)) ([509e925](https://github.com/stuttgart-things/homerun-library/commit/509e92554f71475c622b38165235a72254a30faa))

## [3.0.3](https://github.com/stuttgart-things/homerun-library/compare/v3.0.2...v3.0.3) (2026-03-08)


### Bug Fixes

* **deps:** update module github.com/redisearch/redisearch-go to v2 ([a578c49](https://github.com/stuttgart-things/homerun-library/commit/a578c491cebbbb7c547ca9ab06c218441d67ca47))

## [3.0.2](https://github.com/stuttgart-things/homerun-library/compare/v3.0.1...v3.0.2) (2026-03-07)


### Bug Fixes

* use t.Setenv in TestGetEnv to fix errcheck lint errors ([38c723f](https://github.com/stuttgart-things/homerun-library/commit/38c723f9dc34fface7398a8aa70b9bfebeaf85d1))

## [3.0.1](https://github.com/stuttgart-things/homerun-library/compare/v3.0.0...v3.0.1) (2026-03-07)


### Bug Fixes

* ignore .dagger directory in Renovate config ([3b9ea10](https://github.com/stuttgart-things/homerun-library/commit/3b9ea101273f3dc06d045b567bb759dbbcc78ef8))

# [3.0.0](https://github.com/stuttgart-things/homerun-library/compare/v2.0.0...v3.0.0) (2026-03-07)


* feat!: add /v2 module path for proper Go modules v2 semantics ([0c23391](https://github.com/stuttgart-things/homerun-library/commit/0c23391d05120a982a06deaad4494597ea1af610))


### BREAKING CHANGES

* import path changed to github.com/stuttgart-things/homerun-library/v2

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>

# [2.0.0](https://github.com/stuttgart-things/homerun-library/compare/v1.2.0...v2.0.0) (2026-03-07)


* fix!: replace log.Fatalf and panic with proper error returns ([6b09528](https://github.com/stuttgart-things/homerun-library/commit/6b09528cde204443dca7aaa76438a57a7bee454c)), closes [#51](https://github.com/stuttgart-things/homerun-library/issues/51)
* fix!: return error from SendToHomerun instead of nil, nil ([b39f1e0](https://github.com/stuttgart-things/homerun-library/commit/b39f1e0064ea5a7e0a4a7b7a4b22cd17c8097da8)), closes [#52](https://github.com/stuttgart-things/homerun-library/issues/52)
* fix!: return errors from EnqueueMessageInRedisStreams and StoreInRediSearch ([35d79e0](https://github.com/stuttgart-things/homerun-library/commit/35d79e0f01da91537c85c0aba12111447dfad94c)), closes [#53](https://github.com/stuttgart-things/homerun-library/issues/53)
* refactor!: replace map[string]string with typed RedisConfig struct ([b54ec16](https://github.com/stuttgart-things/homerun-library/commit/b54ec161e775bb27cda5bc714add77d5aa31c2a3)), closes [#54](https://github.com/stuttgart-things/homerun-library/issues/54)


### Bug Fixes

* **deps:** update module github.com/99designs/gqlgen to v0.17.87 ([df6e35b](https://github.com/stuttgart-things/homerun-library/commit/df6e35be17d378fce376dd0fe72e32bb8fb69016))
* **deps:** update module github.com/jedib0t/go-pretty/v6 to v6.6.8 ([e31ba40](https://github.com/stuttgart-things/homerun-library/commit/e31ba409dd17996c8e7034ddcdcbe7731229c8b2))
* **deps:** update module github.com/jedib0t/go-pretty/v6 to v6.7.1 ([43f877c](https://github.com/stuttgart-things/homerun-library/commit/43f877c954ed06e21f393a5220060df8503d6636))
* **deps:** update module github.com/jedib0t/go-pretty/v6 to v6.7.5 ([7e9992b](https://github.com/stuttgart-things/homerun-library/commit/7e9992b659a751f138c8ef86afc729ccca152f27))
* **deps:** update module github.com/jedib0t/go-pretty/v6 to v6.7.8 ([2dea535](https://github.com/stuttgart-things/homerun-library/commit/2dea535855e700fd8014e01775c77eab10e80e77))
* **deps:** update module github.com/pterm/pterm to v0.12.81 ([#21](https://github.com/stuttgart-things/homerun-library/issues/21)) ([9cb86a8](https://github.com/stuttgart-things/homerun-library/commit/9cb86a883b3ce6598f81b15ceb0da9fce8822b20))
* **deps:** update module github.com/pterm/pterm to v0.12.82 ([b25c5d2](https://github.com/stuttgart-things/homerun-library/commit/b25c5d2e9149610875f8d9af235fea2cd3b4b7a0))
* **deps:** update module github.com/pterm/pterm to v0.12.83 ([b00f67a](https://github.com/stuttgart-things/homerun-library/commit/b00f67a2992eb2a6bf7461fb7e08b2b6e47ece68))
* **deps:** update module github.com/redisearch/redisearch-go to v2 ([71fe781](https://github.com/stuttgart-things/homerun-library/commit/71fe781e9635cb29f97e2175e797a9026c968f13))
* **deps:** update module github.com/redisearch/redisearch-go to v2 ([9384331](https://github.com/stuttgart-things/homerun-library/commit/9384331f865a17d2fe9240db9c3570d82548b3eb))
* **deps:** update module github.com/redisearch/redisearch-go to v2 ([734c83d](https://github.com/stuttgart-things/homerun-library/commit/734c83d23a97e2df17728dd2f12e7b6622b58f25))
* **deps:** update module github.com/redisearch/redisearch-go to v2 ([b025c1e](https://github.com/stuttgart-things/homerun-library/commit/b025c1e31ef6dcd39b7dafabcbac51b925bd40e0))
* **deps:** update module github.com/stretchr/testify to v1.11.1 ([9c559d3](https://github.com/stuttgart-things/homerun-library/commit/9c559d3a6757a84b3b13c9fde7c1e4a3f7c6f1a7))
* **deps:** update module github.com/stretchr/testify to v1.11.1 ([ac3819f](https://github.com/stuttgart-things/homerun-library/commit/ac3819f2d1ee0a8eabbaa553f71213031f983179))
* **deps:** update module github.com/vektah/gqlparser/v2 to v2.5.31 ([d44bab9](https://github.com/stuttgart-things/homerun-library/commit/d44bab9a22b5d65e0913ab38b82bc0bdd15e86a2))
* **deps:** update module go.opentelemetry.io/otel/sdk to v1.40.0 [security] ([5bd03a2](https://github.com/stuttgart-things/homerun-library/commit/5bd03a2197272537eed4cd74ed2bf6924d0a798c))
* **deps:** update module go.opentelemetry.io/otel/sdk to v1.42.0 ([6b7c679](https://github.com/stuttgart-things/homerun-library/commit/6b7c679bb3b30851c2ef1a40e3dca239f298a112))
* **deps:** update module go.opentelemetry.io/proto/otlp to v1.9.0 ([d686b1c](https://github.com/stuttgart-things/homerun-library/commit/d686b1cd0594b737439692c7f4dc2bfe44ae3492))
* **deps:** update module golang.org/x/sync to v0.18.0 ([#35](https://github.com/stuttgart-things/homerun-library/issues/35)) ([76f4499](https://github.com/stuttgart-things/homerun-library/commit/76f4499ecf59faf78ed4558e97b757be17bf93c2))
* **deps:** update module google.golang.org/grpc to v1.79.2 ([11f7c15](https://github.com/stuttgart-things/homerun-library/commit/11f7c156545713478f0470a813aa90cbcbe54cbf))
* **deps:** update opentelemetry-go monorepo ([0d4f0b5](https://github.com/stuttgart-things/homerun-library/commit/0d4f0b5ac11f9fcc92052f3dff50c0f1c4d22b90))
* regenerate .dagger go.sum after gqlgen update ([f187940](https://github.com/stuttgart-things/homerun-library/commit/f187940149cd0d3dcbc12cd6696f432eabeb0872))
* regenerate .dagger go.sum after grpc update ([a72dffb](https://github.com/stuttgart-things/homerun-library/commit/a72dffb612c54a3e9e6aba8862220d2baa310a39))
* remove broken verify workflow ([4a004f7](https://github.com/stuttgart-things/homerun-library/commit/4a004f7a72d5b43c4bf6ef0f4877e9024653230a))
* remove deprecated rand seeding pattern in helpers.go ([ed0d1b8](https://github.com/stuttgart-things/homerun-library/commit/ed0d1b8c1aec54885eadbbc77634403c90978075)), closes [#55](https://github.com/stuttgart-things/homerun-library/issues/55)


### Features

* add release and pages deployment workflows ([ffd41c0](https://github.com/stuttgart-things/homerun-library/commit/ffd41c04aa46e5ea3f7033d1b25d1b2f872b9bdc))
* feat/add-dagger-linting ([4c935fe](https://github.com/stuttgart-things/homerun-library/commit/4c935fe15ef9ce54a2c28216d6b2c1490951aa7f))
* feat/add-dagger-module ([0508596](https://github.com/stuttgart-things/homerun-library/commit/0508596cf64a87134977b90b7c699733f14aef58))
* feat/add-dagger-module ([5c42ba4](https://github.com/stuttgart-things/homerun-library/commit/5c42ba47f3f165c50314d77a02c61441bda3e123))
* feat/add-tests ([c7e8797](https://github.com/stuttgart-things/homerun-library/commit/c7e8797fbf28c6c1255fa1a6ea94e16200f79e06))
* feat/update-test-task ([bd33218](https://github.com/stuttgart-things/homerun-library/commit/bd33218ad62e2bde0035c10fc48e72d8291a36f4))
* fix/linting ([5460dd8](https://github.com/stuttgart-things/homerun-library/commit/5460dd8e1d7613d74d687826352318ed0d00971c))
* main ([8837684](https://github.com/stuttgart-things/homerun-library/commit/883768414c9f60538cc2c38b12e8e664d6805cd2))
* main ([69947c0](https://github.com/stuttgart-things/homerun-library/commit/69947c0efaf8e02a2ad0c54fb43ae49fa9e213f0))
* main ([d1097de](https://github.com/stuttgart-things/homerun-library/commit/d1097dee1bab0889fce39a0bba8daffd306e8d29))
* main ([dd0903d](https://github.com/stuttgart-things/homerun-library/commit/dd0903df72571a4a755519ec28171df898fe13e5))
* main ([7ac4aac](https://github.com/stuttgart-things/homerun-library/commit/7ac4aac40ed2da7e074530f508f4cbd34f65c9c0))
* renovate/actions-checkout-6.x ([db66e58](https://github.com/stuttgart-things/homerun-library/commit/db66e58b62766a758a0a80fe5a1022663a9d07a0))


### BREAKING CHANGES

* RenderBody now returns (string, error) instead of string.

- message.go: GetMessageJSON returns error instead of calling log.Fatalf
- message.go: GetMessageJSON returns error when Redis key not found
- send.go: RenderBody returns (string, error) instead of panicking
- send_test.go: update TestRenderBody for new signature
* EnqueueMessageInRedisStreams now returns (objectID, streamID, error).
StoreInRediSearch now returns error.

- pitcher.go: return error on failed enqueue instead of just logging
- redisearch.go: return error on index check failure instead of silently continuing
- send_test.go: update test for new signature
* EnqueueMessageInRedisStreams and StoreInRediSearch now
accept RedisConfig instead of map[string]string.

- Add RedisConfig struct with Addr, Port, Password, Stream, Index fields
- Update pitcher.go, redisearch.go, tests, and godoc examples
* SendToHomerun now returns ([]byte, *http.Response, error)
instead of ([]byte, *http.Response).

- Return wrapped errors instead of printing to stdout via fmt.Println
- Fix double-close bug: body is closed inside SendToHomerun via defer,
  callers no longer need to close it
- Remove redundant resp.Body.Close() from test

# [1.2.0](https://github.com/stuttgart-things/homerun-library/compare/v1.1.0...v1.2.0) (2025-02-22)


### Bug Fixes

* **deps:** update module github.com/jedib0t/go-pretty/v6 to v6.6.6 ([63ed640](https://github.com/stuttgart-things/homerun-library/commit/63ed640a564b93c9d9a5196811d346e174d65c3e))
* **deps:** update module github.com/redisearch/redisearch-go to v2 ([3cbc10f](https://github.com/stuttgart-things/homerun-library/commit/3cbc10f045d678d25a90dcbec1fef7089ba97f1e))
* **deps:** update module github.com/stuttgart-things/sthingscli to v0.3.0 ([3b1e6ec](https://github.com/stuttgart-things/homerun-library/commit/3b1e6ecfab496a343c281b93a838afa902d5c431))
* fix/fix-release ([0bcd0cb](https://github.com/stuttgart-things/homerun-library/commit/0bcd0cb9739e1ab3ff9ab868832420e8ed5d95e1))


### Features

* fix/fixed-readme-desc ([fa75c83](https://github.com/stuttgart-things/homerun-library/commit/fa75c83ddc9a7d8feb29ecb9343ebdb830abd228))

# [1.1.0](https://github.com/stuttgart-things/homerun-library/compare/v1.0.0...v1.1.0) (2025-02-22)


### Bug Fixes

* **deps:** update module github.com/jedib0t/go-pretty/v6 to v6.6.5 ([bae0964](https://github.com/stuttgart-things/homerun-library/commit/bae0964179d49a6bc58e52f181709a82778d193c))
* **deps:** update module github.com/pterm/pterm to v0.12.80 ([3f9c81c](https://github.com/stuttgart-things/homerun-library/commit/3f9c81cdf3f8893068a4be4e8c018e9751a569a8))
* **deps:** update module github.com/redisearch/redisearch-go to v2 ([2d56570](https://github.com/stuttgart-things/homerun-library/commit/2d56570e15d7f4aff6c1b5eb82560d42f3792d24))
* **deps:** update module github.com/stuttgart-things/sthingscli to v0.1.125 ([30df8ee](https://github.com/stuttgart-things/homerun-library/commit/30df8ee1f1254f76eea8f1c3b39b0cf81f5f2914))


### Features

* feat/add-send-feature ([03fc168](https://github.com/stuttgart-things/homerun-library/commit/03fc16850f2a4e4401c28e25c077a58074269402))
* feat/update-readme-send ([d21cefc](https://github.com/stuttgart-things/homerun-library/commit/d21cefc939b13056483a8b352aeb7cbfcc6bba70))

# 1.0.0 (2025-01-09)


### Features

* add new helper function for data parsing ([15f27ab](https://github.com/stuttgart-things/homerun-library/commit/15f27ab918a4bd85a54419bdef573eceddcadc2f))
