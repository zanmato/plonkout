// Package food is what can be eaten: the user's own foods, and the foods of
// Livsmedelsverket's database, searched as one list.
package food

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/food/fooddb"
	"github.com/zanmato/plonkout/server/internal/food/lmv"
	"github.com/zanmato/plonkout/server/internal/platform/api"
	"github.com/zanmato/plonkout/server/internal/platform/db"
)

// Sources of a food.
const (
	SourceMine = "mine"
	SourceLMV  = "lmv"
)

// Nutrients are per 100 g. The four main values are always known, the rest
// only when the label or the database has them.
type Nutrients struct {
	Kcal         float64  `json:"kcal" minimum:"0" maximum:"900"`
	Protein      float64  `json:"protein" minimum:"0" maximum:"100" doc:"Grams."`
	Carbs        float64  `json:"carbs" minimum:"0" maximum:"100" doc:"Grams of available carbohydrates."`
	Fat          float64  `json:"fat" minimum:"0" maximum:"100" doc:"Grams."`
	Fiber        *float64 `json:"fiber" required:"false" minimum:"0" maximum:"100"`
	Sugars       *float64 `json:"sugars" required:"false" minimum:"0" maximum:"100"`
	SaturatedFat *float64 `json:"saturatedFat" required:"false" minimum:"0" maximum:"100"`
	Salt         *float64 `json:"salt" required:"false" minimum:"0" maximum:"100"`
}

// Portion is a household measure of a food.
type Portion struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name" example:"tbsp"`
	Grams float64   `json:"grams"`
}

// PortionInput names a portion of a food.
type PortionInput struct {
	Name  string  `json:"name" minLength:"1" maxLength:"40" example:"slice"`
	Grams float64 `json:"grams" exclusiveMinimum:"0" maximum:"5000"`
}

// Match is a search result.
type Match struct {
	Source    string     `json:"source" enum:"mine,lmv" doc:"mine for the user's own foods, lmv for Livsmedelsverket's."`
	FoodID    *uuid.UUID `json:"foodId" doc:"Set for the user's own foods."`
	LMVNumber *int32     `json:"lmvNumber" doc:"Set for Livsmedelsverket's foods."`
	Name      string     `json:"name"`
	Brand     string     `json:"brand"`
	Group     string     `json:"group" doc:"Livsmedelsverket's food group."`
	Per100g   Nutrients  `json:"per100g"`
	Portions  []Portion  `json:"portions" doc:"The user's household measures of this food."`
	Uses      int32      `json:"uses" doc:"How many times the user logged it in the last 90 days."`
}

// DataSource credits the food database, as its license asks.
type DataSource struct {
	Name        string     `json:"name"`
	License     string     `json:"license"`
	LicenseURL  string     `json:"licenseUrl"`
	URL         string     `json:"url"`
	Version     string     `json:"version" doc:"The release imported, empty until the first import."`
	Attribution string     `json:"attribution" doc:"Show this wherever its foods are shown."`
	ImportedAt  *time.Time `json:"importedAt"`
}

// SearchResult is a search.
type SearchResult struct {
	Matches []Match    `json:"matches"`
	Source  DataSource `json:"source"`
}

// Food is one of the user's own foods.
type Food struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Brand    string    `json:"brand"`
	GTIN     *string   `json:"gtin" doc:"The barcode."`
	Per100g  Nutrients `json:"per100g"`
	Notes    string    `json:"notes"`
	Portions []Portion `json:"portions"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

// FoodInput is what a client writes. Portions are saved by name, so giving
// one again changes its weight and leaving one out keeps it.
type FoodInput struct {
	Name     string         `json:"name" minLength:"1" maxLength:"120"`
	Brand    string         `json:"brand,omitempty" maxLength:"80"`
	GTIN     *string        `json:"gtin,omitempty" pattern:"^[0-9]{8,14}$" doc:"The barcode, 8 to 14 digits."`
	Per100g  Nutrients      `json:"per100g" doc:"As on the label, per 100 g."`
	Notes    string         `json:"notes,omitempty" maxLength:"500"`
	Portions []PortionInput `json:"portions,omitempty" maxItems:"20"`
}

// Ref names a food of either kind.
type Ref struct {
	FoodID    *uuid.UUID `json:"foodId,omitempty" doc:"One of the user's own foods."`
	LMVNumber *int32     `json:"lmvNumber,omitempty" doc:"A food of Livsmedelsverket's database."`
}

// Snapshot is a food as it reads now, for logging it.
type Snapshot struct {
	Ref
	Name    string
	Per100g Nutrients
}

// Service is the food module.
type Service struct {
	pool *pgxpool.Pool
	q    *fooddb.Queries

	// The vocabulary of the latest release, to split compounds with. Loaded
	// once per release.
	mu         sync.Mutex
	release    uuid.UUID
	vocabulary lmv.Vocabulary
}

// NewService builds the food module.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, q: fooddb.New(pool)}
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// Search finds foods by name, in Swedish with its inflections and
// compounds, the user's own by brand and barcode too, and foods by what the
// user has called them.
func (s *Service) Search(ctx context.Context, query string, limit int32) (SearchResult, error) {
	query = strings.TrimSpace(query)
	out := SearchResult{Matches: []Match{}}
	release, err := s.latestRelease(ctx)
	if err != nil {
		return SearchResult{}, err
	}
	out.Source = source(release)
	if query == "" {
		return out, nil
	}
	vocabulary, err := s.vocabularyOf(ctx, release)
	if err != nil {
		return SearchResult{}, err
	}

	escaped := likeEscaper.Replace(query)
	params := fooddb.SearchFoodsParams{Query: escaped, Words: strings.Fields(escaped), MaxResults: limit}
	if tsquery := vocabulary.Query(query); tsquery != "" {
		params.Tsquery = &tsquery
	}
	rows, err := s.q.SearchFoods(ctx, params)
	if err != nil {
		return SearchResult{}, err
	}
	var foodIDs []uuid.UUID
	var numbers []int32
	for _, row := range rows {
		match := Match{
			Source: row.Source, Name: row.Name, Brand: row.Brand, Group: row.FoodGroup, Uses: row.Uses,
			Per100g: Nutrients{
				Kcal: row.Kcal, Protein: row.Protein, Carbs: row.Carbs, Fat: row.Fat, Fiber: row.Fiber,
				Sugars: row.Sugars, SaturatedFat: row.SaturatedFat, Salt: row.Salt,
			},
		}
		if row.Source == SourceMine {
			id, err := uuid.FromString(row.Ref)
			if err != nil {
				return SearchResult{}, err
			}
			match.FoodID = &id
			foodIDs = append(foodIDs, id)
		} else {
			number, err := strconv.ParseInt(row.Ref, 10, 32)
			if err != nil {
				return SearchResult{}, err
			}
			n := int32(number)
			match.LMVNumber = &n
			numbers = append(numbers, n)
		}
		out.Matches = append(out.Matches, match)
	}

	portions, err := s.portions(ctx, foodIDs, numbers)
	if err != nil {
		return SearchResult{}, err
	}
	for i := range out.Matches {
		out.Matches[i].Portions = portions.of(out.Matches[i].FoodID, out.Matches[i].LMVNumber)
	}
	return out, nil
}

// Source describes the food database and the release imported.
func (s *Service) Source(ctx context.Context) (DataSource, error) {
	release, err := s.latestRelease(ctx)
	if err != nil {
		return DataSource{}, err
	}
	return source(release), nil
}

func source(release *fooddb.LatestReleaseRow) DataSource {
	out := DataSource{
		Name: "Livsmedelsverkets livsmedelsdatabas", License: "CC BY 4.0",
		LicenseURL: "https://creativecommons.org/licenses/by/4.0/",
		URL:        "https://www.livsmedelsverket.se/livsmedelsdatabasen", Attribution: lmv.Attribution,
	}
	if release != nil {
		out.Version = release.Version
		out.ImportedAt = &release.ImportedAt
		out.Attribution = fmt.Sprintf("%s version %s, CC BY 4.0", out.Name, release.Version)
	}
	return out
}

// latestRelease returns the release imported last, nil before the first.
func (s *Service) latestRelease(ctx context.Context) (*fooddb.LatestReleaseRow, error) {
	release, err := s.q.LatestRelease(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &release, nil
}

// vocabularyOf returns the words of a release, empty before the first.
func (s *Service) vocabularyOf(ctx context.Context, release *fooddb.LatestReleaseRow) (lmv.Vocabulary, error) {
	if release == nil {
		return lmv.Vocabulary{}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.release != release.ID {
		words, err := s.q.ReleaseVocabulary(ctx, release.ID)
		if err != nil {
			return nil, err
		}
		s.release, s.vocabulary = release.ID, lmv.VocabularyOf(words)
	}
	return s.vocabulary, nil
}

// searchTerms derives the compound parts of a food of the user's own.
func (s *Service) searchTerms(ctx context.Context, in FoodInput) (string, error) {
	release, err := s.latestRelease(ctx)
	if err != nil {
		return "", err
	}
	vocabulary, err := s.vocabularyOf(ctx, release)
	if err != nil {
		return "", err
	}
	return vocabulary.Terms(in.Name + " " + in.Brand), nil
}

// SaveAlias remembers that the user calls a food this, so searching for it
// finds the food first. An alias that is just the food's name is not kept.
func (s *Service) SaveAlias(ctx context.Context, alias string, snapshot Snapshot) error {
	alias = strings.TrimSpace(alias)
	if alias == "" || strings.EqualFold(alias, snapshot.Name) || len([]rune(alias)) > 60 {
		return nil
	}
	return s.q.SaveAlias(ctx, fooddb.SaveAliasParams{
		Alias: alias, FoodID: snapshot.FoodID, LmvNumber: snapshot.LMVNumber,
	})
}

// List returns the user's own foods.
func (s *Service) List(ctx context.Context) ([]Food, error) {
	rows, err := s.q.ListFoods(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	portions, err := s.portions(ctx, ids, nil)
	if err != nil {
		return nil, err
	}
	out := make([]Food, len(rows))
	for i, row := range rows {
		out[i] = toFood(row, portions.of(&row.ID, nil))
	}
	return out, nil
}

// Create adds a food with its portions.
func (s *Service) Create(ctx context.Context, in FoodInput) (Food, error) {
	terms, err := s.searchTerms(ctx, in)
	if err != nil {
		return Food{}, err
	}
	var id uuid.UUID
	err = db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		n := in.Per100g
		row, err := q.CreateFood(ctx, fooddb.CreateFoodParams{
			Name: strings.TrimSpace(in.Name), Brand: strings.TrimSpace(in.Brand), Gtin: in.GTIN,
			Kcal: n.Kcal, Protein: n.Protein, Carbs: n.Carbs, Fat: n.Fat, Fiber: n.Fiber, Sugars: n.Sugars,
			SaturatedFat: n.SaturatedFat, Salt: n.Salt, Notes: in.Notes, SearchTerms: terms,
		})
		if err != nil {
			return duplicate(err)
		}
		id = row.ID
		return savePortions(ctx, q, &row.ID, nil, in.Portions)
	})
	if err != nil {
		return Food{}, err
	}
	return s.get(ctx, id)
}

// Update changes a food. Days it was logged on keep what they logged.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in FoodInput) (Food, error) {
	terms, err := s.searchTerms(ctx, in)
	if err != nil {
		return Food{}, err
	}
	err = db.RunInTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		n := in.Per100g
		_, err := q.UpdateFood(ctx, fooddb.UpdateFoodParams{
			ID: id, Name: strings.TrimSpace(in.Name), Brand: strings.TrimSpace(in.Brand), Gtin: in.GTIN,
			Kcal: n.Kcal, Protein: n.Protein, Carbs: n.Carbs, Fat: n.Fat, Fiber: n.Fiber, Sugars: n.Sugars,
			SaturatedFat: n.SaturatedFat, Salt: n.Salt, Notes: in.Notes, SearchTerms: terms,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: no such food", api.ErrNotFound)
		} else if err != nil {
			return duplicate(err)
		}
		return savePortions(ctx, q, &id, nil, in.Portions)
	})
	if err != nil {
		return Food{}, err
	}
	return s.get(ctx, id)
}

// Delete removes a food. Days it was logged on keep what they logged.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeleteFood(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: no such food", api.ErrNotFound)
	}
	return nil
}

// SavePortion adds a portion to a food, or changes the weight of the one
// with that name.
func (s *Service) SavePortion(ctx context.Context, ref Ref, in PortionInput) (Portion, error) {
	if _, err := s.Resolve(ctx, ref); err != nil {
		return Portion{}, err
	}
	row, err := s.q.SavePortion(ctx, fooddb.SavePortionParams{
		FoodID: ref.FoodID, LmvNumber: ref.LMVNumber, Name: strings.TrimSpace(in.Name), Grams: in.Grams,
	})
	if err != nil {
		return Portion{}, err
	}
	return Portion{ID: row.ID, Name: row.Name, Grams: row.Grams}, nil
}

// DeletePortion removes a portion.
func (s *Service) DeletePortion(ctx context.Context, id uuid.UUID) error {
	n, err := s.q.DeletePortion(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: no such portion", api.ErrNotFound)
	}
	return nil
}

// Resolve reads the food a reference names, as it is now.
func (s *Service) Resolve(ctx context.Context, ref Ref) (Snapshot, error) {
	switch {
	case ref.FoodID != nil && ref.LMVNumber != nil:
		return Snapshot{}, api.Errorf(http.StatusUnprocessableEntity, "ambiguous_food", "name a food by foodId or lmvNumber, not both")
	case ref.FoodID != nil:
		row, err := s.q.GetFood(ctx, *ref.FoodID)
		if errors.Is(err, pgx.ErrNoRows) {
			return Snapshot{}, api.Errorf(http.StatusUnprocessableEntity, "unknown_food", "no food has the id %s", ref.FoodID)
		} else if err != nil {
			return Snapshot{}, err
		}
		name := row.Name
		if row.Brand != "" {
			name += ", " + row.Brand
		}
		return Snapshot{Ref: Ref{FoodID: &row.ID}, Name: name, Per100g: foodNutrients(row)}, nil
	case ref.LMVNumber != nil:
		row, err := s.q.GetLMVFood(ctx, *ref.LMVNumber)
		if errors.Is(err, pgx.ErrNoRows) {
			return Snapshot{}, api.Errorf(http.StatusUnprocessableEntity, "unknown_food", "Livsmedelsverket has no food number %d", *ref.LMVNumber)
		} else if err != nil {
			return Snapshot{}, err
		}
		return Snapshot{Ref: Ref{LMVNumber: &row.Number}, Name: row.Name, Per100g: Nutrients{
			Kcal: row.Kcal, Protein: row.Protein, Carbs: row.Carbs, Fat: row.Fat, Fiber: row.Fiber,
			Sugars: row.Sugars, SaturatedFat: row.SaturatedFat, Salt: row.Salt,
		}}, nil
	}
	return Snapshot{}, api.Errorf(http.StatusUnprocessableEntity, "unknown_food", "name a food by foodId or lmvNumber")
}

func (s *Service) get(ctx context.Context, id uuid.UUID) (Food, error) {
	row, err := s.q.GetFood(ctx, id)
	if err != nil {
		return Food{}, err
	}
	portions, err := s.portions(ctx, []uuid.UUID{id}, nil)
	if err != nil {
		return Food{}, err
	}
	return toFood(row, portions.of(&id, nil)), nil
}

type portionIndex struct {
	byFood map[uuid.UUID][]Portion
	byLMV  map[int32][]Portion
}

func (s *Service) portions(ctx context.Context, foodIDs []uuid.UUID, numbers []int32) (portionIndex, error) {
	index := portionIndex{byFood: map[uuid.UUID][]Portion{}, byLMV: map[int32][]Portion{}}
	if len(foodIDs) == 0 && len(numbers) == 0 {
		return index, nil
	}
	if foodIDs == nil {
		foodIDs = []uuid.UUID{}
	}
	if numbers == nil {
		numbers = []int32{}
	}
	rows, err := s.q.PortionsOf(ctx, fooddb.PortionsOfParams{FoodIds: foodIDs, LmvNumbers: numbers})
	if err != nil {
		return index, err
	}
	for _, row := range rows {
		portion := Portion{ID: row.ID, Name: row.Name, Grams: row.Grams}
		if row.FoodID != nil {
			index.byFood[*row.FoodID] = append(index.byFood[*row.FoodID], portion)
		} else if row.LmvNumber != nil {
			index.byLMV[*row.LmvNumber] = append(index.byLMV[*row.LmvNumber], portion)
		}
	}
	return index, nil
}

func (p portionIndex) of(foodID *uuid.UUID, number *int32) []Portion {
	var out []Portion
	if foodID != nil {
		out = p.byFood[*foodID]
	} else if number != nil {
		out = p.byLMV[*number]
	}
	if out == nil {
		return []Portion{}
	}
	return out
}

func savePortions(ctx context.Context, q *fooddb.Queries, foodID *uuid.UUID, number *int32, portions []PortionInput) error {
	for _, p := range portions {
		if _, err := q.SavePortion(ctx, fooddb.SavePortionParams{
			FoodID: foodID, LmvNumber: number, Name: strings.TrimSpace(p.Name), Grams: p.Grams,
		}); err != nil {
			return err
		}
	}
	return nil
}

func duplicate(err error) error {
	if api.MapError(err).Code == "already_exists" {
		return api.Errorf(http.StatusConflict, "food_exists", "a food with that name and brand, or that barcode, already exists")
	}
	return err
}

func foodNutrients(row fooddb.Food) Nutrients {
	return Nutrients{
		Kcal: row.Kcal, Protein: row.Protein, Carbs: row.Carbs, Fat: row.Fat, Fiber: row.Fiber,
		Sugars: row.Sugars, SaturatedFat: row.SaturatedFat, Salt: row.Salt,
	}
}

func toFood(row fooddb.Food, portions []Portion) Food {
	return Food{
		ID: row.ID, Name: row.Name, Brand: row.Brand, GTIN: row.Gtin, Per100g: foodNutrients(row),
		Notes: row.Notes, Portions: portions, Created: row.CreatedAt, Updated: row.UpdatedAt,
	}
}

// Scale returns a nutrient amount for a weight, rounded to a tenth.
func Scale(per100g, grams float64) float64 {
	return math.Round(per100g*grams/10) / 10
}
