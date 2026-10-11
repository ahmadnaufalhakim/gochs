package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const progressInterval = 500_000

type themeCount struct {
	Key         string `json:"key"`
	PuzzleCount int    `json:"puzzleCount"`
	XMLLabel    string `json:"xmlLabel,omitempty"`
	InXML       bool   `json:"inXML"`
}

type themeReport struct {
	PuzzlesRead             int          `json:"puzzlesRead"`
	Themes                  []themeCount `json:"themes"`
	CSVThemesMissingFromXML []string     `json:"csvThemesMissingFromXML"`
	XMLEntriesAbsentFromCSV []string     `json:"xmlEntriesAbsentFromCSV"`
}

type xmlResources struct {
	XMLName xml.Name    `xml:"resources"`
	Strings []xmlString `xml:"string"`
}

type xmlString struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

func main() {
	inputPath := flag.String("input", "data/puzzle/lichess_db_puzzle.csv", "path to the Lichess puzzle CSV")
	catalogPath := flag.String("catalog", "data/puzzle/puzzleTheme.xml", "path to the Lichess puzzle theme XML catalog")
	jsonPath := flag.String("json", "data/puzzle/themes.json", "path to write counted theme metadata")
	xmlPath := flag.String("xml", "data/puzzle/puzzleTheme.actual.xml", "path to write themes occurring in the CSV")
	flag.Parse()

	if err := run(*inputPath, *catalogPath, *jsonPath, *xmlPath, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inputPath, catalogPath, jsonPath, xmlPath string, progress io.Writer) error {
	catalog, err := readCatalog(catalogPath)
	if err != nil {
		return fmt.Errorf("read XML catalog: %w", err)
	}

	counts, puzzlesRead, err := countCSVThemes(inputPath, progress)
	if err != nil {
		return fmt.Errorf("read puzzle CSV: %w", err)
	}

	report := buildReport(counts, puzzlesRead, catalog)
	if err := writeJSON(jsonPath, report); err != nil {
		return fmt.Errorf("write JSON report: %w", err)
	}
	if err := writeActualThemesXML(xmlPath, report.Themes); err != nil {
		return fmt.Errorf("write actual-theme XML: %w", err)
	}

	fmt.Fprintf(progress, "Processed %d puzzles and found %d themes.\n", puzzlesRead, len(report.Themes))
	fmt.Fprintf(progress, "CSV themes missing from XML: %d; XML catalog entries absent from CSV: %d.\n", len(report.CSVThemesMissingFromXML), len(report.XMLEntriesAbsentFromCSV))
	return nil
}

func readCatalog(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var resources xmlResources
	if err := xml.NewDecoder(file).Decode(&resources); err != nil {
		return nil, err
	}
	catalog := make(map[string]string, len(resources.Strings))
	for _, entry := range resources.Strings {
		catalog[entry.Name] = entry.Value
	}
	return catalog, nil
}

func countCSVThemes(path string, progress io.Writer) (map[string]int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReaderSize(file, 1024*1024))
	header, err := reader.Read()
	if err != nil {
		return nil, 0, err
	}
	themesColumn := -1
	for index, name := range header {
		if name == "Themes" {
			themesColumn = index
			break
		}
	}
	if themesColumn == -1 {
		return nil, 0, errors.New("CSV header has no Themes column")
	}

	counts := make(map[string]int)
	puzzlesRead := 0
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("record %d: %w", puzzlesRead+2, err)
		}
		if themesColumn >= len(record) {
			return nil, 0, fmt.Errorf("record %d has no Themes value", puzzlesRead+2)
		}
		for _, theme := range strings.Fields(record[themesColumn]) {
			counts[theme]++
		}
		puzzlesRead++
		if puzzlesRead%progressInterval == 0 {
			fmt.Fprintf(progress, "Processed %d puzzles...\n", puzzlesRead)
		}
	}
	return counts, puzzlesRead, nil
}

func buildReport(counts map[string]int, puzzlesRead int, catalog map[string]string) themeReport {
	themes := make([]themeCount, 0, len(counts))
	missingFromXML := make([]string, 0)
	for key, count := range counts {
		label, inXML := catalog[key]
		themes = append(themes, themeCount{Key: key, PuzzleCount: count, XMLLabel: label, InXML: inXML})
		if !inXML {
			missingFromXML = append(missingFromXML, key)
		}
	}
	sort.Slice(themes, func(i, j int) bool { return themes[i].Key < themes[j].Key })
	sort.Strings(missingFromXML)

	absentFromCSV := make([]string, 0)
	for key := range catalog {
		if _, exists := counts[key]; !exists && !strings.HasSuffix(key, "Description") {
			absentFromCSV = append(absentFromCSV, key)
		}
	}
	sort.Strings(absentFromCSV)

	return themeReport{
		PuzzlesRead:             puzzlesRead,
		Themes:                  themes,
		CSVThemesMissingFromXML: missingFromXML,
		XMLEntriesAbsentFromCSV: absentFromCSV,
	}
}

func writeJSON(path string, report themeReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, append(data, '\n'))
}

func writeActualThemesXML(path string, themes []themeCount) error {
	resources := xmlResources{Strings: make([]xmlString, 0, len(themes))}
	for _, theme := range themes {
		label := theme.XMLLabel
		if label == "" {
			label = theme.Key
		}
		resources.Strings = append(resources.Strings, xmlString{Name: theme.Key, Value: label})
	}
	data, err := xml.MarshalIndent(resources, "", "  ")
	if err != nil {
		return err
	}
	data = append([]byte(xml.Header), data...)
	data = append(data, '\n')
	return writeFile(path, data)
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".themes-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
