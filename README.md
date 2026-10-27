<img src="assets/logo/mark-blade-rounded.svg" width="96" height="96" alt="The kanna mark: a Japanese hand plane seen from the side">

# kanna

kanna is a family of code generators for Go. Each one reads what you have already written, from struct tags to locale
files, and writes the plain Go you would otherwise write by hand: no reflection, nothing hidden from review. A generator
that needs a runtime ships a small one in the same module, so the two never drift apart.

This repository is its site: **https://go-kanna.github.io/kanna/**

| Repository                                                  | What it is                                                                   |
|-------------------------------------------------------------|------------------------------------------------------------------------------|
| [go-kanna/validate](https://github.com/go-kanna/validate)   | kanna-validate: `Validate` methods from struct tags, with Rails' error codes |
| [go-kanna/validated](https://github.com/go-kanna/validated) | Values held to a rule wherever they go                                       |
| [go-kanna/i18n](https://github.com/go-kanna/i18n)           | kanna-i18n: typed message constructors, the translations compiled in         |
| [go-kanna/fixture](https://github.com/go-kanna/fixture)     | kanna-fixture: a constructor for every struct, every field filled            |
| [go-kanna/mapper](https://github.com/go-kanna/mapper)       | kanna-mapper: the functions you declare between domain and wire types        |

kanna-di is on its way.

## What is here

- `index.html` is the page. There is no build step: open it in a browser to preview it, and push to `main` to publish it
  with GitHub Pages.
- [`assets/logo`](assets/logo) holds the mark and the script that draws it.

The Go module that used to live here, kanna v0.2.x with kanna-orm, is kept on the
[`archive/v0.2`](https://github.com/go-kanna/kanna/tree/archive/v0.2) branch and its tags.

## License

[MIT](LICENSE)
