package importer_test

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/zanmato/plonkout/server/internal/importer"
	"github.com/zanmato/plonkout/server/internal/platform/apitest"
	"github.com/zanmato/plonkout/server/internal/workout"
)

// testdata/legacy-export.json is cut from a real export of the local only app,
// picked for its quirks: an RPE saved as "", a workout never ended, single arm
// sets, intensities, warmups, fractional weights and a record with no created
// date.
func loadExport(t *testing.T) json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile("testdata/legacy-export.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestImportLegacyExport(t *testing.T) {
	h := apitest.New(t)
	u := h.NewUser()
	export := loadExport(t)

	var first importer.Result
	h.Expect(h.Do(http.MethodPost, "/import/legacy", export, u.Session), http.StatusOK).Decode(t, &first)
	if first.Imported != 7 || first.Skipped != 0 || first.ExercisesCreated != 8 {
		t.Fatalf("unexpected first import %+v", first)
	}

	var again importer.Result
	h.Expect(h.Do(http.MethodPost, "/import/legacy", export, u.Session), http.StatusOK).Decode(t, &again)
	if again.Imported != 0 || again.Skipped != 7 || again.ExercisesCreated != 0 {
		t.Fatalf("a second import should skip everything, got %+v", again)
	}

	var workouts []workout.Workout
	h.Expect(h.Do(http.MethodGet, "/workouts", nil, u.Session), http.StatusOK).Decode(t, &workouts)
	if len(workouts) != 7 {
		t.Fatalf("expected 7 workouts, got %d", len(workouts))
	}

	var sets, emptyRPE, unended, linked, rightArm int
	for _, w := range workouts {
		if w.Ended == nil {
			unended++
		}
		for _, e := range w.Exercises {
			if e.ExerciseID != nil {
				linked++
			}
			for _, s := range e.Sets {
				sets++
				if s.RPE == nil {
					emptyRPE++
				}
				if s.Arm == "right" {
					rightArm++
				}
			}
		}
	}
	if sets != 100 {
		t.Fatalf("expected 100 sets, got %d", sets)
	}
	if unended != 1 || emptyRPE == 0 || rightArm == 0 {
		t.Fatalf("quirks lost: unended %d, empty rpe %d, right arm %d", unended, emptyRPE, rightArm)
	}

	var exercises []struct{ ID string }
	h.Expect(h.Do(http.MethodGet, "/exercises", nil, u.Session), http.StatusOK).Decode(t, &exercises)
	if len(exercises) != 8 {
		t.Fatalf("expected 8 exercises, got %d", len(exercises))
	}
	var logged int
	for _, w := range workouts {
		logged += len(w.Exercises)
	}
	if linked != logged {
		t.Fatalf("only %d of %d logged exercises link to the exercise list", linked, logged)
	}
}

func TestImportRefusesGarbage(t *testing.T) {
	h := apitest.New(t)
	u := h.NewUser()

	r := h.Expect(h.Do(http.MethodPost, "/import/legacy", map[string]any{
		"workouts": []any{map[string]any{"id": 1, "name": "x", "started": "yesterday", "exercises": []any{}}},
	}, u.Session), http.StatusUnprocessableEntity)
	if r.Code(t) != "invalid_export" {
		t.Fatalf("unexpected problem %s", r.Body)
	}
}
