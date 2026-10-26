package validate

import (
	"errors"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-kanna/kanna/internal/diag"
	"github.com/go-kanna/kanna/internal/ir"
)

// runtimePath is the import path of the package generated code calls.
const runtimePath = "github.com/go-kanna/kanna/validate"

// Plan is what the generated Validate method of one struct does.
type Plan struct {
	Struct ir.Struct

	// Fields are the fields with anything to check, in declaration order.
	Fields []FieldPlan

	// Hook reports whether the struct has a validate(*validate.Errors) method,
	// which the generated method calls last.
	Hook bool
}

// FieldPlan is what the generated method checks of one field.
type FieldPlan struct {
	// Name is the Go name of the field.
	Name string

	// Path is the name errors report the field by: its JSON name, or the Go
	// name when the json tag gives none.
	Path string

	// Flatten keeps what the field's own Validate reports at the level of the
	// struct, as encoding/json flattens an embedded struct without a JSON name.
	Flatten bool

	Value *ValuePlan
}

// ValuePlan is what the generated code checks of one value: a field, the value
// a pointer points to, or an element of a slice, array, or map.
type ValuePlan struct {
	Type types.Type

	// Required reports blank for the zero value, which Zero tells.
	Required bool

	// OmitEmpty lets the zero value, which Zero tells, past everything below.
	OmitEmpty bool

	Zero ZeroTest

	// Checks are the other rules, each reported on its own.
	Checks []Check

	// Nest calls the value's Validate method and nests what it reports under
	// the value's path.
	Nest bool

	// Pointee is what the rest of a pointer's rules check: the value it points
	// to, when it is not nil.
	Pointee *ValuePlan

	// Elem checks each element of a slice or array, or each value of a map.
	Elem *ValuePlan
}

// planner holds what deciding a value's plan needs beyond its type.
type planner struct {
	fset *token.FileSet

	// targets are the structs this run generates Validate for, which have it
	// even before it is written.
	targets map[*types.TypeName]bool

	// regenerated are the import paths of the packages whose output this run
	// rewrites, so a Validate an earlier run left there may be about to go.
	regenerated map[string]bool
}

// Plans decides what the Validate method of each target checks. fset resolves
// positions in the loaded packages, whose import paths regenerated names: their
// generated output is about to be rewritten.
func Plans(fset *token.FileSet, targets []ir.Struct, regenerated map[string]bool) ([]Plan, []diag.Diag) {
	p := planner{fset: fset, targets: make(map[*types.TypeName]bool, len(targets)), regenerated: regenerated}
	for _, s := range targets {
		p.targets[s.Named.Obj()] = true
	}

	var (
		plans []Plan
		diags []diag.Diag
	)
	for _, s := range targets {
		plan, ds := p.plan(s)
		diags = append(diags, ds...)
		if !diag.HasErrors(ds) {
			plans = append(plans, plan)
		}
	}
	return plans, diags
}

func (p planner) plan(s ir.Struct) (Plan, []diag.Diag) {
	plan := Plan{Struct: s}
	var diags []diag.Diag

	hook, err := hookOf(s.Named)
	if err != nil {
		diags = append(diags, diag.Errorf(s.Pos, "%s: %v", s.Name, err))
	}
	plan.Hook = hook

	siblings := make([]Sibling, len(s.Fields))
	for i, f := range s.Fields {
		path, _ := jsonName(f)
		siblings[i] = Sibling{Name: f.Name, JSONName: path, Type: f.Type}
	}

	for i, f := range s.Fields {
		others := slices.Delete(slices.Clone(siblings), i, i+1)
		fp, errs := p.field(f, others)
		for _, err := range errs {
			diags = append(diags, fieldDiag(f, err))
		}
		if len(errs) == 0 && fp.Value != nil {
			plan.Fields = append(plan.Fields, fp)
		}
	}
	return plan, diags
}

func (p planner) field(f ir.Field, siblings []Sibling) (FieldPlan, []error) {
	path, explicit := jsonName(f)
	fp := FieldPlan{Name: f.Name, Path: path, Flatten: f.Embedded && !explicit}

	value, found, err := LookupTag(f.Tag)
	if err != nil {
		return fp, []error{err}
	}
	var levels []Level
	if found {
		tag, err := ParseTag(value)
		if err != nil {
			return fp, []error{err}
		}
		levels = tag.Levels
	}

	var errs []error
	fp.Value, errs = p.value(f.Type, levels, siblings)
	return fp, errs
}

// value plans the checks of a value of type t. levels[0] holds the rules for
// the value itself and the rest those for its elements, one level per dive.
// siblings are the struct's other fields, for eqfield, at the field's own level
// only. The plan is nil when there is nothing to check.
func (p planner) value(t types.Type, levels []Level, siblings []Sibling) (*ValuePlan, []error) {
	var level Level
	if len(levels) > 0 {
		level = levels[0]
	}
	var below []Level
	if len(levels) > 1 {
		below = levels[1:]
	}

	vp := &ValuePlan{Type: t, OmitEmpty: level.OmitEmpty}
	var errs []error

	for _, r := range level.Rules {
		if r.Name == "required" {
			vp.Required = true
			if err := noValue(r); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if vp.Required && vp.OmitEmpty {
		errs = append(errs, errors.New("omitempty lets through the empty value required rejects; keep one of them"))
	}
	if vp.Required || vp.OmitEmpty {
		zero, ok := zeroTestOf(t)
		if !ok {
			rule := "omitempty"
			if vp.Required {
				rule = "required"
			}
			return nil, append(errs, cannotTellEmpty(rule, t))
		}
		vp.Zero = zero
	}

	if ptr, ok := t.Underlying().(*types.Pointer); ok {
		// The pointer itself is only ever nil or not; every rule but required
		// and omitempty is about what it points to.
		rest := Level{Rules: slices.DeleteFunc(slices.Clone(level.Rules), func(r Rule) bool { return r.Name == "required" })}
		pointee, es := p.value(ptr.Elem(), append([]Level{rest}, below...), siblings)
		vp.Pointee = pointee
		return pruned(vp), append(errs, es...)
	}

	for _, r := range level.Rules {
		if r.Name == "required" {
			continue
		}
		check, err := ResolveRule(r, t, siblings)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		vp.Checks = append(vp.Checks, check)
	}

	vp.Nest = p.hasValidate(t)

	elem, ok := elemOf(t)
	switch {
	case len(below) > 0 && !ok:
		errs = append(errs, &RuleError{Msg: "dive needs a slice, array, or map, not " + typeString(t)})
	case len(below) > 0:
		var es []error
		vp.Elem, es = p.value(elem, below, nil)
		errs = append(errs, es...)
	case ok && !vp.Nest:
		// The elements are validated whether or not the tag dives: a slice of
		// structs that validate themselves is checked as the structs would be.
		vp.Elem, _ = p.value(elem, nil, nil)
	}

	return pruned(vp), errs
}

// pruned returns vp, or nil when it checks nothing.
func pruned(vp *ValuePlan) *ValuePlan {
	if !vp.Required && len(vp.Checks) == 0 && !vp.Nest && vp.Pointee == nil && vp.Elem == nil {
		return nil
	}
	return vp
}

// elemOf returns the element type of a slice or array, or the value type of a
// map.
func elemOf(t types.Type) (types.Type, bool) {
	switch u := t.Underlying().(type) {
	case *types.Slice:
		return u.Elem(), true
	case *types.Array:
		return u.Elem(), true
	case *types.Map:
		return u.Elem(), true
	}
	return nil, false
}

// hasValidate reports whether the generated code can call Validate() error on
// a value of type t.
func (p planner) hasValidate(t types.Type) bool {
	if named, ok := types.Unalias(t).(*types.Named); ok && p.targets[named.Obj()] {
		return true
	}

	obj, _, _ := types.LookupFieldOrMethod(t, true, nil, "Validate")
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	sig := fn.Signature()
	if sig.Params().Len() != 0 || sig.Results().Len() != 1 ||
		!types.Identical(sig.Results().At(0).Type(), types.Universe.Lookup("error").Type()) {
		return false
	}

	// An earlier run's output in a package this run rewrites keeps only the
	// methods of today's targets, which the check above already found.
	if pkg := fn.Pkg(); pkg != nil && p.regenerated[pkg.Path()] &&
		filepath.Base(p.fset.Position(fn.Pos()).Filename) == outputFile {
		return false
	}
	return true
}

// hookOf reports whether named has a validate method for the generated
// Validate to call last, and why one by that name cannot be called.
func hookOf(named *types.Named) (bool, error) {
	obj, _, _ := types.LookupFieldOrMethod(named, true, named.Obj().Pkg(), "validate")
	fn, ok := obj.(*types.Func)
	if !ok {
		return false, nil
	}

	sig := fn.Signature()
	if sig.Params().Len() == 1 && sig.Results().Len() == 0 && isErrorsPointer(sig.Params().At(0).Type()) {
		return true, nil
	}
	return false, errors.New("the validate method must be func(*validate.Errors) for Validate to call it")
}

func isErrorsPointer(t types.Type) bool {
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	return ok && named.Obj().Name() == "Errors" && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == runtimePath
}

// jsonName returns the name encoding/json gives f, and whether its json tag
// spells it out; without one, it is the Go name.
func jsonName(f ir.Field) (string, bool) {
	tag, ok := f.Tag.Lookup("json")
	if !ok || tag == "-" {
		return f.Name, false
	}
	if name, _, _ := strings.Cut(tag, ","); name != "" {
		return name, true
	}
	return f.Name, false
}

// fieldDiag reports err at f, naming the field as the sketch of a run's output
// does: the rule first, the field it sits on after.
func fieldDiag(f ir.Field, err error) diag.Diag {
	if rerr, ok := errors.AsType[*RuleError](err); ok {
		return diag.Errorf(f.Pos, "%s (field %s)", rerr.Msg, f.Name).WithHints(rerr.Hints...)
	}
	return diag.Errorf(f.Pos, "%v (field %s)", err, f.Name)
}
