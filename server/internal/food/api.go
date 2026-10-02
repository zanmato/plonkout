package food

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid/v5"
	"github.com/zanmato/plonkout/server/internal/platform/api"
)

// Register declares the food operations.
func Register(reg *api.Registry, s *Service) {
	tags := []string{"foods"}
	type oneFood struct{ Body Food }

	api.Register(reg, api.Op{
		ID: "search-foods", Method: http.MethodGet, Path: "/foods/search",
		Summary: "Search foods by name", Tags: tags, MCPTool: "search_foods",
		Description: "The user's own foods and Livsmedelsverket's database in one list, best match first, " +
			"with nutrients per 100 g and the user's portions. Livsmedelsverket names foods in Swedish and " +
			"generically, e.g. \"Pasta kokt u. salt\" or \"Korv falukorv kött 58%\", and often lists a food " +
			"both raw and cooked: pick the state it was eaten in. Inflections and most compounds are understood " +
			"(\"kokta potatisar\", \"kycklingfilé\"). When a search finds nothing, try fewer or more general " +
			"words. Foods the user logs often and names they used before come first, and a barcode finds the " +
			"user's food with it.",
	}, func(ctx context.Context, in *struct {
		Query string `query:"q" maxLength:"100" doc:"Part of a name, in Swedish for Livsmedelsverket's foods, or a barcode."`
		Limit int32  `query:"limit" minimum:"1" maximum:"50" default:"20"`
	}) (*struct{ Body SearchResult }, error) {
		result, err := s.Search(ctx, in.Query, in.Limit)
		if err != nil {
			return nil, err
		}
		return &struct{ Body SearchResult }{Body: result}, nil
	})

	api.Register(reg, api.Op{
		ID: "list-foods", Method: http.MethodGet, Path: "/foods",
		Summary: "The user's own foods", Tags: tags,
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []Food }, error) {
		foods, err := s.List(ctx)
		if err != nil {
			return nil, err
		}
		return &struct{ Body []Food }{Body: foods}, nil
	})

	api.Register(reg, api.Op{
		ID: "create-food", Method: http.MethodPost, Path: "/foods",
		Summary: "Add a food of the user's own", Tags: tags, MCPTool: "create_food",
		Description: "For a product Livsmedelsverket does not have, from the nutrition label per 100 g. " +
			"Search first: the user may have added it before.",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusConflict},
	}, func(ctx context.Context, in *struct{ Body FoodInput }) (*oneFood, error) {
		f, err := s.Create(ctx, in.Body)
		if err != nil {
			return nil, err
		}
		return &oneFood{Body: f}, nil
	})

	api.Register(reg, api.Op{
		ID: "update-food", Method: http.MethodPut, Path: "/foods/{id}",
		Summary: "Change a food of the user's own", Tags: tags, MCPTool: "update_food",
		Description: "Days it was already logged on keep the values they were logged with.",
		Errors:      []int{http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *struct {
		ID   uuid.UUID `path:"id"`
		Body FoodInput
	}) (*oneFood, error) {
		f, err := s.Update(ctx, in.ID, in.Body)
		if err != nil {
			return nil, err
		}
		return &oneFood{Body: f}, nil
	})

	api.Register(reg, api.Op{
		ID: "delete-food", Method: http.MethodDelete, Path: "/foods/{id}",
		Summary:     "Delete a food of the user's own",
		Description: "Days it was logged on keep what they logged.",
		Tags:        tags, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*struct{}, error) {
		return nil, s.Delete(ctx, in.ID)
	})

	type portionBody struct {
		Ref
		PortionInput
	}
	api.Register(reg, api.Op{
		ID: "save-portion", Method: http.MethodPost, Path: "/food-portions",
		Summary: "Save a household measure of a food", Tags: tags, MCPTool: "save_portion",
		Description: "E.g. a tablespoon of pesto is 15 g. Saving a name again changes its weight. " +
			"For a food of either kind, by foodId or lmvNumber.",
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *struct{ Body portionBody }) (*struct{ Body Portion }, error) {
		p, err := s.SavePortion(ctx, in.Body.Ref, in.Body.PortionInput)
		if err != nil {
			return nil, err
		}
		return &struct{ Body Portion }{Body: p}, nil
	})

	api.Register(reg, api.Op{
		ID: "delete-portion", Method: http.MethodDelete, Path: "/food-portions/{id}",
		Summary: "Delete a household measure", Tags: tags, DefaultStatus: http.StatusNoContent,
		Errors: []int{http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID uuid.UUID `path:"id"`
	}) (*struct{}, error) {
		return nil, s.DeletePortion(ctx, in.ID)
	})
}
