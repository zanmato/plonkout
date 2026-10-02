package schedule_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/zanmato/plonkout/server/internal/platform/schedule"
)

func TestAMalformedSpecIsRefused(t *testing.T) {
	s := schedule.New(slog.New(slog.DiscardHandler))
	err := s.Add(schedule.Job{Name: "typo", Spec: "@montly", Run: func(context.Context) error { return nil }})
	if err == nil {
		t.Fatal("a malformed spec should be refused")
	}
}

func TestJobsRunAtStartAndStopWithTheContext(t *testing.T) {
	s := schedule.New(slog.New(slog.DiscardHandler))
	started := make(chan struct{})
	if err := s.Add(schedule.Job{
		Name: "failing", Spec: "@yearly",
		Run: func(context.Context) error { return nil },
		AtStart: func(context.Context) error {
			close(started)
			return errors.New("logged, not fatal")
		},
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the start job did not run")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the scheduler did not stop with its context")
	}
}
