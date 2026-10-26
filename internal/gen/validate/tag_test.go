package validate_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/go-kanna/kanna/internal/gen/validate"
)

func TestLookupTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tag       reflect.StructTag
		want      string
		wantFound bool
		wantErr   string
	}{
		{name: "absent", tag: `json:"name"`},
		{name: "present", tag: `json:"name" validate:"required,max=50"`, want: "required,max=50", wantFound: true},
		{name: "escaped backslash", tag: `validate:"pattern=^\\d+$"`, want: `pattern=^\d+$`, wantFound: true},
		{
			name:      "unescaped backslash",
			tag:       `json:"zip" validate:"pattern=^\d+$"`,
			wantFound: true,
			wantErr:   `write a backslash as \\`,
		},
		{name: "unterminated", tag: `validate:"required`, wantFound: true, wantErr: "not a valid quoted string"},
		{name: "another tag broken", tag: `json:"a\d" validate:"required"`, want: "required", wantFound: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, found, err := validate.LookupTag(tt.tag)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("LookupTag() error = %v, want it to mention %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("LookupTag() error = %v", err)
			}
			if got != tt.want || found != tt.wantFound {
				t.Errorf("LookupTag() = %q, %v; want %q, %v", got, found, tt.want, tt.wantFound)
			}
		})
	}
}

func TestParseTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want []validate.Level
	}{
		{name: "empty", in: "", want: []validate.Level{{}}},
		{name: "one rule", in: "required", want: []validate.Level{{Rules: []validate.Rule{{Name: "required"}}}}},
		{
			name: "rules with values",
			in:   "required,max=50,oneof=admin member",
			want: []validate.Level{{Rules: []validate.Rule{
				{Name: "required"}, {Name: "max", Param: "50"}, {Name: "oneof", Param: "admin member"},
			}}},
		},
		{
			name: "omitempty",
			in:   "omitempty,email",
			want: []validate.Level{{OmitEmpty: true, Rules: []validate.Rule{{Name: "email"}}}},
		},
		{
			name: "dive",
			in:   "max=5,dive,omitempty,max=20",
			want: []validate.Level{
				{Rules: []validate.Rule{{Name: "max", Param: "5"}}},
				{OmitEmpty: true, Rules: []validate.Rule{{Name: "max", Param: "20"}}},
			},
		},
		{
			name: "dive twice",
			in:   "dive,dive,required",
			want: []validate.Level{{}, {}, {Rules: []validate.Rule{{Name: "required"}}}},
		},
		{
			name: "pattern keeps its commas",
			in:   `required,pattern=^\d{1,3}(,\d{3})*$`,
			want: []validate.Level{{Rules: []validate.Rule{
				{Name: "required"}, {Name: "pattern", Param: `^\d{1,3}(,\d{3})*$`},
			}}},
		},
		{
			name: "pattern after dive",
			in:   "dive,pattern=^a,b$",
			want: []validate.Level{{}, {Rules: []validate.Rule{{Name: "pattern", Param: "^a,b$"}}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := validate.ParseTag(tt.in)
			if err != nil {
				t.Fatalf("ParseTag(%q) error = %v", tt.in, err)
			}
			if !reflect.DeepEqual(got.Levels, tt.want) {
				t.Errorf("ParseTag(%q) = %+v, want %+v", tt.in, got.Levels, tt.want)
			}
		})
	}
}

func TestParseTag_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "trailing comma", in: "required,", want: "empty rule"},
		{name: "leading comma", in: ",required", want: "empty rule"},
		{name: "double comma", in: "required,,max=5", want: "empty rule"},
		{name: "space after comma", in: "required, max=50", want: "contains a space"},
		{name: "no name", in: "=5", want: "has no name"},
		{name: "no value", in: "max=", want: "no value after ="},
		{name: "dive with a value", in: "dive=1", want: "dive takes no value"},
		{name: "dive at the end", in: "max=5,dive", want: "dive must be followed"},
		{name: "omitempty with a value", in: "omitempty=1", want: "omitempty takes no value"},
		{name: "omitempty twice", in: "omitempty,omitempty", want: "given twice"},
		{name: "omitempty after a rule", in: "email,omitempty", want: "must come before email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := validate.ParseTag(tt.in)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("ParseTag(%q) error = %v, want it to mention %q", tt.in, err, tt.want)
			}
		})
	}
}
