package validate_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/go-kanna/kanna/validate"
)

// typeError stands in for *validated.Error, which is Coded without this package
// knowing about it.
type typeError struct{ code string }

func (e *typeError) Error() string          { return "validated: " + e.code }
func (e *typeError) Code() string           { return e.code }
func (e *typeError) Params() map[string]any { return nil }

func TestErrors_Add(t *testing.T) {
	t.Parallel()

	var errs validate.Errors
	errs.Add("name", nil)
	errs.Add("name", validate.Blank())

	want := validate.Errors{{Field: "name", Err: validate.Blank()}}
	if !equalErrors(errs, want) {
		t.Errorf("Errors = %v, want %v", errs, want)
	}
}

func TestErrors_Nest(t *testing.T) {
	t.Parallel()

	address := validate.Errors{
		{Field: "zip", Err: validate.Invalid()},
		{Field: validate.Base, Err: validate.Fail("unknown_area")},
	}
	plain := errors.New("lookup failed")

	tests := []struct {
		name  string
		field string
		err   error
		want  validate.Errors
	}{
		{name: "nil", field: "address", err: nil, want: nil},
		{
			name:  "Errors move under the field",
			field: "address",
			err:   address.Err(),
			want: validate.Errors{
				{Field: "address.zip", Err: validate.Invalid()},
				{Field: "address.base", Err: validate.Fail("unknown_area")},
			},
		},
		{
			name:  "empty field keeps the paths",
			field: "",
			err:   address.Err(),
			want: validate.Errors{
				{Field: "zip", Err: validate.Invalid()},
				{Field: "base", Err: validate.Fail("unknown_area")},
			},
		},
		{
			name:  "element of a slice",
			field: validate.Index("items", 2),
			err:   address.Err(),
			want: validate.Errors{
				{Field: "items[2].zip", Err: validate.Invalid()},
				{Field: "items[2].base", Err: validate.Fail("unknown_area")},
			},
		},
		{
			name:  "Coded error lands on the field",
			field: "email",
			err:   &typeError{code: "email"},
			want:  validate.Errors{{Field: "email", Err: &typeError{code: "email"}}},
		},
		{
			name:  "plain error lands on the field",
			field: "address",
			err:   plain,
			want:  validate.Errors{{Field: "address", Err: plain}},
		},
		{
			name:  "error without a field lands on base",
			field: "",
			err:   validate.Fail("unknown_area"),
			want:  validate.Errors{{Field: validate.Base, Err: validate.Fail("unknown_area")}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got validate.Errors
			got.Nest(tt.field, tt.err)
			if !equalErrors(got, tt.want) {
				t.Errorf("Errors = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		got, want string
	}{
		{got: validate.Index("tags", 2), want: "tags[2]"},
		{got: validate.Key("labels", "primary"), want: "labels.primary"},
		{got: validate.Key("scores", 3), want: "scores.3"},
		{got: validate.Index(validate.Key("groups", "a"), 0), want: "groups.a[0]"},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("path = %q, want %q", tt.got, tt.want)
		}
	}
}

func TestErrors_Err(t *testing.T) {
	t.Parallel()

	var empty validate.Errors
	if err := empty.Err(); err != nil {
		t.Errorf("empty Err() = %v, want nil", err)
	}

	errs := validate.Errors{{Field: "name", Err: validate.Blank()}}
	err := errs.Err()

	got, ok := errors.AsType[validate.Errors](err)
	if !ok || !equalErrors(got, errs) {
		t.Errorf("errors.AsType[Errors](Err()) = %v, %v; want %v", got, ok, errs)
	}
	if verr, ok := errors.AsType[*validate.Error](err); !ok || verr.Code() != "blank" {
		t.Errorf("errors.AsType[*Error](Err()) = %v, %v; want the blank error behind the field", verr, ok)
	}
}

func TestErrors_Error(t *testing.T) {
	t.Parallel()

	errs := validate.Errors{
		{Field: "name", Err: validate.Blank()},
		{Field: "tags[2]", Err: validate.TooLong(20)},
		{Field: "address", Err: errors.New("lookup failed")},
	}

	want := "validate: name: blank; tags[2]: too_long (count=20); address: lookup failed"
	if got := errs.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestFieldError_Code(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantCode   string
		wantParams map[string]any
	}{
		{name: "built-in", err: validate.TooLong(20), wantCode: "too_long", wantParams: map[string]any{"count": 20}},
		{name: "Coded from elsewhere", err: &typeError{code: "email"}, wantCode: "email"},
		{name: "not Coded", err: errors.New("lookup failed"), wantCode: "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fe := validate.FieldError{Field: "f", Err: tt.err}
			if got := fe.Code(); got != tt.wantCode {
				t.Errorf("Code() = %q, want %q", got, tt.wantCode)
			}
			if got := fe.Params(); len(got) != len(tt.wantParams) || got["count"] != tt.wantParams["count"] {
				t.Errorf("Params() = %v, want %v", got, tt.wantParams)
			}
		})
	}
}

func TestErrors_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		errs validate.Errors
		want string
	}{
		{name: "nil", errs: nil, want: `{}`},
		{
			name: "fields in the order they first broke",
			errs: validate.Errors{
				{Field: "password", Err: validate.Blank()},
				{Field: "name", Err: validate.Blank()},
				{Field: "password", Err: validate.TooShort(12)},
			},
			want: `{"password":[{"error":"blank"},{"error":"too_short","count":12}],"name":[{"error":"blank"}]}`,
		},
		{
			name: "params in key order beside the code",
			errs: validate.Errors{
				{Field: "confirm", Err: validate.FailWith("between", map[string]any{"min": 1, "max": 9})},
			},
			want: `{"confirm":[{"error":"between","max":9,"min":1}]}`,
		},
		{
			name: "a param named error is left out",
			errs: validate.Errors{
				{Field: "x", Err: validate.FailWith("odd", map[string]any{"error": "shadow"})},
			},
			want: `{"x":[{"error":"odd"}]}`,
		},
		{
			name: "error that is not Coded",
			errs: validate.Errors{{Field: "address", Err: errors.New("lookup failed")}},
			want: `{"address":[{"error":"invalid"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(tt.errs)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestErrors_MarshalJSON_UnencodableParam(t *testing.T) {
	t.Parallel()

	errs := validate.Errors{{Field: "x", Err: validate.FailWith("odd", map[string]any{"f": func() {}})}}
	if _, err := json.Marshal(errs); err == nil {
		t.Error("Marshal() error = nil, want an error for the func param")
	}
}

// equalErrors compares by field and by the text of each error, which is enough
// to tell the errors these tests build apart.
func equalErrors(a, b validate.Errors) bool {
	return slices.EqualFunc(a, b, func(x, y validate.FieldError) bool {
		return x.Field == y.Field && x.Err.Error() == y.Err.Error()
	})
}
