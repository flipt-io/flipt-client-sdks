# Changelog

## [0.1.6](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-engine-ffi-v0.1.5...flipt-engine-ffi-v0.1.6) (2024-03-26)

### Bug Fixes

* use committed c-header file and copy to correct place expected by Go SDK ([#190](https://github.com/flipt-io/flipt-client-sdks/issues/190)) ([183dffd](https://github.com/flipt-io/flipt-client-sdks/commit/183dffdf8481410e2aa50c10ae040219f0098694))

### Miscellaneous Chores

* release 0.1.6 ([395b785](https://github.com/flipt-io/flipt-client-sdks/commit/395b785961b44d2d49a36b242e6bf9b1ff1c2c70))

## [0.1.6](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-engine-ffi-v0.1.5...flipt-engine-ffi-v0.1.6) (2024-03-23)

### Miscellaneous Chores

* release 0.1.6 ([395b785](https://github.com/flipt-io/flipt-client-sdks/commit/395b785961b44d2d49a36b242e6bf9b1ff1c2c70))

## [0.1.5](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-engine-ffi-v0.1.4...flipt-engine-ffi-v0.1.5) (2024-03-14)

### Miscellaneous Chores

* release 0.1.5 ([ebe9788](https://github.com/flipt-io/flipt-client-sdks/commit/ebe9788ff57c0230bb6b66f692dc44e7bbdbf14b))

## [0.1.4](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-engine-ffi-v0.1.3...flipt-engine-ffi-v0.1.4) (2024-03-05)

### Bug Fixes

* **engine-ffi:** prevent memory leaks ([#157](https://github.com/flipt-io/flipt-client-sdks/issues/157)) ([60b4ec9](https://github.com/flipt-io/flipt-client-sdks/commit/60b4ec9c51204c29d1eae403ea1612c8f6a5faab))

## [0.1.3](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-engine-ffi-v0.1.2...flipt-engine-ffi-v0.1.3) (2024-03-01)

### Features

* list flags + single namespace ([#151](https://github.com/flipt-io/flipt-client-sdks/issues/151)) ([cf09c85](https://github.com/flipt-io/flipt-client-sdks/commit/cf09c857a851c70aea97bccc9a159c4c7b5540a0))

## [0.1.2](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-engine-ffi-v0.1.1...flipt-engine-ffi-v0.1.2) (2024-02-24)

### Miscellaneous Chores

* release 0.1.2 ([fea6c5a](https://github.com/flipt-io/flipt-client-sdks/commit/fea6c5a894cb2f138dbf1bb3badc855f2910e7a4))

## [0.1.1](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-engine-ffi-v0.1.0...flipt-engine-ffi-v0.1.1) (2024-02-06)

### Features

* **client-sdks:** add batch evaluation to engine ([5dec28b](https://github.com/flipt-io/flipt-client-sdks/commit/5dec28bdd62d97080f53004e022dc196ae0959a3))

## [0.1.0](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-engine-ffi-v0.0.5...flipt-engine-ffi-v0.1.0) (2024-01-25)

### Features

* **authentication:** add support for substitutable authentication methods ([c01cb48](https://github.com/flipt-io/flipt-client-sdks/commit/c01cb4859d4e51a7f04eaa46ca36256c81f69a3c))

### Miscellaneous Chores

* release 0.1.0 ([fb9053a](https://github.com/flipt-io/flipt-client-sdks/commit/fb9053aeb21538b8ccb85dd67518e07a45c2f1b6))

## [0.0.5](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-engine-ffi-v0.0.4...flipt-engine-ffi-v0.0.5) (2024-01-18)

### Features

* Client refs ([#104](https://github.com/flipt-io/flipt-client-sdks/issues/104)) ([de4fc26](https://github.com/flipt-io/flipt-client-sdks/commit/de4fc265872fb8e6b20a39ef026324501763341d))

## [0.0.4](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-engine-v0.0.3...flipt-engine-v0.0.4) (2024-01-16)

### Bug Fixes

* **cargo:** move workspace reference up to root toml ([9bdb2b5](https://github.com/flipt-io/flipt-client-sdks/commit/9bdb2b5ae1745cb2d3a171ec5ff9313ef10b254b))
* dont wipe out snapshot if error in parser ([#96](https://github.com/flipt-io/flipt-client-sdks/issues/96)) ([75850ab](https://github.com/flipt-io/flipt-client-sdks/commit/75850ab0bcaf61185a7523efa7201ce67d673174))
* **examples:** rewire the use statements ([f3457a0](https://github.com/flipt-io/flipt-client-sdks/commit/f3457a0033473908864f1aa1485c5c4e3fb7edce))

### Miscellaneous Chores

* release 0.0.3 ([cd97903](https://github.com/flipt-io/flipt-client-sdks/commit/cd979032e1844f162a0317f50e9bed0a5570bfcc))
* release 0.0.4 ([73e760e](https://github.com/flipt-io/flipt-client-sdks/commit/73e760e1df5255f642e15865e3bf38f3b7af2d27))

## [0.0.3](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-engine-v0.0.2...flipt-engine-v0.0.3) (2024-01-03)

### Bug Fixes

* instead of using empty dictionary use actual EngineOpts instance ([#62](https://github.com/flipt-io/flipt-client-sdks/issues/62)) ([9aae5e9](https://github.com/flipt-io/flipt-client-sdks/commit/9aae5e91216a0c101551340567a251e9aa6a240b))
* return error on http status &gt;= 400 ([#65](https://github.com/flipt-io/flipt-client-sdks/issues/65)) ([100c7d2](https://github.com/flipt-io/flipt-client-sdks/commit/100c7d2f7417b3bee6fa5f5fb55fd4cd27a682c2))

## [0.0.2](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-engine-v0.0.1...flipt-engine-v0.0.2) (2023-12-27)

### Miscellaneous Chores

* release 0.0.2 ([0cecaba](https://github.com/flipt-io/flipt-client-sdks/commit/0cecaba72e7ee5465048df7a6dc23a4e0419781c))

## 0.0.1 (2023-12-24)

### Features

* add ability to pass opts to the engine ([#7](https://github.com/flipt-io/flipt-client-sdks/issues/7)) ([1abd50d](https://github.com/flipt-io/flipt-client-sdks/commit/1abd50daa2af036b2ba396a1ad85496c5f1d574e))
* add build script for repeatability in generating C header file ([c08d2fd](https://github.com/flipt-io/flipt-client-sdks/commit/c08d2fd4e357cb9cfcb0187309845967e9fb0fbb))
* add filter for empty namespaces for Rust core ([afef630](https://github.com/flipt-io/flipt-client-sdks/commit/afef63017c9e729ec3af97242c52d243b899cbb0))
* Add ruby client ([#3](https://github.com/flipt-io/flipt-client-sdks/issues/3)) ([1e1d5ac](https://github.com/flipt-io/flipt-client-sdks/commit/1e1d5ac2a3299ef78400e84da7fabc97d1fe6a4e))
* Dependency Injection ([#6](https://github.com/flipt-io/flipt-client-sdks/issues/6)) ([90a374d](https://github.com/flipt-io/flipt-client-sdks/commit/90a374d93aa8a8c4110b001f038b99f0115a5497))
* do not fail on non-existent namespace in Rust core ([#26](https://github.com/flipt-io/flipt-client-sdks/issues/26)) ([51ef54b](https://github.com/flipt-io/flipt-client-sdks/commit/51ef54b74c82e1cd4f1a0ac6157c9cd468eab653))
* Error types ([#30](https://github.com/flipt-io/flipt-client-sdks/issues/30)) ([1cbd333](https://github.com/flipt-io/flipt-client-sdks/commit/1cbd333d710cfbcb518897777972428c55c68259))
* remove box dynamic dispatch and use generic trait bounds ([0f337b4](https://github.com/flipt-io/flipt-client-sdks/commit/0f337b43383d8859ee6b2d8ab320c72f7f63af6c))
* rename methods ([#21](https://github.com/flipt-io/flipt-client-sdks/issues/21)) ([d6e669e](https://github.com/flipt-io/flipt-client-sdks/commit/d6e669e4bbde5a92ea71a7fa5609f5af661277da))
* update tests for node, go, and python to account for auth token ([136891b](https://github.com/flipt-io/flipt-client-sdks/commit/136891ba634259bd6fa28afac8cd3c1fd3b21368))
* **wip:** support passing auth token down to engine ([3181a32](https://github.com/flipt-io/flipt-client-sdks/commit/3181a32981f967310c993d1cedd63a8b81ce5969))

### Miscellaneous Chores

* release 0.0.1 ([96da7f1](https://github.com/flipt-io/flipt-client-sdks/commit/96da7f1b8ab04c7eaba8d5093f0e67af2e967e13))
