DROP TABLE IF EXISTS
  settings, templates, workout_sets, workout_exercises, workouts,
  planned_targets, planned_exercises, planned_sessions, plans, exercises;
DROP SCHEMA IF EXISTS auth CASCADE;
DROP FUNCTION IF EXISTS current_user_id();
