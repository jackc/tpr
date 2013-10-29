package main

import (
	"fmt"
)

type MissingError string

func (e MissingError) Error() string {
	return string(e)
}

type TooShortError struct {
	Minimum int
}

func (e TooShortError) Error() string {
	return fmt.Sprintf("too short (minimum: %d)", e.Minimum)
}

type TooLongError struct {
	Maximum int
}

func (e TooLongError) Error() string {
	return fmt.Sprintf("too long (maximum: %d)", e.Maximum)
}

type FieldErrors struct {
	Errors      []error
	IsFinalized bool
}

type FormErrors struct {
	fieldErrorCount int
}

func (fe *FormErrors) AddFieldError() {
	fe.fieldErrorCount++
}

func (fe *FormErrors) IsValid() bool {
	return fe.fieldErrorCount == 0
}

func (fe *FieldErrors) Add(e error) {
	if fe.IsFinalized {
		return
	}

	if fe.Errors == nil {
		fe.Errors = []error{e}
	} else {
		fe.Errors = append(fe.Errors, e)
	}
}

type StringField struct {
	Value  string
	Errors FieldErrors
}

func (f *StringField) ValidatePresence() {
	if f.Value == "" {
		f.Errors.Add(MissingError("missing"))
		f.Errors.IsFinalized = true
	}
}

func (f *StringField) ValidateMinimumLength(min int) {
	if len(f.Value) < min {
		f.Errors.Add(TooShortError{Minimum: min})
	}
}

func (f *StringField) ValidateMaximumLength(max int) {
	if len(f.Value) > max {
		f.Errors.Add(TooLongError{Maximum: max})
	}
}
