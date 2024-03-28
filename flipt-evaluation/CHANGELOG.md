# Changelog

## [0.1.7](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-evaluation-v0.1.7...flipt-evaluation-v0.1.7) (2024-03-28)


### Features

* add authentication helpers for Ruby and Go ([#119](https://github.com/flipt-io/flipt-client-sdks/issues/119)) ([d41a1a8](https://github.com/flipt-io/flipt-client-sdks/commit/d41a1a82082ea5e5066055b3c731034542e00e16))
* **authentication:** add support for substitutable authentication methods ([c01cb48](https://github.com/flipt-io/flipt-client-sdks/commit/c01cb4859d4e51a7f04eaa46ca36256c81f69a3c))
* Client refs ([#104](https://github.com/flipt-io/flipt-client-sdks/issues/104)) ([de4fc26](https://github.com/flipt-io/flipt-client-sdks/commit/de4fc265872fb8e6b20a39ef026324501763341d))
* **client-sdks:** add batch evaluation to engine ([5dec28b](https://github.com/flipt-io/flipt-client-sdks/commit/5dec28bdd62d97080f53004e022dc196ae0959a3))
* **evaluation:** add entityId for constraint comparisons ([#145](https://github.com/flipt-io/flipt-client-sdks/issues/145)) ([fdf9217](https://github.com/flipt-io/flipt-client-sdks/commit/fdf921761fb0a2ef2a5661f58b0a569ace50a886))
* list flags + single namespace ([#151](https://github.com/flipt-io/flipt-client-sdks/issues/151)) ([cf09c85](https://github.com/flipt-io/flipt-client-sdks/commit/cf09c857a851c70aea97bccc9a159c4c7b5540a0))


### Bug Fixes

* **evaluation:** add isoneof isnotoneof operators to string evaluation ([#149](https://github.com/flipt-io/flipt-client-sdks/issues/149)) ([2b17830](https://github.com/flipt-io/flipt-client-sdks/commit/2b178307b334335c026853d257b8d37c3c5ef023)), closes [#147](https://github.com/flipt-io/flipt-client-sdks/issues/147)
* **flipt-evaluation:** increment bucket after casting ([6b062b2](https://github.com/flipt-io/flipt-client-sdks/commit/6b062b28b3a2417c55f8c088b5f141c1659a81c1))
* **flipt-evaluation:** increment bucket search index by one ([6fe8ef9](https://github.com/flipt-io/flipt-client-sdks/commit/6fe8ef9b716799c1b6c4dbd73dd570b71540d0f0))
* **flipt-evaluation:** stop casting crc32 checksum as i32 ([b3b5c0a](https://github.com/flipt-io/flipt-client-sdks/commit/b3b5c0a4bef8c2a260f16b49559fd14a488ddf9e))
* remove continue for properties that are not in map ([#166](https://github.com/flipt-io/flipt-client-sdks/issues/166)) ([aea3bb3](https://github.com/flipt-io/flipt-client-sdks/commit/aea3bb31a28e8bb1c9aa53cbd7ad55662d5cec1d))


### Miscellaneous Chores

* release 0.0.1 ([96da7f1](https://github.com/flipt-io/flipt-client-sdks/commit/96da7f1b8ab04c7eaba8d5093f0e67af2e967e13))
* release 0.0.3 ([cd97903](https://github.com/flipt-io/flipt-client-sdks/commit/cd979032e1844f162a0317f50e9bed0a5570bfcc))
* release 0.0.4 ([73e760e](https://github.com/flipt-io/flipt-client-sdks/commit/73e760e1df5255f642e15865e3bf38f3b7af2d27))
* release 0.1.0 ([fb9053a](https://github.com/flipt-io/flipt-client-sdks/commit/fb9053aeb21538b8ccb85dd67518e07a45c2f1b6))
* release 0.1.2 ([fea6c5a](https://github.com/flipt-io/flipt-client-sdks/commit/fea6c5a894cb2f138dbf1bb3badc855f2910e7a4))
* release 0.1.5 ([ebe9788](https://github.com/flipt-io/flipt-client-sdks/commit/ebe9788ff57c0230bb6b66f692dc44e7bbdbf14b))
* release 0.1.6 ([395b785](https://github.com/flipt-io/flipt-client-sdks/commit/395b785961b44d2d49a36b242e6bf9b1ff1c2c70))
* release 0.1.7 ([1d00bd6](https://github.com/flipt-io/flipt-client-sdks/commit/1d00bd6114ebe0844d6744898253aca9540a7053))

## [0.1.7](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-evaluation-v0.0.3...flipt-evaluation-v0.1.7) (2024-03-27)


### Features

* add authentication helpers for Ruby and Go ([#119](https://github.com/flipt-io/flipt-client-sdks/issues/119)) ([d41a1a8](https://github.com/flipt-io/flipt-client-sdks/commit/d41a1a82082ea5e5066055b3c731034542e00e16))
* **authentication:** add support for substitutable authentication methods ([c01cb48](https://github.com/flipt-io/flipt-client-sdks/commit/c01cb4859d4e51a7f04eaa46ca36256c81f69a3c))
* Client refs ([#104](https://github.com/flipt-io/flipt-client-sdks/issues/104)) ([de4fc26](https://github.com/flipt-io/flipt-client-sdks/commit/de4fc265872fb8e6b20a39ef026324501763341d))
* **client-sdks:** add batch evaluation to engine ([5dec28b](https://github.com/flipt-io/flipt-client-sdks/commit/5dec28bdd62d97080f53004e022dc196ae0959a3))
* **evaluation:** add entityId for constraint comparisons ([#145](https://github.com/flipt-io/flipt-client-sdks/issues/145)) ([fdf9217](https://github.com/flipt-io/flipt-client-sdks/commit/fdf921761fb0a2ef2a5661f58b0a569ace50a886))
* list flags + single namespace ([#151](https://github.com/flipt-io/flipt-client-sdks/issues/151)) ([cf09c85](https://github.com/flipt-io/flipt-client-sdks/commit/cf09c857a851c70aea97bccc9a159c4c7b5540a0))


### Bug Fixes

* **evaluation:** add isoneof isnotoneof operators to string evaluation ([#149](https://github.com/flipt-io/flipt-client-sdks/issues/149)) ([2b17830](https://github.com/flipt-io/flipt-client-sdks/commit/2b178307b334335c026853d257b8d37c3c5ef023)), closes [#147](https://github.com/flipt-io/flipt-client-sdks/issues/147)
* **flipt-evaluation:** increment bucket after casting ([6b062b2](https://github.com/flipt-io/flipt-client-sdks/commit/6b062b28b3a2417c55f8c088b5f141c1659a81c1))
* **flipt-evaluation:** increment bucket search index by one ([6fe8ef9](https://github.com/flipt-io/flipt-client-sdks/commit/6fe8ef9b716799c1b6c4dbd73dd570b71540d0f0))
* **flipt-evaluation:** stop casting crc32 checksum as i32 ([b3b5c0a](https://github.com/flipt-io/flipt-client-sdks/commit/b3b5c0a4bef8c2a260f16b49559fd14a488ddf9e))
* remove continue for properties that are not in map ([#166](https://github.com/flipt-io/flipt-client-sdks/issues/166)) ([aea3bb3](https://github.com/flipt-io/flipt-client-sdks/commit/aea3bb31a28e8bb1c9aa53cbd7ad55662d5cec1d))


### Miscellaneous Chores

* release 0.0.1 ([96da7f1](https://github.com/flipt-io/flipt-client-sdks/commit/96da7f1b8ab04c7eaba8d5093f0e67af2e967e13))
* release 0.0.3 ([cd97903](https://github.com/flipt-io/flipt-client-sdks/commit/cd979032e1844f162a0317f50e9bed0a5570bfcc))
* release 0.0.4 ([73e760e](https://github.com/flipt-io/flipt-client-sdks/commit/73e760e1df5255f642e15865e3bf38f3b7af2d27))
* release 0.1.0 ([fb9053a](https://github.com/flipt-io/flipt-client-sdks/commit/fb9053aeb21538b8ccb85dd67518e07a45c2f1b6))
* release 0.1.2 ([fea6c5a](https://github.com/flipt-io/flipt-client-sdks/commit/fea6c5a894cb2f138dbf1bb3badc855f2910e7a4))
* release 0.1.5 ([ebe9788](https://github.com/flipt-io/flipt-client-sdks/commit/ebe9788ff57c0230bb6b66f692dc44e7bbdbf14b))
* release 0.1.6 ([395b785](https://github.com/flipt-io/flipt-client-sdks/commit/395b785961b44d2d49a36b242e6bf9b1ff1c2c70))
* release 0.1.7 ([1d00bd6](https://github.com/flipt-io/flipt-client-sdks/commit/1d00bd6114ebe0844d6744898253aca9540a7053))

## [0.1.7](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-evaluation-v0.0.3...flipt-evaluation-v0.1.7) (2024-03-26)


### Features

* add authentication helpers for Ruby and Go ([#119](https://github.com/flipt-io/flipt-client-sdks/issues/119)) ([d41a1a8](https://github.com/flipt-io/flipt-client-sdks/commit/d41a1a82082ea5e5066055b3c731034542e00e16))
* **authentication:** add support for substitutable authentication methods ([c01cb48](https://github.com/flipt-io/flipt-client-sdks/commit/c01cb4859d4e51a7f04eaa46ca36256c81f69a3c))
* Client refs ([#104](https://github.com/flipt-io/flipt-client-sdks/issues/104)) ([de4fc26](https://github.com/flipt-io/flipt-client-sdks/commit/de4fc265872fb8e6b20a39ef026324501763341d))
* **client-sdks:** add batch evaluation to engine ([5dec28b](https://github.com/flipt-io/flipt-client-sdks/commit/5dec28bdd62d97080f53004e022dc196ae0959a3))
* **evaluation:** add entityId for constraint comparisons ([#145](https://github.com/flipt-io/flipt-client-sdks/issues/145)) ([fdf9217](https://github.com/flipt-io/flipt-client-sdks/commit/fdf921761fb0a2ef2a5661f58b0a569ace50a886))
* list flags + single namespace ([#151](https://github.com/flipt-io/flipt-client-sdks/issues/151)) ([cf09c85](https://github.com/flipt-io/flipt-client-sdks/commit/cf09c857a851c70aea97bccc9a159c4c7b5540a0))


### Bug Fixes

* **evaluation:** add isoneof isnotoneof operators to string evaluation ([#149](https://github.com/flipt-io/flipt-client-sdks/issues/149)) ([2b17830](https://github.com/flipt-io/flipt-client-sdks/commit/2b178307b334335c026853d257b8d37c3c5ef023)), closes [#147](https://github.com/flipt-io/flipt-client-sdks/issues/147)
* **flipt-evaluation:** increment bucket after casting ([6b062b2](https://github.com/flipt-io/flipt-client-sdks/commit/6b062b28b3a2417c55f8c088b5f141c1659a81c1))
* **flipt-evaluation:** increment bucket search index by one ([6fe8ef9](https://github.com/flipt-io/flipt-client-sdks/commit/6fe8ef9b716799c1b6c4dbd73dd570b71540d0f0))
* **flipt-evaluation:** stop casting crc32 checksum as i32 ([b3b5c0a](https://github.com/flipt-io/flipt-client-sdks/commit/b3b5c0a4bef8c2a260f16b49559fd14a488ddf9e))
* remove continue for properties that are not in map ([#166](https://github.com/flipt-io/flipt-client-sdks/issues/166)) ([aea3bb3](https://github.com/flipt-io/flipt-client-sdks/commit/aea3bb31a28e8bb1c9aa53cbd7ad55662d5cec1d))


### Miscellaneous Chores

* release 0.0.1 ([96da7f1](https://github.com/flipt-io/flipt-client-sdks/commit/96da7f1b8ab04c7eaba8d5093f0e67af2e967e13))
* release 0.0.3 ([cd97903](https://github.com/flipt-io/flipt-client-sdks/commit/cd979032e1844f162a0317f50e9bed0a5570bfcc))
* release 0.0.4 ([73e760e](https://github.com/flipt-io/flipt-client-sdks/commit/73e760e1df5255f642e15865e3bf38f3b7af2d27))
* release 0.1.0 ([fb9053a](https://github.com/flipt-io/flipt-client-sdks/commit/fb9053aeb21538b8ccb85dd67518e07a45c2f1b6))
* release 0.1.2 ([fea6c5a](https://github.com/flipt-io/flipt-client-sdks/commit/fea6c5a894cb2f138dbf1bb3badc855f2910e7a4))
* release 0.1.5 ([ebe9788](https://github.com/flipt-io/flipt-client-sdks/commit/ebe9788ff57c0230bb6b66f692dc44e7bbdbf14b))
* release 0.1.6 ([395b785](https://github.com/flipt-io/flipt-client-sdks/commit/395b785961b44d2d49a36b242e6bf9b1ff1c2c70))
* release 0.1.7 ([1d00bd6](https://github.com/flipt-io/flipt-client-sdks/commit/1d00bd6114ebe0844d6744898253aca9540a7053))

## [0.1.7](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-evaluation-v0.0.3...flipt-evaluation-v0.1.7) (2024-03-26)


### Features

* add authentication helpers for Ruby and Go ([#119](https://github.com/flipt-io/flipt-client-sdks/issues/119)) ([d41a1a8](https://github.com/flipt-io/flipt-client-sdks/commit/d41a1a82082ea5e5066055b3c731034542e00e16))
* **authentication:** add support for substitutable authentication methods ([c01cb48](https://github.com/flipt-io/flipt-client-sdks/commit/c01cb4859d4e51a7f04eaa46ca36256c81f69a3c))
* Client refs ([#104](https://github.com/flipt-io/flipt-client-sdks/issues/104)) ([de4fc26](https://github.com/flipt-io/flipt-client-sdks/commit/de4fc265872fb8e6b20a39ef026324501763341d))
* **client-sdks:** add batch evaluation to engine ([5dec28b](https://github.com/flipt-io/flipt-client-sdks/commit/5dec28bdd62d97080f53004e022dc196ae0959a3))
* **evaluation:** add entityId for constraint comparisons ([#145](https://github.com/flipt-io/flipt-client-sdks/issues/145)) ([fdf9217](https://github.com/flipt-io/flipt-client-sdks/commit/fdf921761fb0a2ef2a5661f58b0a569ace50a886))
* list flags + single namespace ([#151](https://github.com/flipt-io/flipt-client-sdks/issues/151)) ([cf09c85](https://github.com/flipt-io/flipt-client-sdks/commit/cf09c857a851c70aea97bccc9a159c4c7b5540a0))


### Bug Fixes

* **evaluation:** add isoneof isnotoneof operators to string evaluation ([#149](https://github.com/flipt-io/flipt-client-sdks/issues/149)) ([2b17830](https://github.com/flipt-io/flipt-client-sdks/commit/2b178307b334335c026853d257b8d37c3c5ef023)), closes [#147](https://github.com/flipt-io/flipt-client-sdks/issues/147)
* **flipt-evaluation:** increment bucket after casting ([6b062b2](https://github.com/flipt-io/flipt-client-sdks/commit/6b062b28b3a2417c55f8c088b5f141c1659a81c1))
* **flipt-evaluation:** increment bucket search index by one ([6fe8ef9](https://github.com/flipt-io/flipt-client-sdks/commit/6fe8ef9b716799c1b6c4dbd73dd570b71540d0f0))
* **flipt-evaluation:** stop casting crc32 checksum as i32 ([b3b5c0a](https://github.com/flipt-io/flipt-client-sdks/commit/b3b5c0a4bef8c2a260f16b49559fd14a488ddf9e))
* remove continue for properties that are not in map ([#166](https://github.com/flipt-io/flipt-client-sdks/issues/166)) ([aea3bb3](https://github.com/flipt-io/flipt-client-sdks/commit/aea3bb31a28e8bb1c9aa53cbd7ad55662d5cec1d))


### Miscellaneous Chores

* release 0.0.1 ([96da7f1](https://github.com/flipt-io/flipt-client-sdks/commit/96da7f1b8ab04c7eaba8d5093f0e67af2e967e13))
* release 0.0.3 ([cd97903](https://github.com/flipt-io/flipt-client-sdks/commit/cd979032e1844f162a0317f50e9bed0a5570bfcc))
* release 0.0.4 ([73e760e](https://github.com/flipt-io/flipt-client-sdks/commit/73e760e1df5255f642e15865e3bf38f3b7af2d27))
* release 0.1.0 ([fb9053a](https://github.com/flipt-io/flipt-client-sdks/commit/fb9053aeb21538b8ccb85dd67518e07a45c2f1b6))
* release 0.1.2 ([fea6c5a](https://github.com/flipt-io/flipt-client-sdks/commit/fea6c5a894cb2f138dbf1bb3badc855f2910e7a4))
* release 0.1.5 ([ebe9788](https://github.com/flipt-io/flipt-client-sdks/commit/ebe9788ff57c0230bb6b66f692dc44e7bbdbf14b))
* release 0.1.6 ([395b785](https://github.com/flipt-io/flipt-client-sdks/commit/395b785961b44d2d49a36b242e6bf9b1ff1c2c70))
* release 0.1.7 ([1d00bd6](https://github.com/flipt-io/flipt-client-sdks/commit/1d00bd6114ebe0844d6744898253aca9540a7053))

## [0.1.6](https://github.com/flipt-io/flipt-client-sdks/compare/flipt-evaluation-v0.0.3...flipt-evaluation-v0.1.6) (2024-03-26)


### Features

* add authentication helpers for Ruby and Go ([#119](https://github.com/flipt-io/flipt-client-sdks/issues/119)) ([d41a1a8](https://github.com/flipt-io/flipt-client-sdks/commit/d41a1a82082ea5e5066055b3c731034542e00e16))
* **authentication:** add support for substitutable authentication methods ([c01cb48](https://github.com/flipt-io/flipt-client-sdks/commit/c01cb4859d4e51a7f04eaa46ca36256c81f69a3c))
* Client refs ([#104](https://github.com/flipt-io/flipt-client-sdks/issues/104)) ([de4fc26](https://github.com/flipt-io/flipt-client-sdks/commit/de4fc265872fb8e6b20a39ef026324501763341d))
* **client-sdks:** add batch evaluation to engine ([5dec28b](https://github.com/flipt-io/flipt-client-sdks/commit/5dec28bdd62d97080f53004e022dc196ae0959a3))
* **evaluation:** add entityId for constraint comparisons ([#145](https://github.com/flipt-io/flipt-client-sdks/issues/145)) ([fdf9217](https://github.com/flipt-io/flipt-client-sdks/commit/fdf921761fb0a2ef2a5661f58b0a569ace50a886))
* list flags + single namespace ([#151](https://github.com/flipt-io/flipt-client-sdks/issues/151)) ([cf09c85](https://github.com/flipt-io/flipt-client-sdks/commit/cf09c857a851c70aea97bccc9a159c4c7b5540a0))


### Bug Fixes

* **evaluation:** add isoneof isnotoneof operators to string evaluation ([#149](https://github.com/flipt-io/flipt-client-sdks/issues/149)) ([2b17830](https://github.com/flipt-io/flipt-client-sdks/commit/2b178307b334335c026853d257b8d37c3c5ef023)), closes [#147](https://github.com/flipt-io/flipt-client-sdks/issues/147)
* **flipt-evaluation:** increment bucket after casting ([6b062b2](https://github.com/flipt-io/flipt-client-sdks/commit/6b062b28b3a2417c55f8c088b5f141c1659a81c1))
* **flipt-evaluation:** increment bucket search index by one ([6fe8ef9](https://github.com/flipt-io/flipt-client-sdks/commit/6fe8ef9b716799c1b6c4dbd73dd570b71540d0f0))
* **flipt-evaluation:** stop casting crc32 checksum as i32 ([b3b5c0a](https://github.com/flipt-io/flipt-client-sdks/commit/b3b5c0a4bef8c2a260f16b49559fd14a488ddf9e))
* remove continue for properties that are not in map ([#166](https://github.com/flipt-io/flipt-client-sdks/issues/166)) ([aea3bb3](https://github.com/flipt-io/flipt-client-sdks/commit/aea3bb31a28e8bb1c9aa53cbd7ad55662d5cec1d))


### Miscellaneous Chores

* release 0.0.1 ([96da7f1](https://github.com/flipt-io/flipt-client-sdks/commit/96da7f1b8ab04c7eaba8d5093f0e67af2e967e13))
* release 0.0.3 ([cd97903](https://github.com/flipt-io/flipt-client-sdks/commit/cd979032e1844f162a0317f50e9bed0a5570bfcc))
* release 0.0.4 ([73e760e](https://github.com/flipt-io/flipt-client-sdks/commit/73e760e1df5255f642e15865e3bf38f3b7af2d27))
* release 0.1.0 ([fb9053a](https://github.com/flipt-io/flipt-client-sdks/commit/fb9053aeb21538b8ccb85dd67518e07a45c2f1b6))
* release 0.1.2 ([fea6c5a](https://github.com/flipt-io/flipt-client-sdks/commit/fea6c5a894cb2f138dbf1bb3badc855f2910e7a4))
* release 0.1.5 ([ebe9788](https://github.com/flipt-io/flipt-client-sdks/commit/ebe9788ff57c0230bb6b66f692dc44e7bbdbf14b))
* release 0.1.6 ([395b785](https://github.com/flipt-io/flipt-client-sdks/commit/395b785961b44d2d49a36b242e6bf9b1ff1c2c70))
