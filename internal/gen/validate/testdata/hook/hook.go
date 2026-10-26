package hook

import "github.com/go-kanna/kanna/validate"

//kanna:validate
type Signup struct {
	Email string `json:"email" validate:"required"`
}

// validate is the hook the generated Validate calls last.
func (s Signup) validate(errs *validate.Errors) {
	if s.Email == "admin@example.com" {
		errs.Add("email", validate.Fail("reserved"))
	}
}
