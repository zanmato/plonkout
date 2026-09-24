package exercise_test

import (
	"encoding/json"
	"net/http"
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
