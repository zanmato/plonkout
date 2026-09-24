import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { flushPromises, mount, type VueWrapper } from "@vue/test-utils";
import PrimeVue from "primevue/config";
import Plan from "@/views/Plan.vue";
import { backend, type PlanSeed } from "../mocks/backend";
import { mockPush } from "../setup";

const mockShowError = vi.fn();

vi.mock("@/composables/useToast", () => ({
  useToast: () => ({
    showSuccess: vi.fn(),
    showError: mockShowError,
    showInfo: vi.fn(),
  }),
}));

const IDS = {
  plan: "10000000-0000-4000-8000-000000000001",
  a: "20000000-0000-4000-8000-000000000001",
  b: "20000000-0000-4000-8000-000000000002",
  c: "20000000-0000-4000-8000-000000000003",
  pull: "30000000-0000-4000-8000-000000000001",
  top: "40000000-0000-4000-8000-000000000001",
  backOff: "40000000-0000-4000-8000-000000000002",
};

const peakBlock = (): PlanSeed => ({
  id: IDS.plan,
  name: "Peak block",
  goal: "Pull 150 on the table",
  sessions: [
    {
      id: IDS.a,
      label: "W3 A",
      week: 3,
      day: "A",
      intensity: "heavy",
      exercises: [
        {
          id: IDS.pull,
          exercise: "Table Pull",
          offArmPercent: 82.5,
          notes: "Keep the wrist cupped",
          targets: [
            { id: IDS.top, setType: "Top set", sets: 1, reps: 5, weight: 145, rpeMin: 8, rpeMax: 9 },
            { id: IDS.backOff, setType: "Back-off", sets: 3, reps: 5, weight: 130, rpeMin: 7, rpeMax: 8 },
          ],
        },
      ],
    },
    {
      id: IDS.b,
      label: "W3 B",
      week: 3,
      day: "B",
      intensity: "light",
      exercises: [{ exercise: "Pronation", targets: [{ setType: "Volume", sets: 4, reps: 12, weight: 20 }] }],
    },
    {
      id: IDS.c,
      label: "W4 A",
      week: 4,
      day: "A",
      exercises: [{ exercise: "Pronation", targets: [{ setType: "Volume", sets: 4, reps: 10, weight: 22 }] }],
    },
  ],
});

describe("Plan.vue", () => {
  let wrapper: VueWrapper;

  const createWrapper = async () => {
    wrapper = mount(Plan, { global: { plugins: [PrimeVue] } });
    await flushPromises();
    return wrapper;
  };

  const byTestId = (id: string) => wrapper.find(`[data-testid="${id}"]`);

  beforeEach(() => {
    vi.clearAllMocks();
    backend.seed({
      exercises: [
        { name: "Table Pull", singleArm: true, muscleGroup: "Forearm" },
        { name: "Pronation", muscleGroup: "Forearm" },
      ],
      settings: { weightUnit: "kg" },
    });
  });

  afterEach(() => wrapper?.unmount());

  it("explains where plans come from when there is none", async () => {
    await createWrapper();
    expect(byTestId("plan-empty").text()).toContain("No plan yet");
    expect(byTestId("plan-empty").text()).toContain("Ask your AI assistant");
    expect(byTestId("next-up").exists()).toBe(false);
  });

  it("shows the next session with its targets and cues", async () => {
    backend.seed({ plans: [peakBlock()] });
    await createWrapper();

    const nextUp = byTestId("next-up").text();
    expect(wrapper.text()).toContain("Peak block");
    expect(nextUp).toContain("W3 A");
    expect(nextUp).toContain("Week 3 · Day A");
    expect(nextUp).toContain("Heavy");
    expect(nextUp).toContain("Table Pull");
    expect(nextUp).toContain("Top set");
    expect(nextUp).toContain("1 × 5 @ 145 kg, RPE 8 to 9");
    expect(nextUp).toContain("3 × 5 @ 130 kg, RPE 7 to 8");
    expect(nextUp).toContain("Off arm 82.5%");
    expect(nextUp).toContain("Keep the wrist cupped");
    expect(byTestId("start-session").text()).toContain("Start");

    const comingUp = wrapper.findAll('[data-testid="coming-up"]');
    expect(comingUp).toHaveLength(2);
    expect(comingUp[0]!.text()).toContain("W3 B");
    expect(comingUp[1]!.text()).toContain("W4 A");
  });

  it("starts the next session and opens its workout", async () => {
    backend.seed({ plans: [peakBlock()] });
    await createWrapper();

    await byTestId("start-session").trigger("click");
    await flushPromises();

    expect(backend.workouts).toHaveLength(1);
    const workout = backend.workouts[0]!;
    expect(mockPush).toHaveBeenCalledWith({ name: "workout-edit", params: { id: workout.id } });
    expect(backend.session(IDS.a).status).toBe("in_progress");
    expect(backend.session(IDS.a).workoutId).toBe(workout.id);

    // One set per arm for each prescribed set, dominant arm first
    expect(workout.plannedSessionId).toBe(IDS.a);
    const sets = workout.exercises[0]!.sets;
    expect(sets).toHaveLength(8);
    expect(sets.slice(0, 2).map((set) => [set.arm, set.targetId, set.targetSeq])).toEqual([
      ["right", IDS.top, 1],
      ["left", IDS.top, 1],
    ]);
  });

  it("offers to continue a session in progress, without skipping it", async () => {
    const plan = peakBlock();
    plan.sessions![0]!.status = "in_progress";
    backend.seed({ plans: [plan] });
    await createWrapper();

    expect(byTestId("next-up").text()).toContain("In progress");
    expect(byTestId("start-session").text()).toContain("Continue");
    expect(byTestId("next-up").find('[data-testid="skip-session"]').exists()).toBe(false);
  });

  it("starts a session further down the queue", async () => {
    backend.seed({ plans: [peakBlock()] });
    await createWrapper();

    const card = wrapper.findAll('[data-testid="coming-up"]')[0]!;
    expect(card.text()).not.toContain("Volume");
    await card.find("button").trigger("click");
    expect(card.text()).toContain("4 × 12 @ 20 kg");

    await card.find('[data-testid="start-session"]').trigger("click");
    await flushPromises();

    expect(backend.session(IDS.b).status).toBe("in_progress");
    expect(mockPush).toHaveBeenCalledWith({
      name: "workout-edit",
      params: { id: backend.session(IDS.b).workoutId },
    });
  });

  it("skips a session with a reason and puts it back", async () => {
    backend.seed({ plans: [peakBlock()] });
    await createWrapper();

    await byTestId("next-up").find('[data-testid="skip-session"]').trigger("click");
    await byTestId("skip-reason").setValue("Elbow is sore");
    await byTestId("confirm-skip").trigger("click");
    await flushPromises();

    expect(backend.session(IDS.a).status).toBe("skipped");
    expect(backend.session(IDS.a).skipReason).toBe("Elbow is sore");
    expect(byTestId("next-up").text()).toContain("W3 B");

    await byTestId("done-toggle").trigger("click");
    const done = byTestId("done-session");
    expect(done.text()).toContain("W3 A");
    expect(done.text()).toContain("Skipped");
    expect(done.text()).toContain("Elbow is sore");

    await byTestId("put-back").trigger("click");
    await flushPromises();

    expect(backend.session(IDS.a).status).toBe("pending");
    expect(byTestId("next-up").text()).toContain("W3 A");
  });

  it("shows how completed sessions went and opens their workout", async () => {
    const plan = peakBlock();
    plan.sessions![0]!.status = "completed";
    plan.sessions![0]!.completedAt = "2026-09-20T10:00:00Z";
    backend.seed({ plans: [plan] });
    const [workout] = backend.seed({
      workouts: [
        {
          name: "Peak block W3 A",
          ended: "2026-09-20T10:00:00Z",
          plannedSessionId: IDS.a,
          exercises: [
            {
              name: "Table Pull",
              singleArm: true,
              plannedExerciseId: IDS.pull,
              sets: [
                // The top set is met on both arms
                { targetId: IDS.top, targetSeq: 1, arm: "right", weight: 145, reps: 5, rpe: 9 },
                { targetId: IDS.top, targetSeq: 1, arm: "left", weight: 120, reps: 5 },
                // Only one of three back-off sets was done
                { targetId: IDS.backOff, targetSeq: 1, arm: "right", weight: 130, reps: 5 },
                { targetId: IDS.backOff, targetSeq: 1, arm: "left", weight: 107.5, reps: 5 },
              ],
            },
          ],
        },
      ],
    }).workouts;
    await createWrapper();

    await byTestId("done-toggle").trigger("click");
    const chip = byTestId("compliance-chip");
    expect(chip.text()).toBe("1/2 targets met");
    expect(chip.classes()).toContain("bg-amber-300");

    await byTestId("done-session").trigger("click");
    expect(mockPush).toHaveBeenCalledWith({ name: "workout-edit", params: { id: workout!.id } });
  });

  it("colors the chip green when every target was met", async () => {
    const plan = peakBlock();
    plan.sessions![1]!.status = "completed";
    backend.seed({ plans: [plan] });
    backend.seed({
      workouts: [
        {
          ended: "2026-09-21T10:00:00Z",
          plannedSessionId: IDS.b,
          exercises: [
            {
              name: "Pronation",
              plannedExerciseId: backend.plans[0]!.sessions[1]!.exercises[0]!.id,
              sets: [1, 2, 3, 4].map((seq) => ({
                targetId: backend.plans[0]!.sessions[1]!.exercises[0]!.targets[0]!.id,
                targetSeq: seq,
                weight: 20,
                reps: 12,
              })),
            },
          ],
        },
      ],
    });
    await createWrapper();

    await byTestId("done-toggle").trigger("click");
    expect(byTestId("compliance-chip").text()).toBe("1/1 targets met");
    expect(byTestId("compliance-chip").classes()).toContain("bg-green-400");
  });

  it("lets the user pick between active plans", async () => {
    backend.seed({
      plans: [
        { name: "Old block", status: "archived", sessions: [{ label: "X", exercises: [] }] },
        peakBlock(),
        { name: "Grip", created: "2027-01-01T00:00:00Z", sessions: [{ label: "G1", exercises: [] }] },
      ],
    });
    await createWrapper();

    const picker = byTestId("plan-picker");
    expect(picker.exists()).toBe(true);
    expect(picker.text()).toContain("Grip");
    expect(picker.text()).not.toContain("Old block");
    // The plan whose session is first in the queue is picked first
    expect(byTestId("next-up").text()).toContain("W3 A");

    const grip = picker.findAll("button").find((button) => button.text().includes("Grip"))!;
    await grip.trigger("click");
    await flushPromises();
    expect(byTestId("next-up").text()).toContain("G1");
  });
});
