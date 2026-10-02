package food_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/zanmato/plonkout/server/internal/food"
	"github.com/zanmato/plonkout/server/internal/platform/apitest"
)

func ptr[T any](v T) *T { return &v }

// seedLMV stores a few of Livsmedelsverket's foods, with the search terms
// and vocabulary an import would derive.
func seedLMV(t *testing.T, h *apitest.Harness) {
	t.Helper()
	_, err := h.DB.Owner.Exec(t.Context(), `
		INSERT INTO lmv.foods (number, name, food_group, kcal, protein, carbs, fat, fiber, salt, search_terms) VALUES
			(4064, 'Pasta okokt', 'Pasta, ris', 358, 11.9, 71.5, 1.3, 3, 0, ''),
			(4065, 'Pasta kokt u. salt', 'Pasta, ris', 128, 4.2, 25.8, 0.5, 1.6, 0, ''),
			(6201, 'Pesto hemlagad', 'Såser', 545, 10.6, 1.9, 55, 1.5, 1.6, ''),
			(1700, 'Korv falukorv kött 58%', 'Korv', 232, 10.1, 6.4, 19, NULL, 2, ''),
			(1500, 'Potatis kokt u. salt', 'Potatis', 77, 1.9, 15.6, 0.1, 1.5, 0, ''),
			(1210, 'Kyckling bröstfilé rå u. skinn', 'Fågel', 106, 23.1, 0, 1.4, 0, 0.2, 'filé'),
			(1211, 'Kyckling lever rå', 'Fågel', 111, 17, 1, 4.8, 0, 0.2, '');
		INSERT INTO lmv.releases (version, sha256, foods, vocabulary)
			VALUES ('2026-07-01', '\x01', 7, '{filé,korv,kyckling,pasta,potatis}')`)
	if err != nil {
		t.Fatal(err)
	}
}

func search(t *testing.T, h *apitest.Harness, u apitest.User, query string) food.SearchResult {
	t.Helper()
	var result food.SearchResult
	h.Expect(h.Do(http.MethodGet, "/foods/search?q="+url.QueryEscape(query), nil, u.Session), http.StatusOK).Decode(t, &result)
	return result
}

func TestSearchFindsOwnAndLivsmedelsverketFoods(t *testing.T) {
	h := apitest.New(t)
	seedLMV(t, h)
	u := h.NewUser()

	var olkorv food.Food
	h.Expect(h.Do(http.MethodPost, "/foods", food.FoodInput{
		Name: "Ölkorv", Brand: "Delikatess", GTIN: ptr("7310500000017"),
		Per100g:  food.Nutrients{Kcal: 298, Protein: 14, Carbs: 2, Fat: 26, Salt: ptr(2.5)},
		Portions: []food.PortionInput{{Name: "slice", Grams: 10}},
	}, u.Session), http.StatusCreated).Decode(t, &olkorv)
	if len(olkorv.Portions) != 1 || olkorv.Portions[0].Grams != 10 {
		t.Fatalf("unexpected portions %+v", olkorv.Portions)
	}

	result := search(t, h, u, "korv")
	if result.Source.Version != "2026-07-01" || result.Source.License != "CC BY 4.0" ||
		result.Source.Attribution != "Livsmedelsverkets livsmedelsdatabas version 2026-07-01, CC BY 4.0" {
		t.Fatalf("unexpected source %+v", result.Source)
	}
	if len(result.Matches) != 2 {
		t.Fatalf("expected the user's sausage and Livsmedelsverket's, got %+v", result.Matches)
	}
	if m := result.Matches[1]; m.Source != food.SourceLMV || *m.LMVNumber != 1700 || m.Per100g.Fiber != nil || m.FoodID != nil {
		t.Fatalf("unexpected lmv match %+v", m)
	}
	if m := result.Matches[0]; m.Source != food.SourceMine || *m.FoodID != olkorv.ID || len(m.Portions) != 1 {
		t.Fatalf("unexpected own match %+v", m)
	}

	// A name starting with the query comes before one merely containing it.
	result = search(t, h, u, "pasta")
	if len(result.Matches) != 2 || result.Matches[0].Name != "Pasta kokt u. salt" && result.Matches[0].Name != "Pasta okokt" {
		t.Fatalf("unexpected pasta matches %+v", result.Matches)
	}
	// Every word, in any order.
	if result = search(t, h, u, "kokt pasta"); result.Matches[0].Name != "Pasta kokt u. salt" {
		t.Fatalf("unexpected matches for words out of order %+v", result.Matches)
	}
	// By barcode, and the LIKE wildcards are taken literally.
	if result = search(t, h, u, "7310500000017"); len(result.Matches) != 1 || result.Matches[0].Name != "Ölkorv" {
		t.Fatalf("unexpected barcode matches %+v", result.Matches)
	}
	if result = search(t, h, u, "_"); len(result.Matches) != 0 {
		t.Fatalf("a wildcard should match nothing, got %+v", result.Matches)
	}
	if result = search(t, h, u, "58%"); len(result.Matches) != 1 {
		t.Fatalf("a percent sign should match itself, got %+v", result.Matches)
	}

	// Another user's food is invisible.
	if result = search(t, h, h.NewUser(), "ölkorv"); len(result.Matches) != 0 {
		t.Fatalf("another user sees %+v", result.Matches)
	}
}

func TestFoodsAndPortions(t *testing.T) {
	h := apitest.New(t)
	seedLMV(t, h)
	u := h.NewUser()
	in := food.FoodInput{Name: "Ölkorv", Per100g: food.Nutrients{Kcal: 298, Protein: 14, Carbs: 2, Fat: 26}}

	var created food.Food
	h.Expect(h.Do(http.MethodPost, "/foods", in, u.Session), http.StatusCreated).Decode(t, &created)
	in.Name = "ölkorv"
	if r := h.Expect(h.Do(http.MethodPost, "/foods", in, u.Session), http.StatusConflict); r.Code(t) != "food_exists" {
		t.Fatalf("unexpected problem %s", r.Body)
	}

	in.Per100g.Kcal = 305
	in.Portions = []food.PortionInput{{Name: "pkg", Grams: 400}}
	var updated food.Food
	h.Expect(h.Do(http.MethodPut, "/foods/"+created.ID.String(), in, u.Session), http.StatusOK).Decode(t, &updated)
	if updated.Per100g.Kcal != 305 || len(updated.Portions) != 1 {
		t.Fatalf("unexpected update %+v", updated)
	}

	// A portion of one of Livsmedelsverket's foods, saved twice by name.
	for _, grams := range []float64{14, 15} {
		h.Expect(h.Do(http.MethodPost, "/food-portions", map[string]any{"lmvNumber": 6201, "name": "tbsp", "grams": grams}, u.Session), http.StatusOK)
	}
	result := search(t, h, u, "pesto")
	if p := result.Matches[0].Portions; len(p) != 1 || p[0].Grams != 15 {
		t.Fatalf("unexpected pesto portions %+v", p)
	}
	r := h.Expect(h.Do(http.MethodPost, "/food-portions", map[string]any{"lmvNumber": 99999, "name": "tbsp", "grams": 15}, u.Session), http.StatusUnprocessableEntity)
	if r.Code(t) != "unknown_food" {
		t.Fatalf("unexpected problem %s", r.Body)
	}

	h.Expect(h.Do(http.MethodDelete, "/foods/"+created.ID.String(), nil, u.Session), http.StatusNoContent)
	var foods []food.Food
	h.Expect(h.Do(http.MethodGet, "/foods", nil, u.Session), http.StatusOK).Decode(t, &foods)
	if len(foods) != 0 {
		t.Fatalf("expected no foods, got %+v", foods)
	}
}

func names(matches []food.Match) []string {
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m.Name
	}
	return out
}

func TestSearchUnderstandsSwedish(t *testing.T) {
	h := apitest.New(t)
	seedLMV(t, h)
	u := h.NewUser()

	// Inflected forms meet the database's base forms.
	if got := names(search(t, h, u, "kokta potatisar").Matches); len(got) == 0 || got[0] != "Potatis kokt u. salt" {
		t.Fatalf("unexpected matches %v", got)
	}
	// A compound finds its parts.
	if got := names(search(t, h, u, "kycklingfilé").Matches); len(got) == 0 || got[0] != "Kyckling bröstfilé rå u. skinn" {
		t.Fatalf("unexpected matches %v", got)
	}
	// Search as you type.
	if got := names(search(t, h, u, "kyckl").Matches); len(got) != 2 {
		t.Fatalf("unexpected matches %v", got)
	}

	// A food of the user's own is split with the same words.
	h.Expect(h.Do(http.MethodPost, "/foods", food.FoodInput{
		Name: "Kycklingkorv", Brand: "Scan", Per100g: food.Nutrients{Kcal: 180, Protein: 13, Carbs: 3, Fat: 13},
	}, u.Session), http.StatusCreated)
	if got := names(search(t, h, u, "korv").Matches); len(got) != 2 || got[0] != "Kycklingkorv" {
		t.Fatalf("expected the user's sausage first, got %v", got)
	}
}

func TestSearchRemembersTheUser(t *testing.T) {
	h := apitest.New(t)
	seedLMV(t, h)
	u := h.NewUser()

	if got := names(search(t, h, u, "pasta").Matches); got[0] != "Pasta okokt" {
		t.Fatalf("unexpected order before any logging %v", got)
	}

	// Logging cooked pasta twice puts it first.
	for range 2 {
		h.Expect(h.Do(http.MethodPost, "/diary/2026-10-02/entries", map[string]any{"entries": []any{
			map[string]any{"meal": "lunch", "lmvNumber": 4065, "grams": 100},
		}}, u.Session), http.StatusCreated)
	}
	result := search(t, h, u, "pasta")
	if result.Matches[0].Name != "Pasta kokt u. salt" || result.Matches[0].Uses != 2 {
		t.Fatalf("expected the logged pasta first, got %+v", result.Matches[0])
	}

	// What the user called a food finds it first from then on.
	if got := names(search(t, h, u, "ölkorv").Matches); len(got) != 0 {
		t.Fatalf("nothing should be called ölkorv yet, got %v", got)
	}
	h.Expect(h.Do(http.MethodPost, "/diary/2026-10-02/entries", map[string]any{"entries": []any{
		map[string]any{"meal": "lunch", "lmvNumber": 1700, "grams": 70, "alias": "Ölkorv"},
	}}, u.Session), http.StatusCreated)
	if got := names(search(t, h, u, "ölkorv").Matches); len(got) != 1 || got[0] != "Korv falukorv kött 58%" {
		t.Fatalf("expected the aliased sausage, got %v", got)
	}

	// Another user's history and aliases do not count.
	other := h.NewUser()
	if got := names(search(t, h, other, "ölkorv").Matches); len(got) != 0 {
		t.Fatalf("another user finds %v", got)
	}
	if result := search(t, h, other, "pasta"); result.Matches[0].Uses != 0 {
		t.Fatalf("another user sees uses %+v", result.Matches[0])
	}
}
