// Package importer brings data from the local only version of the app into an
// account.
package importer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/exercise/exercisedb"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/db"
	"github.com/zanmato/plonkout/server/internal/workout"
)

// Export is the file the old app's "Export data" button wrote. Workouts are
// parsed leniently one by one, because years of IndexedDB records hold every
// shape the app ever wrote: numbers as strings, empty strings for nothing,
// fields that did not exist yet.
type Export struct {
	Version    string            `json:"version,omitempty"`
	ExportDate string            `json:"exportDate,omitempty"`
	Workouts   []json.RawMessage `json:"workouts" maxItems:"20000"`
}

// Result says what an import did.
type Result struct {
	Imported         int `json:"imported"`
	Skipped          int `json:"skipped" doc:"Workouts imported before, recognized by their old id."`
	ExercisesCreated int `json:"exercisesCreated"`
}

type legacyWorkout struct {
	ID        int64            `json:"id"`
	Name      string           `json:"name"`
	Started   string           `json:"started"`
	Ended     *string          `json:"ended"`
	Notes     string           `json:"notes"`
	Created   *string          `json:"created"`
	Updated   *string          `json:"updated"`
	Exercises []legacyExercise `json:"exercises"`
}

type legacyExercise struct {
	Name        string      `json:"name"`
	MuscleGroup string      `json:"muscleGroup"`
	Type        string      `json:"type"`
	DisplayType string      `json:"displayType"`
	SingleArm   bool        `json:"singleArm"`
	Intensity   *string     `json:"intensity"`
	Sets        []legacySet `json:"sets"`
}

type legacySet struct {
	Type     string     `json:"type"`
	Weight   flexNumber `json:"weight"`
	Distance flexNumber `json:"distance"`
	Reps     flexNumber `json:"reps"`
	Time     string     `json:"time"`
	RPE      flexNumber `json:"rpe"`
	Arm      string     `json:"arm"`
	Notes    string     `json:"notes"`
}

// flexNumber accepts a number, a numeric string, an empty string or null.
type flexNumber struct {
	value *float64
}

func (f *flexNumber) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	text := string(b)
	if b[0] == '"' {
		if err := json.Unmarshal(b, &text); err != nil {
			return err
		}
		text = strings.TrimSpace(strings.TrimPrefix(strings.ToUpper(text), "RPE "))
		if text == "" {
			return nil
		}
	}
	value, err := strconv.ParseFloat(strings.ReplaceAll(text, ",", "."), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		// A value that was never a number is treated as not entered.
		return nil
	}
	f.value = &value
	return nil
}

// Service is the importer.
type Service struct {
	pool *pgxpool.Pool
}

// NewService builds the importer.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ImportLegacy stores every workout of an export, creating exercises for
// names the account does not have yet. Running it again skips what is there.
func (s *Service) ImportLegacy(ctx context.Context, export Export) (Result, error) {
	workouts := make([]legacyWorkout, 0, len(export.Workouts))
	for i, raw := range export.Workouts {
		var w legacyWorkout
		if err := json.Unmarshal(raw, &w); err != nil {
			return Result{}, api.Errorf(http.StatusUnprocessableEntity, "invalid_export",
				"workout %d could not be read: %v", i+1, err)
		}
		workouts = append(workouts, w)
	}

	var result Result
	err := db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		created, err := ensureExercises(ctx, exercisedb.New(tx), workouts)
		if err != nil {
			return err
		}
		result.ExercisesCreated = created

		for i, w := range workouts {
			in, opts, err := convert(w)
			if err != nil {
				return api.Errorf(http.StatusUnprocessableEntity, "invalid_export", "workout %d: %v", i+1, err)
			}
			_, err = workout.CreateInTx(ctx, tx, in, opts)
			switch {
			case errors.Is(err, workout.ErrAlreadyImported):
				result.Skipped++
			case err != nil:
				return fmt.Errorf("import workout %d: %w", i+1, err)
			default:
				result.Imported++
			}
		}
		return nil
	})
	return result, err
}

// ensureExercises creates an exercise for every logged name the account does
// not have, from the attributes the workout snapshot carries.
func ensureExercises(ctx context.Context, q *exercisedb.Queries, workouts []legacyWorkout) (int, error) {
	existing, err := q.ListExercises(ctx)
	if err != nil {
		return 0, err
	}
	known := make(map[string]bool, len(existing))
	for _, e := range existing {
		known[strings.ToLower(e.Name)] = true
	}

	created := 0
	for _, w := range workouts {
		for _, e := range w.Exercises {
			name := strings.TrimSpace(e.Name)
			if name == "" || known[strings.ToLower(name)] {
				continue
			}
			known[strings.ToLower(name)] = true
			if _, err := q.CreateExercise(ctx, exercisedb.CreateExerciseParams{
				Name:        name,
				MuscleGroup: e.MuscleGroup,
				Type:        oneOf(e.Type, "strength", "strength", "cardio"),
				DisplayType: oneOf(e.DisplayType, "reps", "reps", "time"),
				SingleArm:   e.SingleArm,
			}); err != nil {
				return 0, fmt.Errorf("create exercise %q: %w", name, err)
			}
			created++
		}
	}
	return created, nil
}

func convert(w legacyWorkout) (workout.WorkoutInput, workout.CreateOptions, error) {
	started, err := parseTime(w.Started)
	if err != nil {
		return workout.WorkoutInput{}, workout.CreateOptions{}, fmt.Errorf("started: %w", err)
	}
	in := workout.WorkoutInput{
		Name:      w.Name,
		Started:   started,
		Notes:     w.Notes,
		Exercises: make([]workout.WorkoutExercise, 0, len(w.Exercises)),
	}
	if w.Ended != nil && *w.Ended != "" {
		ended, err := parseTime(*w.Ended)
		if err != nil {
			return workout.WorkoutInput{}, workout.CreateOptions{}, fmt.Errorf("ended: %w", err)
		}
		in.Ended = &ended
	}

	for _, e := range w.Exercises {
		if strings.TrimSpace(e.Name) == "" {
			continue
		}
		exercise := workout.WorkoutExercise{
			Name:        strings.TrimSpace(e.Name),
			MuscleGroup: e.MuscleGroup,
			Type:        oneOf(e.Type, "strength", "strength", "cardio"),
			DisplayType: oneOf(e.DisplayType, "reps", "reps", "time"),
			SingleArm:   e.SingleArm,
			Sets:        make([]workout.WorkoutSet, 0, len(e.Sets)),
		}
		if e.Intensity != nil && (*e.Intensity == "heavy" || *e.Intensity == "light") {
			intensity := *e.Intensity
			exercise.Intensity = &intensity
		}
		for _, set := range e.Sets {
			exercise.Sets = append(exercise.Sets, workout.WorkoutSet{
				Type:     oneOf(set.Type, "regular", "regular", "warmup"),
				Weight:   nonNegative(set.Weight.value),
				Distance: nonNegative(set.Distance.value),
				Reps:     wholeReps(set.Reps.value),
				Time:     set.Time,
				RPE:      rpe(set.RPE.value),
				Arm:      oneOf(set.Arm, "", "", "left", "right", "both"),
				Notes:    set.Notes,
			})
		}
		in.Exercises = append(in.Exercises, exercise)
	}

	opts := workout.CreateOptions{LegacyID: &w.ID}
	if w.Created != nil {
		if created, err := parseTime(*w.Created); err == nil {
			opts.Created = &created
		}
	}
	if w.Updated != nil {
		if updated, err := parseTime(*w.Updated); err == nil {
			opts.Updated = &updated
		}
	}
	return in, opts, nil
}

func parseTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("%q is not a date", value)
}

// oneOf returns value when it is one of allowed, and fallback otherwise.
func oneOf(value, fallback string, allowed ...string) string {
	for _, a := range allowed {
		if value == a {
			return value
		}
	}
	return fallback
}

func nonNegative(v *float64) *float64 {
	if v == nil || *v < 0 {
		return nil
	}
	return v
}

func wholeReps(v *float64) *int32 {
	if v == nil || *v < 0 || *v > 10000 {
		return nil
	}
	reps := int32(math.Round(*v))
	return &reps
}

func rpe(v *float64) *float64 {
	if v == nil || *v < 0 || *v > 10 {
		return nil
	}
	return v
}

// Register declares the import operation.
func Register(reg *api.Registry, s *Service) {
	api.Register(reg, api.Op{
		ID: "import-legacy-export", Method: http.MethodPost, Path: "/import/legacy",
		Summary: "Import the export file of the local only app",
		Description: "Creates exercises for names the account does not have, then stores every workout. " +
			"Importing the same file again skips workouts already imported.",
		Tags:         []string{"import"},
		MaxBodyBytes: 50 << 20,
		Errors:       []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct{ Body Export }) (*struct{ Body Result }, error) {
		result, err := s.ImportLegacy(ctx, in.Body)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Result }{Body: result}, nil
	})
}
