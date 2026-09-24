import { describe, it, expect } from "vitest";
import type { Exercise, Workout } from "@/types/domain";
import {
  getWorkouts,
  saveWorkout,
  getExercises,
  saveExercise,
  getSetting,
  saveSetting,
} from "@/utils/database";

describe("Database Mocks", () => {
  it("should mock getWorkouts function", async () => {
    const workouts = await getWorkouts();
    expect(Array.isArray(workouts)).toBe(true);
    expect(getWorkouts).toHaveBeenCalled();
  });

  it("should mock saveWorkout function", async () => {
    const workout: Workout = {
      name: "Test Workout",
      started: new Date(),
      notes: "",
      exercises: [],
    };
    const id = await saveWorkout(workout);
    expect(typeof id).toBe("number");
    expect(saveWorkout).toHaveBeenCalledWith(workout);
  });

  it("should mock getExercises function", async () => {
    const exercises = await getExercises();
    expect(Array.isArray(exercises)).toBe(true);
    expect(getExercises).toHaveBeenCalled();
  });

  it("should mock saveExercise function", async () => {
    const exercise: Exercise = {
      name: "Test Exercise",
      muscleGroup: "Test",
      singleArm: true,
      type: "strength",
      displayType: "reps",
    };
    const id = await saveExercise(exercise);
    expect(typeof id).toBe("number");
    expect(saveExercise).toHaveBeenCalledWith(exercise);
  });

  it("should mock getSetting function", async () => {
    const setting = await getSetting("language", "en");
    expect(typeof setting).toBe("string");
    expect(getSetting).toHaveBeenCalledWith("language", "en");
  });

  it("should mock saveSetting function", async () => {
    await saveSetting("language", "sv");
    expect(saveSetting).toHaveBeenCalledWith("language", "sv");
  });
});
