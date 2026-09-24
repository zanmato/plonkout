package workout

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

// WorkoutSet is one logged set. Empty weight and reps mean the set was not done yet,
// which is how a set prefilled from a plan's target starts out.
type WorkoutSet struct {
	Type     string   `json:"type" enum:"regular,warmup"`
	Weight   *float64 `json:"weight" minimum:"0" maximum:"99999"`
	Distance *float64 `json:"distance" minimum:"0"`
	Reps     *int32   `json:"reps" minimum:"0" maximum:"10000"`
	Time     string   `json:"time" maxLength:"32"`
	RPE      *float64 `json:"rpe" minimum:"0" maximum:"10"`
	Arm      string   `json:"arm" enum:",left,right,both"`
	Notes    string   `json:"notes" maxLength:"1000"`
	// TargetID and TargetSeq link the set to the planned target it was
	// prescribed by, and say which of that target's sets it is.
	TargetID  *uuid.UUID `json:"targetId,omitempty"`
	TargetSeq *int32     `json:"targetSeq,omitempty" minimum:"1"`
}

// WorkoutExercise is an exercise as performed in a workout. Name and attributes are a
// snapshot, so history reads the way it was logged.
type WorkoutExercise struct {
	// ExerciseID links to the exercise list. The server fills it in by name
	// when it is missing.
	ExerciseID        *uuid.UUID   `json:"exerciseId,omitempty"`
	Name              string       `json:"name" minLength:"1" maxLength:"100"`
	MuscleGroup       string       `json:"muscleGroup" maxLength:"50"`
	Type              string       `json:"type" enum:"strength,cardio"`
	DisplayType       string       `json:"displayType" enum:"reps,time"`
	SingleArm         bool         `json:"singleArm"`
	Intensity         *string      `json:"intensity" enum:"heavy,light"`
	Notes             string       `json:"notes,omitempty" maxLength:"1000"`
	PlannedExerciseID *uuid.UUID   `json:"plannedExerciseId,omitempty"`
	Sets              []WorkoutSet `json:"sets" maxItems:"100"`
}

// WorkoutInput is what a client writes.
type WorkoutInput struct {
	Name      string            `json:"name" maxLength:"200"`
	Started   time.Time         `json:"started"`
	Ended     *time.Time        `json:"ended"`
	Notes     string            `json:"notes" maxLength:"5000"`
	Exercises []WorkoutExercise `json:"exercises" maxItems:"50"`
}

// WorkoutUpdate is a write to an existing workout. Revision is the one the client
// last read, so a stale editor cannot overwrite a newer version.
type WorkoutUpdate struct {
	WorkoutInput
	Revision int32 `json:"revision" minimum:"1"`
}

// Workout is a whole workout as the API shows it.
type Workout struct {
	ID               uuid.UUID         `json:"id"`
	Name             string            `json:"name"`
	Started          time.Time         `json:"started"`
	Ended            *time.Time        `json:"ended"`
	Notes            string            `json:"notes"`
	PlannedSessionID *uuid.UUID        `json:"plannedSessionId,omitempty"`
	Revision         int32             `json:"revision"`
	Created          time.Time         `json:"created"`
	Updated          time.Time         `json:"updated"`
	Exercises        []WorkoutExercise `json:"exercises"`
}
