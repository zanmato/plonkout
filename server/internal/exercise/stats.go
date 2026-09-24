package exercise

import (
	"context"
	"math"
	"net/http"
	"slices"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/exercise/exercisedb"
	"github.com/zanmato/plonkout/server/internal/platform/api"
)

// Stats summarizes how an exercise has gone, from its working sets.
type Stats struct {
	Exercise string `json:"exercise"`
	Arm      string `json:"arm,omitempty" doc:"The arm the numbers are for, when asked for one."`
	Workouts int    `json:"workouts" doc:"Workouts that logged the exercise."`
	// BestEstimated1RM is the best Epley estimate of any working set.
	BestEstimated1RM *SetRecord `json:"bestEstimated1RM"`
	// Heaviest is the heaviest working set.
	Heaviest *SetRecord `json:"heaviest"`
	// BestByReps is the heaviest weight lifted for each rep count up to 12.
	BestByReps []SetRecord `json:"bestByReps"`
	// Weekly is one entry per week the exercise was trained, oldest first.
	Weekly []WeekStats `json:"weekly"`
}

// SetRecord is one notable set.
type SetRecord struct {
	Weight       float64   `json:"weight"`
	Reps         int32     `json:"reps"`
	RPE          *float64  `json:"rpe"`
	Arm          string    `json:"arm"`
	Date         time.Time `json:"date"`
	WorkoutID    uuid.UUID `json:"workoutId"`
	Estimated1RM float64   `json:"estimated1RM"`
}

// WeekStats is one training week of an exercise.
type WeekStats struct {
	WeekStart   string    `json:"weekStart" format:"date" doc:"The Monday of the week."`
	TopSet      SetRecord `json:"topSet" doc:"The set with the best estimated 1RM that week."`
	Volume      float64   `json:"volume" doc:"Weight times reps over every working set."`
	WorkingSets int       `json:"workingSets"`
}

// epley matches calculateEpley1RM in the web app and plan.Epley1RM.
func epley(weight float64, reps int32) float64 {
	if weight <= 0 || reps <= 0 {
		return 0
	}
	return weight * (1 + float64(min(reps, 12))/30)
}

// Stats summarizes an exercise's working sets, optionally for one arm and
// from a date. An exercise never logged answers with the closest names.
func (s *Service) Stats(ctx context.Context, name, arm string, from *time.Time) (Stats, error) {
	rows, err := s.q.ExerciseSets(ctx, exercisedb.ExerciseSetsParams{Name: name, FromTime: from})
	if err != nil {
		return Stats{}, err
	}
	if len(rows) == 0 {
		if _, err := s.Resolve(ctx, name); err != nil {
			return Stats{}, err
		}
	}

	out := Stats{Exercise: name, Arm: arm, BestByReps: []SetRecord{}, Weekly: []WeekStats{}}
	// Answer with the list's spelling of the name, however it was asked for.
	if known, err := s.q.FindExerciseByName(ctx, name); err == nil {
		out.Exercise = known.Name
	}
	workouts := map[uuid.UUID]bool{}
	byReps := map[int32]SetRecord{}
	weeks := map[string]*WeekStats{}

	for _, row := range rows {
		workouts[row.WorkoutID] = true
		if row.Type != "regular" || row.Weight == nil || row.Reps == nil || *row.Weight <= 0 || *row.Reps <= 0 {
			continue
		}
		// A set for both arms counts for either one.
		if arm != "" && row.Arm != "" && row.Arm != arm && row.Arm != "both" {
			continue
		}
		set := SetRecord{
			Weight: *row.Weight, Reps: *row.Reps, RPE: row.Rpe, Arm: row.Arm, Date: row.Started,
			WorkoutID: row.WorkoutID, Estimated1RM: math.Round(epley(*row.Weight, *row.Reps)*10) / 10,
		}
		if out.BestEstimated1RM == nil || set.Estimated1RM > out.BestEstimated1RM.Estimated1RM {
			best := set
			out.BestEstimated1RM = &best
		}
		if out.Heaviest == nil || set.Weight > out.Heaviest.Weight {
			heaviest := set
			out.Heaviest = &heaviest
		}
		if set.Reps <= 12 {
			if current, ok := byReps[set.Reps]; !ok || set.Weight > current.Weight {
				byReps[set.Reps] = set
			}
		}

		week := weekStart(row.Started)
		w, ok := weeks[week]
		if !ok {
			w = &WeekStats{WeekStart: week, TopSet: set}
			weeks[week] = w
		}
		if set.Estimated1RM > w.TopSet.Estimated1RM {
			w.TopSet = set
		}
		w.Volume += set.Weight * float64(set.Reps)
		w.WorkingSets++
	}

	out.Workouts = len(workouts)
	for reps := int32(1); reps <= 12; reps++ {
		if set, ok := byReps[reps]; ok {
			out.BestByReps = append(out.BestByReps, set)
		}
	}
	for _, w := range weeks {
		out.Weekly = append(out.Weekly, *w)
	}
	slices.SortFunc(out.Weekly, func(a, b WeekStats) int {
		switch {
		case a.WeekStart < b.WeekStart:
			return -1
		case a.WeekStart > b.WeekStart:
			return 1
		}
		return 0
	})
	return out, nil
}

func weekStart(t time.Time) string {
	day := t.UTC()
	offset := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -offset).Format("2006-01-02")
}

// registerStats declares the statistics operation.
func registerStats(reg *api.Registry, s *Service) {
	api.Register(reg, api.Op{
		ID: "get-exercise-stats", Method: http.MethodGet, Path: "/exercises/stats",
		Summary: "How an exercise has gone",
		Description: "Best estimated 1RM (Epley, capped at 12 reps), the heaviest set, the heaviest weight " +
			"for each rep count, and the top set and volume of every week, from working sets only. " +
			"Optionally for one arm and from a date.",
		Tags: []string{"exercises"}, MCPTool: "get_exercise_stats",
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct {
		Name string    `query:"name" minLength:"1" doc:"The exercise's name."`
		Arm  string    `query:"arm" required:"false" enum:"left,right" doc:"Only sets of this arm, for single arm exercises."`
		From time.Time `query:"from" required:"false" doc:"Only sets logged at or after this time."`
	}) (*struct{ Body Stats }, error) {
		var from *time.Time
		if !in.From.IsZero() {
			from = &in.From
		}
		stats, err := s.Stats(ctx, in.Name, in.Arm, from)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Stats }{Body: stats}, nil
	})
}
