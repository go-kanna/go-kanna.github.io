package validate_test

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"strings"
	"testing"

	"github.com/go-kanna/kanna/validate"
)

// email stands in for validated.String[rules.Email]: its UnmarshalJSON rejects
// a value with a Coded error and leaves the field as it was.
type email struct{ v string }

func (e *email) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err //nolint:wrapcheck // the type error is what the test expects back
	}
	if !strings.Contains(s, "@") {
		return &typeError{code: "email"}
	}
	e.v = s
	return nil
}

type address struct {
	Zip email `json:"zip"`
}

type signUp struct {
	Email   email   `json:"email"`
	Name    string  `json:"name"`
	Address address `json:"address"`
	Backups []email `json:"backups"`
}

// Validate stands in for what kanna-validate generates for
// `validate:"required"` on Email and Name.
func (s signUp) Validate() error {
	var errs validate.Errors
	if s.Email.v == "" {
		errs.Add("email", validate.Blank())
	}
	if s.Name == "" {
		errs.Add("name", validate.Blank())
	}
	return errs.Err()
}

func TestDecodeJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want validate.Errors
	}{
		{
			name: "valid",
			in:   `{"email":"a@example.com","name":"kanna"}`,
			want: nil,
		},
		{
			name: "every rejection with its path, then what Validate finds",
			in:   `{"email":"x","name":"","address":{"zip":"y"},"backups":["a@example.com","z"]}`,
			want: validate.Errors{
				{Field: "email", Err: &typeError{code: "email"}},
				{Field: "address.zip", Err: &typeError{code: "email"}},
				{Field: "backups[1]", Err: &typeError{code: "email"}},
				{Field: "name", Err: validate.Blank()},
			},
		},
		{
			name: "Validate on a field that was never sent",
			in:   `{"name":"kanna"}`,
			want: validate.Errors{{Field: "email", Err: validate.Blank()}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got signUp
			err := validate.DecodeJSON(strings.NewReader(tt.in), &got)
			if tt.want == nil {
				if err != nil {
					t.Fatalf("DecodeJSON() error = %v, want nil", err)
				}
				return
			}

			errs, ok := errors.AsType[validate.Errors](err)
			if !ok || !equalErrors(errs, tt.want) {
				t.Errorf("DecodeJSON() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestDecodeJSON_KeepsDecoding(t *testing.T) {
	t.Parallel()

	got := signUp{Email: email{v: "old@example.com"}}
	if err := validate.DecodeJSON(strings.NewReader(`{"email":"x","name":"kanna"}`), &got); err == nil {
		t.Fatal("DecodeJSON() error = nil, want the rejected email")
	}
	if got.Email.v != "old@example.com" {
		t.Errorf("Email = %q, want it left at %q", got.Email.v, "old@example.com")
	}
	if got.Name != "kanna" {
		t.Errorf("Name = %q, want %q decoded after the rejected email", got.Name, "kanna")
	}
}

func TestDecodeJSON_Unreadable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		as   func(error) bool
	}{
		{
			name: "malformed",
			in:   `{"email":`,
			as:   func(err error) bool { _, ok := errors.AsType[*jsontext.SyntacticError](err); return ok },
		},
		{
			name: "does not fit the Go type",
			in:   `{"name":1}`,
			as:   func(err error) bool { _, ok := errors.AsType[*jsonv2.SemanticError](err); return ok },
		},
		{
			name: "does not fit the type behind an UnmarshalJSON",
			in:   `{"email":1}`,
			as:   func(err error) bool { _, ok := errors.AsType[*json.UnmarshalTypeError](err); return ok },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got signUp
			err := validate.DecodeJSON(strings.NewReader(tt.in), &got)
			if _, ok := errors.AsType[validate.Errors](err); ok || !tt.as(err) {
				t.Errorf("DecodeJSON() error = %v (%T), want encoding/json's own error", err, err)
			}
		})
	}
}

func TestDecodeJSON_WithoutValidate(t *testing.T) {
	t.Parallel()

	var got address
	err := validate.DecodeJSON(strings.NewReader(`{"zip":"x"}`), &got)

	want := validate.Errors{{Field: "zip", Err: &typeError{code: "email"}}}
	if errs, ok := errors.AsType[validate.Errors](err); !ok || !equalErrors(errs, want) {
		t.Errorf("DecodeJSON() error = %v, want %v", err, want)
	}
}

func TestDecodeJSON_TopLevel(t *testing.T) {
	t.Parallel()

	var got email
	err := validate.DecodeJSON(strings.NewReader(`"x"`), &got)

	want := validate.Errors{{Field: validate.Base, Err: &typeError{code: "email"}}}
	if errs, ok := errors.AsType[validate.Errors](err); !ok || !equalErrors(errs, want) {
		t.Errorf("DecodeJSON() error = %v, want %v", err, want)
	}
}
