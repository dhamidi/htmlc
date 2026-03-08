package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_NoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "htmlc") {
		t.Errorf("stdout missing help content, got: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("unexpected stderr: %q", stderr.String())
	}
}

func TestRun_HelpFlag_Long(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "SUBCOMMANDS") {
		t.Errorf("stdout missing SUBCOMMANDS section, got: %q", stdout.String())
	}
}

func TestRun_HelpFlag_Short(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-h"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "SUBCOMMANDS") {
		t.Errorf("stdout missing SUBCOMMANDS section, got: %q", stdout.String())
	}
}

func TestRun_HelpSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "SUBCOMMANDS") {
		t.Errorf("stdout missing SUBCOMMANDS section, got: %q", stdout.String())
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

func TestRun_RenderHelpFlag_Long(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"render", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "render") {
		t.Errorf("stdout missing 'render', got: %q", out)
	}
}

func TestRun_RenderHelpFlag_Short(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"render", "-h"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "render") {
		t.Errorf("stdout missing 'render', got: %q", out)
	}
}

func TestRun_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"unknowncmd"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "unknowncmd") {
		t.Errorf("stderr missing subcommand name, got: %q", errOut)
	}
	if !strings.Contains(errOut, "htmlc help") {
		t.Errorf("stderr missing hint to run 'htmlc help', got: %q", errOut)
	}
}

func TestRun_UnknownSubcommand_ExitMessage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"foo"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `"foo"`) {
		t.Errorf("stderr should quote the unknown subcommand, got: %q", stderr.String())
	}
}
