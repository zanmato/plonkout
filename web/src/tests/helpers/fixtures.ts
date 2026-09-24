/**
 * Test fixtures usually fill in only the fields the code under test reads.
 * DeepPartial keeps the field names and value types checked while letting
 * the rest be left out.
 */
export type DeepPartial<T> = T extends (infer U)[]
  ? DeepPartial<U>[]
  : T extends Date
    ? T
    : T extends object
      ? { [K in keyof T]?: DeepPartial<T[K]> }
      : T;

/** Treat a partial fixture as the full type */
export const partial = <T>(value: DeepPartial<T>): T => value as T;

/** Workout fields the WorkoutLog sorting and grouping simulations use */
export interface SortableWorkout {
  id: number;
  name?: string;
  started: string | Date;
  ended?: string;
  exercises: unknown[];
}

/** A month group as built by the WorkoutLog grouping logic */
export interface MonthGroup {
  monthYear: string;
  workouts: SortableWorkout[];
  sortDate: Date;
}
