package policy_test

import (
	"os"
	"path/filepath"
	"testing"

	"mcp-host/internal/policy"
)

func mustNew(t *testing.T, cmd string, args, roots []string) *policy.FilesystemPolicy {
	t.Helper()
	p, err := policy.New(cmd, args, roots)
	if err != nil {
		t.Fatalf("New(%q) returned an error: %v", cmd, err)
	}
	return p
}

func TestPolicy_NewValidCommand(t *testing.T) {
	p := mustNew(t, "npx", []string{"-y", "@modelcontextprotocol/server-filesystem"}, []string{t.TempDir()})
	if p.Command() != "npx" {
		t.Errorf("Command() = %q, want npx", p.Command())
	}
	if len(p.Args()) != 2 {
		t.Errorf("Args() len = %d, want 2", len(p.Args()))
	}
}

func TestPolicy_NewRejectsUnsafeCommand(t *testing.T) {
	cases := []string{
		"/usr/bin/npx",
		`C:\bin\npx.exe`,
		"npx -y",
		"npx;rm -rf /",
		"npx|sh",
		"npx&&sh",
		"npx$(sh)",
		`npx\bin`,
	}
	roots := []string{t.TempDir()}
	for _, cmd := range cases {
		if _, err := policy.New(cmd, []string{"-y"}, roots); err == nil {
			t.Errorf("New accepted unsafe command %q", cmd)
		}
	}
}

func TestPolicy_NewRejectsEmptyCommand(t *testing.T) {
	if _, err := policy.New("", []string{"-y"}, []string{t.TempDir()}); err == nil {
		t.Error("New accepted empty command")
	}
}

func TestPolicy_CanonicalizesRoots(t *testing.T) {
	root := t.TempDir()
	p := mustNew(t, "npx", []string{"-y"}, []string{root})

	got := p.Roots()
	if len(got) != 1 {
		t.Fatalf("Roots() len = %d, want 1", len(got))
	}
	if !filepath.IsAbs(got[0]) {
		t.Errorf("root %q is not absolute", got[0])
	}

	// An existing file directly under the root is allowed.
	fileName := filepath.Join(root, "README.md")
	if err := os.WriteFile(fileName, []byte("placeholder"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	ok, err := p.Contains(fileName)
	if err != nil {
		t.Fatalf("Contains returned error: %v", err)
	}
	if !ok {
		t.Errorf("Contains(%q) = false, want true", fileName)
	}
}

func TestPolicy_ContainsDeniedOutsideRoot(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	p := mustNew(t, "npx", []string{"-y"}, []string{root})

	ok, err := p.Contains(other)
	if err != nil {
		t.Fatalf("Contains returned error: %v", err)
	}
	if ok {
		t.Errorf("Contains(%q) = true, want false", other)
	}
}

func TestPolicy_ContainsDeniedParentTraversal(t *testing.T) {
	root := t.TempDir()
	p := mustNew(t, "npx", []string{"-y"}, []string{root})

	escape := filepath.Join(root, "..", "..", "..", "etc")
	ok, err := p.Contains(escape)
	if err != nil {
		t.Fatalf("Contains returned error: %v", err)
	}
	if ok {
		t.Errorf("Contains(%q) = true, want false (parent traversal)", escape)
	}
}

func TestPolicy_ParseArgs_ValidJSONArray(t *testing.T) {
	args, err := policy.ParseArgs(`["-y","@modelcontextprotocol/server-filesystem"]`)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	if len(args) != 2 || args[0] != "-y" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestPolicy_ParseArgs_RejectsNonStringArray(t *testing.T) {
	for _, raw := range []string{
		`["a", 42]`,
		`"not-an-array"`,
		`{`,
	} {
		if _, err := policy.ParseArgs(raw); err == nil {
			t.Errorf("ParseArgs(%q) should fail", raw)
		}
	}
}
