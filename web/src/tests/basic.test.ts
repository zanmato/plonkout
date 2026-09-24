import { describe, it, expect } from "vitest";
import type { ExerciseDraft, Workout } from "@/types/domain";
import { ApiError } from "@/api";
import {
  deleteWorkout,
  getMostRecentWorkoutByName,
  getWorkout,
  getWorkouts,
  saveWorkout,
  getExercises,
  saveExercise,
  getSetting,
  saveSetting,
} from "@/api/data";
import { backend } from "./mocks/backend";

describe("Data layer against the fake backend", () => {
  it("lists workouts newest first", async () => {
    backend.seed({
      workouts: [
        { name: "Older", started: "2024-01-15T10:00:00Z" },
        { name: "Newer", started: "2024-01-16T10:00:00Z" },
      ],
    });

    const workouts = await getWorkouts();

    expect(workouts.map((w) => w.name)).toEqual(["Newer", "Older"]);
  });

  it("creates a workout and answers with its id and first revision", async () => {
    const workout: Workout = {
      name: "Test Workout",
      started: new Date("2024-01-15T10:00:00Z"),
      notes: "",
      exercises: [],
    };

    const saved = await saveWorkout(workout);

    expect(saved.id).toMatch(/^[0-9a-f-]{36}$/);
    expect(saved.revision).toBe(1);
    expect(backend.workout(saved.id!).name).toBe("Test Workout");
  });

  it("updates a workout and bumps its revision", async () => {
    const [stored] = backend.seed({ workouts: [{ name: "Before", started: "2024-01-15T10:00:00Z" }] }).workouts;

    const saved = await saveWorkout({ ...stored!, name: "After" });

    expect(saved.id).toBe(stored!.id);
    expect(saved.revision).toBe(2);
    expect(backend.workouts).toHaveLength(1);
    expect(backend.workout(stored!.id).name).toBe("After");
  });

  it("refuses to overwrite a workout changed elsewhere", async () => {
    const [stored] = backend.seed({ workouts: [{ name: "Mine", started: "2024-01-15T10:00:00Z", revision: 3 }] }).workouts;

    const error = await saveWorkout({ ...stored!, revision: 2, name: "Stale" }).catch((e) => e);

    expect(error).toBeInstanceOf(ApiError);
    expect(error.status).toBe(409);
    expect(error.code).toBe("stale_revision");
    expect(backend.workout(stored!.id).name).toBe("Mine");
  });

  it("answers undefined for a missing workout and deletes stored ones", async () => {
    const [stored] = backend.seed({ workouts: [{ name: "Gone soon" }] }).workouts;

    expect(await getWorkout(crypto.randomUUID())).toBeUndefined();
    await deleteWorkout(stored!.id);
    expect(backend.workouts).toEqual([]);
  });

  it("finds the latest workout by name, skipping the one being edited", async () => {
    const { workouts } = backend.seed({
      workouts: [
        { name: "Pull", started: "2024-01-10T10:00:00Z" },
        { name: "Pull", started: "2024-01-12T10:00:00Z" },
        { name: "Push", started: "2024-01-14T10:00:00Z" },
      ],
    });

    expect((await getMostRecentWorkoutByName("Pull"))?.id).toBe(workouts[1]!.id);
    expect((await getMostRecentWorkoutByName("Pull", workouts[1]!.id))?.id).toBe(workouts[0]!.id);
    expect(await getMostRecentWorkoutByName("Legs")).toBeNull();
  });

  it("lists exercises without archived ones", async () => {
    backend.seed({
      exercises: [
        { name: "Wrist Curl", muscleGroup: "Forearm" },
        { name: "Old Curl", muscleGroup: "Forearm", archived: true },
      ],
    });

    const exercises = await getExercises();

    expect(exercises.map((e) => e.name)).toEqual(["Wrist Curl"]);
  });

  it("creates an exercise", async () => {
    const exercise: ExerciseDraft = {
      name: "Test Exercise",
      muscleGroup: "Forearm",
      singleArm: true,
      type: "strength",
      displayType: "reps",
    };

    const saved = await saveExercise(exercise);

    expect(saved.id).toMatch(/^[0-9a-f-]{36}$/);
    expect(backend.exercises).toEqual([expect.objectContaining({ ...exercise, id: saved.id, archived: false })]);
  });

  it("reads settings with a default for missing keys", async () => {
    backend.seed({ settings: { language: "sv" } });

    expect(await getSetting("language", "en")).toBe("sv");
    expect(await getSetting("weightUnit", "kg")).toBe("kg");
  });

  it("saves a setting and reads it back", async () => {
    await saveSetting("language", "sv");

    expect(backend.settings).toEqual({ language: "sv" });
    expect(await getSetting("language", "en")).toBe("sv");
  });
});
