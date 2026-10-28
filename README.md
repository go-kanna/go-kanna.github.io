<img src="assets/logo/mark-blade-rounded.svg" width="96" height="96" alt="The kanna mark: a Japanese hand plane seen from the side">

# kanna

kanna is a family of code generators for Go. Each one reads what you have already written, from struct tags to locale
files, and writes the plain Go you would otherwise write by hand: no reflection, nothing hidden from review. A generator
that needs a runtime ships a small one in the same module, so the two never drift apart.

This repository is its site: **https://go-kanna.github.io/kanna/**

| Repository                                                  | What it is                                                                              |
|-------------------------------------------------------------|-----------------------------------------------------------------------------------------|
| [go-kanna/validate](https://github.com/go-kanna/validate)   | kanna-validate: `Validate` methods from struct tags, with Rails' error codes            |
| [go-kanna/i18n](https://github.com/go-kanna/i18n)           | kanna-i18n: typed message constructors, the translations compiled in                    |
| [go-kanna/fixture](https://github.com/go-kanna/fixture)     | kanna-fixture: a constructor for every struct, every field filled                       |
| [go-kanna/di](https://github.com/go-kanna/di)               | kanna-di: the constructor of a container struct, from the functions next to your types |
| [go-kanna/mapper](https://github.com/go-kanna/mapper)       | kanna-mapper: the functions you declare between domain and wire types                   |
| [go-kanna/validated](https://github.com/go-kanna/validated) | Values held to a rule wherever they go                                                  |

## What is here

- `index.html` is the landing page. Open it in a browser to preview it.
- Each tool's page, `validate/` and so on, is its README: `pages/render.py` fetches the README of each repository's
  default branch through the GitHub API, renders it into `pages/page.html`, and fills in the versions on the landing
  page from the latest tags and the `go.mod` files. None of that is committed. The Pages workflow runs the script on
  every push to `main` and once a day, so the site follows the repositories. To preview the whole site, with the gh CLI
  logged in:

  ```sh
  python3 pages/render.py --base / && python3 -m http.server -d _site
  ```

- `assets/site.css` and `assets/site.js` are shared by every page. The site is set in
  [Go's own typeface](https://go.dev/blog/go-fonts), which `assets/fonts` holds with its license.
- [`assets/logo`](assets/logo) holds the mark and the script that draws it.

The Go module that used to live here, kanna v0.2.x with kanna-orm, is kept on the
[`archive/v0.2`](https://github.com/go-kanna/kanna/tree/archive/v0.2) branch and its tags.

## License

[MIT](LICENSE)
