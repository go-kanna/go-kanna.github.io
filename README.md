<img src="assets/logo/mark-blade-rounded.svg" width="96" height="96" alt="The kanna mark: a Japanese hand plane seen from the side">

# kanna

kanna is a family of code generators for Go. Each one reads the structs you already have and writes the plain Go you
would otherwise write by hand: no reflection, no runtime to learn, nothing hidden from review.

This repository is its site: **https://go-kanna.github.io/kanna/**

| Repository                                                  | What it is                                                                   |
|-------------------------------------------------------------|------------------------------------------------------------------------------|
| [go-kanna/validate](https://github.com/go-kanna/validate)   | kanna-validate: `Validate` methods from struct tags, with Rails' error codes |
| [go-kanna/validated](https://github.com/go-kanna/validated) | Values held to a rule wherever they go                                       |
| [go-kanna/i18n](https://github.com/go-kanna/i18n)           | kanna-i18n: typed message constructors, the translations compiled in         |

kanna-di, kanna-fixture, and kanna-mapper are on their way.

## What is here

- `index.html` is the page. There is no build step: open it in a browser to preview it, and push to `main` to publish it
  with GitHub Pages.
- [`assets/logo`](assets/logo) holds the mark and the script that draws it.

The Go module that used to live here, kanna v0.2.x with kanna-orm, is kept on the
[`archive/v0.2`](https://github.com/go-kanna/kanna/tree/archive/v0.2) branch and its tags.

## License

[MIT](LICENSE)
