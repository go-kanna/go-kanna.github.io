<img src="assets/logo/mark-kon-rounded.svg" width="96" height="96" alt="kanna のロゴ: 平らなところから巻いていく木の削り屑">

# kanna

[![English](https://img.shields.io/badge/lang-English-blue.svg)](README.md)

kanna は、Go のための 8 つのコードジェネレータのファミリーです。バリデーション、値オブジェクト、データベース、マッピング、メッセージ、設定、依存関係の組み立て、テストデータを扱います。必要なものを規約どおりの名前で宣言すれば、いずれも本来手で書くはずの素の Go を生成します。リフレクションはなく、レビューから隠れるものもありません。規約に合わないところは、その部分だけ Go や SQL で書きます。ランタイムが必要なジェネレータは小さなものを同じモジュールで提供するので、両者がずれることはありません。ただし kanna-config が生成するコードは kanna-validator のランタイムも呼び、ランタイムを持たない kanna-fixture は、テンプレートを指定したフィールドで gofakeit を呼びます。これらは、利用するモジュールが選ぶバージョンで動きます。

このリポジトリはそのサイトです: **https://go-kanna.github.io/ja/**

| リポジトリ | 内容 |
|---|---|
| [go-kanna/validator](https://github.com/go-kanna/validator) | kanna-validator: 構造体タグから生成する `Validate` メソッドと、フィールドの JSON 名をキーにしたコードのエラー |
| [go-kanna/value](https://github.com/go-kanna/value) | kanna-value: ルールを守り続ける値オブジェクトと、定数の値だけを取る enum |
| [go-kanna/orm](https://github.com/go-kanna/orm) | kanna-orm: repository のインターフェースの実装と、テーブルごとの連結できるスコープ |
| [go-kanna/mapper](https://github.com/go-kanna/mapper) | kanna-mapper: ドメイン型とワイヤ型の間に宣言した関数の実装 |
| [go-kanna/i18n](https://github.com/go-kanna/i18n) | kanna-i18n: 翻訳を組み込んだ型付きのメッセージコンストラクタ |
| [go-kanna/config](https://github.com/go-kanna/config) | kanna-config: 設定の構造体を、フラグ、環境変数、ファイルから読む関数 |
| [go-kanna/di](https://github.com/go-kanna/di) | kanna-di: 型の隣にある関数から生成するコンテナ構造体のコンストラクタ |
| [go-kanna/fixture](https://github.com/go-kanna/fixture) | kanna-fixture: 構造体ごとの、全フィールドが埋まったコンストラクタ |
| [go-kanna/kanna](https://github.com/go-kanna/kanna) | モジュールのジェネレータを必要な順に 1 回ずつ実行する `kanna gen` と、その vet を実行する `kanna vet` |

## 使い始める

9 行で、8 つのジェネレータと kanna gen をモジュールに入れます。各ジェネレータが必要とする Go の版は、[要件と互換性](#要件と互換性) にあります。

```sh
go get -tool github.com/go-kanna/validator/cmd/kanna-validator
go get -tool github.com/go-kanna/value/cmd/kanna-value
go get -tool github.com/go-kanna/orm/cmd/kanna-orm
go get -tool github.com/go-kanna/mapper/cmd/kanna-mapper
go get -tool github.com/go-kanna/i18n/cmd/kanna-i18n
go get -tool github.com/go-kanna/config/cmd/kanna-config
go get -tool github.com/go-kanna/di/cmd/kanna-di
go get -tool github.com/go-kanna/fixture/cmd/kanna-fixture
go get -tool github.com/go-kanna/kanna
```

kanna-orm はスキーマをスナップショットから読むので、マイグレーションしたデータベースから `kanna-orm describe` で 1 回取っておきます。マイグレーションのたびに取り直します。宣言をどのパッケージに置き、各ジェネレータがその隣に何を書き、kanna gen がどの順で動かすかは、サイトの [8 つ全部を 1 つのモジュールで](https://go-kanna.github.io/ja/#start) に図があります。

## 要件と互換性

- **Go:** kanna-validator、kanna-value、kanna-fixture、kanna-config、kanna-mapper は Go 1.27 以降、kanna-orm、kanna-di、kanna-i18n、kanna コマンドは Go 1.26 以降が必要です。8 つ全部を使うなら 1.27 です。`go get -tool` は、go.mod の go 行を、そのツールが求める版に上げます。
- **OS:** Linux と macOS で確かめています。Windows は確かめていません。
- **版:** v0.x の間は、マイナー版（v0.2.0）で、通っていた宣言が止まったり、意味が変わったりすることがあります。その変更は Release ノートに書きます。パッチ版（v0.1.1）では壊しません。
- **上げるとき:** 使っているジェネレータは、まとめてそれぞれ最新にします。内部モジュールを共有していて、モジュールにはその 1 つの版しか入らないためです。版の番号を揃える必要はありません。
- **上げたあと:** `go tool kanna gen` で生成し直します。古い版が書いたファイルは、`go tool kanna gen -check` が知らせます。

## 考え方

**規約に従えば、宣言だけで動く。** 実装は読める素の Go として生成し、誤りはできるだけ早く知らせます。

1. **規約で簡潔に、必要なところは手で書く。** 定型のコードは規約に任せ、規約に合わないところは、設定を足さずに Go や SQL で書きます。
2. **生成物は、素の Go。** kanna を知らなくても読めて、手で書いたコードと同じように扱えます。
3. **知っている言葉で、予想どおりに。** Go と SQL を知っていれば、宣言の意味も、生成されるコードの動きも予想できます。
4. **誤りは早く、明確に。** 静的に検出できる誤りは実行前に、実行時にしか判断できない誤りは実行時に知らせます。

## ここにあるもの

- `index.html` はランディングページで、`ja/index.html` はその日本語版です。どちらもブラウザで開けばプレビューできます。
- `validator/` などの各ツールのページは README を、`ja/validator/` などは `README.ja.md` を表示します。`pages/render.py` は、これらを各リポジトリのデフォルトブランチから GitHub API で取得して `pages/page.html` のテンプレートに入れ、生成コードのブロックをサイトの他の部分と同じように木の色にし、ランディングページのバージョンを最新のタグと、そのタグの `go.mod` ファイルから埋めます。スクリプトが作ったものはコミットされません。Pages ワークフローが `main` への push ごとと 1 日 1 回スクリプトを実行するので、サイトはリポジトリに追従します。サイト全体をプレビューするには、gh CLI にログインした状態で次を実行します:

  ```sh
  python3 pages/render.py && python3 -m http.server -d _site
  ```

- `assets/site.css` と `assets/site.js` はすべてのページで共有します。サイトの書体は [M PLUS 1 と M PLUS 1 Code](https://github.com/coz-m/MPLUS_FONTS) で、`assets/fonts` にライセンスとともに置いています。それぞれラテン文字と日本語に分けてあるので、ページは日本語があるときだけ日本語のフォントを読み込みます。日本語のページは、`assets/fonts/subset.py` が作る、より字数の多いフォントを使います。
- [`assets/logo`](assets/logo) にはロゴと、それを描くスクリプトがあります。
- go-kanna/kanna の [`DEVELOPING.md`](https://github.com/go-kanna/kanna/blob/main/DEVELOPING.md) には、複数のリポジトリにまたがる変更のしかたを書いています。全チェックアウトのワークスペース、それに対して全リポジトリのテストを実行するスクリプト、リリースの順序です。

以前ここにあった Go モジュール（kanna v0.2.x と初代の kanna-orm）は、[`archive/v0.2`](https://github.com/go-kanna/go-kanna.github.io/tree/archive/v0.2) ブランチとそのタグに残しています。kanna-orm はその後 [go-kanna/orm](https://github.com/go-kanna/orm) で新しく書き直されました。

## 生成コードのチェック

ジェネレータは、何かを書き出す前に、与えられたものと、生成物を呼び出す手書きのコードをチェックします。書き出したものがコンパイルでき、ジェネレータが説明するとおりに動くことは、さらに 2 箇所でチェックしています:

- 各ジェネレータのテストは、扱うすべての種類のパッケージについて生成コードをコンパイルし、呼び出します。
- [go-kanna/showcase](https://github.com/go-kanna/showcase) は 8 つすべてを一緒に動かします。CI は、生成コードと protobuf のコードがコミット済みの内容と一致することをチェックし、生成したものを呼び出すコードを kanna vet で検査し、サービスを PostgreSQL に対してテストします。

すべてのジェネレータの土台である [go-kanna/internal](https://github.com/go-kanna/internal) を変更すると、タグを打つ前に、その変更で各ジェネレータのテストが実行されます。

## ライセンス

[MIT](LICENSE)
