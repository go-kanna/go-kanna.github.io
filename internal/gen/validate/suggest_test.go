package validate_test

import (
	"testing"

	"github.com/go-kanna/kanna/internal/gen/validate"
)

func TestDistance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		a, b string
		want int
	}{
		{a: "", b: "", want: 0},
		{a: "max", b: "", want: 3},
		{a: "max", b: "max", want: 0},
		{a: "max", b: "maxx", want: 1},
		{a: "emial", b: "email", want: 1},
		{a: "reqiured", b: "required", want: 1},
		{a: "kitten", b: "sitting", want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.a+"/"+tt.b, func(t *testing.T) {
			t.Parallel()

			if got := validate.Distance(tt.a, tt.b); got != tt.want {
				t.Errorf("distance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
			if got := validate.Distance(tt.b, tt.a); got != tt.want {
				t.Errorf("distance(%q, %q) = %d, want %d", tt.b, tt.a, got, tt.want)
			}
		})
	}
}

func TestSuggest(t *testing.T) {
	t.Parallel()

	candidates := []string{"required", "email", "max", "min", "Password"}

	tests := []struct {
		name   string
		want   string
		wantOK bool
	}{
		{name: "reqiured", want: "required", wantOK: true},
		{name: "emial", want: "email", wantOK: true},
		{name: "maxx", want: "max", wantOK: true},
		{name: "Pasword", want: "Password", wantOK: true},
		{name: "mxa", want: "max", wantOK: true},
		{name: "xyz", wantOK: false},
		{name: "unrelated", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := validate.Suggest(tt.name, candidates)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("suggest(%q) = %q, %v; want %q, %v", tt.name, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
