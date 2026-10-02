/**
 * The food diary part of the fake backend: foods, Livsmedelsverket's foods,
 * diary entries, activities, body weight and the goal. It does the server's
 * arithmetic (nutrients scaled to tenths, meal aims as shares of the goal) so
 * views show what they would against the real thing. Search is simpler than
 * the server's: every word must appear in the name, the user's foods first.
 */
import { http, HttpResponse, type HttpHandler } from "msw";
import type {
  Activity,
  DataSource,
  Day,
  Entry,
  Food,
  Goal,
  Match,
  Meal,
  Nutrients,
  Portion,
  Problem,
  Totals,
} from "@/api/gen/types.gen";

/** A food of Livsmedelsverket's database. */
export interface LmvFoodSeed {
  number: number;
  name: string;
  group?: string;
  per100g: Nutrients;
  portions?: { name: string; grams: number }[];
}

export type FoodSeed = Partial<Omit<Food, "portions">> & {
  name: string;
  per100g: Nutrients;
  portions?: { name: string; grams: number }[];
};

export type EntrySeed = Partial<Omit<Entry, "kcal" | "protein" | "carbs" | "fat">> & {
  day: string;
  meal: Entry["meal"];
  name: string;
  grams: number;
  per100g: Nutrients;
};

export interface FoodBackendSeed {
  lmvFoods?: LmvFoodSeed[];
  foods?: FoodSeed[];
  entries?: EntrySeed[];
  activities?: (Partial<Activity> & { day: string; label: string; kcal: number })[];
  weights?: Record<string, number>;
  goal?: Partial<Goal> & { kcal: number; protein: number; carbs: number; fat: number };
}

export interface LmvFood {
  number: number;
  name: string;
  group: string;
  per100g: Nutrients;
  portions: Portion[];
}

const MEALS: Meal["meal"][] = ["breakfast", "lunch", "dinner", "snack"];
const AIMS: Record<Meal["meal"], [number, number]> = {
  breakfast: [25, 30],
  lunch: [30, 35],
  dinner: [25, 30],
  snack: [10, 15],
};

export const SOURCE: DataSource = {
  name: "Livsmedelsverkets livsmedelsdatabas",
  license: "CC BY 4.0",
  licenseUrl: "https://creativecommons.org/licenses/by/4.0/",
  url: "https://www.livsmedelsverket.se/livsmedelsdatabasen",
  version: "2026-07-01",
  attribution: "Livsmedelsverkets livsmedelsdatabas version 2026-07-01, CC BY 4.0",
  importedAt: "2026-07-02T00:00:00.000Z",
};

const state = {
  lmvFoods: [] as LmvFood[],
  foods: [] as Food[],
  entries: [] as Entry[],
  activities: [] as Activity[],
  weights: {} as Record<string, number>,
  goal: null as Goal | null,
};

const uuid = () => crypto.randomUUID();
const now = () => new Date().toISOString();
const clone = <T>(value: T): T => structuredClone(value);
const scale = (per100g: number, grams: number) => Math.round((per100g * grams) / 10) / 10;
const round = (n: number) => Math.round(n * 10) / 10;
const url = (path: string) => `http://localhost/api${path}`;
const noContent = () => new HttpResponse(null, { status: 204 });

function problem(status: number, code: string, detail: string) {
  const body: Problem = { type: "about:blank", title: "Error", status, code, detail };
  return HttpResponse.json(body, { status, headers: { "Content-Type": "application/problem+json" } });
}

type Body = Record<string, unknown>;

function unknownFields(body: Body, allowed: string[], at: string): string[] {
  return Object.keys(body)
    .filter((key) => !allowed.includes(key))
    .map((key) => `${at}.${key}`);
}

const portionsOf = (list: { name: string; grams: number }[] = []): Portion[] =>
  list.map((p) => ({ id: uuid(), name: p.name, grams: p.grams }));

function toEntry(seed: EntrySeed): Entry {
  const n = seed.per100g;
  return {
    id: uuid(),
    foodId: null,
    lmvNumber: null,
    amount: "",
    loggedBy: "app",
    created: now(),
    ...seed,
    kcal: scale(n.kcal, seed.grams),
    protein: scale(n.protein, seed.grams),
    carbs: scale(n.carbs, seed.grams),
    fat: scale(n.fat, seed.grams),
  };
}

function rescale(entry: Entry): Entry {
  return toEntry(entry as unknown as EntrySeed);
}

function emptyTotals(): Totals {
  return { kcal: 0, protein: 0, carbs: 0, fat: 0, fiber: 0, sugars: 0, saturatedFat: 0, salt: 0 };
}

function add(totals: Totals, entry: Entry) {
  totals.kcal = round(totals.kcal + entry.kcal);
  totals.protein = round(totals.protein + entry.protein);
  totals.carbs = round(totals.carbs + entry.carbs);
  totals.fat = round(totals.fat + entry.fat);
}

function dayOf(day: string): Day {
  const goal = state.goal;
  const meals: Meal[] = MEALS.map((meal) => ({
    meal,
    ...(goal
      ? { aim: { min: Math.round((goal.kcal * AIMS[meal][0]) / 100), max: Math.round((goal.kcal * AIMS[meal][1]) / 100) } }
      : {}),
    totals: emptyTotals(),
    entries: [],
  }));
  const eaten = emptyTotals();
  for (const entry of state.entries.filter((e) => e.day === day)) {
    const meal = meals.find((m) => m.meal === entry.meal)!;
    meal.entries.push(clone(entry));
    add(meal.totals, entry);
    add(eaten, entry);
  }
  const activities = state.activities.filter((a) => a.day === day);
  const burned = activities.reduce((sum, a) => sum + a.kcal, 0);
  const budget = goal ? goal.kcal + (goal.addActivities ? burned : 0) : null;
  return {
    day,
    ...(goal ? { goal: clone(goal) } : {}),
    eaten,
    burned,
    budget,
    remaining: budget === null ? null : round(budget - eaten.kcal),
    meals,
    activities: clone(activities),
    weight: state.weights[day] ?? null,
  };
}

const lmvMatch = (food: LmvFood): Match => ({
  source: "lmv",
  foodId: null,
  lmvNumber: food.number,
  name: food.name,
  brand: "",
  group: food.group,
  per100g: clone(food.per100g),
  portions: clone(food.portions),
  uses: 0,
});

const ownMatch = (food: Food): Match => ({
  source: "mine",
  foodId: food.id,
  lmvNumber: null,
  name: food.name,
  brand: food.brand,
  group: "",
  per100g: clone(food.per100g),
  portions: clone(food.portions),
  uses: 0,
});

const entryFields = ["meal", "foodId", "lmvNumber", "name", "per100g", "grams", "amount", "alias"];
const foodFields = ["name", "brand", "gtin", "per100g", "notes", "portions"];
const goalFields = [
  "direction", "kcal", "protein", "carbs", "fat", "addActivities", "targetWeight", "weeklyChange", "notes",
];

export const foodHandlers: HttpHandler[] = [
  http.get(url("/foods/search"), ({ request }) => {
    const query = (new URL(request.url).searchParams.get("q") ?? "").trim().toLowerCase();
    const words = query.split(/\s+/).filter(Boolean);
    const has = (name: string) => words.length > 0 && words.every((w) => name.toLowerCase().includes(w));
    const matches = [
      ...state.foods.filter((f) => has(`${f.name} ${f.brand}`)).map(ownMatch),
      ...state.lmvFoods.filter((f) => has(f.name)).map(lmvMatch),
    ];
    for (const match of matches) {
      match.uses = state.entries.filter(
        (e) => (match.foodId && e.foodId === match.foodId) || (match.lmvNumber && e.lmvNumber === match.lmvNumber),
      ).length;
    }
    matches.sort((a, b) => b.uses - a.uses);
    return HttpResponse.json({ matches, source: SOURCE });
  }),

  http.get(url("/foods"), () => HttpResponse.json(clone(state.foods))),

  http.post(url("/foods"), async ({ request }) => {
    const body = (await request.json()) as Body;
    const errors = unknownFields(body, foodFields, "body");
    if (errors.length) return problem(422, "invalid_request", errors.join(", "));
    const name = String(body.name).trim();
    if (state.foods.some((f) => f.name.toLowerCase() === name.toLowerCase() && f.brand === (body.brand ?? ""))) {
      return problem(409, "food_exists", "a food with that name and brand already exists");
    }
    const food: Food = {
      id: uuid(),
      name,
      brand: (body.brand as string) ?? "",
      gtin: (body.gtin as string) ?? null,
      per100g: body.per100g as Nutrients,
      notes: (body.notes as string) ?? "",
      portions: portionsOf(body.portions as { name: string; grams: number }[]),
      created: now(),
      updated: now(),
    };
    state.foods.push(food);
    return HttpResponse.json(clone(food), { status: 201 });
  }),

  http.post(url("/food-portions"), async ({ request }) => {
    const body = (await request.json()) as Body;
    const target = body.foodId
      ? state.foods.find((f) => f.id === body.foodId)
      : state.lmvFoods.find((f) => f.number === body.lmvNumber);
    if (!target) return problem(422, "unknown_food", "no such food");
    const existing = target.portions.find((p) => p.name.toLowerCase() === String(body.name).toLowerCase());
    const portion = existing ?? { id: uuid(), name: String(body.name), grams: 0 };
    portion.grams = Number(body.grams);
    if (!existing) target.portions.push(portion);
    return HttpResponse.json(clone(portion));
  }),

  http.get(url("/diary/:day"), ({ params }) => HttpResponse.json(dayOf(params.day as string))),

  http.post(url("/diary/:day/entries"), async ({ params, request }) => {
    const body = (await request.json()) as Body;
    const errors = unknownFields(body, ["entries"], "body");
    const inputs = (body.entries ?? []) as Body[];
    inputs.forEach((input, i) => errors.push(...unknownFields(input, entryFields, `body.entries[${i}]`)));
    if (errors.length) return problem(422, "invalid_request", errors.join(", "));

    const entries: Entry[] = [];
    for (const input of inputs) {
      const own = input.foodId ? state.foods.find((f) => f.id === input.foodId) : undefined;
      const lmv = input.lmvNumber ? state.lmvFoods.find((f) => f.number === input.lmvNumber) : undefined;
      const food = own ?? lmv;
      if (!food && !(input.name && input.per100g)) return problem(422, "unknown_food", "no such food");
      entries.push(
        toEntry({
          day: params.day as string,
          meal: input.meal as Entry["meal"],
          foodId: own?.id ?? null,
          lmvNumber: lmv?.number ?? null,
          name: own ? (own.brand ? `${own.name}, ${own.brand}` : own.name) : (lmv?.name ?? String(input.name)),
          grams: Number(input.grams),
          amount: (input.amount as string) ?? "",
          per100g: clone(food?.per100g ?? (input.per100g as Nutrients)),
        }),
      );
    }
    state.entries.push(...entries);
    return HttpResponse.json(dayOf(params.day as string), { status: 201 });
  }),

  http.put(url("/food-entries/:id"), async ({ params, request }) => {
    const entry = state.entries.find((e) => e.id === params.id);
    if (!entry) return problem(404, "not_found", "no such entry");
    const body = (await request.json()) as Body;
    const errors = unknownFields(body, ["day", "meal", "grams", "amount"], "body");
    if (errors.length) return problem(422, "invalid_request", errors.join(", "));
    Object.assign(entry, rescale({ ...entry, ...(body as Partial<Entry>) }), { id: entry.id });
    return HttpResponse.json(clone(entry));
  }),

  http.delete(url("/food-entries/:id"), ({ params }) => {
    const index = state.entries.findIndex((e) => e.id === params.id);
    if (index === -1) return problem(404, "not_found", "no such entry");
    state.entries.splice(index, 1);
    return noContent();
  }),

  http.post(url("/diary/:day/activities"), async ({ params, request }) => {
    const body = (await request.json()) as Body;
    const errors = unknownFields(body, ["label", "kcal", "workoutId"], "body");
    if (errors.length) return problem(422, "invalid_request", errors.join(", "));
    const activity: Activity = {
      id: uuid(),
      day: params.day as string,
      label: String(body.label),
      kcal: Number(body.kcal),
      workoutId: (body.workoutId as string) ?? null,
      loggedBy: "app",
      created: now(),
    };
    state.activities.push(activity);
    return HttpResponse.json(clone(activity), { status: 201 });
  }),

  http.delete(url("/activities/:id"), ({ params }) => {
    const index = state.activities.findIndex((a) => a.id === params.id);
    if (index === -1) return problem(404, "not_found", "no such activity");
    state.activities.splice(index, 1);
    return noContent();
  }),

  http.put(url("/diary/:day/weight"), async ({ params, request }) => {
    const body = (await request.json()) as Body;
    state.weights[params.day as string] = Number(body.weight);
    return noContent();
  }),

  http.delete(url("/diary/:day/weight"), ({ params }) => {
    if (!(params.day as string in state.weights)) return problem(404, "not_found", "no such weight");
    delete state.weights[params.day as string];
    return noContent();
  }),

  http.get(url("/nutrition-goal"), () =>
    state.goal ? HttpResponse.json(clone(state.goal)) : problem(404, "not_found", "no goal is set"),
  ),

  http.put(url("/nutrition-goal"), async ({ request }) => {
    const body = (await request.json()) as Body;
    const errors = unknownFields(body, goalFields, "body");
    if (errors.length) return problem(422, "invalid_request", errors.join(", "));
    state.goal = { ...(body as unknown as Goal), updated: now() };
    return HttpResponse.json(clone(state.goal));
  }),
];

export const foodBackend = {
  get lmvFoods() {
    return state.lmvFoods;
  },
  get foods() {
    return state.foods;
  },
  get entries() {
    return state.entries;
  },
  get activities() {
    return state.activities;
  },
  get weights() {
    return state.weights;
  },
  get goal() {
    return state.goal;
  },

  seed(seed: FoodBackendSeed) {
    state.lmvFoods.push(
      ...(seed.lmvFoods ?? []).map((f) => ({
        number: f.number,
        name: f.name,
        group: f.group ?? "",
        per100g: f.per100g,
        portions: portionsOf(f.portions),
      })),
    );
    state.foods.push(
      ...(seed.foods ?? []).map((f) => ({
        id: uuid(),
        brand: "",
        gtin: null,
        notes: "",
        created: now(),
        updated: now(),
        ...f,
        portions: portionsOf(f.portions),
      })),
    );
    state.entries.push(...(seed.entries ?? []).map(toEntry));
    state.activities.push(
      ...(seed.activities ?? []).map((a) => ({
        id: uuid(),
        workoutId: null,
        loggedBy: "app" as const,
        created: now(),
        ...a,
      })),
    );
    Object.assign(state.weights, seed.weights ?? {});
    if (seed.goal) {
      state.goal = {
        direction: "maintain",
        addActivities: true,
        targetWeight: null,
        weeklyChange: null,
        notes: "",
        updated: now(),
        ...seed.goal,
      };
    }
  },

  reset() {
    state.lmvFoods = [];
    state.foods = [];
    state.entries = [];
    state.activities = [];
    state.weights = {};
    state.goal = null;
  },
};
