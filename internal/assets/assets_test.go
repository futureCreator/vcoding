package assets_test

import (
	"strings"
	"testing"

	"github.com/futureCreator/vcoding/internal/assets"
	"gopkg.in/yaml.v3"
)

func TestLoadSkillTemplate(t *testing.T) {
	content, err := assets.LoadTemplate("SKILL.md")
	if err != nil {
		t.Fatalf("LoadTemplate(SKILL.md) error: %v", err)
	}
	for _, want := range []string{
		`name: "vcoding"`,
		"## Commands",
		"## Workflow",
		"## Output Files",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("SKILL.md missing expected content: %q", want)
		}
	}
}

func TestSkillTemplateFrontmatter(t *testing.T) {
	content, err := assets.LoadTemplate("SKILL.md")
	if err != nil {
		t.Fatalf("LoadTemplate(SKILL.md) error: %v", err)
	}

	lines := strings.Split(content, "\n")
	if !strings.HasPrefix(lines[0], "---") {
		t.Fatal("SKILL.md must start with --- frontmatter delimiter")
	}

	endIdx := 0
	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "---") {
			endIdx = i
			break
		}
	}
	if endIdx == 0 {
		t.Fatal("SKILL.md frontmatter closing --- not found")
	}

	yamlContent := strings.Join(lines[1:endIdx], "\n")
	var meta map[string]any
	if err := yaml.Unmarshal([]byte(yamlContent), &meta); err != nil {
		t.Fatalf("frontmatter must be valid YAML: %v", err)
	}
	if meta["name"] != "vcoding" {
		t.Errorf("expected name=vcoding, got %v", meta["name"])
	}
	if meta["protocol"] != "file" {
		t.Errorf("expected protocol=file, got %v", meta["protocol"])
	}
}

func TestRenderSkillTemplate(t *testing.T) {
	data := struct{ Version string }{Version: "1.2.3"}
	content, err := assets.RenderTemplate("SKILL.md", data)
	if err != nil {
		t.Fatalf("RenderTemplate(SKILL.md) error: %v", err)
	}
	if !strings.Contains(content, `version: "1.2.3"`) {
		t.Error("rendered SKILL.md missing injected version")
	}
}
