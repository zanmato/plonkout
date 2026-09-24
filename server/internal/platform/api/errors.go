package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// The kinds a service error can have. A service wraps one of these rather than
// building a Problem, so it stays free of HTTP:
// fmt.Errorf("%w: no such plan", api.ErrNotFound).
var (
	// ErrNotFound means the row does not exist or belongs to somebody else,
	// which from outside must look the same.
	ErrNotFound = errors.New("not found")
	// ErrConflict means the request collides with the current state.
	ErrConflict = errors.New("conflict")
	// ErrInvalid means the request was understood and is not acceptable.
	ErrInvalid = errors.New("invalid")
	// ErrUnauthorized means the caller is not known.
	ErrUnauthorized = errors.New("unauthorized")
)

var kinds = []struct {
	sentinel error
	status   int
	code     string
}{
	{ErrNotFound, http.StatusNotFound, "not_found"},
	{ErrConflict, http.StatusConflict, "conflict"},
	{ErrInvalid, http.StatusUnprocessableEntity, "invalid_request"},
	{ErrUnauthorized, http.StatusUnauthorized, "unauthorized"},
}

// Postgres error classes the mapper answers for, by SQLSTATE and never by
// message.
const (
	sqlStateUniqueViolation     = "23505"
	sqlStateForeignKeyViolation = "23503"
	sqlStateNotNullViolation    = "23502"
	sqlStateCheckViolation      = "23514"
	sqlStateStringTooLong       = "22001"
	sqlStateNumericOutOfRange   = "22003"
	sqlStateInvalidText         = "22P02"
	sqlStateRLSViolation        = "42501"
)

// MapError turns anything a handler returns into a problem document. A Problem
// passes through, a known kind becomes its status, a Postgres error becomes
// its SQLSTATE's answer, and everything else is a 500 that says nothing,
// because an unmapped error text is as likely to hold a row as anything useful.
func MapError(err error) *Problem {
	if err == nil {
		return nil
	}
	if problem, ok := AsProblem(err); ok {
		return problem
	}

	var status huma.StatusError
	if errors.As(err, &status) {
		return Errorf(status.GetStatus(), codeForStatus(status.GetStatus()), "%s", err.Error())
	}

	for _, kind := range kinds {
		if errors.Is(err, kind.sentinel) {
			return Errorf(kind.status, kind.code, "%s", trimKind(err.Error(), kind.sentinel.Error()))
		}
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return Errorf(http.StatusNotFound, "not_found", "not found")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case sqlStateUniqueViolation:
			return Errorf(http.StatusConflict, "already_exists", "a record with these values already exists")
		case sqlStateForeignKeyViolation:
			return Errorf(http.StatusUnprocessableEntity, "invalid_reference", "a referenced record does not exist")
		case sqlStateNotNullViolation, sqlStateCheckViolation, sqlStateStringTooLong,
			sqlStateNumericOutOfRange, sqlStateInvalidText:
			return Errorf(http.StatusUnprocessableEntity, "invalid_value", "a value is not acceptable")
		case sqlStateRLSViolation:
			// A write the policies refused looks exactly like a missing row.
			return Errorf(http.StatusNotFound, "not_found", "not found")
		}
	}

	if errors.Is(err, context.Canceled) {
		return Errorf(http.StatusServiceUnavailable, "request_canceled", "the request was cancelled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return Errorf(http.StatusGatewayTimeout, "timeout", "the request took too long")
	}

	return Errorf(http.StatusInternalServerError, "internal_error", "the request could not be completed")
}

// trimKind drops the sentinel's own text from a wrapped message, so
// "not found: no such plan" reads "no such plan".
func trimKind(message, kind string) string {
	trimmed := strings.TrimPrefix(message, kind+": ")
	if trimmed == "" || trimmed == kind {
		return kind
	}
	return trimmed
}
