package validate_test

import (
	"errors"
	"go/types"
	"reflect"
	"strings"
	"testing"

	"github.com/go-kanna/kanna/internal/gen/validate"
	"github.com/go-kanna/kanna/internal/pkgtest"
)

const rulesSrc = `package test

import "time"

type Role string
type Age uint8

type Stamp struct{ t time.Time }

func (s Stamp) IsZero() bool { return s.t.IsZero() }

type Point struct{ X, Y int }

type Bag struct{ Items []string }

type OddZero struct{ X int }

func (OddZero) IsZero(int) bool { return false }

type Fields struct {
	S     string
	Role  Role
	I     int
	Age   Age
	F     float64
	F32   float32
	B     bool
	P     *string
	Sl    []string
	M     map[string]int
	A     [3]int
	Any   any
	When  time.Time
	Stamp Stamp
	Point Point
	Bag   Bag
	Odd   OddZero
	I8    int8
	U16   uint16
	R     rune
}
`

// fieldTypes returns the type of each field of Fields in rulesSrc, by name.
func fieldTypes(t *testing.T) map[string]types.Type {
	t.Helper()

	pkg := pkgtest.LoadFile(t, rulesSrc)
	st, ok := pkg.Types.Scope().Lookup("Fields").Type().Underlying().(*types.Struct)
	if !ok {
		t.Fatal("Fields is not a struct")
	}
	out := make(map[string]types.Type, st.NumFields())
	for f := range st.Fields() {
		out[f.Name()] = f.Type()
	}
	return out
}

func TestResolveRule(t *testing.T) {
	t.Parallel()

	typeOf := fieldTypes(t)

	tests := []struct {
		rule  string
		field string
		want  validate.Check
	}{
		{rule: "required", field: "S", want: validate.RequiredCheck{Zero: validate.ZeroString}},
		{rule: "required", field: "Role", want: validate.RequiredCheck{Zero: validate.ZeroString}},
		{rule: "required", field: "I", want: validate.RequiredCheck{Zero: validate.ZeroNumber}},
		{rule: "required", field: "B", want: validate.RequiredCheck{Zero: validate.ZeroBool}},
		{rule: "required", field: "P", want: validate.RequiredCheck{Zero: validate.ZeroNil}},
		{rule: "required", field: "Sl", want: validate.RequiredCheck{Zero: validate.ZeroNil}},
		{rule: "required", field: "M", want: validate.RequiredCheck{Zero: validate.ZeroNil}},
		{rule: "required", field: "Any", want: validate.RequiredCheck{Zero: validate.ZeroNil}},
		{rule: "required", field: "When", want: validate.RequiredCheck{Zero: validate.ZeroMethod}},
		{rule: "required", field: "Stamp", want: validate.RequiredCheck{Zero: validate.ZeroMethod}},
		{rule: "required", field: "Point", want: validate.RequiredCheck{Zero: validate.ZeroComposite}},
		{rule: "required", field: "A", want: validate.RequiredCheck{Zero: validate.ZeroComposite}},
		{rule: "required", field: "Odd", want: validate.RequiredCheck{Zero: validate.ZeroComposite}},

		{rule: "min=3", field: "S", want: validate.LengthCheck{Op: validate.OpGTE, Bound: 3, Runes: true}},
		{rule: "max=5", field: "Sl", want: validate.LengthCheck{Op: validate.OpLTE, Bound: 5}},
		{rule: "len=2", field: "M", want: validate.LengthCheck{Op: validate.OpEQ, Bound: 2}},
		{rule: "min=18", field: "I", want: validate.CompareCheck{Op: validate.OpGTE, Bound: "18"}},
		{rule: "max=1.5", field: "F", want: validate.CompareCheck{Op: validate.OpLTE, Bound: "1.5"}},
		{rule: "max=255", field: "Age", want: validate.CompareCheck{Op: validate.OpLTE, Bound: "255"}},
		{rule: "max=1e3", field: "I", want: validate.CompareCheck{Op: validate.OpLTE, Bound: "1e3"}},
		{rule: "min=-128", field: "I8", want: validate.CompareCheck{Op: validate.OpGTE, Bound: "-128"}},
		{rule: "max=65535", field: "U16", want: validate.CompareCheck{Op: validate.OpLTE, Bound: "65535"}},

		{rule: "gt=0", field: "F32", want: validate.CompareCheck{Op: validate.OpGT, Bound: "0"}},
		{rule: "gte=-10", field: "I", want: validate.CompareCheck{Op: validate.OpGTE, Bound: "-10"}},
		{rule: "lt=0x10", field: "I", want: validate.CompareCheck{Op: validate.OpLT, Bound: "0x10"}},
		{rule: "lte=9", field: "I", want: validate.CompareCheck{Op: validate.OpLTE, Bound: "9"}},
		{rule: "eq=1", field: "I", want: validate.CompareCheck{Op: validate.OpEQ, Bound: "1"}},
		{rule: "ne=0", field: "F", want: validate.CompareCheck{Op: validate.OpNE, Bound: "0"}},

		{rule: "oneof=admin member", field: "Role", want: validate.OneOfCheck{Values: []string{`"admin"`, `"member"`}}},
		{rule: "oneof=1 2", field: "I", want: validate.OneOfCheck{Values: []string{"1", "2"}}},

		{rule: `pattern=^\d{3}-\d{4}$`, field: "S", want: validate.PatternCheck{Expr: `^\d{3}-\d{4}$`}},

		{rule: "email", field: "S", want: validate.FormatCheck{Rule: "Email"}},
		{rule: "url", field: "Role", want: validate.FormatCheck{Rule: "URL"}},
		{rule: "uuid", field: "S", want: validate.FormatCheck{Rule: "UUID"}},
	}

	for _, tt := range tests {
		t.Run(tt.rule+" on "+tt.field, func(t *testing.T) {
			t.Parallel()

			got, err := validate.ResolveRule(parseRule(t, tt.rule), typeOf[tt.field], nil)
			if err != nil {
				t.Fatalf("ResolveRule() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveRule() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestResolveRule_Errors(t *testing.T) {
	t.Parallel()

	typeOf := fieldTypes(t)

	tests := []struct {
		rule     string
		field    string
		wantErr  string
		wantHint string
	}{
		{rule: "reqiured", field: "S", wantErr: `unknown rule "reqiured"`, wantHint: `did you mean "required"?`},
		{rule: "emial", field: "S", wantErr: `unknown rule "emial"`, wantHint: `did you mean "email"?`},
		{rule: "xyz", field: "S", wantErr: `unknown rule "xyz"`},
		{rule: "required=1", field: "S", wantErr: `rule "required" takes no value`},
		{rule: "email=x", field: "S", wantErr: `rule "email" takes no value`},

		{rule: "required", field: "Bag", wantErr: "required cannot tell whether test.Bag is empty", wantHint: "IsZero"},

		{rule: "max=fifty", field: "S", wantErr: `rule "max" expects a length, got "fifty"`},
		{rule: "min=-1", field: "S", wantErr: `rule "min" expects a length, got "-1"`},
		{rule: "max=5", field: "A", wantErr: "whose length is fixed"},
		{rule: "len=3", field: "I", wantErr: `rule "len" cannot apply to int`, wantHint: "compare a number with eq"},
		{rule: "max=fifty", field: "I", wantErr: `rule "max" expects a number, got "fifty"`},
		{rule: "max=300", field: "Age", wantErr: "300 does not fit test.Age"},
		{rule: "max=128", field: "I8", wantErr: "128 does not fit int8"},
		{rule: "max=65536", field: "U16", wantErr: "65536 does not fit uint16"},
		{rule: "max=2147483648", field: "R", wantErr: "2147483648 does not fit rune"},
		{rule: "min=-1", field: "Age", wantErr: "-1 does not fit test.Age"},
		{rule: "max=1.5", field: "I", wantErr: "1.5 does not fit int"},
		{rule: "max=1e39", field: "F32", wantErr: "1e39 does not fit float32"},
		{rule: "max=5", field: "B", wantErr: `rule "max" cannot apply to bool`},

		{rule: "gt=0", field: "S", wantErr: `rule "gt" cannot apply to string`, wantHint: "min, max, or len"},
		{rule: "gt=0", field: "B", wantErr: `rule "gt" cannot apply to bool`},
		{rule: "eq", field: "I", wantErr: `rule "eq" needs a value`},
		{rule: "gt=abc", field: "I", wantErr: `rule "gt" expects a number, got "abc"`},
		{rule: "min", field: "S", wantErr: `rule "min" needs a value, as in min=…`},
		{rule: "oneof", field: "S", wantErr: `rule "oneof" needs a value`},
		{rule: "eqfield", field: "S", wantErr: `rule "eqfield" needs a value`},
		{rule: "pattern", field: "S", wantErr: `rule "pattern" needs a value`},

		{rule: "oneof=1 2", field: "F", wantErr: `rule "oneof" cannot apply to float64`},
		{rule: "oneof=a b a", field: "S", wantErr: "lists a twice"},
		{rule: "oneof=1 300", field: "Age", wantErr: "300 does not fit test.Age"},

		{rule: "pattern=[a-z", field: "S", wantErr: "invalid pattern: error parsing regexp: missing closing ]"},
		{rule: "pattern=^a$", field: "I", wantErr: `rule "pattern" cannot apply to int`},

		{rule: "email", field: "I", wantErr: `rule "email" cannot apply to int`},
		{rule: "url", field: "When", wantErr: `rule "url" cannot apply to time.Time`},
	}

	for _, tt := range tests {
		t.Run(tt.rule+" on "+tt.field, func(t *testing.T) {
			t.Parallel()

			_, err := validate.ResolveRule(parseRule(t, tt.rule), typeOf[tt.field], nil)
			assertRuleError(t, err, tt.wantErr, tt.wantHint)
		})
	}
}

func TestResolveRule_EqField(t *testing.T) {
	t.Parallel()

	typeOf := fieldTypes(t)
	siblings := []validate.Sibling{
		{Name: "Password", JSONName: "password", Type: typeOf["S"]},
		{Name: "Age", JSONName: "age", Type: typeOf["I"]},
		{Name: "Tags", JSONName: "tags", Type: typeOf["Sl"]},
	}

	got, err := validate.ResolveRule(validate.Rule{Name: "eqfield", Param: "Password"}, typeOf["S"], siblings)
	if err != nil {
		t.Fatalf("ResolveRule() error = %v", err)
	}
	if want := (validate.ConfirmCheck{Field: "Password", Attribute: "password"}); got != want {
		t.Errorf("ResolveRule() = %#v, want %#v", got, want)
	}

	tests := []struct {
		name     string
		param    string
		field    string
		siblings []validate.Sibling
		wantErr  string
		wantHint string
	}{
		{
			name: "misspelled", param: "Pasword", field: "S", siblings: siblings,
			wantErr: "eqfield=Pasword: no such field", wantHint: `did you mean "Password"?`,
		},
		{name: "different type", param: "Age", field: "S", siblings: siblings, wantErr: "compares string with int"},
		{name: "not comparable", param: "Tags", field: "Sl", siblings: siblings, wantErr: "not comparable"},
		{name: "after dive", param: "Password", field: "S", siblings: nil, wantErr: "not to the elements after dive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := validate.ResolveRule(validate.Rule{Name: "eqfield", Param: tt.param}, typeOf[tt.field], tt.siblings)
			assertRuleError(t, err, tt.wantErr, tt.wantHint)
		})
	}
}

// parseRule parses a tag holding the single rule s.
func parseRule(t *testing.T, s string) validate.Rule {
	t.Helper()

	tag, err := validate.ParseTag(s)
	if err != nil {
		t.Fatalf("ParseTag(%q) error = %v", s, err)
	}
	return tag.Levels[0].Rules[0]
}

// assertRuleError checks that err is a RuleError whose message contains
// wantErr and, when wantHint is set, whose hints contain it.
func assertRuleError(t *testing.T, err error, wantErr, wantHint string) {
	t.Helper()

	rerr, ok := errors.AsType[*validate.RuleError](err)
	if !ok {
		t.Fatalf("error = %v, want a *RuleError", err)
	}
	if !strings.Contains(rerr.Error(), wantErr) {
		t.Errorf("Error() = %q, want it to contain %q", rerr.Error(), wantErr)
	}
	if hints := strings.Join(rerr.Hints, "\n"); !strings.Contains(hints, wantHint) {
		t.Errorf("Hints = %q, want them to contain %q", rerr.Hints, wantHint)
	}
}
