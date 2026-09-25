package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// skillRefPattern matches relative references/, assets/, scripts/ paths cited
// in SKILL.md prose — the conventional Agent Skill attachment directories.
var skillRefPattern = regexp.MustCompile(`(?:references|assets|scripts)/[\w][\w./-]*[\w.]`)

// CheckSkillIntegrity inspects the SKILL.md of the skill rooted at skillDir
// and returns one human-readable warning per finding: a missing or
// unterminated YAML frontmatter block, an empty name or description field,
// and references/, assets/, scripts/ paths cited in the markdown body that do
// not exist on disk. Paths inside fenced code blocks are ignored —
// documentation examples are not real references.
//
// It returns nil when skillDir has no SKILL.md (the directory is not a skill
// root, so there is nothing to check). Every finding is a warning; callers
// decide whether to tolerate or enforce them (skill-up validate --strict).
func CheckSkillIntegrity(skillDir string) []string {
	data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return []string{fmt.Sprintf("SKILL.md: unreadable: %v", err)}
	}

	var warnings []string
	body := data

	lines := bytes.Split(data, []byte("\n"))
	hasOpeningFence := len(lines) > 0 && isFence(lines[0])
	frontmatter, rest, closed := splitFrontmatter(lines)
	switch {
	case !hasOpeningFence:
		warnings = append(warnings, "SKILL.md: YAML frontmatter is missing (file must start with a --- fence)")
	case !closed:
		warnings = append(warnings, "SKILL.md: YAML frontmatter is not closed (missing closing --- fence)")
	default:
		warnings = append(warnings, frontmatterFieldWarnings(frontmatter)...)
		body = rest
	}

	for _, ref := range missingSkillRefs(skillDir, body) {
		warnings = append(warnings, fmt.Sprintf("SKILL.md: references %q but it does not exist on disk", ref))
	}

	return warnings
}

// splitFrontmatter extracts the YAML frontmatter block and the markdown body
// from SKILL.md content lines. ok is false when the first line is not a ---
// fence or no closing --- fence exists.
func splitFrontmatter(lines [][]byte) (frontmatter, body []byte, ok bool) {
	if len(lines) == 0 || !isFence(lines[0]) {
		return nil, nil, false
	}
	for i := 1; i < len(lines); i++ {
		if isFence(lines[i]) {
			return bytes.Join(lines[1:i], []byte("\n")), bytes.Join(lines[i+1:], []byte("\n")), true
		}
	}
	return nil, nil, false
}

// frontmatterFieldWarnings reports required SKILL.md frontmatter fields
// (name, description) that are missing or empty.
func frontmatterFieldWarnings(frontmatter []byte) []string {
	var meta skillFrontmatter
	if err := yaml.Unmarshal(frontmatter, &meta); err != nil {
		return []string{fmt.Sprintf("SKILL.md: YAML frontmatter is invalid: %v", err)}
	}

	var warnings []string
	if strings.TrimSpace(meta.Name) == "" {
		warnings = append(warnings, "SKILL.md: frontmatter field 'name' is missing or empty")
	}
	if strings.TrimSpace(meta.Description) == "" {
		warnings = append(warnings, "SKILL.md: frontmatter field 'description' is missing or empty")
	}
	return warnings
}

// missingSkillRefs returns the sorted, de-duplicated references/, assets/,
// scripts/ paths cited in body that do not exist under skillDir. Fenced code
// blocks are stripped first so example paths in documentation snippets are
// not treated as references.
func missingSkillRefs(skillDir string, body []byte) []string {
	seen := make(map[string]struct{})
	var missing []string
	for _, ref := range skillRefPattern.FindAllString(string(stripCodeFences(body)), -1) {
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		if _, err := os.Stat(filepath.Join(skillDir, filepath.FromSlash(ref))); err != nil {
			missing = append(missing, ref)
		}
	}
	slices.Sort(missing)
	return missing
}

// stripCodeFences removes fenced code blocks (``` fences, possibly indented)
// from markdown content.
func stripCodeFences(text []byte) []byte {
	lines := bytes.Split(text, []byte("\n"))
	out := make([][]byte, 0, len(lines))
	inFence := false
	for _, line := range lines {
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("```")) {
			inFence = !inFence
			continue
		}
		if !inFence {
			out = append(out, line)
		}
	}
	return bytes.Join(out, []byte("\n"))
}
