package exercise_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/zanmato/plonkout/server/internal/exercise"
	"github.com/zanmato/plonkout/server/internal/platform/apitest"
)

func TestRenameCarriesHistoryAlong(t *testing.T) {
	h := apitest.New(t)
	u := h.NewUser()

	var created exercise.Exercise
	h.Expect(h.Do(http.MethodPost, "/exercises", exercise.ExerciseInput{
		Name: "Hantel curl", MuscleGroup: "Biceps", Type: "strength", DisplayType: "reps",
	}, u.Session), http.StatusCreated).Decode(t, &created)

	h.Expect(h.Do(http.MethodPost, "/workouts", map[string]any{
		"name": "Armar", "started": time.Now(), "ended": nil, "notes": "",
		"exercises": []map[string]any{{
			"name": "Hantel curl", "muscleGroup": "Biceps", "type": "strength", "displayType": "reps",
			"singleArm": false, "intensity": nil, "sets": []any{},
		}},
	}, u.Session), http.StatusCreated)
	h.Expect(h.Do(http.MethodPut, "/settings/blockPeriodization_exercises", map[string]any{
		"value": map[string]any{"Hantel curl": map[string]any{"blockWorkoutCount": 3}},
	}, u.Session), http.StatusNoContent)

	h.Expect(h.Do(http.MethodPut, "/exercises/"+created.ID.String(), exercise.ExerciseInput{
		Name: "Dumbbell Curl", MuscleGroup: "Biceps", Type: "strength", DisplayType: "reps",
	}, u.Session), http.StatusOK)

	var workouts []struct {
		Exercises []struct{ Name string } `json:"exercises"`
	}
	h.Expect(h.Do(http.MethodGet, "/workouts", nil, u.Session), http.StatusOK).Decode(t, &workouts)
	if workouts[0].Exercises[0].Name != "Dumbbell Curl" {
		t.Fatalf("the logged exercise kept its old name %q", workouts[0].Exercises[0].Name)
	}

	var settings map[string]json.RawMessage
	h.Expect(h.Do(http.MethodGet, "/settings", nil, u.Session), http.StatusOK).Decode(t, &settings)
	var blocks map[string]map[string]int
	if err := json.Unmarshal(settings["blockPeriodization_exercises"], &blocks); err != nil {
		t.Fatal(err)
	}
	if blocks["Dumbbell Curl"]["blockWorkoutCount"] != 3 || blocks["Hantel curl"] != nil {
		t.Fatalf("block state was not renamed: %v", blocks)
	}
}

func TestExerciseNamesAreUniquePerUser(t *testing.T) {
	h := apitest.New(t)
	alice, bob := h.NewUser(), h.NewUser()
	in := exercise.ExerciseInput{Name: "Side Pressure", MuscleGroup: "Shoulders", Type: "strength", DisplayType: "reps", SingleArm: true}

	h.Expect(h.Do(http.MethodPost, "/exercises", in, alice.Session), http.StatusCreated)
	in.Name = "side pressure"
	r := h.Expect(h.Do(http.MethodPost, "/exercises", in, alice.Session), http.StatusConflict)
	if r.Code(t) != "exercise_exists" {
		t.Fatalf("unexpected problem %s", r.Body)
	}
	// Another user may use the same name.
	h.Expect(h.Do(http.MethodPost, "/exercises", in, bob.Session), http.StatusCreated)
}

func TestExerciseStats(t *testing.T) {
	h := apitest.New(t)
	u := h.NewUser()
	h.Expect(h.Do(http.MethodPost, "/exercises", exercise.ExerciseInput{
		Name: "Side Pressure", MuscleGroup: "Shoulders", Type: "strength", DisplayType: "reps", SingleArm: true,
	}, u.Session), http.StatusCreated)

	set := func(kind string, weight float64, reps int, arm string) map[string]any {
		return map[string]any{"type": kind, "weight": weight, "distance": nil, "reps": reps, "time": "", "rpe": nil, "arm": arm, "notes": ""}
	}
	log := func(day int, sets ...map[string]any) {
		h.Expect(h.Do(http.MethodPost, "/workouts", map[string]any{
			"name": "Arms", "started": time.Date(2026, 9, day, 18, 0, 0, 0, time.UTC), "ended": nil, "notes": "",
			"exercises": []map[string]any{{
				"name": "Side Pressure", "muscleGroup": "Shoulders", "type": "strength", "displayType": "reps",
				"singleArm": true, "intensity": nil, "sets": sets,
			}},
		}, u.Session), http.StatusCreated)
	}
	log(1, set("warmup", 30, 10, "right"), set("regular", 21, 5, "right"), set("regular", 17.5, 5, "left"))
	log(9, set("regular", 22.5, 4, "right"), set("regular", 23, 5, "left"))

	var stats exercise.Stats
	h.Expect(h.Do(http.MethodGet, "/exercises/stats?name=side%20pressure&arm=right", nil, u.Session), http.StatusOK).Decode(t, &stats)
	if stats.Exercise != "Side Pressure" || stats.Workouts != 2 {
		t.Fatalf("unexpected stats %+v", stats)
	}
	// The warmup is heavier but does not count, and the left arm is filtered out.
	if stats.Heaviest.Weight != 22.5 || stats.BestEstimated1RM.Estimated1RM != 25.5 {
		t.Fatalf("unexpected records %+v %+v", stats.Heaviest, stats.BestEstimated1RM)
	}
	if len(stats.BestByReps) != 2 || stats.BestByReps[0].Reps != 4 || len(stats.Weekly) != 2 {
		t.Fatalf("unexpected rep bests %+v and weeks %+v", stats.BestByReps, stats.Weekly)
	}

	h.Expect(h.Do(http.MethodGet, "/exercises/stats?name=side%20pressure", nil, u.Session), http.StatusOK).Decode(t, &stats)
	if stats.Heaviest.Weight != 23 {
		t.Fatalf("both arms count without an arm filter, got %v", stats.Heaviest.Weight)
	}

	r := h.Expect(h.Do(http.MethodGet, "/exercises/stats?name=side%20presure", nil, u.Session), http.StatusUnprocessableEntity)
	if r.Code(t) != "unknown_exercise" || !strings.Contains(string(r.Body), "Side Pressure") {
		t.Fatalf("expected a suggestion, got %s", r.Body)
	}
}
