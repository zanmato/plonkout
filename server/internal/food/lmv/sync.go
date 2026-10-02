package lmv

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmato/plonkout/server/internal/food/fooddb"
	"github.com/zanmato/plonkout/server/internal/platform/db"
	"github.com/zanmato/plonkout/server/internal/platform/schedule"
)

// SourceURL answers a POST with the whole database as a spreadsheet.
const SourceURL = "https://soknaringsinnehall.livsmedelsverket.se/Spara/HamtaHelaDatabasen"

// maxFileBytes bounds a download. The file is under a megabyte.
const maxFileBytes = 50 << 20

// Syncer keeps the lmv schema in step with the published database.
type Syncer struct {
	Pool   *pgxpool.Pool
	Client *http.Client
	URL    string
	Logger *slog.Logger
}

// Result is what a sync did.
type Result struct {
	Version  string
	Imported bool
	Foods    int
	Retired  int64
}

// Job syncs on the schedule, and at start when nothing was ever imported, so
// a new installation has foods to search without waiting for the schedule.
func (s *Syncer) Job(spec string) schedule.Job {
	return schedule.Job{
		Name: "lmv-sync", Spec: spec,
		Run: func(ctx context.Context) error {
			_, err := s.Sync(ctx)
			return err
		},
		AtStart: func(ctx context.Context) error {
			if _, err := fooddb.New(s.Pool).LatestRelease(ctx); err == nil {
				return nil
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			_, err := s.Sync(ctx)
			return err
		},
	}
}

// Sync downloads the database and imports it unless that exact file was
// imported before.
func (s *Syncer) Sync(ctx context.Context) (Result, error) {
	file, err := s.download(ctx)
	if err != nil {
		return Result{}, err
	}
	result, err := Import(ctx, s.Pool, file)
	if err != nil {
		return Result{}, err
	}
	if result.Imported {
		s.Logger.Info("imported the Livsmedelsverket food database",
			"version", result.Version, "foods", result.Foods, "retired", result.Retired)
	} else {
		s.Logger.Debug("the Livsmedelsverket food database is unchanged")
	}
	return result, nil
}

func (s *Syncer) download(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	// The endpoint is a form post and answers 411 without a Content-Length,
	// which an empty, non nil body makes the client send.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.ContentLength = 0
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download the food database: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download the food database: %s", resp.Status)
	}
	file, err := io.ReadAll(io.LimitReader(resp.Body, maxFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("download the food database: %w", err)
	}
	if len(file) > maxFileBytes {
		return nil, fmt.Errorf("the food database is larger than %d bytes", maxFileBytes)
	}
	return file, nil
}

// Import stores a release file in one transaction. Foods are upserted by
// number, and those the release no longer has are retired rather than
// deleted, since portions refer to them. The words of the release's names are
// learned to split compounds with, see Vocabulary.
func Import(ctx context.Context, pool *pgxpool.Pool, file []byte) (Result, error) {
	sum := sha256.Sum256(file)
	q := fooddb.New(pool)
	seen, err := q.ReleaseImported(ctx, sum[:])
	if err != nil {
		return Result{}, err
	}
	if seen {
		return Result{}, nil
	}

	release, err := Parse(file)
	if err != nil {
		return Result{}, fmt.Errorf("parse the food database: %w", err)
	}

	names := make([]string, len(release.Foods))
	for i, food := range release.Foods {
		names[i] = food.Name
	}
	vocabulary := NewVocabulary(names)

	result := Result{Version: release.Version, Imported: true, Foods: len(release.Foods)}
	err = db.RunInTx(ctx, pool, func(tx pgx.Tx) error {
		q := q.WithTx(tx)
		numbers := make([]int32, len(release.Foods))
		for i, food := range release.Foods {
			numbers[i] = food.Number
			nutrients, err := json.Marshal(food.Nutrients)
			if err != nil {
				return err
			}
			if err := q.UpsertLMVFood(ctx, fooddb.UpsertLMVFoodParams{
				Number: food.Number, Name: food.Name, FoodGroup: food.Group, Kcal: food.Kcal,
				Protein: food.Protein, Carbs: food.Carbs, Fat: food.Fat, Fiber: food.Fiber,
				Sugars: food.Sugars, SaturatedFat: food.SaturatedFat, Salt: food.Salt, Nutrients: nutrients,
				SearchTerms: vocabulary.Terms(food.Name),
			}); err != nil {
				return fmt.Errorf("store food %d: %w", food.Number, err)
			}
		}
		if result.Retired, err = q.RetireLMVFoods(ctx, numbers); err != nil {
			return err
		}
		return q.InsertRelease(ctx, fooddb.InsertReleaseParams{
			Version: release.Version, Sha256: sum[:], Foods: int32(len(release.Foods)),
			Vocabulary: vocabulary.List(),
		})
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}
