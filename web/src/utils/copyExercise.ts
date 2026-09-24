import type { WorkoutExercise, WorkoutSet } from "@/types/domain";

/**
 * A set without the link to the plan target it was logged against, for
 * copying into another workout or template.
 */
export function withoutTarget(set: WorkoutSet): WorkoutSet {
  const copy = { ...set };
  delete copy.targetId;
  delete copy.targetSeq;
  return copy;
}

/**
 * Copies a logged exercise into a new workout or template. The link to the
 * exercise list stays, the links to a planned session do not.
 */
export function copyExercise(exercise: WorkoutExercise, sets: WorkoutSet[]): WorkoutExercise {
  const copy: WorkoutExercise = { ...exercise, sets: sets.map(withoutTarget) };
  delete copy.plannedExerciseId;
  return copy;
}
