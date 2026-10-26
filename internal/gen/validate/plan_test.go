package validate_test

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/go-kanna/kanna/internal/diag"
	"github.com/go-kanna/kanna/internal/gen/validate"
	"github.com/go-kanna/kanna/internal/packages"
	"github.com/go-kanna/kanna/internal/pkgtest"
	"github.com/go-kanna/kanna/internal/scan"
)

// plans loads files as package "test", plans its targets, and returns each
// plan described by describePlan, keyed by struct name, along with every
// diagnostic rendered.
func plans(t *testing.T, files map[string]string) (map[string][]string, []string) {
	t.Helper()

	pkg := pkgtest.LoadPackage(t, files)
	structs, ds := scan.Structs([]*packages.Package{pkg})
	if diag.HasErrors(ds) {
		t.Fatalf("scan: %s", diag.Format(ds))
	}
	targets, ds := validate.Targets(pkg.Fset, structs)
	if diag.HasErrors(ds) {
		t.Fatalf("targets: %s", diag.Format(ds))
	}

	got, diags := validate.Plans(pkg.Fset, targets, map[string]bool{pkg.PkgPath: true})
	out := make(map[string][]string, len(got))
	for _, p := range got {
		out[p.Struct.Name] = describePlan(p)
	}
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.String()
	}
	return out, msgs
}

func describePlan(p validate.Plan) []string {
	lines := make([]string, 0, len(p.Fields)+1)
	for _, f := range p.Fields {
		path := f.Path
		if f.Flatten {
			path += ",flatten"
		}
		lines = append(lines, fmt.Sprintf("%s(%s): %s", f.Name, path, describeValue(f.Value)))
	}
	if p.Hook {
		lines = append(lines, "hook")
	}
	return lines
}

var zeroNames = map[validate.ZeroTest]string{
	validate.ZeroString:    `""`,
	validate.ZeroNumber:    "0",
	validate.ZeroBool:      "false",
	validate.ZeroNil:       "nil",
	validate.ZeroMethod:    "IsZero",
	validate.ZeroComposite: "{}",
}

var opNames = map[validate.CmpOp]string{
	validate.OpEQ: "==", validate.OpNE: "!=",
	validate.OpLT: "<", validate.OpLTE: "<=",
	validate.OpGT: ">", validate.OpGTE: ">=",
}

// describeValue spells a plan out on one line, such as
// `required=="" len>=3 nest []{nest}`.
func describeValue(v *validate.ValuePlan) string {
	var parts []string
	if v.Required {
		parts = append(parts, "required=="+zeroNames[v.Zero])
	}
	if v.OmitEmpty {
		parts = append(parts, "omitempty=="+zeroNames[v.Zero])
	}
	for _, c := range v.Checks {
		parts = append(parts, describeCheck(c))
	}
	if v.Nest {
		parts = append(parts, "nest")
	}
	if v.Pointee != nil {
		parts = append(parts, "*{"+describeValue(v.Pointee)+"}")
	}
	if v.Elem != nil {
		parts = append(parts, "[]{"+describeValue(v.Elem)+"}")
	}
	return strings.Join(parts, " ")
}

func describeCheck(c validate.Check) string {
	switch c := c.(type) {
	case validate.LengthCheck:
		if c.Runes {
			return fmt.Sprintf("runes%s%d", opNames[c.Op], c.Bound)
		}
		return fmt.Sprintf("len%s%d", opNames[c.Op], c.Bound)
	case validate.CompareCheck:
		return opNames[c.Op] + c.Bound
	case validate.OneOfCheck:
		return "oneof(" + strings.Join(c.Values, ",") + ")"
	case validate.ConfirmCheck:
		return "confirms " + c.Field + " as " + c.Attribute
	case validate.PatternCheck:
		return "pattern(" + c.Expr + ")"
	case validate.FormatCheck:
		return "rules." + c.Rule
	}
	return fmt.Sprintf("%#v", c)
}

const planSrc = `package test

import "time"

//kanna:validate
type Request struct {
	Name     string            ` + "`json:\"name\" validate:\"required,min=1,max=50\"`" + `
	Age      int               ` + "`json:\"age\" validate:\"gte=18\"`" + `
	Role     string            ` + "`json:\"role,omitempty\" validate:\"oneof=admin member\"`" + `
	Email    string            ` + "`json:\"email\" validate:\"omitempty,email\"`" + `
	Tags     []string          ` + "`json:\"tags\" validate:\"max=5,dive,max=20\"`" + `
	Nickname *string           ` + "`json:\"nickname\" validate:\"max=20\"`" + `
	Avatar   *string           ` + "`json:\"avatar\" validate:\"required,url\"`" + `
	When     time.Time         ` + "`json:\"when\" validate:\"required\"`" + `
	Password string            ` + "`json:\"password\"`" + `
	Confirm  string            ` + "`json:\"confirm\" validate:\"eqfield=Password\"`" + `
	Secret   string            ` + "`json:\"-\" validate:\"required\"`" + `
	Plain    string            ` + "`json:\",omitempty\" validate:\"required\"`" + `
	Address  Address           ` + "`json:\"address\"`" + `
	Billing  *Address          ` + "`json:\"billing\" validate:\"required\"`" + `
	Items    []Item            ` + "`json:\"items\" validate:\"min=1\"`" + `
	Labels   map[string]*Item  ` + "`json:\"labels\"`" + `
	Grid     [2]Item           ` + "`json:\"grid\"`" + `
	Checked  Checked           ` + "`json:\"checked\"`" + `
	Ignored  string            ` + "`json:\"ignored\"`" + `
	Audit
}

//kanna:validate
type Address struct {
	Zip string ` + "`json:\"zip\" validate:\"required,pattern=^\\\\d{3}-\\\\d{4}$\"`" + `
}

// Item validates itself by hand.
type Item struct{ SKU string }

func (Item) Validate() error { return nil }

// Checked has a Validate the generated code cannot call.
type Checked struct{ OK bool }

func (Checked) Validate(strict bool) error { return nil }

//kanna:validate
type Audit struct {
	By string ` + "`json:\"by\" validate:\"required\"`" + `
}
`

func TestPlans(t *testing.T) {
	t.Parallel()

	got, diags := plans(t, map[string]string{"test.go": planSrc})
	if len(diags) > 0 {
		t.Fatalf("diagnostics: %q", diags)
	}

	want := map[string][]string{
		"Request": {
			`Name(name): required=="" runes>=1 runes<=50`,
			`Age(age): >=18`,
			`Role(role): oneof("admin","member")`,
			`Email(email): omitempty=="" rules.Email`,
			`Tags(tags): len<=5 []{runes<=20}`,
			`Nickname(nickname): *{runes<=20}`,
			`Avatar(avatar): required==nil *{rules.URL}`,
			`When(when): required==IsZero`,
			`Confirm(confirm): confirms Password as password`,
			`Secret(Secret): required==""`,
			`Plain(Plain): required==""`,
			`Address(address): nest`,
			`Billing(billing): required==nil *{nest}`,
			`Items(items): len>=1 []{nest}`,
			`Labels(labels): []{*{nest}}`,
			`Grid(grid): []{nest}`,
			`Audit(Audit,flatten): nest`,
		},
		"Address": {`Zip(zip): required=="" pattern(^\d{3}-\d{4}$)`},
		"Audit":   {`By(by): required==""`},
	}
	for _, name := range slices.Sorted(maps.Keys(want)) {
		if !slices.Equal(got[name], want[name]) {
			t.Errorf("plan of %s:\n got  %q\n want %q", name, got[name], want[name])
		}
	}
}

func TestPlans_Hook(t *testing.T) {
	t.Parallel()

	// The hook takes the runtime's *validate.Errors, which only a package
	// loaded within the module can import.
	res, err := packages.Load([]string{"./testdata/hook"}, packages.Config{})
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

	got, ds := validate.Plans(res.Fset, targets, map[string]bool{res.Packages[0].PkgPath: true})
	if len(ds) > 0 {
		t.Fatalf("diagnostics: %s", diag.Format(ds))
	}
	if len(got) != 1 || !got[0].Hook {
		t.Errorf("plans = %+v, want one plan calling the hook", got)
	}
}

func TestPlans_EarlierOutput(t *testing.T) {
	t.Parallel()

	got, diags := plans(t, map[string]string{
		"test.go": `package test

//kanna:validate
type Order struct {
	Former Former ` + "`json:\"former\"`" + `
}

// Former was a target once, and an earlier run generated its Validate.
type Former struct{ Note string }
`,
		"validate_gen.go": `// Code generated by kanna-validate. DO NOT EDIT.

package test

func (x Order) Validate() error  { return nil }
func (x Former) Validate() error { return nil }
`,
	})
	if len(diags) > 0 {
		t.Fatalf("diagnostics: %q", diags)
	}

	// The rewritten file will not declare Former.Validate, so nothing calls it.
	if lines := got["Order"]; len(lines) > 0 {
		t.Errorf("plan of Order = %q, want no fields", lines)
	}
}

func TestPlans_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		field string
		hook  string
		want  string
	}{
		{
			name:  "rule error names the field",
			field: "Age int `validate:\"emial\"`",
			want:  "error: unknown rule \"emial\" (field Age)\n  did you mean \"email\"?",
		},
		{
			name:  "unreadable tag",
			field: "Zip string `validate:\"pattern=^\\d$\"`",
			want:  "write a backslash as",
		},
		{
			name:  "bad syntax",
			field: "Name string `validate:\"required,\"`",
			want:  "empty rule; check for a stray comma (field Name)",
		},
		{
			name:  "dive on a string",
			field: "Name string `validate:\"dive,required\"`",
			want:  "dive needs a slice, array, or map, not string",
		},
		{
			name:  "required and omitempty",
			field: "Name string `validate:\"omitempty,required\"`",
			want:  "omitempty lets through the empty value required rejects",
		},
		{
			name:  "required on an incomparable struct",
			field: "Bag struct{ S []string } `validate:\"required\"`",
			want:  "required cannot tell whether struct{S []string} is empty",
		},
		{
			name:  "omitempty on an incomparable struct",
			field: "Bag struct{ S []string } `validate:\"omitempty\"`",
			want:  "omitempty cannot tell whether struct{S []string} is empty",
		},
		{
			name:  "value on required of a pointer",
			field: "P *string `validate:\"required=1\"`",
			want:  `rule "required" takes no value`,
		},
		{
			name:  "eqfield after dive",
			field: "Tags []string `validate:\"dive,eqfield=Other\"`\n\tOther string",
			want:  "not to the elements after dive",
		},
		{
			name: "hook without a parameter",
			hook: "func (Target) validate() {}",
			want: "the validate method must be func(*validate.Errors)",
		},
		{
			name: "hook taking a value",
			hook: "func (Target) validate(int) {}",
			want: "the validate method must be func(*validate.Errors)",
		},
		{
			name: "hook taking another pointer",
			hook: "func (Target) validate(*int) {}",
			want: "the validate method must be func(*validate.Errors)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := "package test\n\n//kanna:validate\ntype Target struct {\n\t" + tt.field + "\n}\n\n" + tt.hook + "\n"
			got, diags := plans(t, map[string]string{"test.go": src})
			if _, ok := got["Target"]; ok {
				t.Errorf("Target was planned despite the error")
			}
			if !slices.ContainsFunc(diags, func(d string) bool { return strings.Contains(d, tt.want) }) {
				t.Errorf("diagnostics = %q, want one containing %q", diags, tt.want)
			}
		})
	}
}
