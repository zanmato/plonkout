package diary

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/diary/diarydb"
	"github.com/zanmato/plonkout/server/internal/food"
)

// Most weeks look alike, so what the user ate at a meal lately is the best
// guess at what they will log for it next.
const (
	// recentDays is how far back habits are looked for.
	recentDays = 60
	// recentFoods is how many usual foods a meal lists.
	recentFoods = 15
)

// RecentFood is a food the user ate at a meal lately, with the amount of the
// last time. Logging it again takes foodId or lmvNumber and grams, or name
// and per100g for a one off without a food.
type RecentFood struct {
	FoodID    *uuid.UUID     `nullable:"true" json:"foodId"`
	LMVNumber *int32         `json:"lmvNumber"`
	Name      string         `json:"name"`
	Grams     float64        `json:"grams" doc:"The amount of the last time."`
	Amount    string         `json:"amount"`
	Per100g   food.Nutrients `json:"per100g" doc:"As last logged."`
	Kcal      float64        `json:"kcal" doc:"For the amount of the last time."`
	Uses      int            `json:"uses" doc:"How many times it was logged at this meal lately."`
	LastDay   string         `json:"lastDay" format:"date"`
}

// PastMeal is a meal as logged on an earlier day.
type PastMeal struct {
	Day     string       `json:"day" format:"date"`
	Kcal    float64      `json:"kcal"`
	Entries []RecentFood `json:"entries"`
}

// RecentMeal is what the user usually eats at a meal.
type RecentMeal struct {
	Meal  string       `json:"meal" enum:"breakfast,lunch,dinner,snack"`
	Last  *PastMeal    `json:"last,omitempty" doc:"The latest earlier day with this meal logged, whole."`
	Foods []RecentFood `json:"foods" doc:"Foods eaten at this meal lately, the most often eaten first."`
}

// Recent is what the user usually eats, per meal, before a day.
type Recent struct {
	Day   string       `json:"day" format:"date"`
	Meals []RecentMeal `json:"meals"`
}

// Recent looks at the days before day for the meals the user keeps eating.
func (s *Service) Recent(ctx context.Context, day time.Time) (Recent, error) {
	rows, err := s.q.EntriesBetween(ctx, diarydb.EntriesBetweenParams{
		FromDay: day.AddDate(0, 0, -recentDays), ToDay: day.AddDate(0, 0, -1),
	})
	if err != nil {
		return Recent{}, err
	}

	out := Recent{Day: formatDay(day)}
	for _, meal := range Meals {
		var entries []diarydb.FoodEntry
		for _, row := range rows {
			if row.Meal == meal {
				entries = append(entries, row)
			}
		}
		out.Meals = append(out.Meals, RecentMeal{Meal: meal, Last: lastMeal(entries), Foods: usualFoods(entries)})
	}
	return out, nil
}

// lastMeal is the latest day's entries. Entries come oldest first.
func lastMeal(entries []diarydb.FoodEntry) *PastMeal {
	if len(entries) == 0 {
		return nil
	}
	last := entries[len(entries)-1].Day
	meal := &PastMeal{Day: formatDay(last), Entries: []RecentFood{}}
	for _, row := range entries {
		if row.Day.Equal(last) {
			entry := recentFood(row)
			entry.Uses = 1
			meal.Entries = append(meal.Entries, entry)
			meal.Kcal += entry.Kcal
		}
	}
	meal.Kcal = math.Round(meal.Kcal*10) / 10
	return meal
}

// usualFoods counts each food once per time it was logged, keeping the
// amount of the last time.
func usualFoods(entries []diarydb.FoodEntry) []RecentFood {
	byFood := map[string]*RecentFood{}
	for _, row := range entries {
		key := foodKey(row)
		count := 1
		if seen, ok := byFood[key]; ok {
			count += seen.Uses
		}
		entry := recentFood(row)
		entry.Uses = count
		byFood[key] = &entry
	}
	out := make([]RecentFood, 0, len(byFood))
	for _, entry := range byFood {
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Uses != out[j].Uses {
			return out[i].Uses > out[j].Uses
		}
		if out[i].LastDay != out[j].LastDay {
			return out[i].LastDay > out[j].LastDay
		}
		return out[i].Name < out[j].Name
	})
	if len(out) > recentFoods {
		out = out[:recentFoods]
	}
	return out
}

// foodKey tells foods apart: by reference, or by name for a one off.
func foodKey(row diarydb.FoodEntry) string {
	switch {
	case row.FoodID != nil:
		return "food:" + row.FoodID.String()
	case row.LmvNumber != nil:
		return "lmv:" + strconv.Itoa(int(*row.LmvNumber))
	}
	return "name:" + strings.ToLower(row.Name)
}

func recentFood(row diarydb.FoodEntry) RecentFood {
	entry := toEntry(row)
	return RecentFood{
		FoodID: row.FoodID, LMVNumber: row.LmvNumber, Name: row.Name, Grams: row.Grams, Amount: row.Amount,
		Per100g: entry.Per100g, Kcal: entry.Kcal, LastDay: formatDay(row.Day),
	}
}
