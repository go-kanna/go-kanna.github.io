package validate_test

import (
	"bytes"
	"flag"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/go-kanna/kanna/internal/diag"
	"github.com/go-kanna/kanna/internal/gen/validate"
	"github.com/go-kanna/kanna/internal/ir"
	"github.com/go-kanna/kanna/internal/output"
	"github.com/go-kanna/kanna/internal/packages"
	"github.com/go-kanna/kanna/internal/scan"
)

var update = flag.Bool("update", false, "rewrite golden files")

// generate runs every stage on the package in dir, as the command would, and
// returns what it emits.
func generate(t *testing.T, dir string) []byte {
	t.Helper()

	res, err := packages.Load([]string{"./" + dir}, packages.Config{})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	structs, ds := scan.Structs(res.Packages)
	if diag.HasErrors(ds) {
		t.Fatalf("scan: %s", diag.Format(ds))
	}
	targets, ds := validate.Targets(res.Fset, structs)
	if diag.HasErrors(ds) {
		t.Fatalf("targets: %s", diag.Format(ds))
	}
	pkg := res.Packages[0]
	plans, ds := validate.Plans(res.Fset, targets, map[string]bool{pkg.PkgPath: true})
	if len(ds) > 0 {
		t.Fatalf("plans: %s", diag.Format(ds))
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	taken := slices.Sorted(maps.Keys(output.DeclaredNames(abs, pkg.Name, "validate_gen.go")))
	src, err := validate.Emit(pkg.Name, plans, taken)
	if err != nil {
		t.Fatalf("emit: %v\n%s", err, src)
	}
	return src
}

// TestEmit compares the output for each testdata package with its golden file.
// To accept a change in the output:
//
//	go test ./internal/gen/validate -run TestEmit -update
//
// testdata/emit keeps its golden file as validate_gen.go, a part of the package
// that TestGenerated compiles and calls. testdata/format calls the rules of
// go-kanna/validated, which this module does not require, so its golden file
// is only compared.
func TestEmit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		dir    string
		golden string
	}{
		{dir: "testdata/emit", golden: "validate_gen.go"},
		{dir: "testdata/format", golden: "validate_gen.go.golden"},
	}

	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			t.Parallel()

			got := generate(t, tt.dir)
			golden := filepath.Join(tt.dir, tt.golden)
			if *update {
				//nolint:gosec // golden files are source, readable like the rest of the tree
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatalf("update golden: %v", err)
				}
			}

			want, err := os.ReadFile(filepath.Clean(golden))
			if err != nil {
				t.Fatalf("read golden: %v (run with -update to create)", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("output differs from %s (run with -update to accept)\n--- got ---\n%s", golden, got)
			}
		})
	}
}

func TestEmit_Errors(t *testing.T) {
	t.Parallel()

	if _, err := validate.Emit("p", nil, nil); err == nil || !strings.Contains(err.Error(), "no plans") {
		t.Errorf("Emit(no plans) error = %v, want one about no plans", err)
	}

	spread := []validate.Plan{{Struct: ir.Struct{PkgPath: "a"}}, {Struct: ir.Struct{PkgPath: "b"}}}
	if _, err := validate.Emit("p", spread, nil); err == nil || !strings.Contains(err.Error(), "span packages a and b") {
		t.Errorf("Emit(two packages) error = %v, want one naming both", err)
	}
}

func TestRawString(t *testing.T) {
	t.Parallel()

	tests := []struct{ in, want string }{
		{in: `^\d+$`, want: "`^\\d+$`"},
		{in: "a`b", want: `"a` + "`" + `b"`},
		{in: "a\nb", want: `"a\nb"`},
	}
	for _, tt := range tests {
		if got := validate.RawString(tt.in); got != tt.want {
			t.Errorf("rawString(%q) = %s, want %s", tt.in, got, tt.want)
		}
	}
}
