package bridge

import (
	"fmt"
	"strings"
)

// SourceLocation describes a position within a source file.
// It mirrors htmlc.SourceLocation without importing the parent package.
type SourceLocation struct {
	File    string // source file path
	Line    int    // 1-based line number (0 = unknown)
	Column  int    // 1-based column (0 = unknown)
	Snippet string // ≈3-line context around the error (may be empty)
}

// ConversionError is returned when a .vue→tmpl or tmpl→.vue conversion
// encounters an unsupported construct.  It is always returned wrapped together
// with htmlc.ErrConversion so callers can detect it with either errors.Is or
// errors.As:
//
//	var cerr *bridge.ConversionError
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
