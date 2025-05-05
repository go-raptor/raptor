package core

import (
	"encoding/json"
	"fmt"

	"github.com/go-raptor/raptor/v4/errs"
)

func JSONSerialize(c *Context, i interface{}, indent string) error {
	enc := json.NewEncoder(c.Response())
	if indent != "" {
		enc.SetIndent("", indent)
	}
	return enc.Encode(i)
}

func JSONDeserialize(c *Context, i interface{}) error {
	err := json.NewDecoder(c.Request().Body).Decode(i)
	if ute, ok := err.(*json.UnmarshalTypeError); ok {
		return errs.NewErrorBadRequest(fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset))
	} else if se, ok := err.(*json.SyntaxError); ok {
		return errs.NewErrorBadRequest(fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error()))
	}
	return err
}
