/**
 * The food diary: foods to search and add, what was eaten each day, energy
 * burned, body weight and the daily goal. An assistant logs into the same
 * diary over MCP.
 */
import * as sdk from "./gen";
import type {
  Activity,
  ActivityDraft,
  Day,
  DayKey,
  DiarySummary,
  EntryDraft,
  Food,
  FoodDraft,
  FoodEntry,
  FoodSearch,
  Id,
  NutritionGoal,
  NutritionGoalDraft,
  Portion,
} from "@/types/domain";
import { ApiError, unwrap } from "./index";

/** Foods by name, the user's own and Livsmedelsverket's, best match first. */
export function searchFoods(query: string, limit = 20): Promise<FoodSearch> {
  return unwrap(sdk.searchFoods({ query: { q: query, limit } }));
}

/** The user's own foods. */
export function getFoods(): Promise<Food[]> {
  return unwrap(sdk.listFoods());
}

export function createFood(food: FoodDraft): Promise<Food> {
  return unwrap(sdk.createFood({ body: food }));
}

export function updateFood(id: Id, food: FoodDraft): Promise<Food> {
  return unwrap(sdk.updateFood({ path: { id }, body: food }));
}

export async function deleteFood(id: Id): Promise<void> {
  await unwrap(sdk.deleteFood({ path: { id } }));
}

/** Saves a household measure of a food, e.g. a tablespoon is 15 g. */
export function savePortion(
  food: { foodId?: Id | null; lmvNumber?: number | null },
  name: string,
  grams: number,
): Promise<Portion> {
  const body = food.foodId ? { foodId: food.foodId, name, grams } : { lmvNumber: food.lmvNumber!, name, grams };
  return unwrap(sdk.savePortion({ body }));
}

/** A day of the diary with its meals, activities, weight and totals. */
export function getDay(day: DayKey): Promise<Day> {
  return unwrap(sdk.getDiaryDay({ path: { day } }));
}

/** Every day of a range in brief. */
export function getDiarySummary(from: DayKey, to: DayKey): Promise<DiarySummary> {
  return unwrap(sdk.summarizeDiary({ query: { from, to } }));
}

/** Logs what was eaten, all or nothing, and answers the day as it now is. */
export function logFood(day: DayKey, entries: EntryDraft[]): Promise<Day> {
  return unwrap(sdk.logFood({ path: { day }, body: { entries } }));
}

export function updateEntry(
  id: Id,
  change: { day: DayKey; meal: FoodEntry["meal"]; grams: number; amount?: string },
): Promise<FoodEntry> {
  return unwrap(sdk.updateFoodEntry({ path: { id }, body: change }));
}

export async function deleteEntry(id: Id): Promise<void> {
  await unwrap(sdk.deleteFoodEntry({ path: { id } }));
}

export function logActivity(day: DayKey, activity: ActivityDraft): Promise<Activity> {
  return unwrap(sdk.logActivity({ path: { day }, body: activity }));
}

export async function deleteActivity(id: Id): Promise<void> {
  await unwrap(sdk.deleteActivity({ path: { id } }));
}

/** Sets a day's body weight, in the user's weight unit. */
export async function saveWeight(day: DayKey, weight: number): Promise<void> {
  await unwrap(sdk.logWeight({ path: { day }, body: { weight } }));
}

export async function deleteWeight(day: DayKey): Promise<void> {
  await unwrap(sdk.deleteWeight({ path: { day } }));
}

/** The daily goal, or undefined before one is set. */
export async function getNutritionGoal(): Promise<NutritionGoal | undefined> {
  try {
    return await unwrap(sdk.getNutritionGoal());
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return undefined;
    throw error;
  }
}

export function saveNutritionGoal(goal: NutritionGoalDraft): Promise<NutritionGoal> {
  return unwrap(sdk.setNutritionGoal({ body: goal }));
}
