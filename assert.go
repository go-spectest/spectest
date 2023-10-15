package spectest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Assert is a user defined custom assertion function
type Assert func(*http.Response, *http.Request) error

// TestingT is an interface to wrap the native *testing.T interface, this allows integration with GinkgoT() interface
// GinkgoT interface defined in https://github.com/onsi/ginkgo/blob/55c858784e51c26077949c81b6defb6b97b76944/ginkgo_dsl.go#L91
type TestingT interface {
	// Errorf is equivalent to Log followed by Fail
	Errorf(format string, args ...interface{})
	// Fatal is equivalent to Log followed by FailNow
	Fatal(args ...interface{})
	// Fatalf is equivalent to Log followed by FailNow
	Fatalf(format string, args ...interface{})
}

// failureMessageArgs are passed to the verifier but get stripped out from the user facing error message that gets printed
// it allows the test to pass additional info about the failure such as the test name.
type failureMessageArgs struct {
	// Name is the name of the test. It's is `SpecTest.name``
	Name string
}

// Verifier is the assertion interface allowing consumers to inject a custom assertion implementation.
// It also allows failure scenarios to be tested within spectest
type Verifier interface {
	Equal(t TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool
	True(t TestingT, value bool, msgAndArgs ...interface{}) bool
	JSONEq(t TestingT, expected string, actual string, msgAndArgs ...interface{}) bool
	Fail(t TestingT, failureMessage string, msgAndArgs ...interface{}) bool
	NoError(t TestingT, err error, msgAndArgs ...interface{}) bool
}

// DefaultVerifier is a verifier that uses some code from https://github.com/stretchr/testify to perform assertions
type DefaultVerifier struct{}

var _ Verifier = DefaultVerifier{}

// True asserts that the value is true
func (a DefaultVerifier) True(t TestingT, value bool, msgAndArgs ...interface{}) bool {
	if !value {
		return a.Fail(t, "Should be true", msgAndArgs...)
	}
	return true
}

// JSONEq asserts that two JSON strings are equivalent
func (a DefaultVerifier) JSONEq(t TestingT, expected string, actual string, msgAndArgs ...interface{}) bool {
	var expectedJSONAsInterface, actualJSONAsInterface interface{}

	if err := json.Unmarshal([]byte(expected), &expectedJSONAsInterface); err != nil {
		return a.Fail(t, fmt.Sprintf("Expected value ('%s') is not valid json.\nJSON parsing error: '%s'", expected, err.Error()), msgAndArgs...)
	}

	if err := json.Unmarshal([]byte(actual), &actualJSONAsInterface); err != nil {
		return a.Fail(t, fmt.Sprintf("Input ('%s') needs to be valid json.\nJSON parsing error: '%s'", actual, err.Error()), msgAndArgs...)
	}

	return a.Equal(t, expectedJSONAsInterface, actualJSONAsInterface, msgAndArgs...)
}

// Equal asserts that two values are equal
func (a DefaultVerifier) Equal(t TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	if err := validateEqualArgs(expected, actual); err != nil {
		return a.Fail(t, fmt.Sprintf("Invalid operation: %#v == %#v (%s)",
			expected, actual, err), msgAndArgs...)
	}

	// For non-error values, continue with the existing comparison logic
	if !objectsAreEqual(expected, actual) {
		diff := diff(expected, actual)
		expected, actual = formatUnequalValues(expected, actual)
		return a.Fail(t, fmt.Sprintf("Not equal: \n"+
			"expected: %s\n"+
			"actual  : %s%s", expected, actual, diff), msgAndArgs...)
	}
	return true
}

// Fail reports a failure
func (a DefaultVerifier) Fail(t TestingT, failureMessage string, msgAndArgs ...interface{}) bool {
	content := []labeledContent{
		{"Error Trace", strings.Join(callerInfo(), "\n\t\t\t")},
		{"Error", failureMessage},
	}

	// Add test name if the Go version supports it
	if n, ok := t.(interface {
		Name() string
	}); ok {
		content = append(content, labeledContent{"Test", n.Name()})
	}

	message := messageFromMsgAndArgs(msgAndArgs...)
	if len(message) > 0 {
		content = append(content, message...)
	}

	t.Errorf("\n%s", ""+labeledOutput(content...))

	return false
}

// NoError asserts that a function returned no error
func (a DefaultVerifier) NoError(t TestingT, err error, msgAndArgs ...interface{}) bool {
	if err != nil {
		return a.Fail(t, fmt.Sprintf("Received unexpected error:\n%+v", err), msgAndArgs...)
	}
	return true
}

func formatUnequalValues(expected, actual interface{}) (e string, a string) {
	if reflect.TypeOf(expected) != reflect.TypeOf(actual) {
		return fmt.Sprintf("%T(%s)", expected, truncatingFormat(expected)),
			fmt.Sprintf("%T(%s)", actual, truncatingFormat(actual))
	}
	switch expected.(type) {
	case time.Duration:
		return fmt.Sprintf("%v", expected), fmt.Sprintf("%v", actual)
	}
	return truncatingFormat(expected), truncatingFormat(actual)
}

func truncatingFormat(data interface{}) string {
	value := fmt.Sprintf("%#v", data)
	max := bufio.MaxScanTokenSize - 100 // Give us some space the type info too if needed.
	if len(value) > max {
		value = value[0:max] + "<... truncated>"
	}
	return value
}

func objectsAreEqual(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	exp, ok := expected.([]byte)
	if !ok {
		return reflect.DeepEqual(expected, actual)
	}

	act, ok := actual.([]byte)
	if !ok {
		return false
	}
	if exp == nil || act == nil {
		return exp == nil && act == nil
	}
	return bytes.Equal(exp, act)
}

func isFunction(arg interface{}) bool {
	if arg == nil {
		return false
	}
	return reflect.TypeOf(arg).Kind() == reflect.Func
}

func validateEqualArgs(expected, actual interface{}) error {
	if expected == nil && actual == nil {
		return nil
	}

	if isFunction(expected) || isFunction(actual) {
		return errors.New("cannot take func type as argument")
	}
	return nil
}

func messageFromMsgAndArgs(msgAndArgs ...interface{}) []labeledContent {
	if len(msgAndArgs) == 0 || msgAndArgs == nil {
		return nil
	}

	if len(msgAndArgs) == 1 {
		msg := msgAndArgs[0]
		if msgAsStr, ok := msg.(string); ok {
			return []labeledContent{{"Messages", msgAsStr}}
		}
		if failureMsg, ok := msg.(failureMessageArgs); ok {
			if failureMsg.Name == "" {
				return nil
			}
			return []labeledContent{{"Name", failureMsg.Name}}
		}
		return []labeledContent{{"Messages", fmt.Sprintf("%+v", msg)}}
	}

	if len(msgAndArgs) > 1 {
		var strMsgs []string
		var structuredMsg *labeledContent
		for _, msg := range msgAndArgs {
			if msgAsStr, ok := msg.(string); ok {
				strMsgs = append(strMsgs, msgAsStr)
			}
			if failureMsg, ok := msg.(failureMessageArgs); ok {
				if failureMsg.Name == "" {
					return nil
				}
				structuredMsg = &labeledContent{"Name", failureMsg.Name}
			}
		}
		combinedContent := []labeledContent{}
		if len(strMsgs) > 0 {
			combinedContent = append(combinedContent, labeledContent{"Messages", strings.Join(strMsgs, ", ")})
		}
		if structuredMsg != nil {
			combinedContent = append(combinedContent, *structuredMsg)
		}
		return combinedContent
	}
	return nil
}

func labeledOutput(content ...labeledContent) string {
	longestLabel := 0
	for _, v := range content {
		if len(v.label) > longestLabel {
			longestLabel = len(v.label)
		}
	}
	var output string
	for _, v := range content {
		output += "\t" + v.label + ":" + strings.Repeat(" ", longestLabel-len(v.label)) + "\t" + indentMessageLines(v.content, longestLabel) + "\n"
	}
	return output
}

func indentMessageLines(message string, longestLabelLen int) string {
	outBuf := new(bytes.Buffer)

	for i, scanner := 0, bufio.NewScanner(strings.NewReader(message)); scanner.Scan(); i++ {
		if i != 0 {
			outBuf.WriteString("\n\t" + strings.Repeat(" ", longestLabelLen+1) + "\t")
		}
		outBuf.WriteString(scanner.Text())
	}

	return outBuf.String()
}

func callerInfo() []string {
	var pc uintptr
	var ok bool
	var file string
	var line int
	var name string

	callers := []string{}
	for i := 0; ; i++ {
		pc, file, line, ok = runtime.Caller(i)
		if !ok {
			break
		}

		if file == "<autogenerated>" {
			break
		}

		f := runtime.FuncForPC(pc)
		if f == nil {
			break
		}
		name = f.Name()

		if name == "testing.tRunner" {
			break
		}

		parts := strings.Split(file, "/")
		file = parts[len(parts)-1]
		if len(parts) > 1 {
			dir := parts[len(parts)-2]
			if (dir != "assert" && dir != "mock" && dir != "require") || file == "mock_test.go" {
				callers = append(callers, fmt.Sprintf("%s:%d", file, line))
			}
		}

		segments := strings.Split(name, ".")
		name = segments[len(segments)-1]
		if isTest(name, "Test") ||
			isTest(name, "Benchmark") ||
			isTest(name, "Example") {
			break
		}
	}

	return callers
}

func isTest(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) { // "Test" is ok
		return true
	}
	r, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(r)
}

type labeledContent struct {
	label   string
	content string
}

// NoopVerifier is a verifier that does not perform verification
type NoopVerifier struct{}

var _ Verifier = NoopVerifier{}

// True is always true
func (n NoopVerifier) True(_ TestingT, _ bool, _ ...interface{}) bool {
	return true
}

// Equal does not perform any assertion and always returns true
func (n NoopVerifier) Equal(_ TestingT, _, _ interface{}, _ ...interface{}) bool {
	return true
}

// JSONEq does not perform any assertion and always returns true
func (n NoopVerifier) JSONEq(_ TestingT, _ string, _ string, _ ...interface{}) bool {
	return true
}

// Fail does not perform any assertion and always returns true
func (n NoopVerifier) Fail(_ TestingT, _ string, _ ...interface{}) bool {
	return true
}

// NoError asserts that a function returned no error
func (n NoopVerifier) NoError(_ TestingT, _ error, _ ...interface{}) bool {
	return true
}

// IsSuccess is a convenience function to assert on a range of happy path status codes
var IsSuccess Assert = func(response *http.Response, request *http.Request) error {
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusBadRequest {
		return nil
	}
	return fmt.Errorf("not success. Status code=%d", response.StatusCode)
}

// IsClientError is a convenience function to assert on a range of client error status codes
var IsClientError Assert = func(response *http.Response, request *http.Request) error {
	if response.StatusCode >= http.StatusBadRequest && response.StatusCode < http.StatusInternalServerError {
		return nil
	}
	return fmt.Errorf("not a client error. Status code=%d", response.StatusCode)
}

// IsServerError is a convenience function to assert on a range of server error status codes
var IsServerError Assert = func(response *http.Response, request *http.Request) error {
	if response.StatusCode >= http.StatusInternalServerError {
		return nil
	}
	return fmt.Errorf("not a server error. Status code=%d", response.StatusCode)
}
