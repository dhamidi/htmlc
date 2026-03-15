package bridge

import (
	"fmt"
	"strings"

	"github.com/dhamidi/htmlc"
)

// ConversionError is returned when a .vue→tmpl or tmpl→.vue conversion
// encounters an unsupported construct.
type ConversionError struct {
	Component string                // component name, e.g. "PostPage"
	Directive string                // directive name, e.g. "v-if" (may be empty)
	Message   string                // human-readable cause
	Location  *htmlc.SourceLocation // source position; may be nil
	Cause     error                 // underlying error; may be nil
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
