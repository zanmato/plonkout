import { describe, it, expect } from 'vitest'

/**
 * Check if two arm settings are compatible for comparison (copied from WorkoutEdit.vue for testing)
 */
function isArmCompatible(currentArm, historicalArm) {
  if (!currentArm || !historicalArm) {
    return true;
  }
  
  if (currentArm === historicalArm) {
    return true;
  }
  
  if (historicalArm === 'both' && (currentArm === 'left' || currentArm === 'right')) {
    return true;
  }
  
  return false;
}

/**
 * Get the previous best reps for a given weight and exercise
 * @param {string} exerciseName - Name of the exercise
 * @param {number} weight - Weight to check reps for
 * @param {string} arm - Arm setting
 * @param {Array} allWorkouts - All historical workouts
 * @returns {number|null} Previous best reps or null if no data
 */
function getPreviousReps(exerciseName, weight, arm, allWorkouts) {
  if (!weight || weight <= 0) {
    return null;
  }
  
  let bestReps = 0;
  
  allWorkouts.forEach(historicalWorkout => {
    if (historicalWorkout.exercises) {
      historicalWorkout.exercises.forEach(exercise => {
        if (exercise.name === exerciseName && exercise.sets) {
          exercise.sets.forEach(set => {
            if (set.weight === weight && set.reps && set.type === 'regular') {
              if (isArmCompatible(arm, set.arm) && set.reps > bestReps) {
                bestReps = set.reps;
              }
            }
          });
        }
      });
    }
  });
  
  return bestReps > 0 ? bestReps : null;
}

/**
 * Check if current reps is a new record for the weight
 * @param {string} exerciseName - Name of the exercise
 * @param {number} weight - Weight being lifted
 * @param {number} reps - Reps being performed
 * @param {string} arm - Arm setting
 * @param {Array} allWorkouts - All historical workouts
 * @returns {boolean} Whether this is a new rep record
 */
function isRepRecord(exerciseName, weight, reps, arm, allWorkouts) {
  if (!weight || !reps || weight <= 0 || reps <= 0) {
    return false;
  }
  
  const previousBest = getPreviousReps(exerciseName, weight, arm, allWorkouts);
  return previousBest === null || reps > previousBest;
}

/**
 * Check if current weight is a new record for the exercise
 * @param {string} exerciseName - Name of the exercise
 * @param {number} weight - Weight being lifted
 * @param {string} arm - Arm setting
 * @param {Array} allWorkouts - All historical workouts
 * @returns {boolean} Whether this is a new weight record
 */
function isWeightRecord(exerciseName, weight, arm, allWorkouts) {
  if (!weight || weight <= 0) {
    return false;
  }
  
  let maxWeight = 0;
  
  allWorkouts.forEach(historicalWorkout => {
    if (historicalWorkout.exercises) {
      historicalWorkout.exercises.forEach(exercise => {
        if (exercise.name === exerciseName && exercise.sets) {
          exercise.sets.forEach(set => {
            if (set.weight && set.weight > maxWeight && set.type === 'regular') {
              if (isArmCompatible(arm, set.arm)) {
                maxWeight = set.weight;
              }
            }
          });
        }
      });
    }
  });
  
  return weight > maxWeight;
}

describe('Record Tracking', () => {
  const mockWorkouts = [
    {
      exercises: [
        {
          name: 'Bench Press',
          sets: [
            { type: 'regular', weight: 100, reps: 10, arm: 'both' },
            { type: 'regular', weight: 120, reps: 8, arm: 'both' },
            { type: 'regular', weight: 120, reps: 12, arm: 'both' }, // Best at 120kg
            { type: 'warmup', weight: 120, reps: 15, arm: 'both' },  // Should be ignored
          ]
        },
        {
          name: 'Wrist Curl',
          sets: [
            { type: 'regular', weight: 50, reps: 12, arm: 'left' },
            { type: 'regular', weight: 60, reps: 8, arm: 'left' },
            { type: 'regular', weight: 50, reps: 10, arm: 'right' },
            { type: 'regular', weight: 70, reps: 15, arm: 'both' },
          ]
        }
      ]
    },
    {
      exercises: [
        {
          name: 'Bench Press',
          sets: [
            { type: 'regular', weight: 120, reps: 10, arm: 'both' },
            { type: 'regular', weight: 140, reps: 5, arm: 'both' }, // Max weight
          ]
        },
        {
          name: 'Wrist Curl',
          sets: [
            { type: 'regular', weight: 50, reps: 15, arm: 'left' }, // Best left at 50kg
            { type: 'regular', weight: 65, reps: 6, arm: 'right' },  // Max right weight
          ]
        }
      ]
    }
  ];

  describe('getPreviousReps', () => {
    it('should return the best previous reps for exact weight match', () => {
      expect(getPreviousReps('Bench Press', 120, 'both', mockWorkouts)).toBe(12)
      expect(getPreviousReps('Wrist Curl', 50, 'left', mockWorkouts)).toBe(15)
      expect(getPreviousReps('Wrist Curl', 50, 'right', mockWorkouts)).toBe(10)
    })

    it('should return null for weights never attempted', () => {
      expect(getPreviousReps('Bench Press', 200, 'both', mockWorkouts)).toBe(null)
      expect(getPreviousReps('Wrist Curl', 100, 'left', mockWorkouts)).toBe(null)
      expect(getPreviousReps('Unknown Exercise', 50, 'both', mockWorkouts)).toBe(null)
    })

    it('should respect arm compatibility', () => {
      // Left arm can use 'both' arm data (70kg @ 15 reps)
      expect(getPreviousReps('Wrist Curl', 70, 'left', mockWorkouts)).toBe(15)
      // Right arm can use 'both' arm data (70kg @ 15 reps)
      expect(getPreviousReps('Wrist Curl', 70, 'right', mockWorkouts)).toBe(15)
      // Both arms cannot use single arm data
      expect(getPreviousReps('Wrist Curl', 65, 'both', mockWorkouts)).toBe(null)
    })

    it('should ignore warmup sets', () => {
      // Even though warmup had 15 reps at 120kg, should return 12 (best regular set)
      expect(getPreviousReps('Bench Press', 120, 'both', mockWorkouts)).toBe(12)
    })

    it('should handle edge cases', () => {
      expect(getPreviousReps('Bench Press', 0, 'both', mockWorkouts)).toBe(null)
      expect(getPreviousReps('Bench Press', -10, 'both', mockWorkouts)).toBe(null)
      expect(getPreviousReps('Bench Press', null, 'both', mockWorkouts)).toBe(null)
      expect(getPreviousReps('Bench Press', 120, 'both', [])).toBe(null)
    })
  })

  describe('isRepRecord', () => {
    it('should detect new rep records', () => {
      // Previous best at 120kg was 12 reps
      expect(isRepRecord('Bench Press', 120, 13, 'both', mockWorkouts)).toBe(true)
      expect(isRepRecord('Bench Press', 120, 15, 'both', mockWorkouts)).toBe(true)
      
      // Previous best left arm at 50kg was 15 reps
      expect(isRepRecord('Wrist Curl', 50, 16, 'left', mockWorkouts)).toBe(true)
    })

    it('should not flag existing records', () => {
      // Previous best at 120kg was 12 reps
      expect(isRepRecord('Bench Press', 120, 12, 'both', mockWorkouts)).toBe(false)
      expect(isRepRecord('Bench Press', 120, 10, 'both', mockWorkouts)).toBe(false)
      
      // Previous best left arm at 50kg was 15 reps
      expect(isRepRecord('Wrist Curl', 50, 15, 'left', mockWorkouts)).toBe(false)
      expect(isRepRecord('Wrist Curl', 50, 12, 'left', mockWorkouts)).toBe(false)
    })

    it('should detect records for new weights', () => {
      // Never attempted 200kg before
      expect(isRepRecord('Bench Press', 200, 1, 'both', mockWorkouts)).toBe(true)
      expect(isRepRecord('Bench Press', 200, 5, 'both', mockWorkouts)).toBe(true)
    })

    it('should handle edge cases', () => {
      expect(isRepRecord('Bench Press', 0, 10, 'both', mockWorkouts)).toBe(false)
      expect(isRepRecord('Bench Press', 120, 0, 'both', mockWorkouts)).toBe(false)
      expect(isRepRecord('Bench Press', -10, 10, 'both', mockWorkouts)).toBe(false)
      expect(isRepRecord('Bench Press', 120, -5, 'both', mockWorkouts)).toBe(false)
      expect(isRepRecord('Bench Press', null, 10, 'both', mockWorkouts)).toBe(false)
      expect(isRepRecord('Bench Press', 120, null, 'both', mockWorkouts)).toBe(false)
    })
  })

  describe('isWeightRecord', () => {
    it('should detect new weight records', () => {
      // Previous max was 140kg
      expect(isWeightRecord('Bench Press', 150, 'both', mockWorkouts)).toBe(true)
      expect(isWeightRecord('Bench Press', 141, 'both', mockWorkouts)).toBe(true)
      
      // Previous max left arm was 60kg (can also use 70kg from 'both')
      expect(isWeightRecord('Wrist Curl', 71, 'left', mockWorkouts)).toBe(true)
      
      // Previous max right arm was 65kg (can also use 70kg from 'both') 
      expect(isWeightRecord('Wrist Curl', 71, 'right', mockWorkouts)).toBe(true)
    })

    it('should not flag existing weight records', () => {
      // Previous max was 140kg
      expect(isWeightRecord('Bench Press', 140, 'both', mockWorkouts)).toBe(false)
      expect(isWeightRecord('Bench Press', 120, 'both', mockWorkouts)).toBe(false)
      
      // Previous max compatible with left arm is 70kg (from 'both')
      expect(isWeightRecord('Wrist Curl', 70, 'left', mockWorkouts)).toBe(false)
      expect(isWeightRecord('Wrist Curl', 60, 'left', mockWorkouts)).toBe(false)
    })

    it('should detect records for new exercises', () => {
      expect(isWeightRecord('Unknown Exercise', 50, 'both', mockWorkouts)).toBe(true)
      expect(isWeightRecord('Unknown Exercise', 1, 'both', mockWorkouts)).toBe(true)
    })

    it('should respect arm compatibility for weight records', () => {
      // Both arms cannot use single arm data, so max is only 70kg (from 'both' sets)
      expect(isWeightRecord('Wrist Curl', 71, 'both', mockWorkouts)).toBe(true)
      expect(isWeightRecord('Wrist Curl', 70, 'both', mockWorkouts)).toBe(false)
    })

    it('should handle edge cases', () => {
      expect(isWeightRecord('Bench Press', 0, 'both', mockWorkouts)).toBe(false)
      expect(isWeightRecord('Bench Press', -10, 'both', mockWorkouts)).toBe(false)
      expect(isWeightRecord('Bench Press', null, 'both', mockWorkouts)).toBe(false)
      expect(isWeightRecord('Bench Press', 150, 'both', [])).toBe(true)
    })
  })

  describe('Integration scenarios', () => {
    it('should handle mixed arm scenarios correctly', () => {
      // Left arm at 50kg: previous best was 15 reps (from historical data)
      expect(getPreviousReps('Wrist Curl', 50, 'left', mockWorkouts)).toBe(15)
      expect(isRepRecord('Wrist Curl', 50, 16, 'left', mockWorkouts)).toBe(true)
      expect(isRepRecord('Wrist Curl', 50, 15, 'left', mockWorkouts)).toBe(false)
      
      // Right arm at 50kg: previous best was 10 reps
      expect(getPreviousReps('Wrist Curl', 50, 'right', mockWorkouts)).toBe(10)
      expect(isRepRecord('Wrist Curl', 50, 11, 'right', mockWorkouts)).toBe(true)
    })

    it('should handle exercises with only warmup sets', () => {
      const workoutsWithOnlyWarmups = [
        {
          exercises: [
            {
              name: 'Light Exercise',
              sets: [
                { type: 'warmup', weight: 50, reps: 10, arm: 'both' },
                { type: 'warmup', weight: 50, reps: 12, arm: 'both' }
              ]
            }
          ]
        }
      ];

      expect(getPreviousReps('Light Exercise', 50, 'both', workoutsWithOnlyWarmups)).toBe(null)
      expect(isRepRecord('Light Exercise', 50, 8, 'both', workoutsWithOnlyWarmups)).toBe(true)
      expect(isWeightRecord('Light Exercise', 60, 'both', workoutsWithOnlyWarmups)).toBe(true)
    })
  })
})