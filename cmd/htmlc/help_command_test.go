package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_HelpSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "# htmlc CLI") {
		t.Errorf("stdout missing README content, got: %q", stdout.String())
	}
}

func TestRun_HelpRender(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help", "render"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "render") {
		t.Errorf("stdout missing 'render', got: %q", out)
	}
	if !strings.Contains(out, "FLAGS") {
		t.Errorf("stdout missing FLAGS section, got: %q", out)
	}
	if !strings.Contains(out, "EXAMPLES") {
		t.Errorf("stdout missing EXAMPLES section, got: %q", out)
	}
}

func TestRun_HelpPage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help", "page"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "page") {
		t.Errorf("stdout missing 'page', got: %q", out)
	}
	if !strings.Contains(out, "FLAGS") {
		t.Errorf("stdout missing FLAGS section, got: %q", out)
	}
	if !strings.Contains(out, "EXAMPLES") {
		t.Errorf("stdout missing EXAMPLES section, got: %q", out)
	}
}

func TestRun_HelpProps(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help", "props"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "props") {
		t.Errorf("stdout missing 'props', got: %q", out)
	}
	if !strings.Contains(out, "FLAGS") {
		t.Errorf("stdout missing FLAGS section, got: %q", out)
	}
	if !strings.Contains(out, "EXAMPLES") {
		t.Errorf("stdout missing EXAMPLES section, got: %q", out)
	}
}

func TestRun_HelpUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help", "unknowncmd"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "unknowncmd") {
		t.Errorf("stderr missing subcommand name, got: %q", stderr.String())
	}
}

func TestHelpAst(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help", "ast"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "ast") {
		t.Errorf("stdout missing 'ast', got: %q", out)
	}
}
