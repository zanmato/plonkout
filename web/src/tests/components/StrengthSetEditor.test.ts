import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import StrengthSetEditor from "@/components/StrengthSetEditor.vue";
import type { Target, WorkoutExercise, WorkoutSet } from "@/types/domain";

const target = (fields: Partial<Target> = {}): Target => ({
  id: "40000000-0000-4000-8000-000000000001",
  setType: "Top set",
  sets: 1,
  reps: 5,
  weight: 145,
  time: "",
  rpeMin: 8,
  rpeMax: 9,
  notes: "",
  ...fields,
});

const emptySet = (fields: Partial<WorkoutSet> = {}): WorkoutSet => ({
  type: "regular",
  weight: null,
  distance: null,
  reps: null,
  time: "",
  rpe: null,
  arm: "",
  notes: "",
  ...fields,
});

const exercise = (fields: Partial<WorkoutExercise> = {}): WorkoutExercise => ({
  name: "Table Pull",
  muscleGroup: "Forearm",
  type: "strength",
  displayType: "reps",
  singleArm: false,
  intensity: null,
  sets: [emptySet()],
  ...fields,
});

function createWrapper(props: {
  set?: WorkoutSet;
  exercise?: WorkoutExercise;
  target?: Target | null;
  targetWeight?: number | null;
}) {
  const set = props.set ?? emptySet();
  return mount(StrengthSetEditor, {
    props: {
      set,
      exercise: props.exercise ?? exercise({ sets: [set] }),
      exerciseIndex: 0,
      setIndex: 0,
      setNumber: "1",
      weightUnit: "kg",
      maxPercentage: "",
      target: props.target,
      targetWeight: props.targetWeight,
    },
    global: { stubs: { Popover: true } },
  });
}

describe("StrengthSetEditor.vue with a plan target", () => {
  it("shows nothing of a plan without a target", () => {
    const wrapper = createWrapper({});
    expect(wrapper.find('[data-testid="set-target"]').exists()).toBe(false);
    const inputs = wrapper.findAll('input[type="number"]');
    expect(inputs[0]!.attributes("placeholder")).toBe("0");
    expect(inputs[1]!.attributes("placeholder")).toBe("0");
  });

  it("labels the set with what the target asks for", () => {
    const wrapper = createWrapper({ target: target(), targetWeight: 145 });
    expect(wrapper.find('[data-testid="set-target"]').text()).toContain("Top set · 145 kg × 5 · RPE 8 to 9");
  });

  it("labels an off arm set with its own weight", () => {
    const wrapper = createWrapper({ target: target(), targetWeight: 119.5 });
    expect(wrapper.find('[data-testid="set-target"]').text()).toContain("Top set · 119.5 kg × 5");
  });

  it("uses the target as placeholders while nothing is logged", () => {
    const wrapper = createWrapper({ target: target(), targetWeight: 145 });
    const inputs = wrapper.findAll('input[type="number"]');
    expect(inputs[0]!.attributes("placeholder")).toBe("145");
    expect(inputs[1]!.attributes("placeholder")).toBe("5");
  });

  it("fills the set as planned in one tap", async () => {
    const wrapper = createWrapper({ target: target(), targetWeight: 119.5 });
    await wrapper.find('[data-testid="as-planned"]').trigger("click");
    expect(wrapper.emitted("update:weight")).toEqual([[119.5]]);
    expect(wrapper.emitted("update:reps")).toEqual([[5]]);
  });

  it("fills the time of a planned hold", async () => {
    const hold = exercise({ displayType: "time" });
    const wrapper = createWrapper({
      exercise: hold,
      target: target({ reps: null, time: "0:30", weight: 20 }),
      targetWeight: 20,
    });
    expect(wrapper.find('input[type="text"]').attributes("placeholder")).toBe("0:30");
    await wrapper.find('[data-testid="as-planned"]').trigger("click");
    expect(wrapper.emitted("update:weight")).toEqual([[20]]);
    expect(wrapper.emitted("update:time")).toEqual([["0:30"]]);
  });

  it("hides the as planned button once something is logged", () => {
    const logged = emptySet({ weight: 140 });
    const wrapper = createWrapper({ set: logged, target: target(), targetWeight: 145 });
    expect(wrapper.find('[data-testid="set-target"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="as-planned"]').exists()).toBe(false);

    const withReps = createWrapper({ set: emptySet({ reps: 5 }), target: target(), targetWeight: 145 });
    expect(withReps.find('[data-testid="as-planned"]').exists()).toBe(false);
  });
});
