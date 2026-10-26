// Package validate generates Validate methods from the validate tags of
// structs marked //kanna:validate.
//
// The tags are written as for go-playground/validator, but nothing reads them
// at run time: each rule becomes plain Go in the generated method, and the
// errors it reports carry Rails' codes, through the runtime package
// github.com/go-kanna/kanna/validate. A rule that does not exist or does not
// fit its field stops generation instead.
package validate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// tagKey is the struct tag holding the rules.
const tagKey = "validate"

// Rule is one rule of a validate tag, such as "max=50".
type Rule struct {
	Name string

	// Param is what follows the "=", or "" for a rule written without one.
	Param string
}

// Level holds the rules for one level of a field: the field itself, or, after
// each dive, the elements of the level before.
type Level struct {
	// OmitEmpty lets a zero value through the rules of this level.
	OmitEmpty bool

	Rules []Rule
}

// Tag is a parsed validate tag. Levels[0] is the field itself, and each dive
// adds a level for the elements of the one before.
type Tag struct {
	Levels []Level
}

// LookupTag returns the validate tag in tag and whether there is one.
//
// A tag that is there but is not a valid quoted string is an error. reflect
// treats it as absent, so without this check a field whose tag says
// `validate:"pattern=^\d+$"` would quietly go unchecked: inside a struct tag the
// value is a Go string literal, and \d is not an escape it knows.
func LookupTag(tag reflect.StructTag) (string, bool, error) {
	if value, ok := tag.Lookup(tagKey); ok {
		return value, true, nil
	}
	if !strings.Contains(string(tag), tagKey+`:"`) {
		return "", false, nil
	}
	if strings.Contains(string(tag), `\`) {
		return "", true, errors.New(`the validate tag is not a valid quoted string; write a backslash as \\`)
	}
	return "", true, errors.New("the validate tag is not a valid quoted string")
}

// ParseTag splits a validate tag into its levels and rules. It checks the
// syntax only; whether a rule exists and fits the field is for the caller.
//
// Rules are separated by commas, except that pattern takes everything after
// "pattern=" as its regular expression, which may well hold a comma, as in
// \d{1,3}. That makes pattern the last rule of a tag.
func ParseTag(s string) (Tag, error) {
	tag := Tag{Levels: []Level{{}}}
	if s == "" {
		return tag, nil
	}

	for more := true; more; {
		var item string
		if strings.HasPrefix(s, "pattern=") {
			item, more = s, false
		} else {
			item, s, more = strings.Cut(s, ",")
		}
		if err := tag.add(item); err != nil {
			return Tag{}, err
		}
	}

	if last := tag.Levels[len(tag.Levels)-1]; len(tag.Levels) > 1 && len(last.Rules) == 0 {
		return Tag{}, errors.New("dive must be followed by the rules for the elements")
	}
	return tag, nil
}

func (t *Tag) add(item string) error {
	name, param, hasParam := strings.Cut(item, "=")
	level := &t.Levels[len(t.Levels)-1]

	switch {
	case item == "":
		return errors.New("empty rule; check for a stray comma")
	case strings.ContainsAny(name, " \t"):
		return fmt.Errorf("rule %q contains a space; rules are separated by commas alone", item)
	case name == "":
		return fmt.Errorf("rule %q has no name", item)
	case hasParam && param == "":
		return fmt.Errorf("rule %q has no value after =", item)
	case name == "dive":
		if hasParam {
			return errors.New("dive takes no value")
		}
		t.Levels = append(t.Levels, Level{})
	case name == "omitempty":
		if hasParam {
			return errors.New("omitempty takes no value")
		}
		if level.OmitEmpty {
			return errors.New("omitempty is given twice")
		}
		if len(level.Rules) > 0 {
			return fmt.Errorf("omitempty must come before %s, which it lets an empty value through", level.Rules[0].Name)
		}
		level.OmitEmpty = true
	default:
		level.Rules = append(level.Rules, Rule{Name: name, Param: param})
	}
	return nil
}
