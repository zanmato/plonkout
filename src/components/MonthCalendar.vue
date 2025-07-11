<template>
  <div class="mt-4 bg-nb-overlay border-3 border-nb-border rounded-xl p-4 shadow-brutal dark:bg-zinc-600">
    <div 
      v-for="calendarRow in calendarData"
      :key="`${monthYear}-${calendarRow.type}-${calendarRow.days?.join?.('-') || ''}`"
      class="flex justify-between"
      :class="{ 'mb-2': calendarRow.type === 'header', 'mb-1': calendarRow.type === 'week' }"
    >
      <!-- Day headers (Mon Tue Wed Thu Fri Sat Sun) -->
      <template v-if="calendarRow.type === 'header'">
        <div
          v-for="dayLabel in calendarRow.days"
          :key="dayLabel"
          class="w-12 h-8 flex items-center justify-center text-xs font-bold text-black dark:text-white"
        >
          {{ dayLabel }}
        </div>
      </template>

      <!-- Calendar week -->
      <template v-else-if="calendarRow.type === 'week'">
        <div
          v-for="(day, dayIndex) in calendarRow.days"
          :key="`day-${dayIndex}`"
          class="w-12 h-8 flex items-center justify-center text-xs border-2 border-nb-border rounded-md"
          :class="{
            'bg-purple-500 text-white font-bold shadow-brutal-sm': day.hasWorkout,
            'bg-white dark:bg-zinc-700 text-black dark:text-white': !day.hasWorkout && day.type === 'day',
            'bg-transparent border-transparent': day.type === 'empty'
          }"
        >
          <span v-if="day.type === 'day'">{{ day.day }}</span>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

const props = defineProps({
  monthYear: {
    type: String,
    required: true
  },
  workouts: {
    type: Array,
    required: true
  }
});

const { t } = useI18n();

/**
 * Generate calendar data for the month
 */
const calendarData = computed(() => {
  if (props.workouts.length === 0) {
    return [];
  }

  // Parse the month/year from the first workout
  const sampleDate = props.workouts[0].started;
  const date = new Date(sampleDate);
  const year = date.getFullYear();
  const month = date.getMonth();
  
  // Get first day of month and number of days
  const firstDay = new Date(year, month, 1);
  const lastDay = new Date(year, month + 1, 0);
  const daysInMonth = lastDay.getDate();
  const startDayOfWeek = firstDay.getDay(); // 0 = Sunday, 1 = Monday, etc.
  
  // Create workout day lookup
  const workoutDays = new Set();
  props.workouts.forEach(workout => {
    const workoutDate = new Date(workout.started);
    workoutDays.add(workoutDate.getDate());
  });
  
  // Generate calendar grid (starting from Monday)
  const calendar = [];
  const daysOfWeek = [
    t('calendar.days.mon'),
    t('calendar.days.tue'), 
    t('calendar.days.wed'),
    t('calendar.days.thu'),
    t('calendar.days.fri'),
    t('calendar.days.sat'),
    t('calendar.days.sun')
  ];
  
  // Add day headers
  calendar.push({
    type: 'header',
    days: daysOfWeek
  });
  
  // Calculate offset for Monday start (Sunday = 0, Monday = 1, etc.)
  const mondayOffset = startDayOfWeek === 0 ? 6 : startDayOfWeek - 1;
  
  // Generate weeks
  let currentWeek = [];
  
  // Add empty cells for days before month starts
  for (let i = 0; i < mondayOffset; i++) {
    currentWeek.push({ type: 'empty' });
  }
  
  // Add days of the month
  for (let day = 1; day <= daysInMonth; day++) {
    currentWeek.push({
      type: 'day',
      day,
      hasWorkout: workoutDays.has(day)
    });
    
    // Start new week on Sunday (after 7 days)
    if (currentWeek.length === 7) {
      calendar.push({ type: 'week', days: currentWeek });
      currentWeek = [];
    }
  }
  
  // Add remaining empty cells and final week
  if (currentWeek.length > 0) {
    while (currentWeek.length < 7) {
      currentWeek.push({ type: 'empty' });
    }
    calendar.push({ type: 'week', days: currentWeek });
  }
  
  return calendar;
});
</script>