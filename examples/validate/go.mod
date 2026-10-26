module github.com/go-kanna/kanna/examples/validate

go 1.27.0

toolchain go1.27.3

// There is no matching require yet: kanna is unreleased, so this resolves
// through the repository's go.work. Once it carries a tag, running
// `go get -tool github.com/go-kanna/kanna/cmd/kanna-validate` in your own module
// adds the require and the setup stands on its own.
tool github.com/go-kanna/kanna/cmd/kanna-validate

require (
	github.com/go-kanna/kanna v0.1.6
	github.com/go-kanna/validated v0.0.1
)
