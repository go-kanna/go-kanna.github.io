package validate_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-kanna/kanna/internal/exit"
	"github.com/go-kanna/kanna/internal/gen/validate"
	"github.com/go-kanna/kanna/internal/packages"
)

// writeModule writes files into a fresh module and returns its directory. The
// module resolves github.com/go-kanna/kanna to this checkout, which the
// generated code imports, and borrows its go.sum so that nothing is fetched.
func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	//nolint:gosec // the path is this checkout's own go.sum
	sum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	files["go.mod"] = "module example.com/app\n\ngo 1.27.0\n\n" +
		"require github.com/go-kanna/kanna v0.0.0\n\n" +
		"replace github.com/go-kanna/kanna => " + root + "\n"
	files["go.sum"] = string(sum)
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// run runs the CLI in dir and returns its exit code, stdout, and stderr.
func run(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()

	var out, errOut bytes.Buffer
	code := validate.CLI{Out: &out, Err: &errOut, Version: "v0.0.0-test", Dir: dir}.Run(args)
	return code, out.String(), errOut.String()
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	//nolint:gosec // the path is built by the test from its own temp directory
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

const signupSrc = `package app

import "github.com/go-kanna/kanna/validate"

//kanna:validate
type Signup struct {
	Name string ` + "`json:\"name\" validate:\"required,max=50\"`" + `
}

func (s Signup) validate(errs *validate.Errors) {
	if s.Name == "root" {
		errs.Add("name", validate.Fail("reserved"))
	}
}
`

func TestCLI_Generates(t *testing.T) {
	t.Parallel()

	dir := writeModule(t, map[string]string{"app.go": signupSrc})
	path := filepath.Join(dir, "validate_gen.go")

	code, stdout, stderr := run(t, dir)
	if code != exit.OK {
		t.Fatalf("exit code = %d, want %d\nstderr: %s", code, exit.OK, stderr)
	}
	if strings.TrimSpace(stdout) != path {
		t.Errorf("stdout = %q, want the path written", stdout)
	}
	if got := readFile(t, path); !strings.Contains(got, "func (x Signup) Validate() error {") {
		t.Errorf("validate_gen.go does not declare Validate:\n%s", got)
	}

	// The output compiles against the runtime it imports.
	res, err := packages.Load([]string{"./..."}, packages.Config{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range res.Packages {
		for _, e := range pkg.Errors {
			t.Errorf("generated package does not compile: %v", e)
		}
	}

	if code, _, stderr := run(t, dir, "-check"); code != exit.OK {
		t.Errorf("-check right after generating: exit code = %d, want %d\nstderr: %s", code, exit.OK, stderr)
	}
}

func TestCLI_GeneratesEveryPackage(t *testing.T) {
	t.Parallel()

	other := strings.Replace(signupSrc, "package app", "package other", 1)
	dir := writeModule(t, map[string]string{"app.go": signupSrc, "other/other.go": other})

	code, stdout, stderr := run(t, dir, "./...")
	if code != exit.OK {
		t.Fatalf("exit code = %d, want %d\nstderr: %s", code, exit.OK, stderr)
	}
	for _, want := range []string{
		filepath.Join(dir, "validate_gen.go"),
		filepath.Join(dir, "other", "validate_gen.go"),
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout = %q, want it to name %s", stdout, want)
		}
	}
}

func TestCLI_Check(t *testing.T) {
	t.Parallel()

	dir := writeModule(t, map[string]string{"app.go": signupSrc})
	path := filepath.Join(dir, "validate_gen.go")

	code, _, stderr := run(t, dir, "-check")
	if code != exit.Error || !strings.Contains(stderr, "has not been generated yet") {
		t.Errorf("-check before generating: exit code = %d, stderr = %q; want a failure naming the missing file",
			code, stderr)
	}

	if code, _, stderr := run(t, dir); code != exit.OK {
		t.Fatalf("generate: exit code = %d\nstderr: %s", code, stderr)
	}
	before := readFile(t, path)
	writeFile(t, filepath.Join(dir, "app.go"), strings.Replace(signupSrc, "max=50", "max=40", 1))

	code, _, stderr = run(t, dir, "-check")
	if code != exit.Error || !strings.Contains(stderr, "is out of date") {
		t.Errorf("-check after a change: exit code = %d, stderr = %q; want a failure saying it is out of date", code, stderr)
	}
	if readFile(t, path) != before {
		t.Error("-check rewrote validate_gen.go")
	}
}

func TestCLI_RemovesLeftover(t *testing.T) {
	t.Parallel()

	dir := writeModule(t, map[string]string{"app.go": signupSrc})
	path := filepath.Join(dir, "validate_gen.go")
	if code, _, stderr := run(t, dir); code != exit.OK {
		t.Fatalf("generate: exit code = %d\nstderr: %s", code, stderr)
	}

	writeFile(t, filepath.Join(dir, "app.go"), "package app\n\ntype Signup struct{ Name string }\n")

	code, _, stderr := run(t, dir, "-check")
	if code != exit.Error || !strings.Contains(stderr, "is left over from an earlier run") {
		t.Errorf("-check: exit code = %d, stderr = %q; want a failure naming the leftover", code, stderr)
	}

	code, stdout, stderr := run(t, dir)
	if code != exit.OK {
		t.Fatalf("exit code = %d, want %d\nstderr: %s", code, exit.OK, stderr)
	}
	if !strings.Contains(stdout, "removed "+path) {
		t.Errorf("stdout = %q, want it to report the removal", stdout)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("validate_gen.go still exists: %v", err)
	}
}

// Renaming a field leaves the earlier output referring to one that is gone,
// which must not stop the run that would fix it.
func TestCLI_RegeneratesOverStaleOutput(t *testing.T) {
	t.Parallel()

	dir := writeModule(t, map[string]string{"app.go": signupSrc})
	if code, _, stderr := run(t, dir); code != exit.OK {
		t.Fatalf("generate: exit code = %d\nstderr: %s", code, stderr)
	}

	renamed := strings.ReplaceAll(signupSrc, "Name", "FullName")
	writeFile(t, filepath.Join(dir, "app.go"), renamed)

	code, _, stderr := run(t, dir)
	if code != exit.OK {
		t.Fatalf("exit code = %d, want %d\nstderr: %s", code, exit.OK, stderr)
	}
	if got := readFile(t, filepath.Join(dir, "validate_gen.go")); !strings.Contains(got, "x.FullName") {
		t.Errorf("validate_gen.go was not regenerated:\n%s", got)
	}
}

func TestCLI_Failures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name:  "no target",
			files: map[string]string{"app.go": "package app\n\ntype Signup struct{ Name string }\n"},
			want:  "no struct is marked //kanna:validate in .",
		},
		{
			name:  "rule error",
			files: map[string]string{"app.go": strings.Replace(signupSrc, "max=50", "maxx=50", 1)},
			want:  `app.go:7:2: error: unknown rule "maxx" (field Name)`,
		},
		{
			name: "hand-written output",
			files: map[string]string{
				"app.go":          signupSrc,
				"validate_gen.go": "package app\n",
			},
			want: "refusing to overwrite",
		},
		{
			name:  "type error elsewhere",
			files: map[string]string{"app.go": signupSrc + "\nvar broken int = \"x\"\n"},
			want:  "cannot use \"x\"",
		},
		{
			name:  "directive with arguments",
			files: map[string]string{"app.go": strings.Replace(signupSrc, "//kanna:validate", "//kanna:validate strict", 1)},
			want:  `//kanna:validate takes no arguments, got "strict"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := writeModule(t, tt.files)
			code, _, stderr := run(t, dir)
			if code != exit.Error {
				t.Errorf("exit code = %d, want %d", code, exit.Error)
			}
			if !strings.Contains(stderr, tt.want) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tt.want)
			}
		})
	}
}

// Setting a stale file aside only helps with the errors it caused. Anything
// else still fails the run, with a note on why the file was set aside.
func TestCLI_StaleFileNoteWhenOtherErrorsRemain(t *testing.T) {
	t.Parallel()

	dir := writeModule(t, map[string]string{"app.go": signupSrc})
	if code, _, stderr := run(t, dir); code != exit.OK {
		t.Fatalf("generate: exit code = %d\nstderr: %s", code, stderr)
	}

	// Renaming the struct breaks the receivers of the generated methods, so the
	// whole file is set aside, and the unrelated error remains.
	renamed := strings.ReplaceAll(signupSrc, "Signup", "Account") + "\nvar broken int = \"x\"\n"
	writeFile(t, filepath.Join(dir, "app.go"), renamed)

	code, _, stderr := run(t, dir)
	if code != exit.Error {
		t.Fatalf("exit code = %d, want %d\nstderr: %s", code, exit.Error, stderr)
	}
	for _, want := range []string{`cannot use "x"`, "validate_gen.go is stale and was set aside"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want it to contain %q", stderr, want)
		}
	}
}

func TestCLI_LoadFailure(t *testing.T) {
	t.Parallel()

	code, _, stderr := run(t, filepath.Join(t.TempDir(), "missing"))
	if code != exit.Error || stderr == "" {
		t.Errorf("exit code = %d, stderr = %q; want a failure to load", code, stderr)
	}
}

func TestNewCLI(t *testing.T) {
	t.Parallel()

	c := validate.NewCLI("v1.2.3")
	if c.Version != "v1.2.3" || c.Out != os.Stdout || c.Err != os.Stderr {
		t.Errorf("NewCLI() = %+v, want the version and the standard streams", c)
	}
}

func TestCLI_Warnings(t *testing.T) {
	t.Parallel()

	forgotten := "\ntype Profile struct {\n\tBio string `validate:\"max=200\"`\n}\n"
	dir := writeModule(t, map[string]string{"app.go": signupSrc + forgotten})

	code, _, stderr := run(t, dir)
	if code != exit.OK {
		t.Fatalf("exit code = %d, want %d\nstderr: %s", code, exit.OK, stderr)
	}
	if !strings.Contains(stderr, "warning: Profile has validate tags but no //kanna:validate") {
		t.Errorf("stderr = %q, want the warning", stderr)
	}
}

func TestCLI_Usage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{name: "help", args: []string{"help"}, wantCode: exit.OK, wantStdout: "Usage:"},
		{name: "-h", args: []string{"-h"}, wantCode: exit.OK, wantStdout: "Usage:"},
		{name: "version", args: []string{"--version"}, wantCode: exit.OK, wantStdout: "v0.0.0-test"},
		{name: "unknown flag", args: []string{"-nope"}, wantCode: exit.Usage, wantStderr: "Usage:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code, stdout, stderr := run(t, "", tt.args...)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			if !strings.Contains(stdout, tt.wantStdout) || !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("stdout = %q, stderr = %q; want %q and %q in them", stdout, stderr, tt.wantStdout, tt.wantStderr)
			}
		})
	}
}

func TestCLI_Verbose(t *testing.T) {
	t.Parallel()

	dir := writeModule(t, map[string]string{"app.go": signupSrc})
	code, stdout, stderr := run(t, dir, "-v", "--tags", "integration, extra", ".")
	if code != exit.OK {
		t.Fatalf("exit code = %d, want %d\nstderr: %s", code, exit.OK, stderr)
	}
	for _, want := range []string{"packages: 1", "structs: 1", "generate: "} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout = %q, want it to contain %q", stdout, want)
		}
	}
}
