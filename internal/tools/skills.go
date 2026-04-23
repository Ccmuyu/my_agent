package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Skill struct {
	Name        string
	Description string
	Parameters string
}

func LoadSkills(skillsDir string) map[string]Skill {
	skills := make(map[string]Skill)

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return skills
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		skillName := strings.TrimSuffix(entry.Name(), ".md")
		path := filepath.Join(skillsDir, entry.Name())

		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		skills[skillName] = Skill{
			Name:        skillName,
			Description: string(content),
		}
	}

	return skills
}

func (s Skill) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## %s\n", s.Name))

	lines := strings.Split(s.Description, "\n")
	inParams := false
	for _, line := range lines {
		if strings.HasPrefix(line, "---") {
			inParams = !inParams
			continue
		}
		if inParams && strings.HasPrefix(line, "  ") {
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}