package validate

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-kanna/kanna/internal/diag"
	"github.com/go-kanna/kanna/internal/exit"
	"github.com/go-kanna/kanna/internal/output"
	"github.com/go-kanna/kanna/internal/packages"
	"github.com/go-kanna/kanna/internal/scan"
	"github.com/go-kanna/kanna/internal/stale"
)

// CLI is the command-line entry point for kanna-validate. Out and Err default
// to os.Stdout and os.Stderr when constructed through NewCLI.
type CLI struct {
	Out     io.Writer
	Err     io.Writer
	Version string

	// Dir is the directory package patterns are resolved against. Empty means
	// the process working directory, which go generate sets to the directory
	// of the file carrying the directive.
	Dir string
}

// NewCLI constructs a CLI with the default writers and the given version.
func NewCLI(version string) CLI {
	return CLI{
		Out:     os.Stdout,
		Err:     os.Stderr,
		Version: version,
	}
}

// Run parses args (excluding the program name) and returns one of the codes in
// package exit: Error when generation fails, Usage when the invocation itself
// is wrong.
func (c CLI) Run(args []string) int {
	// Only a leading "help" is the subcommand. Scanning every argument for it
	// would let a package named help turn the run into a silent no-op.
	if len(args) > 0 && args[0] == "help" {
		c.printUsage(c.Out)
		return exit.OK
	}

	fs := flag.NewFlagSet("kanna-validate", flag.ContinueOnError)
	fs.SetOutput(c.Err)

	// The usage is printed here, which knows whether it answers -h (stdout) or
	// reports a mistake (stderr).
	fs.Usage = func() {}

	var (
		verbose     bool
		tagsRaw     string
		check       bool
		showVersion bool
	)
	fs.BoolVar(&verbose, "v", false, "verbose output")
	fs.BoolVar(&verbose, "verbose", false, "verbose output")
	fs.StringVar(&tagsRaw, "tags", "", "comma-separated build tags")
	fs.BoolVar(&check, "check", false, "verify generated files are up to date instead of writing them")
	fs.BoolVar(&showVersion, "version", false, "print version")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			c.printUsage(c.Out)
			return exit.OK
		}
		c.printUsage(c.Err)
		return exit.Usage
	}

	if showVersion {
		fmt.Fprintln(c.Out, c.Version)
		return exit.OK
	}

	// `//go:generate go tool kanna-validate` names no package: it means the one
	// the directive sits in.
	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"."}
	}

	files, ok := c.generate(patterns, splitTags(tagsRaw), verbose)
	if !ok {
		return exit.Error
	}

	if check {
		return c.check(files)
	}
	return c.write(files, verbose)
}

// pendingFile is what a run does to one package's output: write data, or, for
// a package with nothing left to generate, remove what an earlier run wrote.
type pendingFile struct {
	path   string
	data   []byte
	remove bool
}

// generate renders the output of every package matching patterns, touching
// nothing on disk, and reports whether it got that far.
func (c CLI) generate(patterns, tags []string, verbose bool) ([]pendingFile, bool) {
	res, dropped, err := stale.Load(patterns, packages.Config{Dir: c.Dir, BuildTags: tags}, outputFile)
	if err != nil {
		fmt.Fprintln(c.Err, err)
		return nil, false
	}
	pkgs := scan.DedupePackages(res.Packages)
	if verbose {
		fmt.Fprintln(c.Out, "packages:", len(pkgs))
	}

	structs, ds := scan.Structs(pkgs)
	c.printDiags(ds)
	if diag.HasErrors(ds) {
		for _, path := range dropped {
			fmt.Fprintf(c.Err, "note: %s is stale and was set aside; it is regenerated once the errors above are fixed\n",
				path)
		}
		return nil, false
	}

	targets, ds := Targets(res.Fset, structs)
	c.printDiags(ds)
	if diag.HasErrors(ds) {
		return nil, false
	}

	regenerated := make(map[string]bool, len(pkgs))
	for _, pkg := range pkgs {
		regenerated[pkg.PkgPath] = true
	}
	plans, ds := Plans(res.Fset, targets, regenerated)
	c.printDiags(ds)
	if diag.HasErrors(ds) {
		return nil, false
	}
	if verbose {
		fmt.Fprintln(c.Out, "structs:", len(plans))
	}

	byPkg := make(map[string][]Plan)
	for _, p := range plans {
		byPkg[p.Struct.PkgPath] = append(byPkg[p.Struct.PkgPath], p)
	}

	var files []pendingFile
	for _, pkg := range pkgs {
		if len(pkg.GoFiles) == 0 {
			continue
		}
		dir := filepath.Dir(pkg.GoFiles[0])
		path := filepath.Join(dir, outputFile)

		pkgPlans := byPkg[pkg.PkgPath]
		if len(pkgPlans) == 0 {
			// A package whose structs all lost the directive keeps the methods an
			// earlier run gave them unless the file goes as well.
			if scan.GeneratedFiles(pkg)[path] {
				files = append(files, pendingFile{path: path, remove: true})
			}
			continue
		}

		taken := slices.Sorted(maps.Keys(output.DeclaredNames(dir, pkg.Name, outputFile)))
		src, err := Emit(pkg.Name, pkgPlans, taken)
		if err != nil {
			fmt.Fprintln(c.Err, err)
			return nil, false
		}
		files = append(files, pendingFile{path: path, data: src})
	}

	if len(files) == 0 {
		fmt.Fprintf(c.Err, "no struct is marked //kanna:%s in %s\n", directiveKey, strings.Join(patterns, " "))
		return nil, false
	}
	return files, true
}

// check reports every file a run would change, rather than the first: a CI
// failure should name everything one regeneration fixes.
func (c CLI) check(files []pendingFile) int {
	code := exit.OK
	for _, f := range files {
		if f.remove {
			fmt.Fprintf(c.Err, "%s is left over from an earlier run (run go generate)\n", f.path)
			code = exit.Error
			continue
		}
		if err := output.CheckUpToDate(f.path, f.data); err != nil {
			fmt.Fprintln(c.Err, err)
			code = exit.Error
		}
	}
	return code
}

func (c CLI) write(files []pendingFile, verbose bool) int {
	for _, f := range files {
		if f.remove {
			if err := os.Remove(f.path); err != nil {
				fmt.Fprintln(c.Err, err)
				return exit.Error
			}
			fmt.Fprintln(c.Out, "removed", f.path)
			continue
		}

		if err := output.Write(f.path, f.data); err != nil {
			fmt.Fprintln(c.Err, err)
			return exit.Error
		}
		if verbose {
			fmt.Fprintln(c.Out, "generate:", f.path)
		} else {
			fmt.Fprintln(c.Out, f.path)
		}
	}
	return exit.OK
}

func (c CLI) printDiags(ds []diag.Diag) {
	if len(ds) > 0 {
		fmt.Fprintln(c.Err, diag.Format(ds))
	}
}

func (c CLI) printUsage(w io.Writer) {
	fmt.Fprintln(w, "kanna-validate — generate Validate methods from the validate tags of structs marked //kanna:validate")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  kanna-validate [flags] [packages]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "The packages default to the one in the current directory.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --tags <list>      comma-separated build tags")
	fmt.Fprintln(w, "  -check             verify generated files are up to date instead of writing them")
	fmt.Fprintln(w, "  -v, --verbose      verbose output")
	fmt.Fprintln(w, "  --version          print version")
	fmt.Fprintln(w, "  -h, --help         show this help")
}

func splitTags(s string) []string {
	var tags []string
	for tag := range strings.SplitSeq(s, ",") {
		if tag = strings.TrimSpace(tag); tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
