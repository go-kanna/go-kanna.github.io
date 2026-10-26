package validate

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

// DecodeJSON reads one JSON value from r into v, which must be a non-nil
// pointer, and reports everything wrong with it at once.
//
// A field whose UnmarshalJSON rejects its value with a Coded error, as a
// validated type does, does not stop the decoding: the error is recorded at the
// field's path, the field keeps its previous value, and decoding goes on. Then,
// when v has a Validate method, what it finds is added, except on the paths
// decoding rejected. Those fields were never set, and checking them would only
// report the absence of a value that was in fact sent.
//
// The result is the Errors of both, or nil. JSON that is malformed or does not
// fit the Go types fails with encoding/json/v2's own error instead: such a
// request cannot be read, rather than breaking a rule.
//
// Decoding follows encoding/json/v2, which, unlike encoding/json, matches names
// case-sensitively and rejects duplicate names.
func DecodeJSON(r io.Reader, v any) error {
	var errs Errors
	collect := json.UnmarshalFromFunc(func(dec *jsontext.Decoder, u json.Unmarshaler) error {
		raw, err := dec.ReadValue()
		if err != nil {
			return err //nolint:wrapcheck // encoding/json/v2 reports it with the offset it occurred at
		}
		err = u.UnmarshalJSON(raw)
		if _, ok := errors.AsType[Coded](err); ok {
			errs.Add(pathOf(dec), err)
			return nil
		}
		return err //nolint:wrapcheck // encoding/json/v2 reports it with the path it occurred at
	})
	if err := json.UnmarshalRead(r, v, json.WithUnmarshalers(collect)); err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	x, ok := v.(interface{ Validate() error })
	if !ok {
		return errs.Err()
	}

	rejected := make([]string, len(errs))
	for i, fe := range errs {
		rejected[i] = fe.Field
	}
	var found Errors
	found.Nest("", x.Validate())
	for _, fe := range found {
		if !slices.ContainsFunc(rejected, func(path string) bool { return within(fe.Field, path) }) {
			errs = append(errs, fe)
		}
	}
	return errs.Err()
}

// pathOf spells the path to the value dec has just read with JSON names, such
// as "address.zip" or "tags[1]", or returns Base for a top-level value.
//
// A token of a JSON pointer does not tell an array index from an object name,
// so the kind of each level comes from the decoder's stack, which has one level
// per token.
func pathOf(dec *jsontext.Decoder) string {
	var b strings.Builder
	level := 1
	for token := range dec.StackPointer().Tokens() {
		if kind, _ := dec.StackIndex(level); kind == jsontext.KindBeginArray {
			b.WriteString("[" + token + "]")
		} else {
			if b.Len() > 0 {
				b.WriteByte('.')
			}
			b.WriteString(token)
		}
		level++
	}
	if b.Len() == 0 {
		return Base
	}
	return b.String()
}

// within reports whether path is prefix or lies under it.
func within(path, prefix string) bool {
	rest, ok := strings.CutPrefix(path, prefix)
	return ok && (rest == "" || rest[0] == '.' || rest[0] == '[')
}
