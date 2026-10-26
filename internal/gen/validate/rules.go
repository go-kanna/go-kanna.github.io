package validate

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Check is a rule resolved against the type of the value it checks. Its kind
// says what the generated code tests, and so which error it reports.
type Check interface {
	isCheck()
}

// RequiredCheck reports blank for the zero value of the type.
type RequiredCheck struct {
	Zero ZeroTest
}

// ZeroTest is how generated code tells the zero value of a type.
type ZeroTest int

const (
	// ZeroString compares a string with "".
	ZeroString ZeroTest = iota
	// ZeroNumber compares a number with 0.
	ZeroNumber
	// ZeroBool tests a bool for false.
	ZeroBool
	// ZeroNil compares a pointer, slice, map, interface, channel, or function
	// with nil. An empty slice or map that is not nil is not zero.
	ZeroNil
	// ZeroMethod calls the type's own IsZero method.
	ZeroMethod
	// ZeroComposite compares a comparable struct or array with its zero literal.
	ZeroComposite
)

// LengthCheck compares the length of a string, slice, or map with Bound,
// reporting too_short, too_long, or wrong_length.
type LengthCheck struct {
	// Op is OpGTE for min, OpLTE for max, and OpEQ for len.
	Op    CmpOp
	Bound int

	// Runes counts the characters of a string rather than its bytes.
	Runes bool
}

// CompareCheck compares a number with Bound, a Go literal, reporting the Rails
// error named after Op, such as greater_than_or_equal_to.
type CompareCheck struct {
	Op    CmpOp
	Bound string
}

// OneOfCheck accepts only the listed values, Go literals of the field's type,
// reporting inclusion.
type OneOfCheck struct {
	Values []string
}

// ConfirmCheck requires the value to equal another field of the struct,
// reporting confirmation with that field's JSON name as the attribute.
type ConfirmCheck struct {
	Field     string
	Attribute string
}

// PatternCheck requires a string to match Expr, reporting invalid.
type PatternCheck struct {
	Expr string
}

// FormatCheck runs one of the rules of github.com/go-kanna/validated/rules on a
// string, reporting whatever that rule does.
type FormatCheck struct {
	// Rule is the name of the rule type, such as "Email".
	Rule string
}

func (RequiredCheck) isCheck() {}
func (LengthCheck) isCheck()   {}
func (CompareCheck) isCheck()  {}
func (OneOfCheck) isCheck()    {}
func (ConfirmCheck) isCheck()  {}
func (PatternCheck) isCheck()  {}
func (FormatCheck) isCheck()   {}

// CmpOp is the comparison a check requires of the value.
type CmpOp int

const (
	OpEQ CmpOp = iota
	OpNE
	OpLT
	OpLTE
	OpGT
	OpGTE
)

// compareOps maps the comparison rules to what they require.
var compareOps = map[string]CmpOp{
	"eq":  OpEQ,
	"ne":  OpNE,
	"lt":  OpLT,
	"lte": OpLTE,
	"gt":  OpGT,
	"gte": OpGTE,
}

// formatRules maps the format rules to their types in validated/rules.
var formatRules = map[string]string{
	"email": "Email",
	"url":   "URL",
	"uuid":  "UUID",
}

// ruleNames lists every name a validate tag knows, for suggesting one in place
// of a misspelling.
var ruleNames = []string{
	"required", "omitempty", "dive",
	"min", "max", "len", "eq", "ne", "lt", "lte", "gt", "gte",
	"oneof", "eqfield", "pattern", "email", "url", "uuid",
}

// Sibling is another field of the struct, which eqfield can name.
type Sibling struct {
	// Name is the Go name, which is how eqfield spells it.
	Name string

	// JSONName is the name errors report it by.
	JSONName string

	Type types.Type
}

// RuleError explains why a rule does not apply where it is written.
type RuleError struct {
	Msg   string
	Hints []string
}

func (e *RuleError) Error() string {
	return e.Msg
}

// ResolveRule resolves r against t, the type of the value it checks, or
// explains why it does not apply. siblings are the other fields of the struct,
// for eqfield; they are nil below the field's own level, where eqfield has no
// field to compare with.
//
// ResolveRule takes t as it is: whether a pointer is checked itself or through
// what it points to is for the caller to decide.
func ResolveRule(r Rule, t types.Type, siblings []Sibling) (Check, error) {
	if _, ok := sizeOps[r.Name]; ok {
		return resolveSize(r, t)
	}
	if _, ok := compareOps[r.Name]; ok {
		return resolveCompare(r, t)
	}
	if rule, ok := formatRules[r.Name]; ok {
		return resolveFormat(r, t, rule)
	}

	switch r.Name {
	case "required":
		if err := noValue(r); err != nil {
			return nil, err
		}
		return resolveRequired(t)
	case "oneof":
		return resolveOneOf(r, t)
	case "eqfield":
		return resolveEqField(r, t, siblings)
	case "pattern":
		return resolvePattern(r, t)
	}

	err := &RuleError{Msg: fmt.Sprintf("unknown rule %q", r.Name)}
	if s, ok := suggest(r.Name, ruleNames); ok {
		err.Hints = []string{fmt.Sprintf("did you mean %q?", s)}
	}
	return nil, err
}

func resolveRequired(t types.Type) (Check, error) {
	zero, ok := zeroTestOf(t)
	if !ok {
		return nil, cannotTellEmpty("required", t)
	}
	return RequiredCheck{Zero: zero}, nil
}

// zeroTestOf returns how generated code tells the zero value of t, and false
// when it cannot.
func zeroTestOf(t types.Type) (ZeroTest, bool) {
	// A nil pointer or interface cannot have IsZero called on it, so these are
	// compared with nil even when what they hold has the method.
	switch t.Underlying().(type) {
	case *types.Pointer, *types.Interface, *types.Slice, *types.Map, *types.Chan, *types.Signature:
		return ZeroNil, true
	}
	if hasIsZero(t) {
		return ZeroMethod, true
	}

	if b, ok := t.Underlying().(*types.Basic); ok {
		switch info := b.Info(); {
		case info&types.IsString != 0:
			return ZeroString, true
		case info&types.IsNumeric != 0:
			return ZeroNumber, true
		case info&types.IsBoolean != 0:
			return ZeroBool, true
		}
	}
	if types.Comparable(t) {
		return ZeroComposite, true
	}
	return 0, false
}

// cannotTellEmpty reports a rule that needs to tell the zero value of t, which
// has no way to.
func cannotTellEmpty(rule string, t types.Type) error {
	return &RuleError{
		Msg: fmt.Sprintf("%s cannot tell whether %s is empty", rule, typeString(t)),
		Hints: []string{
			"it is not comparable and has no IsZero() bool method; give it one, or use a pointer",
		},
	}
}

// hasIsZero reports whether t, or a pointer to it, has an IsZero() bool method.
// The generated code calls it on an addressable value, so either receiver works.
func hasIsZero(t types.Type) bool {
	obj, _, _ := types.LookupFieldOrMethod(t, true, nil, "IsZero")
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Params().Len() != 0 || sig.Results().Len() != 1 {
		return false
	}
	return types.Identical(sig.Results().At(0).Type(), types.Typ[types.Bool])
}

// sizeOps maps min, max, and len to what they require of a length.
var sizeOps = map[string]CmpOp{"min": OpGTE, "max": OpLTE, "len": OpEQ}

func resolveSize(r Rule, t types.Type) (Check, error) {
	if err := needValue(r); err != nil {
		return nil, err
	}

	switch shape := shapeOf(t); shape { //nolint:exhaustive // nothing else has a length or a size
	case shapeString, shapeSlice, shapeMap:
		n, err := strconv.Atoi(r.Param)
		if err != nil || n < 0 {
			return nil, &RuleError{Msg: fmt.Sprintf("rule %q expects a length, got %q", r.Name, r.Param)}
		}
		return LengthCheck{Op: sizeOps[r.Name], Bound: n, Runes: shape == shapeString}, nil
	case shapeArray:
		return nil, &RuleError{Msg: fmt.Sprintf("rule %q cannot apply to %s, whose length is fixed", r.Name, typeString(t))}
	case shapeInteger, shapeFloat:
		if r.Name == "len" {
			return nil, &RuleError{
				Msg:   fmt.Sprintf("rule %q cannot apply to %s", r.Name, typeString(t)),
				Hints: []string{"len measures strings, slices, and maps; compare a number with eq"},
			}
		}
		bound, err := numberBound(r.Name, r.Param, t)
		if err != nil {
			return nil, err
		}
		return CompareCheck{Op: sizeOps[r.Name], Bound: bound}, nil
	}
	return nil, cannotApply(r, t)
}

func resolveCompare(r Rule, t types.Type) (Check, error) {
	if err := needValue(r); err != nil {
		return nil, err
	}

	switch shapeOf(t) { //nolint:exhaustive // only numbers compare, and only a length is worth a hint
	case shapeInteger, shapeFloat:
		bound, err := numberBound(r.Name, r.Param, t)
		if err != nil {
			return nil, err
		}
		return CompareCheck{Op: compareOps[r.Name], Bound: bound}, nil
	case shapeString, shapeSlice, shapeMap:
		return nil, &RuleError{
			Msg:   fmt.Sprintf("rule %q cannot apply to %s", r.Name, typeString(t)),
			Hints: []string{r.Name + " compares numbers; bound a length with min, max, or len"},
		}
	}
	return nil, cannotApply(r, t)
}

func resolveOneOf(r Rule, t types.Type) (Check, error) {
	if err := needValue(r); err != nil {
		return nil, err
	}

	shape := shapeOf(t)
	if shape != shapeString && shape != shapeInteger {
		return nil, cannotApply(r, t)
	}

	var values []string
	for v := range strings.FieldsSeq(r.Param) {
		lit := strconv.Quote(v)
		if shape == shapeInteger {
			var err error
			if lit, err = numberBound(r.Name, v, t); err != nil {
				return nil, err
			}
		}
		if slices.Contains(values, lit) {
			return nil, &RuleError{Msg: fmt.Sprintf("rule %q lists %s twice", r.Name, v)}
		}
		values = append(values, lit)
	}
	return OneOfCheck{Values: values}, nil
}

func resolveEqField(r Rule, t types.Type, siblings []Sibling) (Check, error) {
	if err := needValue(r); err != nil {
		return nil, err
	}
	if siblings == nil {
		return nil, &RuleError{Msg: "eqfield applies to the field itself, not to the elements after dive"}
	}

	i := slices.IndexFunc(siblings, func(s Sibling) bool { return s.Name == r.Param })
	if i < 0 {
		err := &RuleError{Msg: fmt.Sprintf("eqfield=%s: no such field", r.Param)}
		names := make([]string, len(siblings))
		for j, s := range siblings {
			names[j] = s.Name
		}
		if s, ok := suggest(r.Param, names); ok {
			err.Hints = []string{fmt.Sprintf("did you mean %q?", s)}
		}
		return nil, err
	}

	target := siblings[i]
	if !types.Identical(t, target.Type) {
		return nil, &RuleError{Msg: fmt.Sprintf("eqfield=%s compares %s with %s, which differ",
			r.Param, typeString(t), typeString(target.Type))}
	}
	if !types.Comparable(t) {
		return nil, &RuleError{Msg: fmt.Sprintf("eqfield=%s cannot compare %s, which is not comparable",
			r.Param, typeString(t))}
	}
	return ConfirmCheck{Field: target.Name, Attribute: target.JSONName}, nil
}

func resolvePattern(r Rule, t types.Type) (Check, error) {
	if err := needValue(r); err != nil {
		return nil, err
	}
	if shapeOf(t) != shapeString {
		return nil, cannotApply(r, t)
	}
	if _, err := regexp.Compile(r.Param); err != nil {
		return nil, &RuleError{Msg: fmt.Sprintf("invalid pattern: %v", err)}
	}
	return PatternCheck{Expr: r.Param}, nil
}

func resolveFormat(r Rule, t types.Type, rule string) (Check, error) {
	if err := noValue(r); err != nil {
		return nil, err
	}
	if shapeOf(t) != shapeString {
		return nil, cannotApply(r, t)
	}
	return FormatCheck{Rule: rule}, nil
}

// numberBound checks that param is a number t can hold, and returns it as the
// Go literal to compare with. A literal that does not fit would only fail when
// the generated code is compiled, far from the tag that caused it.
func numberBound(rule, param string, t types.Type) (string, error) {
	digits := strings.TrimPrefix(param, "-")

	var v constant.Value
	for _, tok := range []token.Token{token.INT, token.FLOAT} {
		if v = constant.MakeFromLiteral(digits, tok, 0); v.Kind() != constant.Unknown {
			break
		}
	}
	if v.Kind() == constant.Unknown {
		return "", &RuleError{Msg: fmt.Sprintf("rule %q expects a number, got %q", rule, param)}
	}
	if digits != param {
		v = constant.UnaryOp(token.SUB, v, 0)
	}

	if b, ok := t.Underlying().(*types.Basic); !ok || !fits(v, b) {
		return "", &RuleError{Msg: fmt.Sprintf("rule %q: %s does not fit %s", rule, param, typeString(t))}
	}
	return param, nil
}

// fits reports whether the constant v can be held by a value of type b.
//
// int, uint, and uintptr are taken to be 64 bits wide, so on a 32-bit platform
// a bound beyond 32 bits still fails when the generated code is compiled.
func fits(v constant.Value, b *types.Basic) bool {
	info := b.Info()
	if info&types.IsFloat != 0 {
		f, _ := constant.Float64Val(constant.ToFloat(v))
		if b.Kind() == types.Float32 {
			return math.Abs(f) <= math.MaxFloat32
		}
		return !math.IsInf(f, 0)
	}

	n := constant.ToInt(v)
	if n.Kind() != constant.Int {
		return false
	}
	bits := bitSize(b.Kind())
	if info&types.IsUnsigned != 0 {
		u, exact := constant.Uint64Val(n)
		return exact && u <= uint64(math.MaxUint64)>>(64-bits)
	}
	i, exact := constant.Int64Val(n)
	lo := int64(-1) << (bits - 1)
	return exact && lo <= i && i <= -(lo+1)
}

func bitSize(k types.BasicKind) int {
	switch k { //nolint:exhaustive // int, uint, uintptr, and the 64-bit kinds share the default
	case types.Int8, types.Uint8:
		return 8
	case types.Int16, types.Uint16:
		return 16
	case types.Int32, types.Uint32:
		return 32
	default:
		return 64
	}
}

// shape is what a rule cares about in a type: what its underlying type is.
type shape int

const (
	shapeOther shape = iota
	shapeString
	shapeInteger
	shapeFloat
	shapeSlice
	shapeArray
	shapeMap
)

func shapeOf(t types.Type) shape {
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch info := u.Info(); {
		case info&types.IsString != 0:
			return shapeString
		case info&types.IsInteger != 0:
			return shapeInteger
		case info&types.IsFloat != 0:
			return shapeFloat
		}
	case *types.Slice:
		return shapeSlice
	case *types.Array:
		return shapeArray
	case *types.Map:
		return shapeMap
	}
	return shapeOther
}

func noValue(r Rule) error {
	if r.Param != "" {
		return &RuleError{Msg: fmt.Sprintf("rule %q takes no value, got %q", r.Name, r.Param)}
	}
	return nil
}

func needValue(r Rule) error {
	if r.Param == "" {
		return &RuleError{Msg: fmt.Sprintf("rule %q needs a value, as in %s=…", r.Name, r.Name)}
	}
	return nil
}

func cannotApply(r Rule, t types.Type) error {
	return &RuleError{Msg: fmt.Sprintf("rule %q cannot apply to %s", r.Name, typeString(t))}
}

// typeString spells t with package names rather than import paths, as the
// source declaring the field would.
func typeString(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Name() })
}
