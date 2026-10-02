package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// instructions tell a model how this domain works before it calls anything.
const instructions = `Plonkout is a workout log with training plans, often used for armwrestling, and a food diary.

Start with get_context: it has the weight unit (every weight anywhere is in it), the dominant arm, active plans with their next session, and recent workouts.

A plan is an ordered queue of sessions. The user opens the app, picks the next session (or skips it), and the app logs a workout prefilled with the session's targets. You never log workouts yourself; you plan and review.

- A session has a label such as "W3 A", optional week and day, and an intensity: heavy for a tough day (typically day A), light for a volume day (day B).
- Each planned exercise names an exercise from list_exercises and has targets. A target is a group of sets with one prescription: setType is a free label (Top set, Back-off, Volume (3s hold), Deload, TEST attempt 2), then sets, reps or time, weight and an optional RPE range.
- Single arm exercises: the weight is for the dominant arm. Set offArmPercent (for example 82.5) and the other arm works at that share.
- Put cues and conditional rules in the exercise's notes, e.g. "If week 6 moved fast, make attempt 4 182.5". When the condition has played out, apply it with update_session.
- Only pending sessions can be changed. Started, completed and skipped ones are history.
- Exercise names must match the list, ignoring case. An unknown name is refused with suggestions; set createMissingExercises only when the user really does a new exercise.

To review, call get_plan with actuals: every session with a workout carries planned against done per target, whether it was met, the top RPE and the best estimated 1RM (Epley). get_exercise_stats gives the longer history of one exercise.

The food diary counts what the user eats against a daily goal (nutritionGoal in get_context). Unlike workouts, you log food yourself when asked.

- A day is the user's own calendar day (YYYY-MM-DD). Work it out from what they say and the time of day; ask when it is unclear, for example just after midnight.
- To log "100 g pasta and two tablespoons of pesto for lunch": search_foods for each food, pick the match in the state it was eaten (cooked pasta, not dry), convert household measures to grams (a saved portion if the food has one, else a sensible estimate: a tablespoon of pesto is about 15 g), then log_food once with every entry. Put how the user said it in amount, e.g. "2 tbsp", and what they called a food in alias when it differs from the food's name, e.g. "ölkorv", so the next search finds it first.
- Livsmedelsverket's foods are generic and named in Swedish. For a branded product they lack, ask for the label's values per 100 g and create_food once, then reuse it. A dish eaten out can be logged as a one off with a name and estimated per100g.
- Tell the user what you logged, with grams and kcal, and say which values were estimates.
- Energy burned is not measured. Log it with log_activity only when the user gives it or asks for an estimate.
- Livsmedelsverket's data is CC BY 4.0: credit "Livsmedelsverkets livsmedelsdatabas" when you present its values.`

func addPrompts(server *mcp.Server) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "design_plan",
		Title:       "Design a training plan",
		Description: "Build a new plan from the user's goal, schedule and history.",
		Arguments: []*mcp.PromptArgument{
			{Name: "goal", Description: "What the plan should achieve, e.g. test a 180 kg bench in 9 weeks.", Required: true},
			{Name: "sessions_per_week", Description: "How many sessions a week, e.g. 2."},
			{Name: "notes", Description: "Anything else: injuries, equipment, exercises to include."},
		},
	}, func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := req.Params.Arguments
		var b strings.Builder
		fmt.Fprintf(&b, "Design a training plan in Plonkout. Goal: %s.\n", args["goal"])
		if v := args["sessions_per_week"]; v != "" {
			fmt.Fprintf(&b, "Sessions per week: %s.\n", v)
		}
		if v := args["notes"]; v != "" {
			fmt.Fprintf(&b, "Notes: %s.\n", v)
		}
		b.WriteString(`
1. Call get_context for units, dominant arm and any plan already running.
2. Call list_exercises, and get_exercise_stats for each main exercise to find current levels (best estimated 1RM, heaviest recent sets).
3. Propose the plan to the user in a short table first: weeks, A and B days, exercises, targets. Ask before creating it.
4. Once agreed, call create_plan with every session in the order they should be done. Give single arm exercises an offArmPercent, and put cues and conditional rules in the exercise notes.`)
		return &mcp.GetPromptResult{
			Description: "Design a training plan",
			Messages:    []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: b.String()}}},
		}, nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "nutrition_goal",
		Title:       "Set a nutrition goal",
		Description: "Work out a daily calorie and macro budget for losing, keeping or gaining weight.",
		Arguments: []*mcp.PromptArgument{
			{Name: "goal", Description: "What the user wants, e.g. lose 4 kg by Christmas without losing strength.", Required: true},
		},
	}, func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		text := fmt.Sprintf(`Help me set a nutrition goal in Plonkout. Goal: %s.

1. Call get_context for units, training and any goal already set, and summarize_diary for the last four weeks to see what I eat and weigh.
2. Ask for what you still need: height, age, sex, and how active I am outside training.
3. Estimate maintenance, then a budget for the pace: about 0.5%% of body weight a week at most when losing, slower when gaining. Keep protein high (1.6 to 2.2 g per kg) when strength matters, fat at least 0.6 g per kg, and the rest carbs.
4. Show the numbers and the reasoning, and ask before saving. Then call set_nutrition_goal, with the reasoning in notes.`, req.Params.Arguments["goal"])
		return &mcp.GetPromptResult{
			Description: "Set a nutrition goal",
			Messages:    []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}},
		}, nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "weekly_review",
		Title:       "Review the week",
		Description: "Compare what was planned with what was done and adjust what is coming up.",
	}, func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		text := `Review my training in Plonkout.

1. Call get_context, then get_plan with actuals for each active plan.
2. For the sessions done or skipped since the last review: which targets were met, which were missed, top RPE against the target range, and the best estimated 1RM against earlier weeks.
3. Check the notes of upcoming sessions for conditional rules whose condition has now played out.
4. Suggest concrete changes to upcoming pending sessions, with reasons. After I agree, apply them with update_session, add_sessions or reorder_sessions.`
		return &mcp.GetPromptResult{
			Description: "Weekly review",
			Messages:    []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}},
		}, nil
	})
}
