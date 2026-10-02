package diary_test

import (
	"net/http"
	"testing"

	"github.com/zanmato/plonkout/server/internal/diary"
	"github.com/zanmato/plonkout/server/internal/food"
	"github.com/zanmato/plonkout/server/internal/platform/apitest"
)

func ptr[T any](v T) *T { return &v }

func seedLMV(t *testing.T, h *apitest.Harness) {
	t.Helper()
	_, err := h.DB.Owner.Exec(t.Context(), `
		INSERT INTO lmv.foods (number, name, food_group, kcal, protein, carbs, fat, fiber, salt) VALUES
			(4065, 'Pasta kokt u. salt', 'Pasta, ris', 128, 4.2, 25.8, 0.5, 1.6, 0),
			(6201, 'Pesto hemlagad', 'Såser', 545, 10.6, 1.9, 55, 1.5, 1.6)`)
	if err != nil {
		t.Fatal(err)
	}
}

func goal() diary.GoalInput {
	return diary.GoalInput{
		Direction: "lose", Kcal: 1900, Protein: 97, Carbs: 242, Fat: 65, AddActivities: true,
		TargetWeight: ptr(88.0), WeeklyChange: ptr(-0.5), Notes: "Maintenance about 2450, minus 550.",
	}
}

func TestADayOfEating(t *testing.T) {
	h := apitest.New(t)
	seedLMV(t, h)
	u := h.NewUser()

	h.Expect(h.Do(http.MethodPut, "/nutrition-goal", goal(), u.Session), http.StatusOK)

	var olkorv food.Food
	h.Expect(h.Do(http.MethodPost, "/foods", food.FoodInput{
		Name: "Ölkorv", Per100g: food.Nutrients{Kcal: 298, Protein: 14, Carbs: 2, Fat: 26},
	}, u.Session), http.StatusCreated).Decode(t, &olkorv)

	var day diary.Day
	h.Expect(h.Do(http.MethodPost, "/diary/2026-10-02/entries", map[string]any{"entries": []diary.EntryInput{
		{Meal: "lunch", LMVNumber: ptr(int32(4065)), Grams: 100},
		{Meal: "lunch", FoodID: &olkorv.ID, Grams: 70},
		{Meal: "lunch", LMVNumber: ptr(int32(6201)), Grams: 30, Amount: "2 tbsp"},
		{Meal: "snack", Name: "Kanelbulle från caféet", Per100g: &food.Nutrients{Kcal: 350, Protein: 6, Carbs: 50, Fat: 14}, Grams: 90},
	}}, u.Session), http.StatusCreated).Decode(t, &day)

	if len(day.Meals) != 4 || day.Meals[1].Meal != "lunch" || len(day.Meals[1].Entries) != 3 {
		t.Fatalf("unexpected meals %+v", day.Meals)
	}
	lunch := day.Meals[1]
	// 128 + 208.6 + 163.5
	if lunch.Totals.Kcal != 500.1 || *lunch.Aim != (diary.Range{Min: 570, Max: 665}) {
		t.Fatalf("unexpected lunch totals %+v aim %+v", lunch.Totals, lunch.Aim)
	}
	pesto := lunch.Entries[2]
	if pesto.Name != "Pesto hemlagad" || pesto.Amount != "2 tbsp" || pesto.Fat != 16.5 || pesto.LoggedBy != "app" {
		t.Fatalf("unexpected pesto %+v", pesto)
	}
	if snack := day.Meals[3].Entries[0]; snack.FoodID != nil || snack.LMVNumber != nil || snack.Kcal != 315 {
		t.Fatalf("unexpected one off %+v", snack)
	}
	if day.Eaten.Kcal != 815.1 || *day.Budget != 1900 || *day.Remaining != 1084.9 {
		t.Fatalf("unexpected totals %+v budget %v remaining %v", day.Eaten, *day.Budget, *day.Remaining)
	}

	// Burned energy is added to the budget, since the goal says so.
	h.Expect(h.Do(http.MethodPost, "/diary/2026-10-02/activities", diary.ActivityInput{Label: "Armwrestling", Kcal: 180}, u.Session), http.StatusCreated)
	h.Expect(h.Do(http.MethodPut, "/diary/2026-10-02/weight", map[string]any{"weight": 91.4}, u.Session), http.StatusNoContent)
	h.Expect(h.Do(http.MethodGet, "/diary/2026-10-02", nil, u.Session), http.StatusOK).Decode(t, &day)
	if day.Burned != 180 || *day.Budget != 2080 || *day.Remaining != 1264.9 || *day.Weight != 91.4 {
		t.Fatalf("unexpected day %+v", day)
	}

	// Changing the food later leaves the logged day as it was.
	h.Expect(h.Do(http.MethodPut, "/foods/"+olkorv.ID.String(), food.FoodInput{
		Name: "Ölkorv", Per100g: food.Nutrients{Kcal: 400, Protein: 14, Carbs: 2, Fat: 36},
	}, u.Session), http.StatusOK)
	h.Expect(h.Do(http.MethodGet, "/diary/2026-10-02", nil, u.Session), http.StatusOK).Decode(t, &day)
	if day.Meals[1].Entries[1].Kcal != 208.6 {
		t.Fatalf("the logged sausage changed to %v", day.Meals[1].Entries[1].Kcal)
	}

	// Moving the pasta to dinner with a bigger portion.
	var moved diary.Entry
	h.Expect(h.Do(http.MethodPut, "/food-entries/"+lunch.Entries[0].ID.String(), diary.EntryUpdate{
		Day: "2026-10-02", Meal: "dinner", Grams: 200,
	}, u.Session), http.StatusOK).Decode(t, &moved)
	if moved.Meal != "dinner" || moved.Kcal != 256 {
		t.Fatalf("unexpected moved entry %+v", moved)
	}
	h.Expect(h.Do(http.MethodDelete, "/food-entries/"+lunch.Entries[2].ID.String(), nil, u.Session), http.StatusNoContent)

	var summary diary.Summary
	h.Expect(h.Do(http.MethodGet, "/diary?from=2026-10-01&to=2026-10-03", nil, u.Session), http.StatusOK).Decode(t, &summary)
	if len(summary.Days) != 3 || summary.Days[0].Entries != 0 || summary.Days[1].Entries != 3 ||
		summary.Days[1].Eaten.Kcal != 779.6 || summary.Days[1].Burned != 180 || summary.Goal.Kcal != 1900 {
		t.Fatalf("unexpected summary %+v", summary)
	}

	// Another user sees none of it.
	other := h.NewUser()
	h.Expect(h.Do(http.MethodGet, "/diary/2026-10-02", nil, other.Session), http.StatusOK).Decode(t, &day)
	if day.Eaten.Kcal != 0 || day.Goal != nil || day.Budget != nil || len(day.Activities) != 0 {
		t.Fatalf("another user sees %+v", day)
	}
	h.Expect(h.Do(http.MethodDelete, "/food-entries/"+moved.ID.String(), nil, other.Session), http.StatusNotFound)
}

func TestLoggingIsAllOrNothing(t *testing.T) {
	h := apitest.New(t)
	seedLMV(t, h)
	u := h.NewUser()

	cases := []struct {
		entry diary.EntryInput
		code  string
	}{
		{diary.EntryInput{Meal: "lunch", LMVNumber: ptr(int32(99999)), Grams: 100}, "unknown_food"},
		{diary.EntryInput{Meal: "lunch", Name: "Something", Grams: 100}, "unknown_food"},
		{diary.EntryInput{Meal: "lunch", LMVNumber: ptr(int32(4065)), Per100g: &food.Nutrients{Kcal: 1}, Grams: 100}, "ambiguous_food"},
	}
	for _, c := range cases {
		r := h.Expect(h.Do(http.MethodPost, "/diary/2026-10-02/entries", map[string]any{"entries": []diary.EntryInput{
			{Meal: "breakfast", LMVNumber: ptr(int32(4065)), Grams: 50}, c.entry,
		}}, u.Session), http.StatusUnprocessableEntity)
		if r.Code(t) != c.code {
			t.Errorf("expected %s, got %s", c.code, r.Body)
		}
	}
	var day diary.Day
	h.Expect(h.Do(http.MethodGet, "/diary/2026-10-02", nil, u.Session), http.StatusOK).Decode(t, &day)
	if day.Eaten.Kcal != 0 {
		t.Fatalf("a refused log stored %v kcal", day.Eaten.Kcal)
	}

	if r := h.Expect(h.Do(http.MethodGet, "/diary/2026-13-40", nil, u.Session), http.StatusUnprocessableEntity); r.Code(t) == "" {
		t.Fatalf("expected a problem, got %s", r.Body)
	}
	h.Expect(h.Do(http.MethodGet, "/diary?from=2026-10-03&to=2026-10-01", nil, u.Session), http.StatusUnprocessableEntity)
	h.Expect(h.Do(http.MethodGet, "/nutrition-goal", nil, u.Session), http.StatusNotFound)
}
