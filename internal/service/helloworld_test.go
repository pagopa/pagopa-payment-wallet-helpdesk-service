package helloworld

import "testing"

func TestHelloWorld(t *testing.T) {
	name := "TEST"
	expectedResult := "Formatted: [TEST]"
	formatted := hello(name)
	if expectedResult != formatted {
		t.Fatalf("Expected %s, actual value: %s", expectedResult, formatted)
	}
}
