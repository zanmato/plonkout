package plan

import (
	"context"
	"math"
	"slices"

	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/plan/plandb"
)

// weightTolerance absorbs rounding, e.g. an off arm weight of 17.4 logged as
// 17.5 or a 21.25 target lifted as 21 on a stack that jumps in whole kilos.
const weightTolerance = 0.5

// Epley1RM estimates a one rep max, capped at 12 reps where the formula is
// reliable. It must match calculateEpley1RM in the web app.
func Epley1RM(weight float64, reps int32) float64 {
	if weight <= 0 || reps <= 0 {
		return 0
	}
	return weight * (1 + float64(min(reps, 12))/30)
}

func attachComparisons(ctx context.Context, q *plandb.Queries, sessions []Session) error {
	ids := sessionIDs(sessions)
	if len(ids) == 0 {
		return nil
	}
	rows, err := q.ListActualSets(ctx, ids)
	if err != nil {
		return err
	}
	bySession := map[uuid.UUID][]plandb.ListActualSetsRow{}
	for _, row := range rows {
		if row.SessionID != nil {
			bySession[*row.SessionID] = append(bySession[*row.SessionID], row)
		}
	}
	for i := range sessions {
		if sessions[i].WorkoutID == nil {
			continue
		}
		comparison := compare(sessions[i], bySession[sessions[i].ID])
		sessions[i].Comparison = &comparison
	}
	return nil
}

// compare lines up what a session planned with what its workout logged.
func compare(session Session, rows []plandb.ListActualSetsRow) Comparison {
	out := Comparison{Exercises: []ExerciseComparison{}, Unplanned: []string{}}

	byExercise := map[uuid.UUID][]plandb.ListActualSetsRow{}
	unplanned := map[string]bool{}
	for _, row := range rows {
		if row.PlannedExerciseID == nil {
			if !unplanned[row.Name] {
				unplanned[row.Name] = true
				out.Unplanned = append(out.Unplanned, row.Name)
			}
			continue
		}
		byExercise[*row.PlannedExerciseID] = append(byExercise[*row.PlannedExerciseID], row)
	}

	for _, planned := range session.Exercises {
		logged := byExercise[planned.ID]
		comparison := ExerciseComparison{
			PlannedExerciseID: planned.ID,
			Exercise:          planned.Exercise,
			Targets:           make([]TargetComparison, 0, len(planned.Targets)),
			ExtraSets:         []ActualSet{},
		}

		byTarget := map[uuid.UUID][]plandb.ListActualSetsRow{}
		best := 0.0
		for _, row := range logged {
			if row.Type == "regular" && row.Weight != nil && row.Reps != nil {
				best = math.Max(best, Epley1RM(*row.Weight, *row.Reps))
			}
			if row.TargetID == nil {
				comparison.ExtraSets = append(comparison.ExtraSets, actual(row))
				continue
			}
			byTarget[*row.TargetID] = append(byTarget[*row.TargetID], row)
		}
		if best > 0 {
			rounded := math.Round(best*10) / 10
			comparison.BestEstimated1RM = &rounded
		}

		for _, target := range planned.Targets {
			comparison.Targets = append(comparison.Targets, compareTarget(target, planned, byTarget[target.ID]))
		}
		out.Exercises = append(out.Exercises, comparison)
	}
	return out
}

// compareTarget judges one target. Its sets are grouped by their number in the
// prescription, since a single arm exercise logs one set per arm for each. The
// heaviest set of a number answers for the dominant arm and the rest for the
// off arm at its percentage.
func compareTarget(target Target, planned PlannedExercise, rows []plandb.ListActualSetsRow) TargetComparison {
	out := TargetComparison{Target: target, Sets: make([]ActualSet, 0, len(rows))}

	bySeq := map[int32][]plandb.ListActualSetsRow{}
	var seqs []int32
	for i, row := range rows {
		out.Sets = append(out.Sets, actual(row))
		seq := int32(-1 - i)
		if row.TargetSeq != nil {
			seq = *row.TargetSeq
		}
		if _, seen := bySeq[seq]; !seen {
			seqs = append(seqs, seq)
		}
		bySeq[seq] = append(bySeq[seq], row)
		if row.Rpe != nil && (out.TopRPE == nil || *row.Rpe > *out.TopRPE) {
			rpe := *row.Rpe
			out.TopRPE = &rpe
		}
	}

	met := true
	for _, seq := range seqs {
		sets := bySeq[seq]
		// A number counts once every arm's set of it has something logged.
		if slices.ContainsFunc(sets, func(r plandb.ListActualSetsRow) bool { return !done(r) }) {
			met = false
			continue
		}
		out.Done++

		slices.SortFunc(sets, func(a, b plandb.ListActualSetsRow) int {
			return cmpWeight(b.Weight, a.Weight)
		})
		for i, s := range sets {
			if target.Reps != nil && (s.Reps == nil || *s.Reps < *target.Reps) {
				met = false
			}
			if target.Weight != nil {
				expected := *target.Weight
				if i > 0 && planned.OffArmPercent != nil {
					expected = expected * *planned.OffArmPercent / 100
				}
				if s.Weight == nil || *s.Weight+weightTolerance < expected {
					met = false
				}
			}
		}
	}
	out.Met = met && out.Done >= target.Sets
	return out
}

// done is a set with anything logged.
func done(row plandb.ListActualSetsRow) bool {
	return (row.Reps != nil && *row.Reps > 0) || (row.Weight != nil && *row.Weight > 0) || row.Time != ""
}

func cmpWeight(a, b *float64) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	case *a < *b:
		return -1
	case *a > *b:
		return 1
	}
	return 0
}

func actual(row plandb.ListActualSetsRow) ActualSet {
	return ActualSet{
		Weight: row.Weight, Reps: row.Reps, Time: row.Time, RPE: row.Rpe, Arm: row.Arm, Type: row.Type,
		TargetSeq: row.TargetSeq,
	}
}
