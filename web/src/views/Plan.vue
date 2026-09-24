<template>
  <div class="plan h-full flex flex-col">
    <NeoHeader :title="t('plan.title')" />

    <div class="flex-1 overflow-y-auto hide-scrollbar p-4 space-y-6">
      <div v-if="loading" class="flex items-center justify-center h-32">
        <div class="text-gray-500">{{ t("common.loading") }}</div>
      </div>

      <!-- Plans are written by an assistant, the app only follows them -->
      <NeoPanel
        v-else-if="plans.length === 0"
        class="text-center"
        data-testid="plan-empty"
      >
        <span class="material-icons text-4xl! text-black dark:text-white">
          event_note
        </span>
        <div class="text-base font-bold text-black dark:text-white mt-2">
          {{ t("plan.empty") }}
        </div>
        <p class="text-sm text-black dark:text-white opacity-70 mt-1">
          {{ t("plan.emptyDescription") }}
        </p>
      </NeoPanel>

      <template v-else>
        <!-- Plan picker, when more than one plan is active -->
        <SelectButton
          v-if="plans.length > 1"
          v-model="selectedPlanId"
          :options="planOptions"
          optionLabel="label"
          optionValue="value"
          :allowEmpty="false"
          class="w-full"
          data-testid="plan-picker"
          @change="loadDetail"
        />

        <div v-if="selectedPlan" class="text-black dark:text-white">
          <h2 class="text-xl font-bold">{{ selectedPlan.name }}</h2>
          <p
            v-if="selectedPlan.goal"
            class="text-sm opacity-70 whitespace-pre-line"
          >
            {{ selectedPlan.goal }}
          </p>
        </div>

        <!-- Next up -->
        <NeoPanel v-if="nextUp" data-testid="next-up">
          <div class="flex items-start justify-between gap-3 mb-1">
            <span
              class="bg-purple-500 text-white border-2 border-nb-border rounded-md px-2.5 py-1 text-xs font-bold uppercase shadow-brutal-sm"
            >
              {{ t("plan.nextUp") }}
            </span>
            <span
              v-if="nextUp.status === 'in_progress'"
              class="bg-yellow-300 text-black border-2 border-nb-border rounded-md px-2.5 py-1 text-xs font-bold shadow-brutal-sm"
            >
              {{ t("plan.inProgress") }}
            </span>
          </div>
          <div class="flex items-center gap-2 flex-wrap mt-3">
            <h3 class="text-2xl font-bold text-black dark:text-white">
              {{ nextUp.label }}
            </h3>
            <span
              v-if="nextUp.intensity"
              :class="[
                'border-2 border-nb-border rounded-full px-2 py-0.5 text-xs font-bold',
                intensityClass(nextUp.intensity),
              ]"
            >
              {{ t(`exercise.intensity.${nextUp.intensity}`) }}
            </span>
          </div>
          <div
            v-if="sessionMeta(nextUp)"
            class="text-sm text-black dark:text-white opacity-70"
          >
            {{ sessionMeta(nextUp) }}
          </div>
          <p
            v-if="nextUp.notes"
            class="mt-2 text-sm text-black dark:text-white whitespace-pre-line"
          >
            {{ nextUp.notes }}
          </p>

          <PlannedExerciseList
            :exercises="nextUp.exercises"
            :weight-unit="weightUnit"
            class="mt-4"
          />

          <SessionActions
            :status="nextUp.status"
            :busy="busy"
            class="mt-5"
            @start="start(nextUp.id)"
            @skip="skip(nextUp.id, $event)"
          />
        </NeoPanel>

        <NeoPanel
          v-else
          class="text-center text-black dark:text-white"
          data-testid="plan-finished"
        >
          <span class="material-icons text-4xl!">emoji_events</span>
          <div class="font-bold mt-2">{{ t("plan.allDone") }}</div>
        </NeoPanel>

        <!-- Coming up -->
        <section v-if="comingUp.length" class="space-y-3">
          <h3 class="text-lg font-bold text-black dark:text-white">
            {{ t("plan.comingUp") }}
          </h3>
          <div
            v-for="session in comingUp"
            :key="session.id"
            class="bg-white border-3 border-nb-border rounded-xl shadow-brutal-sm dark:bg-zinc-700 dark:text-white"
            data-testid="coming-up"
          >
            <button
              type="button"
              class="w-full flex items-center justify-between gap-3 p-4 text-left"
              :aria-expanded="expanded.has(session.id)"
              @click="toggle(session.id)"
            >
              <div class="min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-bold text-black dark:text-white">
                    {{ session.label }}
                  </span>
                  <span
                    v-if="session.intensity"
                    :class="[
                      'border-2 border-nb-border rounded-full px-2 py-0.5 text-xs font-bold',
                      intensityClass(session.intensity),
                    ]"
                  >
                    {{ t(`exercise.intensity.${session.intensity}`) }}
                  </span>
                </div>
                <div class="text-xs text-black dark:text-white opacity-70">
                  {{
                    [
                      sessionMeta(session),
                      t(
                        "plan.exerciseCount",
                        { count: session.exercises.length },
                        session.exercises.length,
                      ),
                    ]
                      .filter(Boolean)
                      .join(" · ")
                  }}
                </div>
              </div>
              <span
                class="material-icons text-black dark:text-white transition-transform duration-200"
                :class="{ 'rotate-180': expanded.has(session.id) }"
              >
                expand_more
              </span>
            </button>
            <div v-if="expanded.has(session.id)" class="px-4 pb-4">
              <p
                v-if="session.notes"
                class="mb-3 text-sm text-black dark:text-white whitespace-pre-line"
              >
                {{ session.notes }}
              </p>
              <PlannedExerciseList
                :exercises="session.exercises"
                :weight-unit="weightUnit"
              />
              <SessionActions
                :status="session.status"
                :busy="busy"
                class="mt-4"
                @start="start(session.id)"
                @skip="skip(session.id, $event)"
              />
            </div>
          </div>
        </section>

        <!-- Done, completed and skipped sessions -->
        <section v-if="done.length" class="space-y-3">
          <button
            type="button"
            class="w-full flex justify-between items-center bg-nb-overlay p-4 border-3 border-nb-border rounded-xl shadow-brutal dark:bg-zinc-600"
            :aria-expanded="showDone"
            data-testid="done-toggle"
            @click="showDone = !showDone"
          >
            <span class="text-base font-bold text-black dark:text-white">
              {{ t("plan.done") }}
            </span>
            <span class="flex items-center gap-2">
              <span
                class="bg-purple-300 text-black px-3 py-1 border-2 border-nb-border rounded-md text-xs font-semibold shadow-brutal-sm"
              >
                {{ done.length }}
              </span>
              <span
                class="material-icons text-black dark:text-white transition-transform duration-200"
                :class="{ 'rotate-180': showDone }"
              >
                expand_more
              </span>
            </span>
          </button>

          <template v-if="showDone">
            <div
              v-for="session in done"
              :key="session.id"
              class="bg-white border-3 border-nb-border rounded-xl shadow-brutal-sm p-4 dark:bg-zinc-700 dark:text-white"
              :class="{ 'cursor-pointer': session.workoutId }"
              data-testid="done-session"
              @click="openWorkout(session)"
            >
              <div class="flex items-center justify-between gap-3">
                <div class="min-w-0">
                  <div class="font-bold text-black dark:text-white">
                    {{ session.label }}
                  </div>
                  <div class="text-xs text-black dark:text-white opacity-70">
                    {{
                      [
                        sessionMeta(session),
                        session.completedAt
                          ? d(new Date(session.completedAt), "short")
                          : "",
                      ]
                        .filter(Boolean)
                        .join(" · ")
                    }}
                  </div>
                </div>

                <span
                  v-if="session.status === 'completed' && session.comparison"
                  :class="[
                    'shrink-0 border-2 border-nb-border rounded-md px-2.5 py-1 text-xs font-bold text-black shadow-brutal-sm',
                    compliance(session).met === compliance(session).total
                      ? 'bg-green-400'
                      : 'bg-amber-300',
                  ]"
                  data-testid="compliance-chip"
                >
                  {{ t("plan.targetsMet", { ...compliance(session) }) }}
                </span>
                <span
                  v-else-if="session.status === 'skipped'"
                  class="shrink-0 bg-nb-overlay text-black border-2 border-nb-border rounded-md px-2.5 py-1 text-xs font-bold dark:bg-zinc-800 dark:text-white"
                >
                  {{ t("plan.skipped") }}
                </span>
              </div>

              <div
                v-if="session.status === 'skipped'"
                class="mt-3 flex items-center justify-between gap-3"
              >
                <p
                  class="text-sm text-black dark:text-white opacity-80 italic min-w-0"
                >
                  {{ session.skipReason }}
                </p>
                <NeoButton
                  variant="secondary"
                  size="sm"
                  class="shrink-0"
                  :disabled="busy"
                  data-testid="put-back"
                  @click.stop="putBack(session.id)"
                >
                  <template #icon>
                    <span class="material-icons text-base!">undo</span>
                  </template>
                  {{ t("plan.putBack") }}
                </NeoButton>
              </div>
            </div>
          </template>
        </section>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import {
  getPlan,
  getPlans,
  getQueue,
  restoreSession,
  skipSession,
} from "@/api/plans";
import { useToast } from "@/composables/useToast";
import { useUnits } from "@/composables/useUnits";
import { usePlanSession } from "@/composables/usePlanSession";
import { complianceOf, doneSessions, type Compliance } from "@/utils/plan";
import NeoHeader from "@/components/NeoHeader.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import NeoButton from "@/components/NeoButton.vue";
import PlannedExerciseList from "@/components/PlannedExerciseList.vue";
import SessionActions from "@/components/SessionActions.vue";
import SelectButton from "@/volt/SelectButton.vue";
import type {
  Id,
  Intensity,
  Plan,
  PlannedSession,
  QueueEntry,
} from "@/types/domain";

const router = useRouter();
const { t, d } = useI18n();
const { showError } = useToast();
const { weightUnit } = useUnits();
const { start, starting } = usePlanSession();

useHead({
  title: () => t("plan.title"),
});

const loading = ref(true);
const plans = ref<Plan[]>([]);
const queue = ref<QueueEntry[]>([]);
const selectedPlanId = ref<Id | null>(null);
// The selected plan with each done session's comparison
const detail = ref<Plan | null>(null);
const expanded = ref(new Set<Id>());
const showDone = ref(false);
const updating = ref(false);

const busy = computed(() => updating.value || starting.value !== null);

const planOptions = computed(() =>
  plans.value.map((plan) => ({ label: plan.name, value: plan.id })),
);

const selectedPlan = computed(() =>
  plans.value.find((plan) => plan.id === selectedPlanId.value),
);

/** The selected plan's sessions still to do, in order */
const planQueue = computed(() =>
  queue.value
    .filter((entry) => entry.session.planId === selectedPlanId.value)
    .map((entry) => entry.session),
);

const nextUp = computed(() => planQueue.value[0]);
const comingUp = computed(() => planQueue.value.slice(1));

const done = computed(() =>
  doneSessions(detail.value?.sessions ?? selectedPlan.value?.sessions ?? []),
);

/**
 * Load the active plans and the queue, keeping the picked plan when it is
 * still active
 */
async function load() {
  try {
    const [activePlans, entries] = await Promise.all([
      getPlans("active"),
      getQueue(),
    ]);
    plans.value = activePlans;
    queue.value = entries;
    if (!activePlans.some((plan) => plan.id === selectedPlanId.value)) {
      selectedPlanId.value =
        entries[0]?.session.planId ?? activePlans[0]?.id ?? null;
    }
    await loadDetail();
  } catch (error) {
    console.error("Error loading plans:", error);
    showError(t("plan.loadError"));
  } finally {
    loading.value = false;
  }
}

/**
 * Load the selected plan with how its done sessions went
 */
async function loadDetail() {
  const id = selectedPlanId.value;
  if (!id) {
    detail.value = null;
    return;
  }
  try {
    const plan = await getPlan(id, true);
    // The picker may have moved on while this loaded
    if (selectedPlanId.value === id) detail.value = plan ?? null;
  } catch (error) {
    console.error("Error loading plan:", error);
  }
}

/**
 * "Week 3 · Day A", or empty when the plan does not number its sessions
 */
function sessionMeta(session: PlannedSession): string {
  return [
    session.week !== null ? t("plan.week", { week: session.week }) : "",
    session.day ? t("plan.day", { day: session.day }) : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

function intensityClass(intensity: Intensity): string {
  return intensity === "heavy"
    ? "bg-red-400 text-black"
    : "bg-sky-300 text-black";
}

function compliance(session: PlannedSession): Compliance {
  return session.comparison
    ? complianceOf(session.comparison)
    : { met: 0, total: 0 };
}

function toggle(id: Id) {
  if (expanded.value.has(id)) {
    expanded.value.delete(id);
  } else {
    expanded.value.add(id);
  }
}

async function skip(id: Id, reason: string) {
  updating.value = true;
  try {
    await skipSession(id, reason);
    await load();
  } catch (error) {
    console.error("Error skipping session:", error);
    showError(t("plan.skipError"));
  } finally {
    updating.value = false;
  }
}

async function putBack(id: Id) {
  updating.value = true;
  try {
    await restoreSession(id);
    await load();
  } catch (error) {
    console.error("Error putting session back:", error);
    showError(t("plan.putBackError"));
  } finally {
    updating.value = false;
  }
}

function openWorkout(session: PlannedSession) {
  if (session.workoutId) {
    router.push({ name: "workout-edit", params: { id: session.workoutId } });
  }
}

onMounted(load);
</script>
