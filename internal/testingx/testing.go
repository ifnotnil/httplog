package testingx

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func AssertError(t *testing.T, assertErrFn func(*testing.T, error), err error) {
	t.Helper()
	switch {
	case err != nil && assertErrFn != nil:
		assertErrFn(t, err)
	case err != nil && assertErrFn == nil:
		t.Errorf("unexpected error returned.\nError: %T(%s)", err, err.Error())
	case err == nil && assertErrFn != nil:
		t.Errorf("expected error but none received")
	}
}

func ExpectedErrorChecks(expected ...func(*testing.T, error)) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()
		for _, fn := range expected {
			fn(t, err)
		}
	}
}

func ExpectedErrorIs(allExpectedErrors ...error) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()
		for _, expected := range allExpectedErrors {
			if is := errors.Is(err, expected); !is {
				t.Errorf("error unexpected.\nExpected error: %T(%s) \nGot           : %T(%s)", expected, expected.Error(), err, err.Error())
			}
		}
	}
}

func ExpectedErrorIsOfType(expected error) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()
		if !errorIsOfType(err, expected) {
			t.Errorf("Error type check failed.\nExpected error type: %T\nGot                : %T(%s)", expected, err, err)
		}
	}
}

func errorIsOfType(err, expected error) bool {
	expectedType := reflect.TypeOf(expected)
	return atLeastOneError(err, func(e error) bool {
		tp := reflect.TypeOf(e)
		return tp == expectedType
	})
}

func atLeastOneError(err error, check func(error) bool) bool {
	if err == nil {
		return false
	}

	if check(err) {
		return true
	}

	switch x := err.(type) { //nolint:errorlint
	case interface{ Unwrap() error }:
		return atLeastOneError(x.Unwrap(), check)
	case interface{ Unwrap() []error }:
		for _, err := range x.Unwrap() {
			if atLeastOneError(err, check) {
				return true
			}
		}
		return false
	}

	return false
}

func ExpectedErrorStringContains(s string) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error string check failed. \nExpected to contain: %s\nGot                : %s\n", s, err.Error())
		}
	}
}

func ReadAndClose(t *testing.T, body io.ReadCloser) []byte {
	t.Helper()
	defer func() {
		t.Helper()
		if body == nil {
			return
		}
		err := body.Close()
		if err != nil {
			t.Errorf("error while closing body: %s", err.Error())
		}
	}()

	b, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("error while reading body: %s", err.Error())
	}

	return b
}

func AssertPartialEqualMap(t *testing.T, subject, expected map[string]any) {
	t.Helper()
	for expectedKey, expectedValue := range expected {
		gotValue, exists := subject[expectedKey]

		if !exists {
			t.Errorf("missing field %s", expectedKey)
			continue
		}

		if asMap, is := expectedValue.(map[string]any); is {
			if subjectAsMap, subIs := gotValue.(map[string]any); subIs {
				AssertPartialEqualMap(t, subjectAsMap, asMap)
				continue
			}
		}

		assert.Equal(t, expectedValue, gotValue)
	}
}
