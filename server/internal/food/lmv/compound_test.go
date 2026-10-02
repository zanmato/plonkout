package lmv_test

import (
	"slices"
	"testing"

	"github.com/zanmato/plonkout/server/internal/food/lmv"
)

func TestVocabularyLearnsWordsStandingAlone(t *testing.T) {
	v := lmv.NewVocabulary([]string{
		"Kyckling färs stekt", "Kyckling lever rå", "Nöt färs rå", "Lax filé rå", "Torsk filé ugnsbakad",
		"Korv falukorv", "Korv grillkorv", "Ägg kokt", "Ägg stekt",
	})
	for _, word := range []string{"kyckling", "färs", "filé", "korv", "ägg", "stekt"} {
		if !v[word] {
			t.Errorf("%s stands alone in two names and should be known", word)
		}
	}
	// In one name only, or shorter than three letters.
	for _, word := range []string{"lax", "falukorv", "rå"} {
		if v[word] {
			t.Errorf("%s should not be known", word)
		}
	}
}

func TestCompounds(t *testing.T) {
	v := lmv.VocabularyOf([]string{"kyckling", "färs", "filé", "korv", "havregryn", "gröt", "vit", "kål", "sallad", "olja"})

	cases := []struct {
		word  string
		split []string
		head  string
	}{
		{"kycklingfärs", []string{"kyckling", "färs"}, "färs"},
		// The linking s.
		{"havregrynsgröt", []string{"havregryn", "gröt"}, "gröt"},
		{"vitkålssallad", []string{"vit", "kål", "sallad"}, "sallad"},
		// Not two known words, but a falukorv is a korv.
		{"falukorv", nil, "korv"},
		{"rapsolja", nil, "olja"},
		{"kyckling", nil, ""},
		{"potatis", nil, ""},
	}
	for _, c := range cases {
		if got := v.Split(c.word); !slices.Equal(got, c.split) {
			t.Errorf("Split(%s) = %v, want %v", c.word, got, c.split)
		}
		if got := v.Head(c.word); got != c.head {
			t.Errorf("Head(%s) = %q, want %q", c.word, got, c.head)
		}
	}

	if got := v.Terms("Kyckling färs stekt"); got != "" {
		t.Errorf("a name without compounds has no terms, got %q", got)
	}
	// Parts already in the name are not repeated.
	if got := v.Terms("Korv falukorv kött 58%"); got != "" {
		t.Errorf("korv is in the name already, got %q", got)
	}
	if got := v.Terms("Kycklingfärs stekt m. rapsolja"); got != "kyckling färs olja" {
		t.Errorf("unexpected terms %q", got)
	}
}

func TestQuery(t *testing.T) {
	v := lmv.VocabularyOf([]string{"kyckling", "filé"})
	cases := map[string]string{
		"kycklingfilé":     "(kycklingfilé:* | (kyckling & filé))",
		"kokta potatisar":  "kokta & potatisar:*",
		"Pasta & kokt | !": "pasta & kokt:*",
		"%":                "",
	}
	for search, want := range cases {
		if got := v.Query(search); got != want {
			t.Errorf("Query(%q) = %q, want %q", search, got, want)
		}
	}
}
