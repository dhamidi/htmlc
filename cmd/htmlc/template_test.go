package main

import (
	"bytes"
	"strings"
	"testing"
)

// --- vue-to-tmpl tests ---

func TestTemplateVueToTmpl_Basic(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(
		[]string{"template", "vue-to-tmpl", "-dir", "testdata/template", "Card"},
		&stdout, &stderr,
	)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `{{define "card"}}`) {
		t.Errorf("expected stdout to contain {{define \"card\"}}, got:\n%s", out)
	}
}

func TestTemplateVueToTmpl_UnsupportedDirective(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(
		[]string{"template", "vue-to-tmpl", "-dir", "testdata/template", "Unsupported"},
		&stdout, &stderr,
	)
	if code == 0 {
		t.Fatal("expected non-zero exit for unsupported directive")
	}
	if !strings.Contains(stderr.String(), "conversion error") {
		t.Errorf("expected stderr to contain 'conversion error', got: %s", stderr.String())
	}
}

func TestTemplateVueToTmpl_UnknownComponent(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(
		[]string{"template", "vue-to-tmpl", "-dir", "testdata/template", "NoSuchComponent"},
		&stdout, &stderr,
	)
	if code == 0 {
		t.Fatal("expected non-zero exit for unknown component")
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("expected stderr to contain 'not found', got: %s", stderr.String())
	}
}

func TestTemplateVueToTmpl_StdoutClean(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(
		[]string{"template", "vue-to-tmpl", "-dir", "testdata/template", "Card"},
		&stdout, &stderr,
	)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}
	out := stdout.String()
	if strings.Contains(out, "warning") {
		t.Errorf("stdout should not contain warnings, got: %s", out)
	}
}

func TestTemplateVueToTmpl_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(
		[]string{"template", "vue-to-tmpl", "-help"},
		&stdout, &stderr,
	)
	// -help causes flag.ErrHelp which we handle by printing help and returning nil
	if code != 0 {
		t.Fatalf("expected exit 0 for -help, got %d; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "vue-to-tmpl") {
		t.Errorf("expected help output to contain 'vue-to-tmpl', got: %s", stdout.String())
	}
}

// --- tmpl-to-vue tests ---

func TestTemplateTmplToVue_Simple(t *testing.T) {
	input := `<p>Hello</p>`
	var stdout, stderr bytes.Buffer
	err := runTemplateTmplToVue([]string{"-name", "Greeting"}, strings.NewReader(input), &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "<template>") {
		t.Errorf("expected stdout to contain <template>, got:\n%s", out)
	}
}

func TestTemplateTmplToVue_NameFlag(t *testing.T) {
	input := `<p>Hello {{ .name }}</p>`
	var stdout, stderr bytes.Buffer
	err := runTemplateTmplToVue([]string{"-name", "MyComp"}, strings.NewReader(input), &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "<template>") {
		t.Errorf("expected stdout to contain <template>, got:\n%s", out)
	}
}

func TestTemplateTmplToVue_UnsupportedWith(t *testing.T) {
	input := `{{with .user}}<p>hello</p>{{end}}`
	var stdout, stderr bytes.Buffer
	err := runTemplateTmplToVue([]string{}, strings.NewReader(input), &stdout, &stderr)
	// Either err is non-nil or errSilent was returned (exit code non-zero via stderr)
	if err == nil && stderr.Len() == 0 {
		t.Fatalf("expected error for unsupported {{with}}, got stdout: %s", stdout.String())
	}
}

func TestTemplateTmplToVue_Quiet(t *testing.T) {
	// Use a template that generates a warning (v-html equivalent in tmpl: none, but
	// a range loop produces a list wrapper which is a warning-level behaviour).
	// For this test we just verify -quiet doesn't suppress the output, only warnings.
	input := `<p>Hello</p>`
	var stdout, stderr bytes.Buffer
	err := runTemplateTmplToVue([]string{"-quiet"}, strings.NewReader(input), &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "<template>") {
		t.Errorf("expected stdout to contain <template>, got: %s", stdout.String())
	}
}

func TestTemplateTmplToVue_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runTemplateTmplToVue([]string{"-help"}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error for -help: %v", err)
	}
	if !strings.Contains(stdout.String(), "tmpl-to-vue") {
		t.Errorf("expected help output to contain 'tmpl-to-vue', got: %s", stdout.String())
	}
}

// --- template subcommand dispatch tests ---

func TestTemplateHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"template"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0 for 'template' with no args, got %d", code)
	}
	if !strings.Contains(stdout.String(), "vue-to-tmpl") {
		t.Errorf("expected help output to mention 'vue-to-tmpl', got: %s", stdout.String())
	}
}

func TestTemplateUnknownAction(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"template", "unknown-action"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit for unknown action")
	}
}
