// Package lmv imports Livsmedelsverkets livsmedelsdatabas, the Swedish Food
// Agency's food composition database, published under CC BY 4.0.
//
// The whole database is one spreadsheet: a title row naming the version, a
// heading, then a header row and a row per food with its nutrients per 100 g.
package lmv

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Attribution is the credit CC BY 4.0 asks for, to show wherever the data is.
const Attribution = "Livsmedelsverkets livsmedelsdatabas, CC BY 4.0"

// Food is one food of a release, per 100 g edible part.
type Food struct {
	Number       int32
	Name         string
	Group        string
	Kcal         float64
	Protein      float64
	Carbs        float64
	Fat          float64
	Fiber        *float64
	Sugars       *float64
	SaturatedFat *float64
	Salt         *float64
	// Nutrients holds every value of the row by its column name, the ones
	// above included. An empty cell is left out.
	Nutrients map[string]float64
}

// Release is a parsed database file.
type Release struct {
	Version string
	Foods   []Food
}

// The columns the release is read by.
const (
	colName         = "Livsmedelsnamn"
	colNumber       = "Livsmedelsnummer"
	colGroup        = "Gruppering"
	colKcal         = "Energi (kcal)"
	colProtein      = "Protein (g)"
	colCarbs        = "Kolhydrater, tillgängliga (g)"
	colFat          = "Fett, totalt (g)"
	colFiber        = "Fiber (g)"
	colSugars       = "Sockerarter, totalt (g)"
	colSaturatedFat = "Summa mättade fettsyror (g)"
	colSalt         = "Salt, NaCl (g)"
)

var versionPattern = regexp.MustCompile(`version\s+(\d{4}-\d{2}-\d{2})`)

// Parse reads a release from the published spreadsheet. A file that no longer
// has the columns it is read by is refused, so a change of format keeps the
// foods already imported rather than replacing them with nonsense.
func Parse(file []byte) (Release, error) {
	rows, err := readSheet(file)
	if err != nil {
		return Release{}, err
	}

	var release Release
	header := -1
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		if m := versionPattern.FindStringSubmatch(row[0]); m != nil && release.Version == "" {
			release.Version = m[1]
		}
		if strings.TrimSpace(row[0]) == colName {
			header = i
			break
		}
	}
	if header < 0 {
		return Release{}, errors.New("the file has no header row")
	}
	if release.Version == "" {
		return Release{}, errors.New("the file does not name its version")
	}

	columns := map[string]int{}
	names := make([]string, len(rows[header]))
	for i, name := range rows[header] {
		name = strings.TrimSpace(name)
		names[i] = name
		if name != "" {
			columns[name] = i
		}
	}
	for _, required := range []string{colName, colNumber, colGroup, colKcal, colProtein, colCarbs, colFat} {
		if _, ok := columns[required]; !ok {
			return Release{}, fmt.Errorf("the file has no %q column", required)
		}
	}
	nonNutrients := map[int]bool{columns[colName]: true, columns[colNumber]: true, columns[colGroup]: true}

	seen := map[int32]bool{}
	for i, row := range rows[header+1:] {
		cell := func(column string) string {
			index, ok := columns[column]
			if !ok || index >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[index])
		}
		if cell(colName) == "" && cell(colNumber) == "" {
			continue
		}
		number, err := strconv.ParseInt(cell(colNumber), 10, 32)
		if err != nil || number <= 0 {
			return Release{}, fmt.Errorf("row %d: food number %q", header+i+2, cell(colNumber))
		}
		if seen[int32(number)] {
			return Release{}, fmt.Errorf("row %d: food number %d appears twice", header+i+2, number)
		}
		seen[int32(number)] = true

		food := Food{Number: int32(number), Name: cell(colName), Group: cell(colGroup), Nutrients: map[string]float64{}}
		for index, value := range row {
			if nonNutrients[index] || index >= len(names) || names[index] == "" {
				continue
			}
			if v, ok := parseNumber(value); ok {
				food.Nutrients[names[index]] = v
			}
		}
		food.Kcal = food.Nutrients[colKcal]
		food.Protein = food.Nutrients[colProtein]
		food.Carbs = food.Nutrients[colCarbs]
		food.Fat = food.Nutrients[colFat]
		food.Fiber = optional(food.Nutrients, colFiber)
		food.Sugars = optional(food.Nutrients, colSugars)
		food.SaturatedFat = optional(food.Nutrients, colSaturatedFat)
		food.Salt = optional(food.Nutrients, colSalt)
		release.Foods = append(release.Foods, food)
	}
	if len(release.Foods) == 0 {
		return Release{}, errors.New("the file has no foods")
	}
	return release, nil
}

func parseNumber(text string) (float64, bool) {
	text = strings.ReplaceAll(strings.TrimSpace(text), ",", ".")
	if text == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(text, 64)
	return v, err == nil
}

func optional(values map[string]float64, column string) *float64 {
	if v, ok := values[column]; ok {
		return &v
	}
	return nil
}
