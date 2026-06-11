package tests

import (
	"strings"
	"testing"
)

func TestOpenSourceRoadmapCoversFeatureMatrix(t *testing.T) {
	root := repositoryRoot(t)
	featureMatrix := readRepositoryFile(t, root, "opensource-projects/FEATURE-MATRIX.md")
	roadmap := readRepositoryFile(t, root, "docs/POLYGOLEM-ROADMAP-MATRIX.md")

	capabilities := markdownTableRowsAfterHeader(featureMatrix, "| Capability |")
	if len(capabilities) == 0 {
		t.Fatal("expected feature matrix to contain capability rows")
	}

	roadmapRows := markdownTableRowsAfterHeader(roadmap, "| Capability |")
	roadmapByCapability := make(map[string][]string, len(roadmapRows))
	for _, row := range roadmapRows {
		if len(row) < 5 {
			t.Fatalf("roadmap row for %q must include current state, evidence, next reinforcement, and explicit non-goal: %#v", firstCell(row), row)
		}
		roadmapByCapability[row[0]] = row
	}

	for _, featureRow := range capabilities {
		capability := featureRow[0]
		roadmapRow, ok := roadmapByCapability[capability]
		if !ok {
			t.Fatalf("docs/POLYGOLEM-ROADMAP-MATRIX.md missing capability from FEATURE-MATRIX: %q", capability)
		}
		for i, columnName := range []string{"current polygolem state", "evidence", "next reinforcement", "explicit non-goal"} {
			cell := strings.TrimSpace(roadmapRow[i+1])
			if cell == "" {
				t.Fatalf("roadmap capability %q has empty %s cell", capability, columnName)
			}
			if columnName != "current polygolem state" && cell == "—" {
				t.Fatalf("roadmap capability %q must explain %s instead of using an em dash", capability, columnName)
			}
		}
	}
}

func markdownTableRowsAfterHeader(content, headerPrefix string) [][]string {
	lines := strings.Split(content, "\n")
	rows := [][]string{}
	inTable := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inTable {
			if strings.HasPrefix(trimmed, headerPrefix) {
				inTable = true
			}
			continue
		}
		if strings.HasPrefix(trimmed, "|---") {
			continue
		}
		if !strings.HasPrefix(trimmed, "|") {
			break
		}
		cells := strings.Split(strings.Trim(trimmed, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		rows = append(rows, cells)
	}
	return rows
}

func firstCell(row []string) string {
	if len(row) == 0 {
		return ""
	}
	return row[0]
}
