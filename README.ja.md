<img src="assets/logo/mark-kon-rounded.svg" width="96" height="96" alt="kanna のロゴ: 平らなところから巻いていく木の削り屑">

# kanna

[![English](https://img.shields.io/badge/lang-English-blue.svg)](README.md)

kanna は、Go のための 7 つのコードジェネレータのファミリーです。バリデーション、値オブジェクト、データベース、マッピング、メッセージ、依存関係の組み立て、テストデータを扱います。いずれも、構造体タグからスキーマまで、すでに書いたものを読み取り、本来手で書くはずの素の Go を生成します。リフレクションはなく、レビューから隠れるものもありません。ランタイムが必要なジェネレータは小さなものを同じモジュールで提供するので、両者がずれることはありません。

このリポジトリはそのサイトです: **https://go-kanna.github.io/ja/**

| リポジトリ | 内容 |
|---|---|
| [go-kanna/validate](https://github.com/go-kanna/validate) | kanna-validate: 構造体タグから `Validate` メソッドを生成する。エラーコードは Rails と同じ |
| [go-kanna/value](https://github.com/go-kanna/value) | kanna-value: ルールを守り続ける値オブジェクトと、定数の値だけを取る enum |
| [go-kanna/orm](https://github.com/go-kanna/orm) | kanna-orm: リポジトリのインターフェースの実装と、テーブルごとの連結できるスコープ |
| [go-kanna/mapper](https://github.com/go-kanna/mapper) | kanna-mapper: ドメイン型とワイヤ型の間に宣言した関数の実装 |
| [go-kanna/i18n](https://github.com/go-kanna/i18n) | kanna-i18n: 翻訳を組み込んだ型付きのメッセージコンストラクタ |
| [go-kanna/di](https://github.com/go-kanna/di) | kanna-di: 型の隣にある関数から生成するコンテナ構造体のコンストラクタ |
| [go-kanna/fixture](https://github.com/go-kanna/fixture) | kanna-fixture: 構造体ごとの、全フィールドが埋まったコンストラクタ |

## ここにあるもの

- `index.html` はランディングページで、`ja/index.html` はその日本語版です。どちらもブラウザで開けばプレビューできます。
- `validate/` などの各ツールのページは README を、`ja/validate/` などは `README.ja.md` を表示します。`pages/render.py` は、これらを各リポジトリのデフォルトブランチから GitHub API で取得して `pages/page.html` に描画し、生成コードのブロックをサイトの他の部分と同じように木の色にし、ランディングページのバージョンを最新のタグと `go.mod` ファイルから埋めます。これらはコミットされません。Pages ワークフローが `main` への push ごとと 1 日 1 回スクリプトを実行するので、サイトはリポジトリに追従します。サイト全体をプレビューするには、gh CLI にログインした状態で次を実行します:

  ```sh
  python3 pages/render.py && python3 -m http.server -d _site
  ```

- `assets/site.css` と `assets/site.js` はすべてのページで共有します。サイトの書体は [M PLUS 1 と M PLUS 1 Code](https://github.com/coz-m/MPLUS_FONTS) で、`assets/fonts` にライセンスとともに置いています。それぞれラテン文字と日本語に分けてあるので、ページは日本語があるときだけ日本語のフォントを読み込みます。日本語のページは、`assets/fonts/subset.py` が作る、より字数の多いフォントを使います。
- [`assets/logo`](assets/logo) にはロゴと、それを描くスクリプトがあります。
- [`DEVELOPING.md`](DEVELOPING.md) には、複数のリポジトリにまたがる変更のしかたを書いています。[`dev`](dev) にある全チェックアウトのワークスペース、それに対して全リポジトリのテストを実行するスクリプト、リリースの順序です。

以前ここにあった Go モジュール（kanna v0.2.x と初代の kanna-orm）は、[`archive/v0.2`](https://github.com/go-kanna/go-kanna.github.io/tree/archive/v0.2) ブランチとそのタグに残しています。kanna-orm はその後 [go-kanna/orm](https://github.com/go-kanna/orm) で新しく書き直されました。

## 生成コードのチェック

ジェネレータは、何かを書き出す前に、与えられたものと、生成物を呼び出す手書きのコードをチェックします。書き出したものがコンパイルでき、言ったとおりに動くことは、さらに 2 箇所でチェックしています:

- 各ジェネレータのテストは、扱うすべての種類のパッケージについて生成コードをコンパイルし、呼び出します。
- [go-kanna/showcase](https://github.com/go-kanna/showcase) は 7 つすべてを一緒に動かします。CI は、生成コードと protobuf のコードがコミット済みの内容と一致することをチェックし、ジェネレータが生成したものを削除して 1 回の実行ですべて書き直させて vet し、サービスを PostgreSQL に対してテストします。

すべてのジェネレータの土台である [go-kanna/internal](https://github.com/go-kanna/internal) への変更は、タグを打つ前に各ジェネレータのテストを実行します。

## ライセンス

[MIT](LICENSE)
