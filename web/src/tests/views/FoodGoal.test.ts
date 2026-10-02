import { describe, it, expect, afterEach } from "vitest";
import { flushPromises, mount, type VueWrapper } from "@vue/test-utils";
import FoodGoal from "@/views/FoodGoal.vue";
import { backend } from "../mocks/backend";
import { mockPush } from "../setup";

describe("FoodGoal.vue", () => {
  let wrapper: VueWrapper;

  const createWrapper = async () => {
    wrapper = mount(FoodGoal);
    await flushPromises();
    return wrapper;
  };
  const byTestId = (id: string) => wrapper.find(`[data-testid="${id}"]`);
  const input = (id: string) => byTestId(id).element as HTMLInputElement;

  afterEach(() => wrapper?.unmount());

  it("sets a goal", async () => {
    await createWrapper();
    expect(byTestId("goal-save").attributes("disabled")).toBeDefined();

    await byTestId("direction-lose").trigger("click");
    await byTestId("goal-kcal").setValue("1900");
    await byTestId("goal-protein").setValue("140");
    await byTestId("goal-carbs").setValue("200");
    await byTestId("goal-fat").setValue("60");
    expect(byTestId("macro-sum").text()).toContain("1900 kcal");
    await wrapper.find("#goal-target").setValue("88");
    await wrapper.find("#goal-pace").setValue("-0,5");

    await byTestId("goal-form").trigger("submit");
    await flushPromises();

    expect(backend.food.goal).toMatchObject({
      direction: "lose", kcal: 1900, protein: 140, carbs: 200, fat: 60,
      addActivities: true, targetWeight: 88, weeklyChange: -0.5, notes: "",
    });
    expect(mockPush).toHaveBeenCalledWith({ name: "food" });
  });

  it("shows the goal already set, as an assistant left it", async () => {
    backend.seed({
      food: {
        goal: {
          direction: "gain", kcal: 3000, protein: 180, carbs: 380, fat: 85, addActivities: false,
          notes: "Maintenance about 2700, plus 300.",
        },
      },
    });
    await createWrapper();

    expect(byTestId("direction-gain").attributes("aria-pressed")).toBe("true");
    expect(input("goal-kcal").value).toBe("3000");
    expect(input("goal-add-activities").checked).toBe(false);
    expect((wrapper.find("#goal-notes").element as HTMLTextAreaElement).value).toBe("Maintenance about 2700, plus 300.");
  });

  it("refuses a budget below 800 kcal", async () => {
    await createWrapper();
    await byTestId("goal-kcal").setValue("600");
    await byTestId("goal-protein").setValue("50");
    await byTestId("goal-carbs").setValue("50");
    await byTestId("goal-fat").setValue("20");
    expect(byTestId("goal-save").attributes("disabled")).toBeDefined();
  });
});
