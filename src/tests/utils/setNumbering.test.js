import { describe, it, expect } from 'vitest'

/**
 * Get display number for a set (copied from WorkoutEdit.vue for testing)
 * @param {Array} sets - All sets for the exercise
 * @param {number} setIndex - Index of the current set
 * @param {Object} exercise - The exercise object
 * @param {Object} currentSet - The current set object
 * @returns {string} Set number display
 */
function getSetNumber(sets, setIndex, exercise, currentSet) {
  const set = sets[setIndex];
  
  if (set.type === "warmup") {
    // For warmup sets, count all warmup sets regardless of arm
    const warmupSets = sets
      .slice(0, setIndex + 1)
      .filter((s) => s.type === "warmup");
    let setNumber = `W${warmupSets.length}`;
    
    // Add arm designation for single-arm exercises
    if (exercise.singleArm && currentSet.arm) {
      const armSuffix = currentSet.arm === 'left' ? 'L' : currentSet.arm === 'right' ? 'R' : '';
      if (armSuffix) {
        setNumber += armSuffix;
      }
    }
    
    return setNumber;
  } else {
    // For regular sets
    if (exercise.singleArm && currentSet.arm) {
      // Count sets for the specific arm
      const armSets = sets
        .slice(0, setIndex + 1)
        .filter((s) => s.type === "regular" && s.arm === currentSet.arm);
      const armSuffix = currentSet.arm === 'left' ? 'L' : currentSet.arm === 'right' ? 'R' : '';
      return `${armSets.length}${armSuffix}`;
    } else {
      // For non-single arm exercises or sets without arm specified
      const regularSets = sets
        .slice(0, setIndex + 1)
        .filter((s) => s.type === "regular");
      return `${regularSets.length}`;
    }
  }
}

describe('Set Numbering', () => {
  describe('Single-arm exercises', () => {
    const singleArmExercise = {
      name: 'Wrist Curl',
      singleArm: true
    }

    describe('Regular sets', () => {
      it('should number left arm sets correctly', () => {
        const sets = [
          { type: 'regular', arm: 'left' },
          { type: 'regular', arm: 'left' },
          { type: 'regular', arm: 'left' }
        ]

        expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('1L')
        expect(getSetNumber(sets, 1, singleArmExercise, sets[1])).toBe('2L')
        expect(getSetNumber(sets, 2, singleArmExercise, sets[2])).toBe('3L')
      })

      it('should number right arm sets correctly', () => {
        const sets = [
          { type: 'regular', arm: 'right' },
          { type: 'regular', arm: 'right' },
          { type: 'regular', arm: 'right' }
        ]

        expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('1R')
        expect(getSetNumber(sets, 1, singleArmExercise, sets[1])).toBe('2R')
        expect(getSetNumber(sets, 2, singleArmExercise, sets[2])).toBe('3R')
      })

      it('should count left and right arm sets independently', () => {
        const sets = [
          { type: 'regular', arm: 'left' },    // 1L
          { type: 'regular', arm: 'right' },   // 1R
          { type: 'regular', arm: 'left' },    // 2L
          { type: 'regular', arm: 'right' },   // 2R
          { type: 'regular', arm: 'left' }     // 3L
        ]

        expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('1L')
        expect(getSetNumber(sets, 1, singleArmExercise, sets[1])).toBe('1R')
        expect(getSetNumber(sets, 2, singleArmExercise, sets[2])).toBe('2L')
        expect(getSetNumber(sets, 3, singleArmExercise, sets[3])).toBe('2R')
        expect(getSetNumber(sets, 4, singleArmExercise, sets[4])).toBe('3L')
      })

      it('should handle both arm sets for single-arm exercises', () => {
        const sets = [
          { type: 'regular', arm: 'both' },
          { type: 'regular', arm: 'both' }
        ]

        expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('1')
        expect(getSetNumber(sets, 1, singleArmExercise, sets[1])).toBe('2')
      })

      it('should handle sets without arm specified', () => {
        const sets = [
          { type: 'regular', arm: '' },
          { type: 'regular', arm: null },
          { type: 'regular', arm: undefined }
        ]

        expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('1')
        expect(getSetNumber(sets, 1, singleArmExercise, sets[1])).toBe('2')
        expect(getSetNumber(sets, 2, singleArmExercise, sets[2])).toBe('3')
      })
    })

    describe('Warmup sets', () => {
      it('should number left arm warmup sets correctly', () => {
        const sets = [
          { type: 'warmup', arm: 'left' },
          { type: 'warmup', arm: 'left' },
          { type: 'warmup', arm: 'left' }
        ]

        expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('W1L')
        expect(getSetNumber(sets, 1, singleArmExercise, sets[1])).toBe('W2L')
        expect(getSetNumber(sets, 2, singleArmExercise, sets[2])).toBe('W3L')
      })

      it('should number right arm warmup sets correctly', () => {
        const sets = [
          { type: 'warmup', arm: 'right' },
          { type: 'warmup', arm: 'right' }
        ]

        expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('W1R')
        expect(getSetNumber(sets, 1, singleArmExercise, sets[1])).toBe('W2R')
      })

      it('should count warmup sets across all arms', () => {
        const sets = [
          { type: 'warmup', arm: 'left' },     // W1L
          { type: 'warmup', arm: 'right' },    // W2R
          { type: 'warmup', arm: 'left' },     // W3L
          { type: 'warmup', arm: 'right' }     // W4R
        ]

        expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('W1L')
        expect(getSetNumber(sets, 1, singleArmExercise, sets[1])).toBe('W2R')
        expect(getSetNumber(sets, 2, singleArmExercise, sets[2])).toBe('W3L')
        expect(getSetNumber(sets, 3, singleArmExercise, sets[3])).toBe('W4R')
      })

      it('should handle warmup sets without arm specified', () => {
        const sets = [
          { type: 'warmup', arm: '' },
          { type: 'warmup', arm: null }
        ]

        expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('W1')
        expect(getSetNumber(sets, 1, singleArmExercise, sets[1])).toBe('W2')
      })
    })

    describe('Mixed set types', () => {
      it('should handle mixed warmup and regular sets with different arms', () => {
        const sets = [
          { type: 'warmup', arm: 'left' },     // W1L
          { type: 'warmup', arm: 'right' },    // W2R
          { type: 'regular', arm: 'left' },    // 1L
          { type: 'regular', arm: 'right' },   // 1R
          { type: 'regular', arm: 'left' },    // 2L
          { type: 'warmup', arm: 'left' }      // W3L
        ]

        expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('W1L')
        expect(getSetNumber(sets, 1, singleArmExercise, sets[1])).toBe('W2R')
        expect(getSetNumber(sets, 2, singleArmExercise, sets[2])).toBe('1L')
        expect(getSetNumber(sets, 3, singleArmExercise, sets[3])).toBe('1R')
        expect(getSetNumber(sets, 4, singleArmExercise, sets[4])).toBe('2L')
        expect(getSetNumber(sets, 5, singleArmExercise, sets[5])).toBe('W3L')
      })
    })
  })

  describe('Both-arms exercises', () => {
    const bothArmsExercise = {
      name: 'Resistance Band Pull',
      singleArm: false
    }

    describe('Regular sets', () => {
      it('should number regular sets without arm suffix', () => {
        const sets = [
          { type: 'regular', arm: 'both' },
          { type: 'regular', arm: 'both' },
          { type: 'regular', arm: 'both' }
        ]

        expect(getSetNumber(sets, 0, bothArmsExercise, sets[0])).toBe('1')
        expect(getSetNumber(sets, 1, bothArmsExercise, sets[1])).toBe('2')
        expect(getSetNumber(sets, 2, bothArmsExercise, sets[2])).toBe('3')
      })

      it('should ignore arm values for both-arms exercises', () => {
        const sets = [
          { type: 'regular', arm: 'left' },    // Should still be '1'
          { type: 'regular', arm: 'right' },   // Should still be '2'
          { type: 'regular', arm: 'both' }     // Should still be '3'
        ]

        expect(getSetNumber(sets, 0, bothArmsExercise, sets[0])).toBe('1')
        expect(getSetNumber(sets, 1, bothArmsExercise, sets[1])).toBe('2')
        expect(getSetNumber(sets, 2, bothArmsExercise, sets[2])).toBe('3')
      })
    })

    describe('Warmup sets', () => {
      it('should number warmup sets without arm suffix', () => {
        const sets = [
          { type: 'warmup', arm: 'both' },
          { type: 'warmup', arm: 'both' }
        ]

        expect(getSetNumber(sets, 0, bothArmsExercise, sets[0])).toBe('W1')
        expect(getSetNumber(sets, 1, bothArmsExercise, sets[1])).toBe('W2')
      })
    })

    describe('Mixed set types', () => {
      it('should handle mixed warmup and regular sets for both-arms exercises', () => {
        const sets = [
          { type: 'warmup', arm: 'both' },     // W1
          { type: 'warmup', arm: 'both' },     // W2
          { type: 'regular', arm: 'both' },    // 1
          { type: 'regular', arm: 'both' },    // 2
          { type: 'warmup', arm: 'both' }      // W3
        ]

        expect(getSetNumber(sets, 0, bothArmsExercise, sets[0])).toBe('W1')
        expect(getSetNumber(sets, 1, bothArmsExercise, sets[1])).toBe('W2')
        expect(getSetNumber(sets, 2, bothArmsExercise, sets[2])).toBe('1')
        expect(getSetNumber(sets, 3, bothArmsExercise, sets[3])).toBe('2')
        expect(getSetNumber(sets, 4, bothArmsExercise, sets[4])).toBe('W3')
      })
    })
  })

  describe('Edge cases', () => {
    const singleArmExercise = { name: 'Wrist Curl', singleArm: true }

    it('should handle empty sets array', () => {
      const sets = [{ type: 'regular', arm: 'left' }]
      expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('1L')
    })

    it('should handle single set', () => {
      const sets = [{ type: 'regular', arm: 'right' }]
      expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('1R')
    })

    it('should handle inconsistent arm values', () => {
      const sets = [
        { type: 'regular', arm: 'left' },
        { type: 'regular', arm: '' },        // No arm
        { type: 'regular', arm: 'left' },
        { type: 'regular', arm: 'right' }
      ]

      expect(getSetNumber(sets, 0, singleArmExercise, sets[0])).toBe('1L')
      expect(getSetNumber(sets, 1, singleArmExercise, sets[1])).toBe('2')      // Counts all regular sets
      expect(getSetNumber(sets, 2, singleArmExercise, sets[2])).toBe('2L')     // 2nd left set
      expect(getSetNumber(sets, 3, singleArmExercise, sets[3])).toBe('1R')     // First right
    })

    it('should handle undefined exercise properties', () => {
      const undefinedExercise = { name: 'Test Exercise' } // No singleArm property
      const sets = [{ type: 'regular', arm: 'left' }]
      
      expect(getSetNumber(sets, 0, undefinedExercise, sets[0])).toBe('1')
    })
  })
})