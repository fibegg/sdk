package fibe

import (
	"fmt"
	"regexp"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
}

type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	msgs := make([]string, len(ve))
	for i, e := range ve {
		msgs[i] = fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return "fibe: validation failed: " + strings.Join(msgs, "; ")
}

type validator struct {
	errors ValidationErrors
}

func (v *validator) required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, ValidationError{Field: field, Message: "is required"})
	}
}

func (v *validator) requiredInt(field string, value int64) {
	if value == 0 {
		v.errors = append(v.errors, ValidationError{Field: field, Message: "is required"})
	}
}

func (v *validator) oneOf(field, value string, allowed []string) {
	if value == "" {
		return
	}
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")),
	})
}

var subdomainPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

func (v *validator) subdomain(field, value string) {
	if value == "" || value == "@" {
		return
	}
	if !subdomainPattern.MatchString(value) {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "must contain only lowercase letters, numbers, and hyphens",
		})
	}
}

func (v *validator) port(field string, value int) {
	if value == 0 {
		return
	}
	if value < 1 || value > 65535 {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "must be between 1 and 65535",
		})
	}
}

var secretKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (v *validator) secretKey(field, value string) {
	if value == "" {
		return
	}
	if !secretKeyPattern.MatchString(value) {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "must contain only letters, numbers, underscores, and hyphens",
		})
	}
}

func (v *validator) err() error {
	if len(v.errors) == 0 {
		return nil
	}
	return v.errors
}

type Validatable interface {
	Validate() error
}

func validateParams(p any) error {
	if v, ok := p.(Validatable); ok {
		return v.Validate()
	}
	return nil
}
