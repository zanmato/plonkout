// Package diary is the food diary: what was eaten and burned each day,
// against a daily goal, and body weight.
package diary

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/diary/diarydb"
	"github.com/zanmato/plonkout/server/internal/food"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/db"
)

// Meals of a day, in order.
var Meals = []string{"breakfast", "lunch", "dinner", "snack"}

// mealAims is the share of the day's budget each meal aims for, in percent.
var mealAims = map[string][2]float64{
	"breakfast": {25, 30},
	"lunch":     {30, 35},
	"dinner":    {25, 30},
	"snack":     {10, 15},
}

// maxRangeDays bounds a summary.
const maxRangeDays = 366

// Totals sums nutrients. Optional nutrients count what is known.
type Totals struct {
	Kcal         float64 `json:"kcal"`
	Protein      float64 `json:"protein"`
	Carbs        float64 `json:"carbs"`
	Fat          float64 `json:"fat"`
	Fiber        float64 `json:"fiber"`
	Sugars       float64 `json:"sugars"`
	SaturatedFat float64 `json:"saturatedFat"`
	Salt         float64 `json:"salt"`
}

// Entry is something eaten.
type Entry struct {
	ID        uuid.UUID      `json:"id"`
	Day       string         `json:"day" format:"date"`
	Meal      string         `json:"meal" enum:"breakfast,lunch,dinner,snack"`
	FoodID    *uuid.UUID     `json:"foodId"`
	LMVNumber *int32         `json:"lmvNumber"`
	Name      string         `json:"name" doc:"The food's name when it was logged."`
	Grams     float64        `json:"grams"`
	Amount    string         `json:"amount" doc:"How the amount was put, e.g. 2 tbsp."`
	Per100g   food.Nutrients `json:"per100g" doc:"As logged. Later changes to the food leave this as it was."`
	Kcal      float64        `json:"kcal"`
	Protein   float64        `json:"protein"`
	Carbs     float64        `json:"carbs"`
	Fat       float64        `json:"fat"`
	LoggedBy  string         `json:"loggedBy" enum:"app,assistant"`
	Created   time.Time      `json:"created"`
}

// Range is a span of kcal.
type Range struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// Meal is a meal of a day.
type Meal struct {
	Meal    string  `json:"meal" enum:"breakfast,lunch,dinner,snack"`
	Aim     *Range  `json:"aim" doc:"The kcal this meal aims for, a share of the goal. Null without a goal."`
	Totals  Totals  `json:"totals"`
	Entries []Entry `json:"entries"`
}

// Activity is energy burned by exercise.
type Activity struct {
	ID        uuid.UUID  `json:"id"`
	Day       string     `json:"day" format:"date"`
	Label     string     `json:"label"`
	Kcal      int32      `json:"kcal"`
	WorkoutID *uuid.UUID `json:"workoutId"`
	LoggedBy  string     `json:"loggedBy" enum:"app,assistant"`
	Created   time.Time  `json:"created"`
}

// GoalInput is the daily budget and what it is for.
type GoalInput struct {
	Direction     string   `json:"direction" enum:"lose,maintain,gain"`
	Kcal          int32    `json:"kcal" minimum:"800" maximum:"10000" doc:"The daily budget."`
	Protein       int32    `json:"protein" minimum:"0" maximum:"1000" doc:"Grams a day."`
	Carbs         int32    `json:"carbs" minimum:"0" maximum:"2000" doc:"Grams a day."`
	Fat           int32    `json:"fat" minimum:"0" maximum:"1000" doc:"Grams a day."`
	AddActivities bool     `json:"addActivities" doc:"Whether energy burned by activities is added to the day's budget."`
	TargetWeight  *float64 `json:"targetWeight" required:"false" exclusiveMinimum:"0" maximum:"999" doc:"In the user's weight unit."`
	WeeklyChange  *float64 `json:"weeklyChange" required:"false" minimum:"-2" maximum:"2" doc:"The pace aimed for, in the weight unit per week, negative to lose."`
	Notes         string   `json:"notes" required:"false" maxLength:"2000" doc:"How the goal was worked out, e.g. the maintenance estimate and the deficit."`
}

// Goal is the goal as stored.
type Goal struct {
	GoalInput
	Updated time.Time `json:"updated"`
}

// Day is one day of the diary.
type Day struct {
	Day        string     `json:"day" format:"date"`
	Goal       *Goal      `json:"goal" doc:"Null until a goal is set."`
	Eaten      Totals     `json:"eaten"`
	Burned     int32      `json:"burned" doc:"Kcal burned by activities."`
	Budget     *int32     `json:"budget" doc:"The goal's kcal, plus what was burned when the goal adds activities."`
	Remaining  *float64   `json:"remaining" doc:"Budget minus eaten. Negative when over."`
	Meals      []Meal     `json:"meals"`
	Activities []Activity `json:"activities"`
	Weight     *float64   `json:"weight" doc:"Body weight that day, in the user's weight unit."`
}

// DaySummary is a day in brief, for looking over a stretch of days.
type DaySummary struct {
	Day     string   `json:"day" format:"date"`
	Entries int      `json:"entries"`
	Eaten   Totals   `json:"eaten"`
	Burned  int32    `json:"burned"`
	Weight  *float64 `json:"weight"`
}

// Summary is a stretch of days.
type Summary struct {
	From string       `json:"from" format:"date"`
	To   string       `json:"to" format:"date"`
	Goal *Goal        `json:"goal"`
	Days []DaySummary `json:"days" doc:"Every day of the range, logged or not."`
}

// EntryInput is something eaten, of a food named by foodId or lmvNumber, or
// a one off with its own name and nutrients.
type EntryInput struct {
	Meal      string          `json:"meal" enum:"breakfast,lunch,dinner,snack"`
	FoodID    *uuid.UUID      `json:"foodId,omitempty" doc:"One of the user's own foods, from search_foods."`
	LMVNumber *int32          `json:"lmvNumber,omitempty" doc:"A food of Livsmedelsverket's, from search_foods."`
	Name      string          `json:"name,omitempty" maxLength:"120" doc:"Only for a one off without a food, e.g. a restaurant dish."`
	Per100g   *food.Nutrients `json:"per100g,omitempty" doc:"Only for a one off without a food: its nutrients per 100 g."`
	Grams     float64         `json:"grams" exclusiveMinimum:"0" maximum:"5000" doc:"The weight eaten. Convert household measures to grams."`
	Amount    string          `json:"amount,omitempty" maxLength:"60" doc:"How the user put it, e.g. 2 tbsp or 1 package."`
	Alias     string          `json:"alias,omitempty" maxLength:"60" doc:"What the user called the food when it is not its name, e.g. ölkorv. Remembered, so searching for it finds this food first."`
}

// EntryUpdate changes an entry's amount, meal or day.
type EntryUpdate struct {
	Day    string  `json:"day" format:"date"`
	Meal   string  `json:"meal" enum:"breakfast,lunch,dinner,snack"`
	Grams  float64 `json:"grams" exclusiveMinimum:"0" maximum:"5000"`
	Amount string  `json:"amount,omitempty" maxLength:"60"`
}

// ActivityInput is energy burned.
type ActivityInput struct {
	Label     string     `json:"label" minLength:"1" maxLength:"120" example:"Armwrestling, W3 A"`
	Kcal      int32      `json:"kcal" minimum:"1" maximum:"10000"`
	WorkoutID *uuid.UUID `json:"workoutId,omitempty" doc:"The workout the energy was burned in, if one was logged."`
}

// Service is the diary module.
type Service struct {
	pool  *pgxpool.Pool
	q     *diarydb.Queries
	foods *food.Service
}

// NewService builds the diary module.
func NewService(pool *pgxpool.Pool, foods *food.Service) *Service {
	return &Service{pool: pool, q: diarydb.New(pool), foods: foods}
}

// ParseDay reads a calendar day.
func ParseDay(text string) (time.Time, error) {
	day, err := time.Parse(time.DateOnly, text)
	if err != nil {
		return time.Time{}, api.Errorf(http.StatusUnprocessableEntity, "invalid_day", "%q is not a day like 2026-10-02", text)
	}
	return day, nil
}

func formatDay(day time.Time) string { return day.Format(time.DateOnly) }

// Get returns a day with its meals, activities and weight.
func (s *Service) Get(ctx context.Context, day time.Time) (Day, error) {
	goal, err := s.goal(ctx)
	if err != nil {
		return Day{}, err
	}
	entries, err := s.q.EntriesBetween(ctx, diarydb.EntriesBetweenParams{FromDay: day, ToDay: day})
	if err != nil {
		return Day{}, err
	}
	activities, err := s.q.ActivitiesBetween(ctx, diarydb.ActivitiesBetweenParams{FromDay: day, ToDay: day})
	if err != nil {
		return Day{}, err
	}
	weights, err := s.q.WeightsBetween(ctx, diarydb.WeightsBetweenParams{FromDay: day, ToDay: day})
	if err != nil {
		return Day{}, err
	}

	out := Day{Day: formatDay(day), Goal: goal, Activities: []Activity{}}
	meals := map[string]*Meal{}
	for _, name := range Meals {
		meal := Meal{Meal: name, Entries: []Entry{}}
		if goal != nil {
			aim := mealAims[name]
			meal.Aim = &Range{
				Min: int(math.Round(float64(goal.Kcal) * aim[0] / 100)),
				Max: int(math.Round(float64(goal.Kcal) * aim[1] / 100)),
			}
		}
		out.Meals = append(out.Meals, meal)
	}
	for i := range out.Meals {
		meals[out.Meals[i].Meal] = &out.Meals[i]
	}
	for _, row := range entries {
		entry := toEntry(row)
		meal := meals[entry.Meal]
		meal.Entries = append(meal.Entries, entry)
		add(&meal.Totals, row)
		add(&out.Eaten, row)
	}
	for _, meal := range out.Meals {
		roundTotals(&meal.Totals)
	}
	roundTotals(&out.Eaten)

	for _, row := range activities {
		out.Activities = append(out.Activities, toActivity(row))
		out.Burned += row.Kcal
	}
	if len(weights) > 0 {
		out.Weight = &weights[0].Weight
	}
	if goal != nil {
		budget := goal.Kcal
		if goal.AddActivities {
			budget += out.Burned
		}
		remaining := math.Round((float64(budget)-out.Eaten.Kcal)*10) / 10
		out.Budget, out.Remaining = &budget, &remaining
	}
	return out, nil
}

// Summarize returns every day of a range in brief.
func (s *Service) Summarize(ctx context.Context, from, to time.Time) (Summary, error) {
	if to.Before(from) {
		return Summary{}, api.Errorf(http.StatusUnprocessableEntity, "invalid_range", "from is after to")
	}
	if to.Sub(from) > maxRangeDays*24*time.Hour {
		return Summary{}, api.Errorf(http.StatusUnprocessableEntity, "invalid_range", "a summary covers at most %d days", maxRangeDays)
	}
	goal, err := s.goal(ctx)
	if err != nil {
		return Summary{}, err
	}
	entries, err := s.q.EntriesBetween(ctx, diarydb.EntriesBetweenParams{FromDay: from, ToDay: to})
	if err != nil {
		return Summary{}, err
	}
	activities, err := s.q.ActivitiesBetween(ctx, diarydb.ActivitiesBetweenParams{FromDay: from, ToDay: to})
	if err != nil {
		return Summary{}, err
	}
	weights, err := s.q.WeightsBetween(ctx, diarydb.WeightsBetweenParams{FromDay: from, ToDay: to})
	if err != nil {
		return Summary{}, err
	}

	out := Summary{From: formatDay(from), To: formatDay(to), Goal: goal}
	index := map[string]*DaySummary{}
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		out.Days = append(out.Days, DaySummary{Day: formatDay(day)})
	}
	for i := range out.Days {
		index[out.Days[i].Day] = &out.Days[i]
	}
	for _, row := range entries {
		day := index[formatDay(row.Day)]
		day.Entries++
		add(&day.Eaten, row)
	}
	for _, row := range activities {
		index[formatDay(row.Day)].Burned += row.Kcal
	}
	for _, row := range weights {
		index[formatDay(row.Day)].Weight = &row.Weight
	}
	for i := range out.Days {
		roundTotals(&out.Days[i].Eaten)
	}
	return out, nil
}

// Log adds what was eaten to a day, all or nothing.
func (s *Service) Log(ctx context.Context, day time.Time, inputs []EntryInput) error {
	loggedBy := "app"
	if api.ViaAssistant(ctx) {
		loggedBy = "assistant"
	}

	params := make([]diarydb.CreateEntryParams, len(inputs))
	snapshots := make([]food.Snapshot, len(inputs))
	for i, in := range inputs {
		snapshot, err := s.snapshot(ctx, in)
		if err != nil {
			return fmt.Errorf("entry %d: %w", i+1, err)
		}
		snapshots[i] = snapshot
		n := snapshot.Per100g
		params[i] = diarydb.CreateEntryParams{
			Day: day, Meal: in.Meal, FoodID: snapshot.FoodID, LmvNumber: snapshot.LMVNumber, Name: snapshot.Name,
			Grams: in.Grams, Amount: strings.TrimSpace(in.Amount), Kcal: n.Kcal, Protein: n.Protein,
			Carbs: n.Carbs, Fat: n.Fat, Fiber: n.Fiber, Sugars: n.Sugars, SaturatedFat: n.SaturatedFat,
			Salt: n.Salt, LoggedBy: loggedBy,
		}
	}
	// Every food is known to exist by now, and an alias of one is worth
	// keeping even if the entries fail.
	for i, in := range inputs {
		if in.Alias != "" && (snapshots[i].FoodID != nil || snapshots[i].LMVNumber != nil) {
			if err := s.foods.SaveAlias(ctx, in.Alias, snapshots[i]); err != nil {
				return err
			}
		}
	}
	return db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		for _, p := range params {
			if _, err := q.CreateEntry(ctx, p); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) snapshot(ctx context.Context, in EntryInput) (food.Snapshot, error) {
	if in.FoodID != nil || in.LMVNumber != nil {
		if in.Per100g != nil {
			return food.Snapshot{}, api.Errorf(http.StatusUnprocessableEntity, "ambiguous_food",
				"per100g is only for a one off, a food brings its own nutrients")
		}
		return s.foods.Resolve(ctx, food.Ref{FoodID: in.FoodID, LMVNumber: in.LMVNumber})
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || in.Per100g == nil {
		return food.Snapshot{}, api.Errorf(http.StatusUnprocessableEntity, "unknown_food",
			"name a food by foodId or lmvNumber, or give a one off its name and per100g")
	}
	return food.Snapshot{Name: name, Per100g: *in.Per100g}, nil
}

// UpdateEntry changes an entry's amount, meal or day.
func (s *Service) UpdateEntry(ctx context.Context, id uuid.UUID, in EntryUpdate) (Entry, error) {
	day, err := ParseDay(in.Day)
	if err != nil {
		return Entry{}, err
	}
	row, err := s.q.UpdateEntry(ctx, diarydb.UpdateEntryParams{
		ID: id, Day: day, Meal: in.Meal, Grams: in.Grams, Amount: strings.TrimSpace(in.Amount),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, fmt.Errorf("%w: no such entry", api.ErrNotFound)
	} else if err != nil {
		return Entry{}, err
	}
	return toEntry(row), nil
}

// DeleteEntry removes an entry.
func (s *Service) DeleteEntry(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteEntry(ctx, id)
	return deleted(n, err, "entry")
}

// LogActivity adds energy burned to a day.
func (s *Service) LogActivity(ctx context.Context, day time.Time, in ActivityInput) (Activity, error) {
	loggedBy := "app"
	if api.ViaAssistant(ctx) {
		loggedBy = "assistant"
	}
	row, err := s.q.CreateActivity(ctx, diarydb.CreateActivityParams{
		Day: day, Label: strings.TrimSpace(in.Label), Kcal: in.Kcal, WorkoutID: in.WorkoutID, LoggedBy: loggedBy,
	})
	if err != nil {
		return Activity{}, err
	}
	return toActivity(row), nil
}

// DeleteActivity removes an activity.
func (s *Service) DeleteActivity(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteActivity(ctx, id)
	return deleted(n, err, "activity")
}

// SaveWeight sets a day's body weight.
func (s *Service) SaveWeight(ctx context.Context, day time.Time, weight float64) error {
	_, err := s.q.SaveWeight(ctx, diarydb.SaveWeightParams{Day: day, Weight: weight})
	return err
}

// DeleteWeight removes a day's body weight.
func (s *Service) DeleteWeight(ctx context.Context, day time.Time) error {
	n, err := s.q.DeleteWeight(ctx, day)
	return deleted(n, err, "weight")
}

// Goal returns the goal, or not found when none is set.
func (s *Service) Goal(ctx context.Context) (Goal, error) {
	goal, err := s.goal(ctx)
	if err != nil {
		return Goal{}, err
	}
	if goal == nil {
		return Goal{}, fmt.Errorf("%w: no goal is set", api.ErrNotFound)
	}
	return *goal, nil
}

// SaveGoal sets the goal.
func (s *Service) SaveGoal(ctx context.Context, in GoalInput) (Goal, error) {
	row, err := s.q.SaveGoal(ctx, diarydb.SaveGoalParams{
		Direction: in.Direction, Kcal: in.Kcal, Protein: in.Protein, Carbs: in.Carbs, Fat: in.Fat,
		AddActivities: in.AddActivities, TargetWeight: in.TargetWeight, WeeklyChange: in.WeeklyChange,
		Notes: strings.TrimSpace(in.Notes),
	})
	if err != nil {
		return Goal{}, err
	}
	return toGoal(row), nil
}

func (s *Service) goal(ctx context.Context) (*Goal, error) {
	row, err := s.q.GetGoal(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	goal := toGoal(row)
	return &goal, nil
}

// deleted turns a delete that removed nothing into not found.
func deleted(n int64, err error, what string) error {
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: no such %s", api.ErrNotFound, what)
	}
	return nil
}

func add(t *Totals, row diarydb.FoodEntry) {
	t.Kcal += food.Scale(row.Kcal, row.Grams)
	t.Protein += food.Scale(row.Protein, row.Grams)
	t.Carbs += food.Scale(row.Carbs, row.Grams)
	t.Fat += food.Scale(row.Fat, row.Grams)
	for _, pair := range []struct {
		total *float64
		value *float64
	}{{&t.Fiber, row.Fiber}, {&t.Sugars, row.Sugars}, {&t.SaturatedFat, row.SaturatedFat}, {&t.Salt, row.Salt}} {
		if pair.value != nil {
			*pair.total += food.Scale(*pair.value, row.Grams)
		}
	}
}

// roundTotals drops the float noise sums of tenths pick up.
func roundTotals(t *Totals) {
	for _, v := range []*float64{&t.Kcal, &t.Protein, &t.Carbs, &t.Fat, &t.Fiber, &t.Sugars, &t.SaturatedFat, &t.Salt} {
		*v = math.Round(*v*10) / 10
	}
}

func toEntry(row diarydb.FoodEntry) Entry {
	return Entry{
		ID: row.ID, Day: formatDay(row.Day), Meal: row.Meal, FoodID: row.FoodID, LMVNumber: row.LmvNumber,
		Name: row.Name, Grams: row.Grams, Amount: row.Amount,
		Per100g: food.Nutrients{
			Kcal: row.Kcal, Protein: row.Protein, Carbs: row.Carbs, Fat: row.Fat, Fiber: row.Fiber,
			Sugars: row.Sugars, SaturatedFat: row.SaturatedFat, Salt: row.Salt,
		},
		Kcal: food.Scale(row.Kcal, row.Grams), Protein: food.Scale(row.Protein, row.Grams),
		Carbs: food.Scale(row.Carbs, row.Grams), Fat: food.Scale(row.Fat, row.Grams),
		LoggedBy: row.LoggedBy, Created: row.CreatedAt,
	}
}

func toActivity(row diarydb.Activity) Activity {
	return Activity{
		ID: row.ID, Day: formatDay(row.Day), Label: row.Label, Kcal: row.Kcal, WorkoutID: row.WorkoutID,
		LoggedBy: row.LoggedBy, Created: row.CreatedAt,
	}
}

func toGoal(row diarydb.NutritionGoal) Goal {
	return Goal{
		GoalInput: GoalInput{
			Direction: row.Direction, Kcal: row.Kcal, Protein: row.Protein, Carbs: row.Carbs, Fat: row.Fat,
			AddActivities: row.AddActivities, TargetWeight: row.TargetWeight, WeeklyChange: row.WeeklyChange,
			Notes: row.Notes,
		},
		Updated: row.UpdatedAt,
	}
}
