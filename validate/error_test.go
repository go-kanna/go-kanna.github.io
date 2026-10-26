package validate_test

import (
	"maps"
	"testing"

	"github.com/go-kanna/kanna/validate"
)

func TestBuiltInErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        *validate.Error
		wantCode   string
		wantParams map[string]any
	}{
		{name: "Blank", err: validate.Blank(), wantCode: "blank"},
		{name: "TooShort", err: validate.TooShort(3), wantCode: "too_short", wantParams: map[string]any{"count": 3}},
		{name: "TooLong", err: validate.TooLong(50), wantCode: "too_long", wantParams: map[string]any{"count": 50}},
		{name: "WrongLength", err: validate.WrongLength(7), wantCode: "wrong_length", wantParams: map[string]any{"count": 7}},
		{
			name: "GreaterThan", err: validate.GreaterThan(0),
			wantCode: "greater_than", wantParams: map[string]any{"count": 0},
		},
		{
			name: "GreaterThanOrEqualTo", err: validate.GreaterThanOrEqualTo(18),
			wantCode: "greater_than_or_equal_to", wantParams: map[string]any{"count": 18},
		},
		{
			name: "LessThan", err: validate.LessThan(1.5),
			wantCode: "less_than", wantParams: map[string]any{"count": 1.5},
		},
		{
			name: "LessThanOrEqualTo", err: validate.LessThanOrEqualTo(uint8(130)),
			wantCode: "less_than_or_equal_to", wantParams: map[string]any{"count": uint8(130)},
		},
		{name: "EqualTo", err: validate.EqualTo(1), wantCode: "equal_to", wantParams: map[string]any{"count": 1}},
		{name: "OtherThan", err: validate.OtherThan(1), wantCode: "other_than", wantParams: map[string]any{"count": 1}},
		{name: "Inclusion", err: validate.Inclusion(), wantCode: "inclusion"},
		{
			name: "Confirmation", err: validate.Confirmation("password"),
			wantCode: "confirmation", wantParams: map[string]any{"attribute": "password"},
		},
		{name: "Invalid", err: validate.Invalid(), wantCode: "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.err.Code(); got != tt.wantCode {
				t.Errorf("Code() = %q, want %q", got, tt.wantCode)
			}
			if got := tt.err.Params(); !maps.Equal(got, tt.wantParams) {
				t.Errorf("Params() = %v, want %v", got, tt.wantParams)
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *validate.Error
		want string
	}{
		{name: "code only", err: validate.Fail("corp_domain"), want: "validate: corp_domain"},
		{name: "with params", err: validate.TooLong(50), want: "validate: too_long (count=50)"},
		{
			name: "params in key order",
			err:  validate.FailWith("between", map[string]any{"min": 1, "max": 9}),
			want: "validate: between (max=9, min=1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFailWith_CopiesParams(t *testing.T) {
	t.Parallel()

	params := map[string]any{"max": 20}
	err := validate.FailWith("max_len", params)
	params["max"] = 99
	err.Params()["max"] = 99

	if got := err.Params()["max"]; got != 20 {
		t.Errorf(`Params()["max"] = %v, want 20`, got)
	}
}
