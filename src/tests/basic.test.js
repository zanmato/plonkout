import { describe, it, expect } from "vitest";
import {
  getWorkouts,
  saveWorkout,
  getExercises,
  saveExercise,
  getSetting,
  saveSetting,
} from "@/utils/database.js";

describe("Database Mocks", () => {
  it("should mock getWorkouts function", async () => {
    const workouts = await getWorkouts();
    expect(Array.isArray(workouts)).toBe(true);
    expect(getWorkouts).toHaveBeenCalled();
  });

  it("should mock saveWorkout function", async () => {
    const workout = { name: "Test Workout", exercises: [] };
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
    const exercise = {
      name: "Test Exercise",
      muscleGroup: "Test",
      singleArm: true,
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
