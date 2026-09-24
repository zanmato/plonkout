package plan_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/plan"
	"github.com/zanmato/plonkout/server/internal/platform/apitest"
	"github.com/zanmato/plonkout/server/internal/workout"
)

func ptr[T any](v T) *T { return &v }

// armwrestling is two weeks of the spreadsheet plan this feature was built
// for: a heavy A day and a volume B day, bench and side pressure.
func armwrestling() plan.PlanInput {
	session := func(label string, week int32, day, intensity string, bench, side []plan.TargetInput) plan.SessionInput {
		return plan.SessionInput{
			Label: label, Week: &week, Day: &day, Intensity: &intensity,
			Exercises: []plan.PlannedExerciseInput{
				{Exercise: "Bench Press", Targets: bench},
				{Exercise: "side pressure", OffArmPercent: ptr(82.5), Notes: "Elbow pinned, wrist locked", Targets: side},
			},
		}
	}
	return plan.PlanInput{
		Name: "Armwrestling 9 weeks", Goal: "Bench 180, side pressure 30",
		Sessions: []plan.SessionInput{
			session("W1 A", 1, "A", "heavy",
				[]plan.TargetInput{
					{SetType: "Top set", Sets: 1, Reps: ptr(int32(5)), Weight: ptr(145.0), RPEMin: ptr(8.0), RPEMax: ptr(9.0)},
					{SetType: "Back-off", Sets: 3, Reps: ptr(int32(5)), Weight: ptr(130.0)},
				},
				[]plan.TargetInput{{SetType: "Top set", Sets: 1, Reps: ptr(int32(5)), Weight: ptr(21.25)}}),
			session("W1 B", 1, "B", "light",
				[]plan.TargetInput{{SetType: "Volume (paused)", Sets: 4, Reps: ptr(int32(6)), Weight: ptr(125.0)}},
				[]plan.TargetInput{{SetType: "Volume (3s hold)", Sets: 4, Reps: ptr(int32(8)), Weight: ptr(16.25)}}),
			session("W2 A", 2, "A", "heavy",
				[]plan.TargetInput{{SetType: "Top set", Sets: 1, Reps: ptr(int32(4)), Weight: ptr(150.0)}},
				[]plan.TargetInput{{SetType: "Top set", Sets: 1, Reps: ptr(int32(4)), Weight: ptr(22.5)}}),
		},
	}
}

func newUserWithExercises(t *testing.T, h *apitest.Harness) apitest.User {
	t.Helper()
	u := h.NewUser()
	for _, e := range []map[string]any{
		{"name": "Bench Press", "muscleGroup": "Chest", "type": "strength", "displayType": "reps", "singleArm": false},
		{"name": "Side Pressure", "muscleGroup": "Shoulders", "type": "strength", "displayType": "reps", "singleArm": true},
	} {
		h.Expect(h.Do(http.MethodPost, "/exercises", e, u.Session), http.StatusCreated)
	}
	return u
}

func createPlan(t *testing.T, h *apitest.Harness, u apitest.User) plan.Plan {
	t.Helper()
	var p plan.Plan
	h.Expect(h.Do(http.MethodPost, "/plans", armwrestling(), u.Session), http.StatusCreated).Decode(t, &p)
	return p
}

func TestCreatePlanAndQueue(t *testing.T) {
	h := apitest.New(t)
	u := newUserWithExercises(t, h)
	p := createPlan(t, h, u)

	if len(p.Sessions) != 3 || p.Sessions[0].Label != "W1 A" || p.Sessions[2].Position != 2 {
		t.Fatalf("unexpected sessions %+v", p.Sessions)
	}
	side := p.Sessions[0].Exercises[1]
	// Matched ignoring case, and the canonical name comes back.
	if side.Exercise != "Side Pressure" || !side.SingleArm || *side.OffArmPercent != 82.5 {
		t.Fatalf("unexpected planned exercise %+v", side)
	}
	if bench := p.Sessions[0].Exercises[0]; len(bench.Targets) != 2 || bench.Targets[1].Sets != 3 || *bench.Targets[0].RPEMax != 9 {
		t.Fatalf("unexpected targets %+v", bench.Targets)
	}

	var queue []plan.QueueEntry
	h.Expect(h.Do(http.MethodGet, "/queue", nil, u.Session), http.StatusOK).Decode(t, &queue)
	if len(queue) != 3 || queue[0].Session.Label != "W1 A" || queue[0].PlanName != "Armwrestling 9 weeks" {
		t.Fatalf("unexpected queue %+v", queue)
	}
}

func TestUnknownExercisesAreRefusedWithSuggestions(t *testing.T) {
	h := apitest.New(t)
	u := newUserWithExercises(t, h)

	in := armwrestling()
	in.Sessions[0].Exercises[0].Exercise = "Bench Pres"
	r := h.Expect(h.Do(http.MethodPost, "/plans", in, u.Session), http.StatusUnprocessableEntity)
	if r.Code(t) != "unknown_exercise" {
		t.Fatalf("unexpected problem %s", r.Body)
	}
	var problem struct{ Detail string }
	r.Decode(t, &problem)
	if want := `"Bench Press"`; !contains(problem.Detail, want) {
		t.Fatalf("expected a suggestion of %s in %q", want, problem.Detail)
	}

	in.Sessions[0].Exercises[0].Exercise = "Pronation Curl"
	in.Sessions[0].Exercises[0].SingleArm = ptr(true)
	in.CreateMissingExercises = true
	var p plan.Plan
	h.Expect(h.Do(http.MethodPost, "/plans", in, u.Session), http.StatusCreated).Decode(t, &p)
	if e := p.Sessions[0].Exercises[0]; e.Exercise != "Pronation Curl" || !e.SingleArm {
		t.Fatalf("the missing exercise was not created as asked: %+v", e)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestStartSessionPrefillsTargetsAndCompares(t *testing.T) {
	h := apitest.New(t)
	u := newUserWithExercises(t, h)
	h.Expect(h.Do(http.MethodPut, "/settings/dominantArm", map[string]any{"value": "right"}, u.Session), http.StatusNoContent)
	p := createPlan(t, h, u)
	session := p.Sessions[0]

	var started struct{ WorkoutID uuid.UUID }
	h.Expect(h.Do(http.MethodPost, "/planned-sessions/"+session.ID.String()+"/start", nil, u.Session), http.StatusOK).Decode(t, &started)

	// A second tap answers with the same workout.
	var again struct{ WorkoutID uuid.UUID }
	h.Expect(h.Do(http.MethodPost, "/planned-sessions/"+session.ID.String()+"/start", nil, u.Session), http.StatusOK).Decode(t, &again)
	if again.WorkoutID != started.WorkoutID {
		t.Fatal("starting twice logged a second workout")
	}

	var w workout.Workout
	h.Expect(h.Do(http.MethodGet, "/workouts/"+started.WorkoutID.String(), nil, u.Session), http.StatusOK).Decode(t, &w)
	if w.Name != "Armwrestling 9 weeks W1 A" || w.PlannedSessionID == nil || *w.PlannedSessionID != session.ID {
		t.Fatalf("unexpected workout %+v", w)
	}
	bench, side := w.Exercises[0], w.Exercises[1]
	if len(bench.Sets) != 4 || *bench.Intensity != "heavy" || bench.PlannedExerciseID == nil {
		t.Fatalf("expected 1 top set and 3 back-off sets, got %+v", bench)
	}
	if bench.Sets[0].Weight != nil || bench.Sets[0].TargetID == nil || *bench.Sets[3].TargetSeq != 3 {
		t.Fatal("prescribed sets start empty and linked to their target")
	}
	if len(side.Sets) != 2 || side.Sets[0].Arm != "right" || side.Sets[1].Arm != "left" {
		t.Fatalf("a single arm target gets a set per arm, dominant first: %+v", side.Sets)
	}

	var queue []plan.QueueEntry
	h.Expect(h.Do(http.MethodGet, "/queue", nil, u.Session), http.StatusOK).Decode(t, &queue)
	if queue[0].Session.Status != plan.StatusInProgress {
		t.Fatalf("a started session is in progress, got %s", queue[0].Session.Status)
	}

	// Log it: the top set as planned, one back-off set short, and the off arm
	// at 82.5% of 21.25, rounded.
	fill := func(s *workout.WorkoutSet, weight float64, reps int32, rpe float64) {
		s.Weight, s.Reps, s.RPE = &weight, &reps, &rpe
	}
	fill(&bench.Sets[0], 145, 5, 8.5)
	fill(&bench.Sets[1], 130, 5, 7)
	fill(&bench.Sets[2], 130, 5, 7)
	fill(&bench.Sets[3], 130, 3, 9)
	fill(&side.Sets[0], 21.25, 5, 8)
	fill(&side.Sets[1], plan.OffArmWeight(21.25, 82.5), 5, 7)
	extra := int32(8)
	bench.Sets = append(bench.Sets, workout.WorkoutSet{Type: "regular", Weight: ptr(100.0), Reps: &extra})
	w.Exercises = append(w.Exercises[:2:2], workout.WorkoutExercise{
		Name: "Face Pull", MuscleGroup: "Shoulders", Type: "strength", DisplayType: "reps",
		Sets: []workout.WorkoutSet{{Type: "regular", Weight: ptr(20.0), Reps: ptr(int32(15))}},
	})
	w.Exercises[0], w.Exercises[1] = bench, side
	ended := time.Now()
	update := workout.WorkoutUpdate{WorkoutInput: workout.WorkoutInput{
		Name: w.Name, Started: w.Started, Ended: &ended, Notes: w.Notes, Exercises: w.Exercises,
	}, Revision: w.Revision}
	h.Expect(h.Do(http.MethodPut, "/workouts/"+w.ID.String(), update, u.Session), http.StatusOK)

	var withActuals plan.Plan
	h.Expect(h.Do(http.MethodGet, "/plans/"+p.ID.String()+"?actuals=true", nil, u.Session), http.StatusOK).Decode(t, &withActuals)
	done := withActuals.Sessions[0]
	if done.Status != plan.StatusCompleted || done.CompletedAt == nil {
		t.Fatalf("ending the workout completes the session, got %s", done.Status)
	}
	c := done.Comparison
	if c == nil || len(c.Exercises) != 2 || len(c.Unplanned) != 1 || c.Unplanned[0] != "Face Pull" {
		t.Fatalf("unexpected comparison %+v", c)
	}
	top, backOff := c.Exercises[0].Targets[0], c.Exercises[0].Targets[1]
	if !top.Met || top.Done != 1 || *top.TopRPE != 8.5 {
		t.Fatalf("the top set was done as planned: %+v", top)
	}
	if backOff.Met || backOff.Done != 3 {
		t.Fatalf("one back-off set fell short of 5 reps: %+v", backOff)
	}
	if len(c.Exercises[0].ExtraSets) != 1 {
		t.Fatalf("the unprescribed set is extra work: %+v", c.Exercises[0].ExtraSets)
	}
	if got := *c.Exercises[0].BestEstimated1RM; got != 169.2 {
		t.Fatalf("best Epley estimate of 145 × 5 is 169.2, got %v", got)
	}
	if sideTop := c.Exercises[1].Targets[0]; !sideTop.Met {
		t.Fatalf("the off arm at its percentage meets the target: %+v", sideTop)
	}

	var queueAfter []plan.QueueEntry
	h.Expect(h.Do(http.MethodGet, "/queue", nil, u.Session), http.StatusOK).Decode(t, &queueAfter)
	if len(queueAfter) != 2 || queueAfter[0].Session.Label != "W1 B" {
		t.Fatal("a completed session leaves the queue")
	}

	// Deleting the workout puts the session back in the queue.
	h.Expect(h.Do(http.MethodDelete, "/workouts/"+w.ID.String(), nil, u.Session), http.StatusNoContent)
	h.Expect(h.Do(http.MethodGet, "/queue", nil, u.Session), http.StatusOK).Decode(t, &queueAfter)
	if len(queueAfter) != 3 || queueAfter[0].Session.Status != plan.StatusPending {
		t.Fatal("a session whose workout was deleted is pending again")
	}
}

func TestSkipReorderAndEdit(t *testing.T) {
	h := apitest.New(t)
	u := newUserWithExercises(t, h)
	p := createPlan(t, h, u)
	a, b, c := p.Sessions[0].ID.String(), p.Sessions[1].ID.String(), p.Sessions[2].ID.String()

	h.Expect(h.Do(http.MethodPost, "/planned-sessions/"+a+"/status", map[string]any{"status": "skipped", "reason": "elbow"}, u.Session), http.StatusOK)
	r := h.Expect(h.Do(http.MethodPost, "/planned-sessions/"+a+"/start", nil, u.Session), http.StatusConflict)
	if r.Code(t) != "session_skipped" {
		t.Fatalf("unexpected problem %s", r.Body)
	}

	// Only the pending sessions are reordered, and all of them must be named.
	h.Expect(h.Do(http.MethodPut, "/plans/"+p.ID.String()+"/order", map[string]any{"sessionIds": []string{c}}, u.Session), http.StatusUnprocessableEntity)
	var reordered plan.Plan
	h.Expect(h.Do(http.MethodPut, "/plans/"+p.ID.String()+"/order", map[string]any{"sessionIds": []string{c, b}}, u.Session), http.StatusOK).Decode(t, &reordered)
	if reordered.Sessions[0].Label != "W1 A" || reordered.Sessions[1].Label != "W2 A" || reordered.Sessions[2].Label != "W1 B" {
		t.Fatalf("unexpected order %v %v %v", reordered.Sessions[0].Label, reordered.Sessions[1].Label, reordered.Sessions[2].Label)
	}

	var edited plan.Session
	h.Expect(h.Do(http.MethodPatch, "/planned-sessions/"+b, map[string]any{
		"notes": "Pause every rep",
		"exercises": []map[string]any{{
			"exercise": "Bench Press",
			"targets":  []map[string]any{{"setType": "Volume", "sets": 5, "reps": 5, "weight": 127.5}},
		}},
	}, u.Session), http.StatusOK).Decode(t, &edited)
	if edited.Notes != "Pause every rep" || edited.Label != "W1 B" || len(edited.Exercises) != 1 || edited.Exercises[0].Targets[0].Sets != 5 {
		t.Fatalf("unexpected edited session %+v", edited)
	}

	// A skipped session cannot be edited until it is put back.
	h.Expect(h.Do(http.MethodPatch, "/planned-sessions/"+a, map[string]any{"notes": "x"}, u.Session), http.StatusConflict)
	h.Expect(h.Do(http.MethodPost, "/planned-sessions/"+a+"/status", map[string]any{"status": "pending"}, u.Session), http.StatusOK)
	h.Expect(h.Do(http.MethodPatch, "/planned-sessions/"+a, map[string]any{"notes": "x"}, u.Session), http.StatusOK)

	h.Expect(h.Do(http.MethodPost, "/planned-sessions/"+c+"/status", map[string]any{"status": "deleted"}, u.Session), http.StatusOK)
	var after plan.Plan
	h.Expect(h.Do(http.MethodGet, "/plans/"+p.ID.String(), nil, u.Session), http.StatusOK).Decode(t, &after)
	if len(after.Sessions) != 2 {
		t.Fatalf("expected the deleted session gone, got %d sessions", len(after.Sessions))
	}

	var inserted plan.Plan
	h.Expect(h.Do(http.MethodPost, "/plans/"+p.ID.String()+"/sessions", map[string]any{
		"afterSessionId": a,
		"sessions": []map[string]any{{
			"label": "Deload", "exercises": []map[string]any{{
				"exercise": "Bench Press", "targets": []map[string]any{{"setType": "Deload", "sets": 3, "reps": 3, "weight": 120}},
			}},
		}},
	}, u.Session), http.StatusOK).Decode(t, &inserted)
	if inserted.Sessions[1].Label != "Deload" || len(inserted.Sessions) != 3 {
		t.Fatalf("the session was not inserted after the first: %v", inserted.Sessions)
	}
}

func TestPlansBelongToTheirUser(t *testing.T) {
	h := apitest.New(t)
	alice := newUserWithExercises(t, h)
	bob := h.NewUser()
	p := createPlan(t, h, alice)
	session := p.Sessions[0].ID.String()

	h.Expect(h.Do(http.MethodGet, "/plans/"+p.ID.String(), nil, bob.Session), http.StatusNotFound)
	h.Expect(h.Do(http.MethodPost, "/planned-sessions/"+session+"/start", nil, bob.Session), http.StatusNotFound)
	h.Expect(h.Do(http.MethodPost, "/planned-sessions/"+session+"/status", map[string]any{"status": "skipped"}, bob.Session), http.StatusNotFound)

	var queue []plan.QueueEntry
	h.Expect(h.Do(http.MethodGet, "/queue", nil, bob.Session), http.StatusOK).Decode(t, &queue)
	if len(queue) != 0 {
		t.Fatal("bob sees alice's queue")
	}
}

func TestEpleyMatchesTheWebApp(t *testing.T) {
	// The same values the web app's calculateEpley1RM is tested with.
	cases := []struct {
		weight float64
		reps   int32
		want   float64
	}{
		{100, 1, 100 * (1 + 1.0/30)},
		{150, 4, 170},
		{145, 5, 145 * (1 + 5.0/30)},
		{100, 20, 140},
		{0, 5, 0},
		{100, 0, 0},
	}
	for _, c := range cases {
		if got := plan.Epley1RM(c.weight, c.reps); got < c.want-1e-9 || got > c.want+1e-9 {
			t.Errorf("Epley1RM(%v, %v) = %v, want %v", c.weight, c.reps, got, c.want)
		}
	}
}
