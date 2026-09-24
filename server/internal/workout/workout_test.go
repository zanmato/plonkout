package workout_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/zanmato/plonkout/server/internal/platform/apitest"
	"github.com/zanmato/plonkout/server/internal/workout"
)

func ptr[T any](v T) *T { return &v }

func benchDay(started time.Time) workout.WorkoutInput {
	return workout.WorkoutInput{
		Name:    "Bröst",
		Started: started,
		Ended:   ptr(started.Add(40 * time.Minute)),
		Notes:   "felt strong",
		Exercises: []workout.WorkoutExercise{
			{
				Name: "Bench Press", MuscleGroup: "Chest", Type: "strength", DisplayType: "reps",
				Intensity: ptr("heavy"),
				Sets: []workout.WorkoutSet{
					{Type: "warmup", Weight: ptr(60.0), Reps: ptr(int32(10)), Arm: ""},
					{Type: "regular", Weight: ptr(130.0), Reps: ptr(int32(5)), RPE: ptr(8.5), Arm: ""},
					{Type: "regular", Weight: nil, Reps: nil, Arm: ""},
				},
			},
			{
				Name: "Pronation Curl", MuscleGroup: "Forearm", Type: "strength", DisplayType: "reps", SingleArm: true,
				Sets: []workout.WorkoutSet{
					{Type: "regular", Weight: ptr(25.0), Reps: ptr(int32(5)), Arm: "right"},
					{Type: "regular", Weight: ptr(21.25), Reps: ptr(int32(5)), Arm: "left"},
				},
			},
		},
	}
}

// exercises are seeded at signup, so the test user needs the defaults.
func seedExercises(t *testing.T, h *apitest.Harness, u apitest.User) {
	t.Helper()
	for _, name := range []string{"Bench Press", "Pronation Curl"} {
		h.Expect(h.Do(http.MethodPost, "/exercises", map[string]any{
			"name": name, "muscleGroup": "Chest", "type": "strength", "displayType": "reps", "singleArm": false,
		}, u.Session), http.StatusCreated)
	}
}

func TestWorkoutRoundTrip(t *testing.T) {
	h := apitest.New(t)
	u := h.NewUser()
	seedExercises(t, h, u)

	started := time.Date(2026, 9, 21, 15, 41, 0, 0, time.UTC)
	var created workout.Workout
	h.Expect(h.Do(http.MethodPost, "/workouts", benchDay(started), u.Session), http.StatusCreated).Decode(t, &created)

	if created.Revision != 1 || len(created.Exercises) != 2 {
		t.Fatalf("unexpected workout %+v", created)
	}
	bench := created.Exercises[0]
	if bench.ExerciseID == nil {
		t.Fatal("the exercise was not linked to the exercise list by name")
	}
	if *bench.Intensity != "heavy" || len(bench.Sets) != 3 || bench.Sets[0].Type != "warmup" {
		t.Fatalf("unexpected exercise %+v", bench)
	}
	if bench.Sets[2].Weight != nil || bench.Sets[2].Reps != nil {
		t.Fatal("an empty set must stay empty")
	}
	if *created.Exercises[1].Sets[1].Weight != 21.25 || created.Exercises[1].Sets[1].Arm != "left" {
		t.Fatalf("unexpected single arm set %+v", created.Exercises[1].Sets[1])
	}

	var fetched workout.Workout
	h.Expect(h.Do(http.MethodGet, "/workouts/"+created.ID.String(), nil, u.Session), http.StatusOK).Decode(t, &fetched)
	if fetched.Name != "Bröst" || *fetched.Exercises[0].Sets[1].RPE != 8.5 || !fetched.Started.Equal(started) {
		t.Fatalf("unexpected fetched workout %+v", fetched)
	}

	update := workout.WorkoutUpdate{WorkoutInput: benchDay(started), Revision: created.Revision}
	update.Exercises = update.Exercises[:1]
	update.Exercises[0].Sets[2].Weight = ptr(130.0)
	update.Exercises[0].Sets[2].Reps = ptr(int32(4))
	var updated workout.Workout
	h.Expect(h.Do(http.MethodPut, "/workouts/"+created.ID.String(), update, u.Session), http.StatusOK).Decode(t, &updated)
	if updated.Revision != 2 || len(updated.Exercises) != 1 || *updated.Exercises[0].Sets[2].Reps != 4 {
		t.Fatalf("unexpected updated workout %+v", updated)
	}

	// The editor that still holds revision 1 must not overwrite revision 2.
	r := h.Expect(h.Do(http.MethodPut, "/workouts/"+created.ID.String(), update, u.Session), http.StatusConflict)
	if r.Code(t) != "stale_revision" {
		t.Fatalf("unexpected problem %s", r.Body)
	}

	h.Expect(h.Do(http.MethodDelete, "/workouts/"+created.ID.String(), nil, u.Session), http.StatusNoContent)
	h.Expect(h.Do(http.MethodGet, "/workouts/"+created.ID.String(), nil, u.Session), http.StatusNotFound)
}

func TestWorkoutsBelongToTheirUser(t *testing.T) {
	h := apitest.New(t)
	alice, bob := h.NewUser(), h.NewUser()

	var theirs workout.Workout
	h.Expect(h.Do(http.MethodPost, "/workouts", benchDay(time.Now()), alice.Session), http.StatusCreated).Decode(t, &theirs)

	path := "/workouts/" + theirs.ID.String()
	h.Expect(h.Do(http.MethodGet, path, nil, bob.Session), http.StatusNotFound)
	h.Expect(h.Do(http.MethodPut, path, workout.WorkoutUpdate{WorkoutInput: benchDay(time.Now()), Revision: 1}, bob.Session), http.StatusNotFound)
	h.Expect(h.Do(http.MethodDelete, path, nil, bob.Session), http.StatusNotFound)

	var list []workout.Workout
	h.Expect(h.Do(http.MethodGet, "/workouts", nil, bob.Session), http.StatusOK).Decode(t, &list)
	if len(list) != 0 {
		t.Fatalf("bob sees %d of alice's workouts", len(list))
	}
	h.Expect(h.Do(http.MethodGet, "/workouts", nil, ""), http.StatusUnauthorized)
}

func TestWorkoutValidation(t *testing.T) {
	h := apitest.New(t)
	u := h.NewUser()

	cases := map[string]func(*workout.WorkoutInput){
		"unknown intensity": func(in *workout.WorkoutInput) { in.Exercises[0].Intensity = ptr("medium") },
		"unknown arm":       func(in *workout.WorkoutInput) { in.Exercises[0].Sets[0].Arm = "middle" },
		"rpe above ten":     func(in *workout.WorkoutInput) { in.Exercises[0].Sets[1].RPE = ptr(11.0) },
		"negative weight":   func(in *workout.WorkoutInput) { in.Exercises[0].Sets[1].Weight = ptr(-5.0) },
		"nameless exercise": func(in *workout.WorkoutInput) { in.Exercises[0].Name = "" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			in := benchDay(time.Now())
			change(&in)
			h.Expect(h.Do(http.MethodPost, "/workouts", in, u.Session), http.StatusUnprocessableEntity)
		})
	}

	t.Run("no intensity", func(t *testing.T) {
		in := benchDay(time.Now())
		in.Exercises[0].Intensity = nil
		var created workout.Workout
		h.Expect(h.Do(http.MethodPost, "/workouts", in, u.Session), http.StatusCreated).Decode(t, &created)
		if created.Exercises[0].Intensity != nil {
			t.Fatal("expected no intensity")
		}
	})
}

func TestLatestWorkoutAndRange(t *testing.T) {
	h := apitest.New(t)
	u := h.NewUser()

	day := func(d int) time.Time { return time.Date(2026, 9, d, 18, 0, 0, 0, time.UTC) }
	ids := map[int]string{}
	for _, d := range []int{1, 8, 15} {
		var w workout.Workout
		h.Expect(h.Do(http.MethodPost, "/workouts", benchDay(day(d)), u.Session), http.StatusCreated).Decode(t, &w)
		ids[d] = w.ID.String()
	}

	var latest workout.Workout
	h.Expect(h.Do(http.MethodGet, "/workouts/latest?name=Bröst", nil, u.Session), http.StatusOK).Decode(t, &latest)
	if latest.ID.String() != ids[15] {
		t.Fatal("expected the newest workout with the name")
	}
	h.Expect(h.Do(http.MethodGet, "/workouts/latest?name=Bröst&excludeId="+ids[15], nil, u.Session), http.StatusOK).Decode(t, &latest)
	if latest.ID.String() != ids[8] {
		t.Fatal("expected the excluded workout to be skipped")
	}
	h.Expect(h.Do(http.MethodGet, "/workouts/latest?name=Rygg", nil, u.Session), http.StatusNotFound)

	var inRange []workout.Workout
	h.Expect(h.Do(http.MethodGet, "/workouts?from=2026-09-05T00:00:00Z&to=2026-09-15T00:00:00Z", nil, u.Session), http.StatusOK).Decode(t, &inRange)
	if len(inRange) != 1 || inRange[0].ID.String() != ids[8] {
		t.Fatalf("expected only the workout of the 8th, got %d", len(inRange))
	}

	var all []workout.Workout
	h.Expect(h.Do(http.MethodGet, "/workouts", nil, u.Session), http.StatusOK).Decode(t, &all)
	if len(all) != 3 || all[0].ID.String() != ids[15] {
		t.Fatal("expected every workout, newest first")
	}
}
