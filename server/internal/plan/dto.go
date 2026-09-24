package plan

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

// Target is one group of prescribed sets, e.g. "Back-off, 3 × 5 @ 130, RPE 7".
type Target struct {
	ID      uuid.UUID `json:"id"`
	SetType string    `json:"setType" doc:"A free label, e.g. Top set, Back-off, Volume (3s hold), TEST attempt 2."`
	Sets    int32     `json:"sets"`
	Reps    *int32    `json:"reps"`
	Weight  *float64  `json:"weight"`
	Time    string    `json:"time"`
	RPEMin  *float64  `json:"rpeMin"`
	RPEMax  *float64  `json:"rpeMax"`
	Notes   string    `json:"notes"`
}

// PlannedExercise is an exercise of a planned session with its targets.
type PlannedExercise struct {
	ID          uuid.UUID `json:"id"`
	ExerciseID  uuid.UUID `json:"exerciseId"`
	Exercise    string    `json:"exercise" doc:"The exercise's name."`
	MuscleGroup string    `json:"muscleGroup"`
	Type        string    `json:"type" enum:"strength,cardio"`
	DisplayType string    `json:"displayType" enum:"reps,time"`
	SingleArm   bool      `json:"singleArm"`
	Notes       string    `json:"notes" doc:"Cues and conditional rules for this exercise."`
	// OffArmPercent is the share of the listed weight the non dominant arm
	// works with, for single arm exercises.
	OffArmPercent *float64 `json:"offArmPercent"`
	Targets       []Target `json:"targets"`
}

// Session is one entry of a plan's queue.
type Session struct {
	ID          uuid.UUID         `json:"id"`
	PlanID      uuid.UUID         `json:"planId"`
	Position    int32             `json:"position"`
	Label       string            `json:"label" example:"W3 A"`
	Week        *int32            `json:"week"`
	Day         *string           `json:"day" example:"A"`
	Intensity   *string           `json:"intensity" enum:"heavy,light"`
	Notes       string            `json:"notes"`
	Status      string            `json:"status" enum:"pending,in_progress,completed,skipped"`
	SkipReason  string            `json:"skipReason"`
	CompletedAt *time.Time        `json:"completedAt"`
	WorkoutID   *uuid.UUID        `json:"workoutId" nullable:"true" doc:"The workout logged for this session, once started."`
	Exercises   []PlannedExercise `json:"exercises"`
	// Comparison is present when asked for and the session has a workout.
	Comparison *Comparison `json:"comparison,omitempty"`
}

// Plan is a training plan: an ordered queue of sessions.
type Plan struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Goal      string    `json:"goal"`
	Notes     string    `json:"notes"`
	Status    string    `json:"status" enum:"active,completed,archived"`
	StartDate *string   `json:"startDate" format:"date"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
	Sessions  []Session `json:"sessions"`
}

// TargetInput prescribes a group of sets.
type TargetInput struct {
	SetType string   `json:"setType" maxLength:"60" doc:"A free label, e.g. Top set, Back-off, Volume (3s hold), TEST attempt 2."`
	Sets    int32    `json:"sets" minimum:"1" maximum:"50" doc:"How many sets of this prescription."`
	Reps    *int32   `json:"reps,omitempty" minimum:"0" maximum:"1000"`
	Weight  *float64 `json:"weight,omitempty" minimum:"0" doc:"In the user's weight unit, for the dominant arm on single arm exercises."`
	Time    string   `json:"time,omitempty" maxLength:"32" doc:"For holds and cardio, e.g. 0:30."`
	RPEMin  *float64 `json:"rpeMin,omitempty" minimum:"0" maximum:"10"`
	RPEMax  *float64 `json:"rpeMax,omitempty" minimum:"0" maximum:"10"`
	Notes   string   `json:"notes,omitempty" maxLength:"500"`
}

// PlannedExerciseInput plans an exercise, referenced by its name.
type PlannedExerciseInput struct {
	Exercise      string        `json:"exercise" minLength:"1" maxLength:"100" doc:"The exercise's name, matched ignoring case."`
	Notes         string        `json:"notes,omitempty" maxLength:"2000" doc:"Cues and conditional rules, e.g. if week 6 moved fast, make attempt 4 182.5."`
	OffArmPercent *float64      `json:"offArmPercent,omitempty" exclusiveMinimum:"0" maximum:"100" doc:"Non dominant arm works at this percent of the listed weight."`
	Targets       []TargetInput `json:"targets" minItems:"1" maxItems:"20"`
	// MuscleGroup and SingleArm describe a new exercise, used only when it
	// has to be created.
	MuscleGroup string `json:"muscleGroup,omitempty" maxLength:"50" doc:"Only used when createMissingExercises creates this exercise."`
	SingleArm   *bool  `json:"singleArm,omitempty" doc:"Only used when createMissingExercises creates this exercise."`
}

// SessionInput plans one session.
type SessionInput struct {
	Label     string                 `json:"label" maxLength:"60" example:"W3 A"`
	Week      *int32                 `json:"week,omitempty" minimum:"1"`
	Day       *string                `json:"day,omitempty" maxLength:"8" example:"A"`
	Intensity *string                `json:"intensity,omitempty" enum:"heavy,light" doc:"Heavy for a tough day, light for a volume day."`
	Notes     string                 `json:"notes,omitempty" maxLength:"2000"`
	Exercises []PlannedExerciseInput `json:"exercises" minItems:"1" maxItems:"30"`
}

// PlanInput creates a plan with its sessions.
type PlanInput struct {
	Name      string         `json:"name" minLength:"1" maxLength:"200"`
	Goal      string         `json:"goal,omitempty" maxLength:"2000"`
	Notes     string         `json:"notes,omitempty" maxLength:"5000"`
	StartDate *string        `json:"startDate,omitempty" format:"date"`
	Sessions  []SessionInput `json:"sessions" maxItems:"200"`
	// CreateMissingExercises adds exercises the user does not have. Without it
	// an unknown name is refused with the closest matches, so a typo is
	// corrected rather than stored.
	CreateMissingExercises bool `json:"createMissingExercises,omitempty"`
}

// PlanUpdate changes a plan's own fields.
type PlanUpdate struct {
	Name      string  `json:"name" minLength:"1" maxLength:"200"`
	Goal      string  `json:"goal" maxLength:"2000"`
	Notes     string  `json:"notes" maxLength:"5000"`
	Status    string  `json:"status" enum:"active,completed,archived"`
	StartDate *string `json:"startDate" format:"date"`
}

// Comparison is what was planned for a session against what was logged.
type Comparison struct {
	Exercises []ExerciseComparison `json:"exercises"`
	// Unplanned are exercises logged in the workout that the session did not plan.
	Unplanned []string `json:"unplanned"`
}

// ExerciseComparison compares one planned exercise.
type ExerciseComparison struct {
	PlannedExerciseID uuid.UUID          `json:"plannedExerciseId"`
	Exercise          string             `json:"exercise"`
	Targets           []TargetComparison `json:"targets"`
	// ExtraSets are logged sets of this exercise that no target prescribed.
	ExtraSets []ActualSet `json:"extraSets"`
	// BestEstimated1RM is the best Epley estimate of the logged working sets.
	BestEstimated1RM *float64 `json:"bestEstimated1RM"`
}

// TargetComparison compares one target with the sets logged against it.
type TargetComparison struct {
	Target Target      `json:"target"`
	Sets   []ActualSet `json:"sets"`
	// Done counts sets with reps, a weight or a time logged.
	Done int32 `json:"done"`
	// Met is true when every prescribed set was done at or above the target
	// reps and weight.
	Met    bool     `json:"met"`
	TopRPE *float64 `json:"topRpe"`
}

// ActualSet is one logged set.
type ActualSet struct {
	Weight    *float64 `json:"weight"`
	Reps      *int32   `json:"reps"`
	Time      string   `json:"time"`
	RPE       *float64 `json:"rpe"`
	Arm       string   `json:"arm"`
	Type      string   `json:"type"`
	TargetSeq *int32   `json:"targetSeq,omitempty"`
}
