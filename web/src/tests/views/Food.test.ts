import { describe, it, expect, afterEach } from "vitest";
import { flushPromises, mount, type VueWrapper } from "@vue/test-utils";
import Food from "@/views/Food.vue";
import { backend } from "../mocks/backend";
import { mockReplace } from "../setup";
import { addDays, today } from "@/utils/day";
import type { Nutrients } from "@/types/domain";

const pasta: Nutrients = { kcal: 128, protein: 4.2, carbs: 25.8, fat: 0.5 };
const pesto: Nutrients = { kcal: 545, protein: 10.6, carbs: 1.9, fat: 55 };

const lmvFoods = [
  { number: 4065, name: "Pasta kokt u. salt", group: "Pasta, ris", per100g: pasta },
  { number: 6201, name: "Pesto hemlagad", group: "Såser", per100g: pesto, portions: [{ name: "tbsp", grams: 15 }] },
];
const goal = { direction: "lose" as const, kcal: 1900, protein: 97, carbs: 242, fat: 65 };

// Search waits for the person to stop typing.
const searchSettles = async () => {
  await new Promise((resolve) => setTimeout(resolve, 300));
  await flushPromises();
};

describe("Food.vue", () => {
  let wrapper: VueWrapper;

  const createWrapper = async () => {
    wrapper = mount(Food);
    await flushPromises();
    return wrapper;
  };
  const byTestId = (id: string) => wrapper.find(`[data-testid="${id}"]`);
  const allByTestId = (id: string) => wrapper.findAll(`[data-testid="${id}"]`);

  afterEach(() => wrapper?.unmount());

  it("shows the day against the goal", async () => {
    backend.seed({
      food: {
        lmvFoods,
        goal,
        entries: [
          { day: today(), meal: "lunch", name: "Pasta kokt u. salt", lmvNumber: 4065, grams: 100, per100g: pasta },
          { day: today(), meal: "lunch", name: "Pesto hemlagad", lmvNumber: 6201, grams: 30, amount: "2 tbsp", per100g: pesto, loggedBy: "assistant" },
          { day: addDays(today(), -1), meal: "dinner", name: "Pasta kokt u. salt", grams: 500, per100g: pasta },
        ],
        activities: [{ day: today(), label: "Armwrestling", kcal: 180 }],
        weights: { [today()]: 91.4 },
      },
    });
    await createWrapper();

    expect(byTestId("no-goal").exists()).toBe(false);
    // 128 + 163.5 eaten, 1900 + 180 budget
    expect(byTestId("eaten").text()).toBe("292");
    expect(byTestId("burned").text()).toBe("180");
    expect(byTestId("remaining").text()).toBe("1789");
    expect(byTestId("macro-fat").text()).toContain("17 / 65 g");

    const lunch = byTestId("meal-lunch");
    expect(lunch.find('[data-testid="meal-summary"]').text()).toBe("292 kcal · aim 570–665");
    const entries = lunch.findAll('[data-testid="entry"]');
    expect(entries).toHaveLength(2);
    expect(entries[1]!.text()).toContain("2 tbsp (30 g)");
    expect(entries[1]!.text()).toContain("Logged by your assistant");
    expect(byTestId("meal-dinner").find('[data-testid="meal-summary"]').text()).toBe("Aim 475–570 kcal");

    expect(byTestId("activities").text()).toContain("Armwrestling");
    expect(byTestId("weight").text()).toContain("91.4 kg");
  });

  it("asks for a goal until one is set", async () => {
    backend.seed({ food: { lmvFoods } });
    await createWrapper();

    expect(byTestId("no-goal").exists()).toBe(true);
    expect(byTestId("remaining").exists()).toBe(false);
    expect(byTestId("meal-lunch").find('[data-testid="meal-summary"]').text()).toBe("0 kcal");
  });

  it("adds a food by search, with a portion tapped twice", async () => {
    backend.seed({ food: { lmvFoods, goal } });
    await createWrapper();

    await byTestId("add-lunch").trigger("click");
    await byTestId("food-search").setValue("pesto");
    await searchSettles();

    const matches = allByTestId("food-match");
    expect(matches).toHaveLength(1);
    expect(wrapper.text()).toContain("Livsmedelsverkets livsmedelsdatabas");
    await matches[0]!.trigger("click");

    // The one portion is picked to start with, a second tap makes it two.
    const tbsp = wrapper.findAll("button").find((b) => b.text() === "tbsp 15 g")!;
    await tbsp.trigger("click");
    expect((byTestId("grams").element as HTMLInputElement).value).toBe("30");

    await byTestId("add-food").trigger("click");
    await flushPromises();

    expect(backend.food.entries).toHaveLength(1);
    expect(backend.food.entries[0]).toMatchObject({ meal: "lunch", lmvNumber: 6201, grams: 30, amount: "2 tbsp", day: today() });
    expect(byTestId("food-search").exists()).toBe(false);
    expect(byTestId("eaten").text()).toBe("164");
  });

  it("saves an amount as a portion of the food", async () => {
    backend.seed({ food: { lmvFoods: [lmvFoods[0]!] } });
    await createWrapper();

    await byTestId("add-lunch").trigger("click");
    await byTestId("food-search").setValue("pasta");
    await searchSettles();
    await allByTestId("food-match")[0]!.trigger("click");

    await byTestId("grams").setValue("250");
    await byTestId("portion-name").setValue("plate");
    await byTestId("save-portion").trigger("click");
    await flushPromises();

    expect(backend.food.lmvFoods[0]!.portions).toMatchObject([{ name: "plate", grams: 250 }]);
    // Saved and picked, so the entry says how much in the person's words.
    await byTestId("add-food").trigger("click");
    await flushPromises();
    expect(backend.food.entries[0]).toMatchObject({ grams: 250, amount: "1 plate" });
  });

  it("creates a food from its label when search finds nothing", async () => {
    backend.seed({ food: { lmvFoods } });
    await createWrapper();

    await byTestId("add-snack").trigger("click");
    await byTestId("food-search").setValue("ölkorv");
    await searchSettles();
    expect(byTestId("search-status").text()).toContain("Nothing found");

    await byTestId("create-food").trigger("click");
    expect((wrapper.find("#food-name").element as HTMLInputElement).value).toBe("ölkorv");
    await byTestId("food-kcal").setValue("298");
    await byTestId("food-protein").setValue("14");
    await byTestId("food-carbs").setValue("2");
    await byTestId("food-fat").setValue("26,5");
    await byTestId("food-form").trigger("submit");
    await flushPromises();

    expect(backend.food.foods).toHaveLength(1);
    expect(backend.food.foods[0]!.per100g).toEqual({ kcal: 298, protein: 14, carbs: 2, fat: 26.5 });

    await byTestId("grams").setValue("70");
    await byTestId("add-food").trigger("click");
    await flushPromises();
    expect(backend.food.entries[0]).toMatchObject({ meal: "snack", foodId: backend.food.foods[0]!.id, grams: 70 });
  });

  it("changes and removes an entry", async () => {
    backend.seed({
      food: { entries: [{ day: today(), meal: "lunch", name: "Pasta kokt u. salt", grams: 100, amount: "1 plate", per100g: pasta }] },
    });
    await createWrapper();

    await byTestId("entry").trigger("click");
    await byTestId("entry-grams").setValue("200");
    await wrapper.findAll("button").find((b) => b.text() === "Dinner")!.trigger("click");
    await byTestId("entry-save").trigger("click");
    await flushPromises();

    // A new weight leaves the household amount behind.
    expect(backend.food.entries[0]).toMatchObject({ meal: "dinner", grams: 200, amount: "", kcal: 256 });
    expect(byTestId("meal-dinner").text()).toContain("200 g");

    await byTestId("entry").trigger("click");
    await byTestId("entry-delete").trigger("click");
    await byTestId("entry-delete").trigger("click");
    await flushPromises();
    expect(backend.food.entries).toHaveLength(0);
  });

  it("logs exercise and body weight", async () => {
    await createWrapper();

    await byTestId("add-activity").trigger("click");
    await byTestId("activity-label").setValue("Armwrestling");
    await byTestId("activity-kcal").setValue("180");
    await byTestId("activity-save").trigger("click");
    await flushPromises();
    expect(backend.food.activities[0]).toMatchObject({ day: today(), label: "Armwrestling", kcal: 180 });
    expect(byTestId("burned").text()).toBe("180");

    await byTestId("log-weight").trigger("click");
    await byTestId("weight-input").setValue("91,4");
    await byTestId("weight-save").trigger("click");
    await flushPromises();
    expect(backend.food.weights[today()]).toBe(91.4);
  });

  it("moves between days", async () => {
    const yesterday = addDays(today(), -1);
    backend.seed({ food: { entries: [{ day: yesterday, meal: "dinner", name: "Pasta kokt u. salt", grams: 500, per100g: pasta }] } });
    await createWrapper();
    expect(byTestId("eaten").text()).toBe("0");

    await byTestId("previous-day").trigger("click");
    await flushPromises();
    expect(mockReplace).toHaveBeenLastCalledWith({ query: { day: yesterday } });
    expect(byTestId("day-label").text()).toContain("Yesterday");
    expect(byTestId("eaten").text()).toBe("640");

    await byTestId("day-label").trigger("click");
    await flushPromises();
    expect(mockReplace).toHaveBeenLastCalledWith({ query: {} });
    expect(byTestId("eaten").text()).toBe("0");
  });

  it("adds the last meal again with one tap", async () => {
    const yesterday = addDays(today(), -1);
    backend.seed({
      food: {
        lmvFoods,
        entries: [
          { day: addDays(today(), -3), meal: "breakfast", name: "Pasta kokt u. salt", lmvNumber: 4065, grams: 50, per100g: pasta },
          { day: yesterday, meal: "lunch", name: "Pasta kokt u. salt", lmvNumber: 4065, grams: 200, per100g: pasta },
          { day: yesterday, meal: "lunch", name: "Pesto hemlagad", lmvNumber: 6201, grams: 30, amount: "2 tbsp", per100g: pesto },
        ],
      },
    });
    await createWrapper();

    const lunch = byTestId("meal-lunch");
    expect(lunch.find('[data-testid="repeat"]').text()).toContain("Same as yesterday");
    expect(lunch.find('[data-testid="repeat"]').text()).toContain("2 foods · 420 kcal");
    // An older breakfast is named by its day.
    expect(byTestId("meal-breakfast").find('[data-testid="repeat"]').exists()).toBe(true);

    await lunch.find('[data-testid="repeat-meal"]').trigger("click");
    await flushPromises();

    const today_ = backend.food.entries.filter((e) => e.day === today());
    expect(today_).toHaveLength(2);
    expect(today_[1]).toMatchObject({ meal: "lunch", lmvNumber: 6201, grams: 30, amount: "2 tbsp" });
    expect(byTestId("meal-lunch").find('[data-testid="repeat"]').exists()).toBe(false);
    expect(byTestId("eaten").text()).toBe("420");
  });

  it("quick adds usual foods and keeps the sheet open for more", async () => {
    const bun: Nutrients = { kcal: 350, protein: 6, carbs: 50, fat: 14 };
    backend.seed({
      food: {
        lmvFoods,
        entries: [
          { day: addDays(today(), -2), meal: "snack", name: "Pesto hemlagad", lmvNumber: 6201, grams: 15, per100g: pesto },
          { day: addDays(today(), -1), meal: "snack", name: "Kanelbulle", grams: 90, per100g: bun },
          { day: addDays(today(), -1), meal: "snack", name: "Pesto hemlagad", lmvNumber: 6201, grams: 20, per100g: pesto },
        ],
      },
    });
    await createWrapper();

    await byTestId("add-snack").trigger("click");
    expect(byTestId("search-status").text()).toBe("Usual for snacks");
    const usual = allByTestId("usual-food");
    // Eaten most often first, with the amount of the last time.
    expect(usual[0]!.text()).toContain("Pesto hemlagad");
    expect(usual[0]!.text()).toContain("20 g");

    await allByTestId("quick-add")[0]!.trigger("click");
    await flushPromises();
    await allByTestId("quick-add")[1]!.trigger("click");
    await flushPromises();

    expect(byTestId("food-search").exists()).toBe(true);
    expect(allByTestId("quick-add")[0]!.text()).toBe("check");
    expect(backend.food.entries.filter((e) => e.day === today())).toMatchObject([
      { meal: "snack", lmvNumber: 6201, grams: 20 },
      // A one off without a food, logged again by its name and nutrients.
      { meal: "snack", lmvNumber: null, foodId: null, name: "Kanelbulle", grams: 90, kcal: 315 },
    ]);
    expect(byTestId("eaten").text()).toBe("424");
  });

  it("closes a sheet with Escape", async () => {
    await createWrapper();
    await byTestId("add-lunch").trigger("click");
    expect(byTestId("food-search").exists()).toBe(true);

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    await flushPromises();
    expect(byTestId("food-search").exists()).toBe(false);
  });
});
