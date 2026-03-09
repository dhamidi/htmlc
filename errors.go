package htmlc

import (
	"errors"
	"fmt"
)

// Sentinel errors returned by Engine methods.
var (
	// ErrComponentNotFound is returned when the requested component name is not
	// registered in the engine.
	ErrComponentNotFound = errors.New("htmlc: component not found")

	// ErrMissingProp is returned when a required prop is absent from the render
	// scope and no MissingPropFunc has been set.
	ErrMissingProp = errors.New("htmlc: missing required prop")
)

// ParseError is returned when a .vue file cannot be parsed.
type ParseError struct {
	// Path is the source file path.
	Path string
	// Msg is the human-readable description of the parse failure.
	Msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("htmlc: parse %s: %s", e.Path, e.Msg)
}

// RenderError is returned when template rendering fails for a named component.
type RenderError struct {
	// Component is the component name being rendered when the error occurred.
	Component string
	// Expr is the template expression that triggered the error (may be empty).
	Expr string
	// Wrapped is the underlying error.
	Wrapped error
}

func (e *RenderError) Error() string {
	if e.Expr != "" {
		return fmt.Sprintf("htmlc: render %s: expr %q: %s", e.Component, e.Expr, e.Wrapped)
	}
	return fmt.Sprintf("htmlc: render %s: %s", e.Component, e.Wrapped)
}

func (e *RenderError) Unwrap() error { return e.Wrapped }

// ValidationError describes a single problem found by Engine.ValidateAll.
type ValidationError struct {
	// Component is the name of the component that has the problem.
	Component string
	// Message describes the problem.
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("htmlc: validate %s: %s", e.Component, e.Message)
}
