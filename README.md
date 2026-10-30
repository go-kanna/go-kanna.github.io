<img src="assets/logo/mark-kon-rounded.svg" width="96" height="96" alt="The kanna mark: a wood shaving, flat and then curled">

# kanna

kanna is a family of seven code generators for Go: validation, value types, the database, mapping, messages, wiring,
and test data. Each one reads what you have already written, from struct tags to your schema, and writes the plain Go
you would otherwise write by hand: no reflection, nothing hidden from review. A generator that needs a runtime ships a
small one in the same module, so the two never drift apart.

This repository is its site: **https://go-kanna.github.io/kanna/**

| Repository                                                | What it is                                                                                   |
|-----------------------------------------------------------|----------------------------------------------------------------------------------------------|
| [go-kanna/validate](https://github.com/go-kanna/validate) | kanna-validate: `Validate` methods from struct tags, with Rails' error codes                 |
| [go-kanna/value](https://github.com/go-kanna/value)       | kanna-value: value types held to their rule wherever their values come from                  |
| [go-kanna/orm](https://github.com/go-kanna/orm)           | kanna-orm: the implementation of your repository interfaces, and a chainable scope per table |
| [go-kanna/mapper](https://github.com/go-kanna/mapper)     | kanna-mapper: the functions you declare between domain and wire types                        |
| [go-kanna/i18n](https://github.com/go-kanna/i18n)         | kanna-i18n: typed message constructors, the translations compiled in                         |
| [go-kanna/di](https://github.com/go-kanna/di)             | kanna-di: the constructor of a container struct, from the functions next to your types       |
| [go-kanna/fixture](https://github.com/go-kanna/fixture)   | kanna-fixture: a constructor for every struct, every field filled                            |

## What is here

- `index.html` is the landing page. Open it in a browser to preview it.
- Each tool's page, `validate/` and so on, is its README: `pages/render.py` fetches the README of each repository's
  default branch through the GitHub API, renders it into `pages/page.html`, sets its blocks of generated code on wood,
  as the rest of the site does, and fills in the versions on the landing page from the latest tags and the `go.mod`
  files. None of that is committed. The Pages workflow runs the script on every push to `main` and once a day, so the
  site follows the repositories. To preview the whole site, with the gh CLI logged in:

  ```sh
  python3 pages/render.py --base / && python3 -m http.server -d _site
  ```

- `assets/site.css` and `assets/site.js` are shared by every page. The site is set in
  [M PLUS 1 and M PLUS 1 Code](https://github.com/coz-m/MPLUS_FONTS), which `assets/fonts` holds with their license,
  each cut into its Latin and its Japanese, so that a page loads the Japanese only when it has some.
- [`assets/logo`](assets/logo) holds the mark and the script that draws it.

The Go module that used to live here, kanna v0.2.x with the first kanna-orm, is kept on the
[`archive/v0.2`](https://github.com/go-kanna/kanna/tree/archive/v0.2) branch and its tags. kanna-orm has since been
written anew, in [go-kanna/orm](https://github.com/go-kanna/orm).

## License

[MIT](LICENSE)
