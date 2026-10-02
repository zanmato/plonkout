package lmv_test

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zanmato/plonkout/server/internal/food/lmv"
	"github.com/zanmato/plonkout/server/internal/platform/dbtest"
)

var header = []string{
	"Livsmedelsnamn", "Livsmedelsnummer", "Gruppering", "Energi (kcal)", "Energi (kJ)", "Fett, totalt (g)",
	"Protein (g)", "Kolhydrater, tillgängliga (g)", "Fiber (g)", "Sockerarter, totalt (g)",
	"Summa mättade fettsyror (g)", "Salt, NaCl (g)", "Järn, Fe (mg)",
}

// workbook builds a spreadsheet shaped like the published one. Strings go
// through the shared string table and empty cells are left out, as Excel
// writes them.
func workbook(t *testing.T, version string, foods ...[]string) []byte {
	t.Helper()
	rows := [][]string{
		{"Livsmedelsverkets livsmedelsdatabas version " + version},
		{"Näringsinnehåll per 100 gram livsmedel"},
		header,
	}
	rows = append(rows, foods...)

	var shared []string
	index := map[string]int{}
	var sheet strings.Builder
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for r, row := range rows {
		fmt.Fprintf(&sheet, `<row r="%d">`, r+1)
		for c, value := range row {
			if value == "" {
				continue
			}
			ref := fmt.Sprintf("%s%d", column(c), r+1)
			if r < 3 || c == 0 || c == 2 {
				i, ok := index[value]
				if !ok {
					i = len(shared)
					index[value] = i
					shared = append(shared, value)
				}
				fmt.Fprintf(&sheet, `<c r="%s" t="s"><v>%d</v></c>`, ref, i)
			} else {
				fmt.Fprintf(&sheet, `<c r="%s"><v>%s</v></c>`, ref, value)
			}
		}
		sheet.WriteString(`</row>`)
	}
	sheet.WriteString(`</sheetData></worksheet>`)

	var strs strings.Builder
	strs.WriteString(`<?xml version="1.0" encoding="UTF-8"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	for _, s := range shared {
		fmt.Fprintf(&strs, `<si><t>%s</t></si>`, html.EscapeString(s))
	}
	strs.WriteString(`</sst>`)

	parts := map[string]string{
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml":   sheet.String(),
		"xl/sharedStrings.xml":       strs.String(),
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range parts {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func column(i int) string {
	name := ""
	for i++; i > 0; i = (i - 1) / 26 {
		name = string(rune('A'+(i-1)%26)) + name
	}
	return name
}

var (
	pasta = []string{"Pasta kokt u. salt", "4065", "Pasta, ris", "128", "543", "0.5", "4.2", "25.8", "1.6", "0.3", "0.1", "0", "0.3"}
	pesto = []string{"Pesto hemlagad", "6201", "Såser", "545", "2250", "55", "10.6", "1.9", "1.5", "0.9", "", "1.6", ""}
	talg  = []string{"Nöt talg", "1", "Fett, olja", "884", "3700", "100", "0", "0", "0", "0", "54.8", "0", "0"}
)

func TestParse(t *testing.T) {
	release, err := lmv.Parse(workbook(t, "2026-07-01", pasta, pesto))
	if err != nil {
		t.Fatal(err)
	}
	if release.Version != "2026-07-01" || len(release.Foods) != 2 {
		t.Fatalf("unexpected release %s with %d foods", release.Version, len(release.Foods))
	}
	p := release.Foods[1]
	if p.Number != 6201 || p.Name != "Pesto hemlagad" || p.Group != "Såser" || p.Kcal != 545 || p.Fat != 55 || p.Carbs != 1.9 {
		t.Fatalf("unexpected food %+v", p)
	}
	// Empty cells are unknown, not zero.
	if p.SaturatedFat != nil || *p.Salt != 1.6 {
		t.Fatalf("unexpected optional values %v %v", p.SaturatedFat, p.Salt)
	}
	if _, ok := p.Nutrients["Järn, Fe (mg)"]; ok {
		t.Fatal("an empty cell should be left out of the nutrients")
	}
	if release.Foods[0].Nutrients["Energi (kJ)"] != 543 {
		t.Fatalf("every column should be kept, got %v", release.Foods[0].Nutrients)
	}
}

func TestParseRefusesAnotherFormat(t *testing.T) {
	saved := header[6]
	header[6] = "Proteiner"
	broken := workbook(t, "2026-07-01", pasta)
	header[6] = saved
	if _, err := lmv.Parse(broken); err == nil || !strings.Contains(err.Error(), "Protein (g)") {
		t.Fatalf("expected a missing column, got %v", err)
	}
}

func TestImportOnlyWhatChanged(t *testing.T) {
	d := dbtest.New(t)
	ctx := context.Background()

	first := workbook(t, "2026-01-15", pasta, pesto, talg)
	result, err := lmv.Import(ctx, d.App, first)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Imported || result.Foods != 3 || result.Version != "2026-01-15" {
		t.Fatalf("unexpected first import %+v", result)
	}

	// The same file again changes nothing.
	if result, err = lmv.Import(ctx, d.App, first); err != nil || result.Imported {
		t.Fatalf("a file imported before should be skipped, got %+v %v", result, err)
	}

	// The next release revises pesto and drops tallow.
	revised := append([]string(nil), pesto...)
	revised[3] = "530"
	if result, err = lmv.Import(ctx, d.App, workbook(t, "2026-07-01", pasta, revised)); err != nil {
		t.Fatal(err)
	}
	if !result.Imported || result.Retired != 1 {
		t.Fatalf("unexpected second import %+v", result)
	}

	var kcal float64
	var retired bool
	if err := d.App.QueryRow(ctx, `SELECT kcal FROM lmv.foods WHERE number = 6201`).Scan(&kcal); err != nil || kcal != 530 {
		t.Fatalf("pesto should be revised, got %v %v", kcal, err)
	}
	if err := d.App.QueryRow(ctx, `SELECT retired FROM lmv.foods WHERE number = 1`).Scan(&retired); err != nil || !retired {
		t.Fatalf("tallow should be retired, got %v %v", retired, err)
	}
	var releases int
	if err := d.App.QueryRow(ctx, `SELECT count(*) FROM lmv.releases`).Scan(&releases); err != nil || releases != 2 {
		t.Fatalf("expected two releases, got %d %v", releases, err)
	}
}

func TestSyncDownloadsWithAnEmptyPost(t *testing.T) {
	d := dbtest.New(t)
	file := workbook(t, "2026-07-01", pasta)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The real endpoint answers 411 without a Content-Length.
		if r.Method != http.MethodPost || r.Header.Get("Content-Length") != "0" {
			http.Error(w, "length required", http.StatusLengthRequired)
			return
		}
		_, _ = w.Write(file)
	}))
	defer source.Close()

	syncer := &lmv.Syncer{Pool: d.App, URL: source.URL, Logger: slog.New(slog.DiscardHandler)}
	job := syncer.Job("@monthly")
	if err := job.AtStart(t.Context()); err != nil {
		t.Fatal(err)
	}
	var foods int
	if err := d.App.QueryRow(t.Context(), `SELECT count(*) FROM lmv.foods`).Scan(&foods); err != nil || foods != 1 {
		t.Fatalf("the first start should import, got %d %v", foods, err)
	}

	// With a release in place, a start leaves the source alone.
	source.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("a start with foods should not download")
		w.WriteHeader(http.StatusInternalServerError)
	})
	if err := job.AtStart(t.Context()); err != nil {
		t.Fatal(err)
	}
}
