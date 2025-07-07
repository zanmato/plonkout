import { describe, it, expect } from 'vitest'

/**
 * Check if two arm settings are compatible for comparison (copied from WorkoutEdit.vue for testing)
 * @param {string} currentArm - Current arm setting
 * @param {string} historicalArm - Historical arm setting to compare
 * @returns {boolean} Whether the arms are compatible for comparison
 */
function isArmCompatible(currentArm, historicalArm) {
  // If either arm is not specified, they're compatible
  if (!currentArm || !historicalArm) {
    return true;
  }
  
  // Exact match
  if (currentArm === historicalArm) {
    return true;
  }
  
  // 'both' is compatible with individual arms (since both arms working together might be stronger)
  if (historicalArm === 'both' && (currentArm === 'left' || currentArm === 'right')) {
    return true;
  }
  
  // Individual arms are not compatible with 'both' (single arm max should not include both-arm sets)
  // Individual arms are not compatible with each other
  return false;
}

/**
 * Calculate the percentage of current weight vs max weight for an exercise (simplified for testing)
 * @param {string} exerciseName - Name of the exercise
 * @param {number} currentWeight - Current weight being lifted
 * @param {string} currentArm - Current arm setting
 * @param {Array} allWorkouts - All historical workouts
 * @returns {string} Percentage string or '-' if no data
 */
function getMaxPercentage(exerciseName, currentWeight, currentArm, allWorkouts) {
  if (!currentWeight || currentWeight <= 0) {
    return '-';
  }
  
  let maxWeight = 0;
  
  // Check all historical workouts
  allWorkouts.forEach(historicalWorkout => {
    if (historicalWorkout.exercises) {
      historicalWorkout.exercises.forEach(exercise => {
        if (exercise.name === exerciseName && exercise.sets) {
          exercise.sets.forEach(set => {
            if (set.weight && set.weight > maxWeight && set.type === 'regular') {
              // Only consider sets with the same arm setting or compatible arm settings
              if (isArmCompatible(currentArm, set.arm)) {
                maxWeight = set.weight;
              }
            }
          });
        }
      });
    }
  });
  
  if (maxWeight === 0) {
    return '-';
  }
  
  const percentage = Math.round((currentWeight / maxWeight) * 100);
  return `${percentage}%`;
}

describe('Arm Compatibility and Max Percentage', () => {
  describe('isArmCompatible', () => {
    describe('Exact matches', () => {
      it('should match identical arms', () => {
        expect(isArmCompatible('left', 'left')).toBe(true)
        expect(isArmCompatible('right', 'right')).toBe(true)
        expect(isArmCompatible('both', 'both')).toBe(true)
      })
    })

    describe('Empty/null values', () => {
      it('should be compatible when either arm is empty', () => {
        expect(isArmCompatible('', 'left')).toBe(true)
        expect(isArmCompatible('left', '')).toBe(true)
        expect(isArmCompatible(null, 'right')).toBe(true)
        expect(isArmCompatible('right', null)).toBe(true)
        expect(isArmCompatible(undefined, 'both')).toBe(true)
        expect(isArmCompatible('both', undefined)).toBe(true)
        expect(isArmCompatible('', '')).toBe(true)
        expect(isArmCompatible(null, null)).toBe(true)
      })
    })

    describe('Both arms compatibility', () => {
      it('should allow both arms data for single arm comparisons', () => {
        expect(isArmCompatible('left', 'both')).toBe(true)
        expect(isArmCompatible('right', 'both')).toBe(true)
      })

      it('should NOT allow single arm data for both arms comparisons', () => {
        expect(isArmCompatible('both', 'left')).toBe(false)
        expect(isArmCompatible('both', 'right')).toBe(false)
      })
    })

    describe('Cross-arm incompatibility', () => {
      it('should not allow left/right cross comparisons', () => {
        expect(isArmCompatible('left', 'right')).toBe(false)
        expect(isArmCompatible('right', 'left')).toBe(false)
      })
    })
  })

  describe('getMaxPercentage with arm specificity', () => {
    const mockWorkouts = [
      {
        exercises: [
          {
            name: 'Wrist Curl',
            sets: [
              { type: 'regular', weight: 50, arm: 'left' },
              { type: 'regular', weight: 60, arm: 'left' },
              { type: 'regular', weight: 70, arm: 'right' },
              { type: 'regular', weight: 80, arm: 'both' },
              { type: 'warmup', weight: 30, arm: 'left' } // Should be ignored
            ]
          }
        ]
      },
      {
        exercises: [
          {
            name: 'Wrist Curl',
            sets: [
              { type: 'regular', weight: 55, arm: 'left' },
              { type: 'regular', weight: 75, arm: 'right' }
            ]
          }
        ]
      }
    ];

    it('should calculate percentage for left arm using highest compatible max', () => {
      // Left arm can use 'both' arm max (80kg), so 30kg = 38% (rounded from 37.5%)
      expect(getMaxPercentage('Wrist Curl', 30, 'left', mockWorkouts)).toBe('38%')
      // Left arm can use 'both' arm max (80kg), so 60kg = 75%  
      expect(getMaxPercentage('Wrist Curl', 60, 'left', mockWorkouts)).toBe('75%')
      // Left arm can use 'both' arm max (80kg), so 45kg = 56% (rounded from 56.25%)
      expect(getMaxPercentage('Wrist Curl', 45, 'left', mockWorkouts)).toBe('56%')
    })

    it('should calculate percentage for right arm using highest compatible max', () => {
      // Right arm can use 'both' arm max (80kg), so 37.5kg = 47% (rounded from 46.875%)
      expect(getMaxPercentage('Wrist Curl', 37.5, 'right', mockWorkouts)).toBe('47%')
      // Right arm can use 'both' arm max (80kg), so 75kg = 94% (rounded from 93.75%)
      expect(getMaxPercentage('Wrist Curl', 75, 'right', mockWorkouts)).toBe('94%')
    })

    it('should use both arm max for single arm when available', () => {
      // Left arm can use 'both' arm data (80kg), so 40kg = 50%
      expect(getMaxPercentage('Wrist Curl', 40, 'left', mockWorkouts)).toBe('50%')
      // Right arm can use 'both' arm data (80kg), so 60kg = 75%  
      expect(getMaxPercentage('Wrist Curl', 60, 'right', mockWorkouts)).toBe('75%')
    })

    it('should calculate percentage for both arms against both arms max only', () => {
      // Both arms max is 80kg, current is 40kg
      expect(getMaxPercentage('Wrist Curl', 40, 'both', mockWorkouts)).toBe('50%')
      // Both arms max is 80kg, current is 80kg
      expect(getMaxPercentage('Wrist Curl', 80, 'both', mockWorkouts)).toBe('100%')
    })

    it('should handle exercises with no historical data', () => {
      expect(getMaxPercentage('Unknown Exercise', 50, 'left', mockWorkouts)).toBe('-')
    })

    it('should handle zero or negative weights', () => {
      expect(getMaxPercentage('Wrist Curl', 0, 'left', mockWorkouts)).toBe('-')
      expect(getMaxPercentage('Wrist Curl', -10, 'left', mockWorkouts)).toBe('-')
      expect(getMaxPercentage('Wrist Curl', null, 'left', mockWorkouts)).toBe('-')
    })

    it('should ignore warmup sets in max calculations', () => {
      const workoutsWithWarmups = [
        {
          exercises: [
            {
              name: 'Test Exercise',
              sets: [
                { type: 'warmup', weight: 100, arm: 'left' }, // Should be ignored
                { type: 'regular', weight: 50, arm: 'left' }   // This should be max
              ]
            }
          ]
        }
      ];

      expect(getMaxPercentage('Test Exercise', 25, 'left', workoutsWithWarmups)).toBe('50%')
    })

    it('should handle empty workouts array', () => {
      expect(getMaxPercentage('Wrist Curl', 50, 'left', [])).toBe('-')
    })

    it('should handle workouts without exercises', () => {
      const emptyWorkouts = [{ exercises: [] }, { exercises: undefined }];
      expect(getMaxPercentage('Wrist Curl', 50, 'left', emptyWorkouts)).toBe('-')
    })

    it('should round percentages correctly', () => {
      const workouts = [
        {
          exercises: [
            {
              name: 'Test',
              sets: [{ type: 'regular', weight: 33, arm: 'left' }]
            }
          ]
        }
      ];

      // 10/33 = 30.303...% should round to 30%
      expect(getMaxPercentage('Test', 10, 'left', workouts)).toBe('30%')
      // 11/33 = 33.333...% should round to 33%
      expect(getMaxPercentage('Test', 11, 'left', workouts)).toBe('33%')
    })
  })
})