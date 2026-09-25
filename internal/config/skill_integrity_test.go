package config

import (
	"strings"
	"testing"
)

func TestCheckSkillIntegrity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		dir          string
		wantWarnings []string // substrings, in order; empty means no warnings
	}{
		{
			name:         "healthy skill produces no warnings",
			dir:          "testdata/skill-integrity/healthy",
			wantWarnings: nil,
		},
		{
			name: "empty frontmatter reports missing name and description",
			dir:  "testdata/skill-integrity/empty-frontmatter",
			wantWarnings: []string{
				"frontmatter field 'name' is missing or empty",
				"frontmatter field 'description' is missing or empty",
			},
		},
		{
			name: "dangling attachment paths are reported in sorted order",
			dir:  "testdata/skill-integrity/broken-references",
			wantWarnings: []string{
				`references "assets/gone.png" but it does not exist on disk`,
				`references "references/missing.md" but it does not exist on disk`,
			},
		},
		{
			name:         "paths inside fenced code blocks are not references",
			dir:          "testdata/skill-integrity/codeblock-refs",
			wantWarnings: nil,
		},
		{
			name: "missing frontmatter is reported and body references are still checked",
			dir:  "testdata/skill-integrity/missing-frontmatter",
			wantWarnings: []string{
				"YAML frontmatter is missing",
				`references "references/orphan.md" but it does not exist on disk`,
			},
		},
		{
			name: "unterminated frontmatter is reported and body references are still checked",
			dir:  "testdata/skill-integrity/unterminated-frontmatter",
			wantWarnings: []string{
				"YAML frontmatter is not closed",
				`references "references/anything.md" but it does not exist on disk`,
			},
		},
		{
			name:         "directory without SKILL.md is skipped",
			dir:          "testdata/skill-integrity",
			wantWarnings: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := CheckSkillIntegrity(tt.dir)
			if len(got) != len(tt.wantWarnings) {
				t.Fatalf("CheckSkillIntegrity(%q) = %v, want %d warning(s)", tt.dir, got, len(tt.wantWarnings))
			}
			for i, want := range tt.wantWarnings {
				if !strings.Contains(got[i], want) {
					t.Errorf("warning[%d] = %q, want substring %q", i, got[i], want)
				}
			}
		})
	}
}
