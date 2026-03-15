package htmlc

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors returned by Engine methods.
var (
	// ErrComponentNotFound is returned when the requested component name is not
	// registered in the engine.
	ErrComponentNotFound = errors.New("htmlc: component not found")

	// ErrMissingProp is returned when a required prop is absent from the render
	// scope and no MissingPropFunc has been set.
	ErrMissingProp = errors.New("htmlc: missing required prop")

	// ErrConversion is returned (wrapped) when a bridge conversion fails.
	// Callers can use errors.As to extract the underlying *ConversionError:
	//
	//	var cerr *ConversionError
	//	if errors.As(err, &cerr) {
	//	    fmt.Println(cerr.Location)
	//	}
	ErrConversion = errors.New("htmlc: conversion failed")
)

// SourceLocation describes a position within a source file.
type SourceLocation struct {
	File    string // source file path
	Line    int    // 1-based line number (0 = unknown)
	Column  int    // 1-based column (0 = unknown)
	Snippet string // ≈3-line context around the error (may be empty)
}

// ParseError is returned when a .vue file cannot be parsed.
type ParseError struct {
	// Path is the source file path.
	Path string
	// Msg is the human-readable description of the parse failure.
	Msg string
	// Location holds the source position of the error, or nil if unknown.
	Location *SourceLocation
}

func (e *ParseError) Error() string {
	if e.Location != nil && e.Location.Line > 0 {
		return fmt.Sprintf("%s:%d:%d: parse error: %s\n%s",
			e.Location.File, e.Location.Line, e.Location.Column,
			e.Msg, e.Location.Snippet)
	}
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
	// Location holds the source position of the error, or nil if unknown.
	Location *SourceLocation
}

func (e *RenderError) Error() string {
	loc := ""
	snippet := ""
	if e.Location != nil && e.Location.Line > 0 {
		loc = fmt.Sprintf("%s:%d: ", e.Location.File, e.Location.Line)
		snippet = e.Location.Snippet
	}
	if e.Expr != "" {
		return fmt.Sprintf("%srender %s: expr %q: %s\n%s",
			loc, e.Component, e.Expr, e.Wrapped, snippet)
	}
	return fmt.Sprintf("%srender %s: %s", loc, e.Component, e.Wrapped)
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

// ConversionError is returned when a .vue→tmpl or tmpl→.vue conversion
// encounters an unsupported construct.  It is always returned wrapped together
// with ErrConversion so callers can detect it with either errors.Is or
// errors.As:
//
//	var cerr *htmlc.ConversionError
//	if errors.As(err, &cerr) {
//	    log.Printf("conversion failed at %s:%d: %s", cerr.Location.File, cerr.Location.Line, cerr.Message)
//	}
//
// Conversion halts on the first unsupported construct; no partial output is
// produced.
type ConversionError struct {
	Component string          // component name, e.g. "PostPage"
	Directive string          // directive name, e.g. "v-if" (may be empty)
	Message   string          // human-readable cause
	Location  *SourceLocation // source position; may be nil
	Cause     error           // underlying error; may be nil
}

func (e *ConversionError) Error() string {
	var sb strings.Builder
	if e.Location != nil && e.Location.Line > 0 {
		fmt.Fprintf(&sb, "%s:%d:%d: conversion error: ",
			e.Location.File, e.Location.Line, e.Location.Column)
	} else if e.Component != "" {
		fmt.Fprintf(&sb, "%s: conversion error: ", e.Component)
	} else {
		sb.WriteString("conversion error: ")
	}
	sb.WriteString(e.Message)
	if e.Location != nil && e.Location.Snippet != "" {
		sb.WriteString("\n")
		sb.WriteString(e.Location.Snippet)
	}
	return sb.String()
}

func (e *ConversionError) Unwrap() error { return e.Cause }
