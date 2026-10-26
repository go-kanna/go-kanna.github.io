package validate

import (
	"go/token"
	"path/filepath"

	"github.com/go-kanna/kanna/internal/diag"
	"github.com/go-kanna/kanna/internal/directive"
	"github.com/go-kanna/kanna/internal/ir"
)

// directiveKey spells the directive //kanna:validate.
const directiveKey = "validate"

// outputFile is the file generated in each package.
const outputFile = "validate_gen.go"

// Targets returns the structs marked //kanna:validate, in the order scan
// reports them.
//
// A struct carrying validate tags without the directive is warned about: its
// rules would never run, and nothing else would say so. fset resolves the
// position of a Validate method the struct already has.
func Targets(fset *token.FileSet, structs []ir.Struct) ([]ir.Struct, []diag.Diag) {
	var (
		targets []ir.Struct
		diags   []diag.Diag
	)

	for _, s := range structs {
		d, msgs := directive.Find(s.Doc, directiveKey)
		diags = append(diags, msgs.Diags(s.Pos)...)
		if !d.Found {
			if hasValidateTag(s) {
				diags = append(diags, diag.Warningf(s.Pos,
					"%s has validate tags but no //kanna:validate, so they are never checked", s.Name))
			}
			continue
		}

		if ds := checkTarget(fset, s, d); len(ds) > 0 {
			diags = append(diags, ds...)
			continue
		}
		targets = append(targets, s)
	}

	return targets, diags
}

// checkTarget reports why s cannot have a Validate method generated for it.
func checkTarget(fset *token.FileSet, s ir.Struct, d directive.Directive) []diag.Diag {
	var diags []diag.Diag

	if d.Args != "" {
		diags = append(diags, diag.Errorf(s.Pos, "//kanna:%s takes no arguments, got %q", directiveKey, d.Args))
	}

	// A method on a generic type would need its type parameters, and a rule
	// written against a type parameter cannot be checked until it is
	// instantiated.
	if s.Named.TypeParams().Len() > 0 {
		diags = append(diags, diag.Errorf(s.Pos, "generic struct %s is not supported", s.Name))
	}

	for _, f := range s.Fields {
		if f.Name == "Validate" {
			diags = append(diags, diag.Errorf(f.Pos,
				"%s has a field named Validate, which the generated method would collide with", s.Name))
		}
	}

	// The Validate an earlier run generated is about to be replaced; one
	// written by hand would be redeclared.
	for m := range s.Named.Methods() {
		if pos := fset.Position(m.Pos()); m.Name() == "Validate" && filepath.Base(pos.Filename) != outputFile {
			diags = append(diags, diag.Errorf(s.Pos, "%s already has a Validate method", s.Name).
				WithHints("declared at "+pos.String()+"; remove it, or the //kanna:validate directive"))
		}
	}

	return diags
}

func hasValidateTag(s ir.Struct) bool {
	for _, f := range s.Fields {
		if _, ok := f.Tag.Lookup(tagKey); ok {
			return true
		}
	}
	return false
}
