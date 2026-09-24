package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// Problem is an RFC 9457 problem document, the one error shape every endpoint
// answers with.
type Problem struct {
	// Type is a URI reference identifying the problem kind, about:blank when the
	// status says everything there is to say.
	Type string `json:"type" example:"about:blank"`
	// Title is a short human readable summary, stable per status.
	Title string `json:"title" example:"Not Found"`
	// Status repeats the HTTP status code.
	Status int `json:"status" example:"404"`
	// Detail explains this occurrence. It never carries a secret or a token.
	Detail string `json:"detail,omitempty"`
	// Code is the stable machine readable reason clients switch on.
	Code string `json:"code" example:"not_found"`
	// Errors carries per field failures for a form.
	Errors []ProblemDetail `json:"errors,omitempty"`

	headers http.Header
}

// ProblemDetail is one field level failure.
type ProblemDetail struct {
	// Location is a path into the request, for example body.name.
	Location string `json:"location,omitempty"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message"`
}

// WithHeader adds a response header to the refusal, e.g. WWW-Authenticate.
func (p *Problem) WithHeader(name, value string) *Problem {
	if p.headers == nil {
		p.headers = http.Header{}
	}
	p.headers.Set(name, value)
	return p
}

// GetHeaders is how Huma reads headers off an error value.
func (p *Problem) GetHeaders() http.Header { return p.headers }

func (p *Problem) Error() string { return fmt.Sprintf("%s: %s", p.Code, p.Title) }

// GetStatus satisfies huma.StatusError.
func (p *Problem) GetStatus() int { return p.Status }

// ContentType satisfies huma.ContentTypeFilter.
func (p *Problem) ContentType(string) string { return "application/problem+json" }

// Errorf builds a problem with a status, a code and a detail.
func Errorf(status int, code, format string, args ...any) *Problem {
	return &Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Code:   code,
		Detail: fmt.Sprintf(format, args...),
	}
}

// AsProblem unwraps a Problem from an error chain.
func AsProblem(err error) (*Problem, bool) {
	var problem *Problem
	if errors.As(err, &problem) {
		return problem, true
	}
	return nil, false
}

// UseProblemErrors makes Huma's own errors, validation above all, Problems.
// Called once at boot before any API is built.
func UseProblemErrors() {
	huma.NewError = func(status int, message string, errs ...error) huma.StatusError {
		problem := &Problem{
			Type:   "about:blank",
			Title:  http.StatusText(status),
			Status: status,
			Code:   codeForStatus(status),
			Detail: message,
		}
		for _, err := range errs {
			var detail huma.ErrorDetailer
			if errors.As(err, &detail) {
				d := detail.ErrorDetail()
				problem.Errors = append(problem.Errors, ProblemDetail{
					Location: d.Location,
					Code:     "invalid",
					Message:  d.Message,
				})
				continue
			}
			problem.Errors = append(problem.Errors, ProblemDetail{Message: err.Error()})
		}
		return problem
	}
}

func codeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusUnprocessableEntity:
		return "validation_failed"
	default:
		if status >= 500 {
			return "internal_error"
		}
		return "request_failed"
	}
}
