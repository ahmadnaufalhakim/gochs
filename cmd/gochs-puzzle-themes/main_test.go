package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCountsAndCrossChecksThemes(t *testing.T) {
	directory := t.TempDir()
	csvPath := filepath.Join(directory, "puzzles.csv")
	xmlPath := filepath.Join(directory, "themes.xml")
	jsonPath := filepath.Join(directory, "themes.json")
	actualXMLPath := filepath.Join(directory, "actual.xml")

	csv := "PuzzleId,FEN,Moves,Themes\nfirst,fen,moves, fork pin\nsecond,fen,moves,pin missingTheme\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o600); err != nil {
		t.Fatal(err)
	}
	xml := "<?xml version=\"1.0\"?><resources><string name=\"fork\">Fork</string><string name=\"pin\">Pin</string><string name=\"unused\">Unused</string></resources>"
	if err := os.WriteFile(xmlPath, []byte(xml), 0o600); err != nil {
		t.Fatal(err)
	}

	var progress strings.Builder
	if err := run(csvPath, xmlPath, jsonPath, actualXMLPath, &progress); err != nil {
		t.Fatal(err)
	}

	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"\"puzzlesRead\": 2", "\"key\": \"fork\"", "\"puzzleCount\": 2", "\"csvThemesMissingFromXML\": [\n    \"missingTheme\"", "\"xmlEntriesAbsentFromCSV\": [\n    \"unused\""} {
		if !strings.Contains(string(jsonData), want) {
			t.Errorf("JSON report does not contain %q", want)
		}
	}

	actualXML, err := os.ReadFile(actualXMLPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name=\"fork\"", "name=\"pin\"", "name=\"missingTheme\""} {
		if !strings.Contains(string(actualXML), want) {
			t.Errorf("actual XML does not contain %q", want)
		}
	}
	if strings.Contains(string(actualXML), "name=\"unused\"") {
		t.Error("actual XML includes a catalog-only theme")
	}
}
