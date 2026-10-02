package lmv

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// readSheet returns the cells of a workbook's first sheet, row by row. Only
// what the published file uses is supported: shared and inline strings and
// plain values. A row's cells are placed by their reference, since a writer
// may leave empty cells out.
func readSheet(file []byte) ([][]string, error) {
	archive, err := zip.NewReader(bytes.NewReader(file), int64(len(file)))
	if err != nil {
		return nil, fmt.Errorf("open the workbook: %w", err)
	}
	files := map[string]*zip.File{}
	for _, f := range archive.File {
		files[f.Name] = f
	}

	sheetPath, err := firstSheet(files)
	if err != nil {
		return nil, err
	}
	shared, err := sharedStrings(files)
	if err != nil {
		return nil, err
	}

	var sheet struct {
		Rows []struct {
			Cells []struct {
				Ref    string `xml:"r,attr"`
				Type   string `xml:"t,attr"`
				Value  string `xml:"v"`
				Inline struct {
					Text string `xml:"t"`
					Runs []struct {
						Text string `xml:"t"`
					} `xml:"r"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := decode(files, sheetPath, &sheet); err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(sheet.Rows))
	for _, row := range sheet.Rows {
		var cells []string
		for i, cell := range row.Cells {
			column := i
			if cell.Ref != "" {
				if column, err = columnIndex(cell.Ref); err != nil {
					return nil, err
				}
			}
			for len(cells) <= column {
				cells = append(cells, "")
			}
			switch cell.Type {
			case "s":
				var index int
				if _, err := fmt.Sscan(cell.Value, &index); err != nil || index < 0 || index >= len(shared) {
					return nil, fmt.Errorf("cell %s refers to shared string %q", cell.Ref, cell.Value)
				}
				cells[column] = shared[index]
			case "inlineStr":
				text := cell.Inline.Text
				for _, run := range cell.Inline.Runs {
					text += run.Text
				}
				cells[column] = text
			default:
				cells[column] = cell.Value
			}
		}
		rows = append(rows, cells)
	}
	return rows, nil
}

// firstSheet follows the workbook to the part of its first sheet.
func firstSheet(files map[string]*zip.File) (string, error) {
	var workbook struct {
		Sheets []struct {
			ID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := decode(files, "xl/workbook.xml", &workbook); err != nil {
		return "", err
	}
	if len(workbook.Sheets) == 0 {
		return "", errors.New("the workbook has no sheets")
	}
	var rels struct {
		Relationships []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if err := decode(files, "xl/_rels/workbook.xml.rels", &rels); err != nil {
		return "", err
	}
	for _, rel := range rels.Relationships {
		if rel.ID == workbook.Sheets[0].ID {
			if absolute, ok := strings.CutPrefix(rel.Target, "/"); ok {
				return absolute, nil
			}
			return path.Join("xl", rel.Target), nil
		}
	}
	return "", errors.New("the workbook's first sheet has no part")
}

func sharedStrings(files map[string]*zip.File) ([]string, error) {
	if _, ok := files["xl/sharedStrings.xml"]; !ok {
		return nil, nil
	}
	var table struct {
		Items []struct {
			Text string `xml:"t"`
			Runs []struct {
				Text string `xml:"t"`
			} `xml:"r"`
		} `xml:"si"`
	}
	if err := decode(files, "xl/sharedStrings.xml", &table); err != nil {
		return nil, err
	}
	out := make([]string, len(table.Items))
	for i, item := range table.Items {
		text := item.Text
		for _, run := range item.Runs {
			text += run.Text
		}
		out[i] = text
	}
	return out, nil
}

func decode(files map[string]*zip.File, name string, into any) error {
	f, ok := files[name]
	if !ok {
		return fmt.Errorf("the workbook has no %s", name)
	}
	r, err := f.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}
	defer r.Close()
	if err := xml.NewDecoder(io.LimitReader(r, 64<<20)).Decode(into); err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	return nil
}

// columnIndex turns a cell reference such as AB12 into its zero based column.
func columnIndex(ref string) (int, error) {
	column := 0
	letters := 0
	for _, r := range ref {
		if r < 'A' || r > 'Z' {
			break
		}
		column = column*26 + int(r-'A'+1)
		letters++
	}
	if letters == 0 || letters > 3 {
		return 0, fmt.Errorf("malformed cell reference %q", ref)
	}
	return column - 1, nil
}
