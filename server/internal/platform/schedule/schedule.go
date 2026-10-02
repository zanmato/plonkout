// Package schedule runs the server's background jobs on cron schedules.
//
// One server instance is all there is, so a job needs no lock to keep two
// from racing. A run still in progress when the next one is due is skipped,
// and a panic in a job is logged rather than taking the server down.
package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

// Job is one background job.
type Job struct {
	Name string
	// Spec is a standard five field cron expression or a descriptor such as
	// @hourly or @monthly.
	Spec string
	Run  func(context.Context) error
	// AtStart runs once when the scheduler starts, before the first scheduled
	// run. Nil runs nothing at start.
	AtStart func(context.Context) error
}

// Scheduler runs jobs until its context ends.
type Scheduler struct {
	logger *slog.Logger
	jobs   []Job
}

// New builds an empty scheduler.
func New(logger *slog.Logger) *Scheduler {
	return &Scheduler{logger: logger}
}

// Add registers a job. A malformed spec is an error, so a typo in the
// configuration stops the server rather than silently never running.
func (s *Scheduler) Add(job Job) error {
	if _, err := cron.ParseStandard(job.Spec); err != nil {
		return fmt.Errorf("schedule %s: %w", job.Name, err)
	}
	s.jobs = append(s.jobs, job)
	return nil
}

// Run starts the jobs and blocks until the context ends, then waits for any
// job still running.
func (s *Scheduler) Run(ctx context.Context) {
	logger := cronLogger{s.logger}
	c := cron.New(cron.WithChain(cron.Recover(logger), cron.SkipIfStillRunning(logger)))
	for _, job := range s.jobs {
		if _, err := c.AddFunc(job.Spec, func() { s.run(ctx, job.Name, job.Run) }); err != nil {
			// Add parsed the spec already.
			panic(err)
		}
	}
	c.Start()

	for _, job := range s.jobs {
		if job.AtStart != nil {
			go s.run(ctx, job.Name, job.AtStart)
		}
	}

	<-ctx.Done()
	<-c.Stop().Done()
}

func (s *Scheduler) run(ctx context.Context, name string, fn func(context.Context) error) {
	if ctx.Err() != nil {
		return
	}
	started := time.Now()
	if err := fn(ctx); err != nil && ctx.Err() == nil {
		s.logger.Error("job failed", "job", name, "error", err)
		return
	}
	s.logger.Debug("job done", "job", name, "took", time.Since(started))
}

// cronLogger hands the scheduler's own messages to slog.
type cronLogger struct{ logger *slog.Logger }

func (l cronLogger) Info(msg string, keysAndValues ...any) {
	l.logger.Debug("cron: "+msg, keysAndValues...)
}

func (l cronLogger) Error(err error, msg string, keysAndValues ...any) {
	l.logger.Error("cron: "+msg, append(keysAndValues, "error", err)...)
}
