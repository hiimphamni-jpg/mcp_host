package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FilesystemPolicy defines and enforces the MCP process allowlist and
// canonical root containment. It is pure and carries no application or
// mcp-go dependency.
type FilesystemPolicy struct {
	command string
	args    []string
	roots   []string
}

const unsafeCommandChars = "/\\|;&$()<>[]{}*~'\"`=! \t\r\n"

// New builds a FilesystemPolicy. The command must be a single bare
// executable name with no path separators or shell metacharacters; the
// child argv is the fixed parsed JSON array and is never LLM-supplied.
// All allowed roots are canonicalized to absolute, clean paths.
func New(command string, args []string, roots []string) (*FilesystemPolicy, error) {
	if err := validateCommand(command); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, errors.New("policy: at least one argument is required")
	}

	r, err := canonicalRoots(roots)
	if err != nil {
		return nil, err
	}

	return &FilesystemPolicy{command: command, args: args, roots: r}, nil
}

// ParseArgs validates that raw JSON decodes to a JSON array of strings.
func ParseArgs(raw string) ([]string, error) {
	var args []string
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, fmt.Errorf("args must be a JSON array of strings: %w", err)
	}
	return args, nil
}

func (p *FilesystemPolicy) Command() string { return p.command }
func (p *FilesystemPolicy) Args() []string {
	out := make([]string, len(p.args))
	copy(out, p.args)
	return out
}
func (p *FilesystemPolicy) Roots() []string {
	out := make([]string, len(p.roots))
	copy(out, p.roots)
	return out
}

// Contains reports whether the requested path is inside any of the
// canonical allowed roots. The requested path is canonicalized the same
// way as the roots, so a symlink escaping a root is denied.
func (p *FilesystemPolicy) Contains(path string) (bool, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("resolve %q: %w", path, err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err == nil {
		abs = real
	}
	target := filepath.Clean(abs)

	for _, root := range p.roots {
		rel, err := filepath.Rel(root, target)
		if err != nil {
			continue
		}
		if rel == "." {
			return true, nil
		}
		if rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel) {
			return true, nil
		}
	}
	return false, nil
}

func canonicalRoots(roots []string) ([]string, error) {
	if len(roots) == 0 {
		return nil, errors.New("policy: at least one allowed root is required")
	}
	seen := map[string]bool{}
	var out []string
	for _, r := range roots {
		if r == "" {
			continue
		}
		abs, err := filepath.Abs(r)
		if err != nil {
			return nil, fmt.Errorf("resolve root %q: %w", r, err)
		}
		if real, err := filepath.EvalSymlinks(abs); err == nil {
			abs = real
		}
		clean := filepath.Clean(abs)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
	}
	if len(out) == 0 {
		return nil, errors.New("policy: no valid allowed roots")
	}
	return out, nil
}

func validateCommand(cmd string) error {
	if cmd == "" {
		return errors.New("policy: command is empty")
	}
	if strings.ContainsAny(cmd, "/\\") {
		return errors.New("policy: command must be a bare executable name with no path separators")
	}
	if strings.ContainsAny(cmd, unsafeCommandChars) {
		return errors.New("policy: command contains unsafe characters")
	}
	return nil
}
