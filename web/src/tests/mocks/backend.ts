/**
 * An in-memory stand-in for the Go server. Tests seed it, let the app talk
 * to it over MSW like it would to the real API, and then look at what got
 * stored. It follows the server's contract closely enough for the app: uuid
 * ids, revisions on workouts, problem documents on errors and the same
 * ordering of lists.
 */
import { http, HttpResponse, type HttpHandler } from "msw";
import type {
  ActualSet,
  Comparison,
  Exercise,
  ExerciseComparison,
  Me,
  Plan,
  PlannedExercise,
  Problem,
  QueueEntry,
  Result,
  Session,
  Target,
  TargetComparison,
  Template,
  Workout,
  WorkoutExercise,
  WorkoutSet,
} from "@/api/gen/types.gen";
import type { DateLike } from "@/types/domain";

/** Origin the API is served from in tests. Node's fetch needs absolute URLs. */
export const API = "http://localhost";

// Seeds take the fields a test cares about and fill in the rest.

export type SetSeed = Partial<WorkoutSet>;

export type WorkoutExerciseSeed = Partial<Omit<WorkoutExercise, "sets">> & {
  name: string;
  sets?: SetSeed[];
};

export type WorkoutSeed = Partial<
  Omit<Workout, "started" | "ended" | "created" | "updated" | "exercises">
> & {
  started?: DateLike;
  ended?: DateLike | null;
  created?: DateLike;
  updated?: DateLike;
  exercises?: WorkoutExerciseSeed[];
};

export type ExerciseSeed = Partial<Omit<Exercise, "created">> & {
  name: string;
  created?: DateLike;
};

export type TemplateSeed = Partial<
  Omit<Template, "created" | "updated" | "exercises">
> & {
  created?: DateLike;
  updated?: DateLike;
  exercises?: WorkoutExerciseSeed[];
};

export type TargetSeed = Partial<Target> & { setType: string };

export type PlannedExerciseSeed = Partial<Omit<PlannedExercise, "targets">> & {
  exercise: string;
  targets?: TargetSeed[];
};

/**
 * A session's workout is the stored workout whose plannedSessionId points at
 * it, like the server's join, so seed that workout rather than workoutId.
 */
export type SessionSeed = Partial<
  Omit<Session, "exercises" | "completedAt" | "comparison" | "workoutId" | "planId" | "position">
> & {
  label: string;
  completedAt?: DateLike | null;
  exercises?: PlannedExerciseSeed[];
};

export type PlanSeed = Partial<Omit<Plan, "sessions" | "created" | "updated">> & {
  name: string;
  created?: DateLike;
  updated?: DateLike;
  sessions?: SessionSeed[];
};

export interface Seed {
  workouts?: WorkoutSeed[];
  exercises?: ExerciseSeed[];
  templates?: TemplateSeed[];
  plans?: PlanSeed[];
  settings?: Record<string, unknown>;
}

const defaultMe = (): Me => ({
  id: "00000000-0000-4000-8000-000000000001",
  username: "tester",
  created: "2026-01-01T00:00:00.000Z",
  passkeys: 1,
  recoveryCodesLeft: 10,
});

const state = {
  workouts: [] as Workout[],
  exercises: [] as Exercise[],
  templates: [] as Template[],
  // Sessions are stored without workoutId and comparison, both are derived
  plans: [] as Plan[],
  settings: {} as Record<string, unknown>,
  me: defaultMe(),
};

const iso = (date: DateLike) => new Date(date).toISOString();
const now = () => new Date().toISOString();
const uuid = () => crypto.randomUUID();
const clone = <T>(value: T): T => structuredClone(value);

function toSet(set: SetSeed): WorkoutSet {
  return {
    type: "regular",
    weight: null,
    distance: null,
    reps: null,
    time: "",
    rpe: null,
    arm: "",
    notes: "",
    ...set,
  };
}

function toWorkoutExercise(exercise: WorkoutExerciseSeed): WorkoutExercise {
  return {
    muscleGroup: "",
    type: "strength",
    displayType: "reps",
    singleArm: false,
    intensity: null,
    ...exercise,
    sets: (exercise.sets ?? []).map(toSet),
  };
}

/** Links logged exercises to the exercise list by name, like the server. */
function linkExercises(exercises: WorkoutExercise[]): WorkoutExercise[] {
  return exercises.map((exercise) => {
    if (exercise.exerciseId) return exercise;
    const match = findExerciseByName(exercise.name);
    return match ? { ...exercise, exerciseId: match.id } : exercise;
  });
}

function findExerciseByName(name: string, exceptId?: string) {
  const key = name.trim().toLowerCase();
  return state.exercises.find(
    (exercise) => exercise.id !== exceptId && exercise.name.trim().toLowerCase() === key,
  );
}

function toWorkout(seed: WorkoutSeed): Workout {
  const stamp = now();
  return {
    name: "",
    notes: "",
    revision: 1,
    ...seed,
    id: seed.id ?? uuid(),
    started: iso(seed.started ?? stamp),
    ended: seed.ended ? iso(seed.ended) : null,
    created: seed.created ? iso(seed.created) : stamp,
    updated: seed.updated ? iso(seed.updated) : stamp,
    exercises: linkExercises((seed.exercises ?? []).map(toWorkoutExercise)),
  };
}

function toExercise(seed: ExerciseSeed): Exercise {
  return {
    muscleGroup: "",
    type: "strength",
    displayType: "reps",
    singleArm: false,
    archived: false,
    ...seed,
    id: seed.id ?? uuid(),
    created: seed.created ? iso(seed.created) : now(),
  };
}

function toTemplate(seed: TemplateSeed): Template {
  const stamp = now();
  return {
    name: "",
    notes: "",
    ...seed,
    id: seed.id ?? uuid(),
    created: seed.created ? iso(seed.created) : stamp,
    updated: seed.updated ? iso(seed.updated) : stamp,
    exercises: (seed.exercises ?? []).map(toWorkoutExercise),
  };
}

function toTarget(seed: TargetSeed): Target {
  return {
    notes: "",
    reps: null,
    rpeMax: null,
    rpeMin: null,
    sets: 1,
    time: "",
    weight: null,
    ...seed,
    id: seed.id ?? uuid(),
  };
}

/** Plans name exercises, which are linked to the exercise list like the server does. */
function toPlannedExercise(seed: PlannedExerciseSeed): PlannedExercise {
  const match = findExerciseByName(seed.exercise);
  return {
    exerciseId: match?.id ?? uuid(),
    muscleGroup: match?.muscleGroup ?? "",
    type: match?.type ?? "strength",
    displayType: match?.displayType ?? "reps",
    singleArm: match?.singleArm ?? false,
    notes: "",
    offArmPercent: null,
    ...seed,
    id: seed.id ?? uuid(),
    targets: (seed.targets ?? []).map(toTarget),
  };
}

function toSession(seed: SessionSeed, planId: string, position: number): Session {
  return {
    week: null,
    day: null,
    intensity: null,
    notes: "",
    status: "pending",
    skipReason: "",
    ...seed,
    id: seed.id ?? uuid(),
    planId,
    position,
    completedAt: seed.completedAt ? iso(seed.completedAt) : null,
    // Derived on the way out, see withWorkout
    workoutId: "",
    exercises: (seed.exercises ?? []).map(toPlannedExercise),
  };
}

function toPlan(seed: PlanSeed): Plan {
  const stamp = now();
  const id = seed.id ?? uuid();
  return {
    goal: "",
    notes: "",
    status: "active",
    startDate: null,
    ...seed,
    id,
    created: seed.created ? iso(seed.created) : stamp,
    updated: seed.updated ? iso(seed.updated) : stamp,
    sessions: (seed.sessions ?? []).map((session, i) => toSession(session, id, i + 1)),
  };
}

function findSession(id: unknown): { plan: Plan; session: Session } | undefined {
  for (const plan of state.plans) {
    const session = plan.sessions.find((s) => s.id === id);
    if (session) return { plan, session };
  }
  return undefined;
}

const workoutOfSession = (sessionId: string) =>
  state.workouts.find((w) => w.plannedSessionId === sessionId);

/** A session as the server answers with it. The server sends null until it is started. */
function withWorkout(session: Session, actuals = false): Session {
  const workout = workoutOfSession(session.id);
  const out: Session = { ...clone(session), workoutId: (workout?.id ?? null) as string };
  if (actuals && workout) out.comparison = compare(session, workout);
  return out;
}

function planOut(plan: Plan, actuals = false): Plan {
  return {
    ...clone(plan),
    sessions: [...plan.sessions].sort((a, b) => a.position - b.position).map((s) => withWorkout(s, actuals)),
  };
}

// Planned versus done, a port of the server's compare.go.

const weightTolerance = 0.5;

const epley1RM = (weight: number, reps: number) =>
  weight <= 0 || reps <= 0 ? 0 : weight * (1 + Math.min(reps, 12) / 30);

interface LoggedSet extends WorkoutSet {
  name: string;
  plannedExerciseId?: string;
}

const isDone = (set: LoggedSet) => (set.reps ?? 0) > 0 || (set.weight ?? 0) > 0 || set.time !== "";

function actual(set: LoggedSet): ActualSet {
  return {
    weight: set.weight,
    reps: set.reps,
    time: set.time,
    rpe: set.rpe,
    arm: set.arm,
    type: set.type,
    ...(set.targetSeq !== undefined ? { targetSeq: set.targetSeq } : {}),
  };
}

function compareTarget(target: Target, planned: PlannedExercise, sets: LoggedSet[]): TargetComparison {
  const out: TargetComparison = { target: clone(target), sets: sets.map(actual), done: 0, met: false, topRpe: null };
  const bySeq = new Map<number, LoggedSet[]>();
  sets.forEach((set, i) => {
    const seq = set.targetSeq ?? -1 - i;
    bySeq.set(seq, [...(bySeq.get(seq) ?? []), set]);
    if (set.rpe !== null && (out.topRpe === null || set.rpe > out.topRpe)) out.topRpe = set.rpe;
  });

  let met = true;
  for (const group of bySeq.values()) {
    // A number counts once every arm's set of it has something logged
    if (group.some((set) => !isDone(set))) {
      met = false;
      continue;
    }
    out.done++;
    // The heaviest set answers for the dominant arm, the rest for the off arm
    group.sort((a, b) => (b.weight ?? -Infinity) - (a.weight ?? -Infinity));
    group.forEach((set, i) => {
      if (target.reps !== null && (set.reps === null || set.reps < target.reps)) met = false;
      if (target.weight !== null) {
        let expected = target.weight;
        if (i > 0 && planned.offArmPercent !== null) expected = (expected * planned.offArmPercent) / 100;
        if (set.weight === null || set.weight + weightTolerance < expected) met = false;
      }
    });
  }
  out.met = met && out.done >= target.sets;
  return out;
}

function compare(session: Session, workout: Workout): Comparison {
  const sets: LoggedSet[] = workout.exercises.flatMap((exercise) =>
    exercise.sets.map((set) => ({ ...set, name: exercise.name, plannedExerciseId: exercise.plannedExerciseId })),
  );
  const unplanned = [...new Set(sets.filter((set) => !set.plannedExerciseId).map((set) => set.name))];
  const exercises = session.exercises.map((planned): ExerciseComparison => {
    const logged = sets.filter((set) => set.plannedExerciseId === planned.id);
    const best = Math.max(
      0,
      ...logged
        .filter((set) => set.type === "regular" && set.weight !== null && set.reps !== null)
        .map((set) => epley1RM(set.weight!, set.reps!)),
    );
    return {
      plannedExerciseId: planned.id,
      exercise: planned.exercise,
      extraSets: logged.filter((set) => !set.targetId).map(actual),
      bestEstimated1RM: best > 0 ? Math.round(best * 10) / 10 : null,
      targets: planned.targets.map((target) =>
        compareTarget(target, planned, logged.filter((set) => set.targetId === target.id)),
      ),
    };
  });
  return { exercises, unplanned };
}

/** The workout a session starts, every prescribed set empty and linked to its target. */
function prefill(plan: Plan, session: Session): WorkoutSeed {
  const dominant = state.settings.dominantArm === "left" ? "left" : "right";
  const offArm = dominant === "left" ? "right" : "left";
  return {
    name: session.label ? `${plan.name} ${session.label}`.trim() : plan.name,
    notes: session.notes,
    started: now(),
    plannedSessionId: session.id,
    exercises: session.exercises.map((planned) => ({
      exerciseId: planned.exerciseId,
      name: planned.exercise,
      muscleGroup: planned.muscleGroup,
      type: planned.type,
      displayType: planned.displayType,
      singleArm: planned.singleArm,
      intensity: session.intensity,
      plannedExerciseId: planned.id,
      sets: planned.targets.flatMap((target) =>
        Array.from({ length: target.sets }, (_, i) => {
          const set = { targetId: target.id, targetSeq: i + 1 };
          return planned.singleArm
            ? [{ ...set, arm: dominant } as const, { ...set, arm: offArm } as const]
            : [set];
        }).flat(),
      ),
    })),
  };
}

/** A planned session follows its workout: in progress until it ends, completed after. */
function syncSession(workout: Workout) {
  const found = workout.plannedSessionId ? findSession(workout.plannedSessionId) : undefined;
  if (!found || found.session.status === "skipped") return;
  found.session.status = workout.ended ? "completed" : "in_progress";
  found.session.completedAt = workout.ended;
}

// Ordering matches the server's queries.

const byStartedDesc = (a: Workout, b: Workout) =>
  new Date(b.started).getTime() - new Date(a.started).getTime();

const byMuscleGroupThenName = (a: Exercise, b: Exercise) =>
  a.muscleGroup.localeCompare(b.muscleGroup) || a.name.localeCompare(b.name);

const byCreatedDesc = (a: Template, b: Template) =>
  new Date(b.created).getTime() - new Date(a.created).getTime();

// Responses

const titles: Record<number, string> = {
  404: "Not Found",
  409: "Conflict",
  422: "Unprocessable Entity",
};

function problem(status: number, code: string, detail: string) {
  const body: Problem = { type: "about:blank", title: titles[status] ?? "Error", status, code, detail };
  return HttpResponse.json(body, {
    status,
    headers: { "Content-Type": "application/problem+json" },
  });
}

const notFound = (what: string) => problem(404, "not_found", `${what} not found`);
const noContent = () => new HttpResponse(null, { status: 204 });

// The server refuses fields it does not know, so the fake does too. That keeps
// editor only state from leaking into request bodies unnoticed.

const setFields = ["type", "weight", "distance", "reps", "time", "rpe", "arm", "notes", "targetId", "targetSeq"];
const exerciseFields = [
  "exerciseId",
  "name",
  "muscleGroup",
  "type",
  "displayType",
  "singleArm",
  "intensity",
  "notes",
  "plannedExerciseId",
  "sets",
];
const workoutFields = ["name", "started", "ended", "notes", "exercises"];
const templateFields = ["name", "notes", "exercises"];
const exerciseInputFields = ["name", "muscleGroup", "type", "displayType", "singleArm", "archived"];

type Body = Record<string, unknown>;

function unknownFields(body: Body, allowed: string[], at: string): string[] {
  return Object.keys(body)
    .filter((key) => !allowed.includes(key))
    .map((key) => `${at}.${key}`);
}

function checkLoggedExercises(exercises: unknown, at: string): string[] {
  if (!Array.isArray(exercises)) return [`${at}.exercises must be an array`];
  return exercises.flatMap((exercise: Body, i) => {
    const where = `${at}.exercises[${i}]`;
    const errors = unknownFields(exercise, exerciseFields, where);
    for (const field of ["name", "muscleGroup", "type", "displayType", "singleArm", "intensity", "sets"]) {
      if (!(field in exercise)) errors.push(`${where}.${field} is required`);
    }
    if (typeof exercise.name !== "string" || !exercise.name.trim()) errors.push(`${where}.name must not be empty`);
    const sets = Array.isArray(exercise.sets) ? (exercise.sets as Body[]) : [];
    sets.forEach((set, j) => errors.push(...unknownFields(set, setFields, `${where}.sets[${j}]`)));
    return errors;
  });
}

function checkWorkout(body: Body, withRevision: boolean): string[] {
  const allowed = withRevision ? [...workoutFields, "revision"] : workoutFields;
  const errors = unknownFields(body, allowed, "body");
  for (const field of allowed) {
    if (!(field in body)) errors.push(`body.${field} is required`);
  }
  if (typeof body.started !== "string" || Number.isNaN(Date.parse(body.started))) {
    errors.push("body.started must be a date-time");
  }
  return [...errors, ...checkLoggedExercises(body.exercises, "body")];
}

function checkTemplate(body: Body): string[] {
  const errors = unknownFields(body, templateFields, "body");
  for (const field of templateFields) {
    if (!(field in body)) errors.push(`body.${field} is required`);
  }
  return [...errors, ...checkLoggedExercises(body.exercises, "body")];
}

function checkExercise(body: Body): string[] {
  const errors = unknownFields(body, exerciseInputFields, "body");
  if (typeof body.name !== "string" || !body.name.trim()) errors.push("body.name must not be empty");
  return errors;
}

const invalid = (errors: string[]) => problem(422, "invalid_request", errors.join(", "));

async function readBody(request: Request): Promise<Body> {
  return (await request.json()) as Body;
}

const url = (path: string) => `${API}/api${path}`;

export const handlers: HttpHandler[] = [
  // Workouts. "latest" comes before ":id" so it is not taken for an id.
  http.get(url("/workouts"), ({ request }) => {
    const params = new URL(request.url).searchParams;
    const from = params.get("from");
    const to = params.get("to");
    const workouts = state.workouts
      .filter((w) => !from || new Date(w.started) >= new Date(from))
      .filter((w) => !to || new Date(w.started) < new Date(to))
      .sort(byStartedDesc);
    return HttpResponse.json(clone(workouts));
  }),

  http.get(url("/workouts/latest"), ({ request }) => {
    const params = new URL(request.url).searchParams;
    const name = params.get("name");
    const excludeId = params.get("excludeId");
    if (!name) return invalid(["query.name is required"]);
    const latest = state.workouts
      .filter((w) => w.name === name && w.id !== excludeId)
      .sort(byStartedDesc)[0];
    return latest ? HttpResponse.json(clone(latest)) : notFound("workout");
  }),

  http.get(url("/workouts/:id"), ({ params }) => {
    const workout = state.workouts.find((w) => w.id === params.id);
    return workout ? HttpResponse.json(clone(workout)) : notFound("workout");
  }),

  http.post(url("/workouts"), async ({ request }) => {
    const body = await readBody(request);
    const errors = checkWorkout(body, false);
    if (errors.length) return invalid(errors);
    const workout = toWorkout(body as WorkoutSeed);
    state.workouts.push(workout);
    return HttpResponse.json(clone(workout), { status: 201 });
  }),

  http.put(url("/workouts/:id"), async ({ params, request }) => {
    const index = state.workouts.findIndex((w) => w.id === params.id);
    if (index === -1) return notFound("workout");
    const body = await readBody(request);
    const errors = checkWorkout(body, true);
    if (errors.length) return invalid(errors);
    const current = state.workouts[index]!;
    if (body.revision !== current.revision) {
      return problem(409, "stale_revision", "the workout was changed since it was loaded");
    }
    const { revision: _revision, ...fields } = body;
    const updated = toWorkout({
      ...(fields as WorkoutSeed),
      id: current.id,
      created: current.created,
      updated: now(),
      revision: current.revision + 1,
      plannedSessionId: current.plannedSessionId,
    });
    state.workouts[index] = updated;
    syncSession(updated);
    return HttpResponse.json(clone(updated));
  }),

  http.delete(url("/workouts/:id"), ({ params }) => {
    const index = state.workouts.findIndex((w) => w.id === params.id);
    if (index === -1) return notFound("workout");
    const [deleted] = state.workouts.splice(index, 1);
    // The session it was logged for goes back in the queue
    const found = deleted!.plannedSessionId ? findSession(deleted!.plannedSessionId) : undefined;
    if (found && (found.session.status === "in_progress" || found.session.status === "completed")) {
      found.session.status = "pending";
      found.session.completedAt = null;
    }
    return noContent();
  }),

  // Exercises

  http.get(url("/exercises"), () =>
    HttpResponse.json(clone([...state.exercises].sort(byMuscleGroupThenName))),
  ),

  http.post(url("/exercises"), async ({ request }) => {
    const body = await readBody(request);
    const errors = checkExercise(body);
    if (errors.length) return invalid(errors);
    if (findExerciseByName(body.name as string)) {
      return problem(409, "exercise_exists", "an exercise with that name already exists");
    }
    const exercise = toExercise({ ...(body as ExerciseSeed), name: (body.name as string).trim() });
    state.exercises.push(exercise);
    return HttpResponse.json(clone(exercise), { status: 201 });
  }),

  http.put(url("/exercises/:id"), async ({ params, request }) => {
    const index = state.exercises.findIndex((e) => e.id === params.id);
    if (index === -1) return notFound("exercise");
    const body = await readBody(request);
    const errors = checkExercise(body);
    if (errors.length) return invalid(errors);
    if (findExerciseByName(body.name as string, params.id as string)) {
      return problem(409, "exercise_exists", "an exercise with that name already exists");
    }
    const current = state.exercises[index]!;
    const updated = toExercise({ ...current, ...(body as ExerciseSeed), id: current.id, created: current.created });
    state.exercises[index] = updated;
    return HttpResponse.json(clone(updated));
  }),

  // Templates

  http.get(url("/templates"), () =>
    HttpResponse.json(clone([...state.templates].sort(byCreatedDesc))),
  ),

  http.get(url("/templates/:id"), ({ params }) => {
    const template = state.templates.find((t) => t.id === params.id);
    return template ? HttpResponse.json(clone(template)) : notFound("template");
  }),

  http.post(url("/templates"), async ({ request }) => {
    const body = await readBody(request);
    const errors = checkTemplate(body);
    if (errors.length) return invalid(errors);
    const template = toTemplate(body as TemplateSeed);
    state.templates.push(template);
    return HttpResponse.json(clone(template), { status: 201 });
  }),

  http.put(url("/templates/:id"), async ({ params, request }) => {
    const index = state.templates.findIndex((t) => t.id === params.id);
    if (index === -1) return notFound("template");
    const body = await readBody(request);
    const errors = checkTemplate(body);
    if (errors.length) return invalid(errors);
    const current = state.templates[index]!;
    const updated = toTemplate({ ...(body as TemplateSeed), id: current.id, created: current.created, updated: now() });
    state.templates[index] = updated;
    return HttpResponse.json(clone(updated));
  }),

  http.delete(url("/templates/:id"), ({ params }) => {
    const index = state.templates.findIndex((t) => t.id === params.id);
    if (index === -1) return notFound("template");
    state.templates.splice(index, 1);
    return noContent();
  }),

  // Settings

  http.get(url("/settings"), () => HttpResponse.json(clone(state.settings))),

  http.put(url("/settings/:key"), async ({ params, request }) => {
    const key = params.key as string;
    if (!/^[A-Za-z0-9_.-]{1,64}$/.test(key)) return invalid(["path.key is invalid"]);
    const body = await readBody(request);
    if (!("value" in body)) return invalid(["body.value is required"]);
    state.settings[key] = clone(body.value);
    return noContent();
  }),

  // Import of the local only app's export: every workout becomes a new one.

  http.post(url("/import/legacy"), async ({ request }) => {
    const body = await readBody(request);
    if (!Array.isArray(body.workouts)) return invalid(["body.workouts is required"]);
    const before = state.exercises.length;
    for (const legacy of body.workouts as WorkoutSeed[]) {
      const exercises = (legacy.exercises ?? []).map((exercise) => {
        if (!findExerciseByName(exercise.name)) state.exercises.push(toExercise({ ...exercise, id: undefined }));
        return { ...exercise, exerciseId: undefined };
      });
      state.workouts.push(toWorkout({ ...legacy, id: undefined, revision: 1, exercises }));
    }
    const result: Result = {
      imported: body.workouts.length,
      skipped: 0,
      exercisesCreated: state.exercises.length - before,
    };
    return HttpResponse.json(result);
  }),

  // Plans

  http.get(url("/queue"), () => {
    const entries: QueueEntry[] = state.plans
      .filter((plan) => plan.status === "active")
      .sort((a, b) => new Date(a.created).getTime() - new Date(b.created).getTime())
      .flatMap((plan) =>
        [...plan.sessions]
          .sort((a, b) => a.position - b.position)
          .filter((session) => session.status === "pending" || session.status === "in_progress")
          .map((session) => ({ planName: plan.name, session: withWorkout(session) })),
      );
    return HttpResponse.json(entries);
  }),

  http.get(url("/plans"), ({ request }) => {
    const status = new URL(request.url).searchParams.get("status");
    const plans = state.plans
      .filter((plan) => !status || plan.status === status)
      .sort((a, b) => new Date(b.created).getTime() - new Date(a.created).getTime())
      .map((plan) => planOut(plan));
    return HttpResponse.json(plans);
  }),

  http.get(url("/plans/:id"), ({ params, request }) => {
    const plan = state.plans.find((p) => p.id === params.id);
    if (!plan) return notFound("plan");
    const actuals = new URL(request.url).searchParams.get("actuals") === "true";
    return HttpResponse.json(planOut(plan, actuals));
  }),

  http.get(url("/planned-sessions/:id"), ({ params }) => {
    const found = findSession(params.id);
    return found ? HttpResponse.json(withWorkout(found.session)) : notFound("session");
  }),

  http.get(url("/planned-sessions/:id/comparison"), ({ params }) => {
    const found = findSession(params.id);
    if (!found) return notFound("session");
    const workout = workoutOfSession(found.session.id);
    const empty: Comparison = { exercises: [], unplanned: [] };
    return HttpResponse.json(workout ? compare(found.session, workout) : empty);
  }),

  http.post(url("/planned-sessions/:id/start"), ({ params }) => {
    const found = findSession(params.id);
    if (!found) return notFound("session");
    // A double tap answers with the workout already logged
    const existing = workoutOfSession(found.session.id);
    if (existing) return HttpResponse.json({ workoutId: existing.id });
    if (found.session.status === "skipped") {
      return problem(409, "session_skipped", "the session was skipped, put it back before starting it");
    }
    const workout = toWorkout(prefill(found.plan, found.session));
    state.workouts.push(workout);
    found.session.status = "in_progress";
    return HttpResponse.json({ workoutId: workout.id });
  }),

  http.post(url("/planned-sessions/:id/status"), async ({ params, request }) => {
    const body = await readBody(request);
    const errors = unknownFields(body, ["status", "reason"], "body");
    if (errors.length) return invalid(errors);
    const found = findSession(params.id);
    if (!found) return notFound("session");
    const { plan, session } = found;
    switch (body.status) {
      case "skipped":
        if (session.status !== "pending") {
          return problem(409, "session_not_pending", `the session is ${session.status}, it cannot be skipped`);
        }
        session.status = "skipped";
        session.skipReason = typeof body.reason === "string" ? body.reason : "";
        break;
      case "pending":
        if (session.status !== "skipped" && session.status !== "pending") {
          return problem(409, "session_started", `the session is ${session.status}, only a skipped session can be put back`);
        }
        session.status = "pending";
        break;
      case "deleted":
        if (session.status !== "pending" && session.status !== "skipped") {
          return problem(409, "session_started", `the session is ${session.status} and has a workout, it cannot be deleted`);
        }
        plan.sessions.splice(plan.sessions.indexOf(session), 1);
        return HttpResponse.json({});
      default:
        return invalid([`body.status ${String(body.status)} is unknown`]);
    }
    return HttpResponse.json({ session: withWorkout(session) });
  }),

  // Account

  http.get(url("/account"), () => HttpResponse.json(clone(state.me))),
];

export const backend = {
  handlers,

  /** Stored workouts in insertion order. Mutate through seed or the app. */
  get workouts(): Workout[] {
    return state.workouts;
  },
  get exercises(): Exercise[] {
    return state.exercises;
  },
  get templates(): Template[] {
    return state.templates;
  },
  get plans(): Plan[] {
    return state.plans;
  },
  get settings(): Record<string, unknown> {
    return state.settings;
  },
  get me(): Me {
    return state.me;
  },
  set me(me: Me) {
    state.me = me;
  },

  /** Looks up a stored workout, failing the test when it is missing. */
  workout(id: string): Workout {
    const workout = state.workouts.find((w) => w.id === id);
    if (!workout) throw new Error(`no workout ${id} in the fake backend`);
    return workout;
  },

  /** Looks up a stored planned session, failing the test when it is missing. */
  session(id: string): Session {
    const found = findSession(id);
    if (!found) throw new Error(`no planned session ${id} in the fake backend`);
    return withWorkout(found.session);
  },

  /** Adds records, filling in ids, revisions and timestamps. Returns what was stored. */
  seed(seed: Seed) {
    const exercises = (seed.exercises ?? []).map(toExercise);
    state.exercises.push(...exercises);
    const workouts = (seed.workouts ?? []).map(toWorkout);
    state.workouts.push(...workouts);
    const templates = (seed.templates ?? []).map(toTemplate);
    state.templates.push(...templates);
    const plans = (seed.plans ?? []).map(toPlan);
    state.plans.push(...plans);
    Object.assign(state.settings, seed.settings ?? {});
    return { workouts, exercises, templates, plans };
  },

  /** Empties every store. */
  reset() {
    state.workouts = [];
    state.exercises = [];
    state.templates = [];
    state.plans = [];
    state.settings = {};
    state.me = defaultMe();
  },
};
