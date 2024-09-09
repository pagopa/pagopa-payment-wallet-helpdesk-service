package utils

import (
	"os"
	"testing"
)

func TestAnyEmpty(t *testing.T) {
	testCases := []struct {
		description     string
		input           []string
		expectedOutcome bool
	}{
		{
			"Should return false for empty array",
			[]string{},
			false,
		},
		{
			"Should return false for array with all valued strings",
			[]string{"test", "test", "test"},
			false,
		},
		{
			"Should return true for array with an empty string",
			[]string{"test", ""},
			true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			actualValue := AnyEmpty(testCase.input...)
			if actualValue != testCase.expectedOutcome {
				t.Errorf("Expected: [%v], actual value: [%v]", testCase.expectedOutcome, actualValue)
			}
		})
	}
}

func TestReplaceEnvPropertiesInString(t *testing.T) {
	testCases := []struct {
		description     string
		input           string
		expectedOutcome string
		precondition    func()
		expectError     bool
	}{
		{
			"Should return the input string without any modification for missing system properties in input string",
			"test",
			"test",
			func() {},
			false,
		},
		{
			"Should replace system property in string in case of only one system property",
			"test-${SYSTEM_PROPERTY_1}",
			"test-test",
			func() { os.Setenv("SYSTEM_PROPERTY_1", "test") },
			false,
		},
		{
			"Should replace all system property instances in input string",
			"test-${SYSTEM_PROPERTY_1}-${SYSTEM_PROPERTY_1}",
			"test-test-test",
			func() { os.Setenv("SYSTEM_PROPERTY_1", "test") },
			false,
		},
		{
			"Should replace multiple system properties in input string",
			"test-${SYSTEM_PROPERTY_1}-${SYSTEM_PROPERTY_2}",
			"test-test1-test2",
			func() {
				os.Setenv("SYSTEM_PROPERTY_1", "test1")
				os.Setenv("SYSTEM_PROPERTY_2", "test2")
			},
			false,
		},
		{
			"Should fail for missing system property to be replaced",
			"test-${SYSTEM_PROPERTY_1}-${SYSTEM_PROPERTY_3}",
			"",
			func() {
				os.Setenv("SYSTEM_PROPERTY_1", "test1")
				os.Setenv("SYSTEM_PROPERTY_2", "test2")
			},
			true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			testCase.precondition()
			actualValue, err := ReplaceEnvPropertiesInString(testCase.input)
			if testCase.expectError {
				if err == nil {
					t.Errorf("Test expect error but it was null")
				}
			} else {
				replacedString := *actualValue
				if replacedString != testCase.expectedOutcome {
					t.Errorf("Expected: [%v], actual value: [%v]", testCase.expectedOutcome, replacedString)
				}
			}
		})
	}
}

func TestGetEnvVariable(t *testing.T) {
	testCases := []struct {
		description     string
		input           string
		expectedOutcome string
		precondition    func()
		expectError     bool
	}{
		{
			"Should return env variable successfully",
			"ENV_VAR_1",
			"env_var_value1",
			func() {
				os.Setenv("ENV_VAR_1", "env_var_value1")
			},
			false,
		},
		{
			"Should fail for missing env variable",
			"ENV_VAR-2",
			"",
			func() {},
			true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			testCase.precondition()
			actualValue, err := GetEnvVariable(testCase.input)
			if testCase.expectError {
				if err == nil {
					t.Errorf("Test expect error but it was null")
				}
			} else {
				recoveredEnvValue := *actualValue
				if recoveredEnvValue != testCase.expectedOutcome {
					t.Errorf("Expected: [%v], actual value: [%v]", testCase.expectedOutcome, recoveredEnvValue)
				}
			}
		})
	}

}

func TestGetEnvVariableOrDefault(t *testing.T) {
	testCases := []struct {
		description     string
		input           string
		defaultValue    string
		expectedOutcome string
		precondition    func()
	}{
		{
			"Should return env variable successfully",
			"ENV_VAR_1",
			"defaultValue",
			"env_var_value1",
			func() {
				os.Setenv("ENV_VAR_1", "env_var_value1")
			},
		},
		{
			"Should return default value for missing env variable",
			"ENV_VAR-2",
			"defaultValue",
			"defaultValue",
			func() {},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			testCase.precondition()
			actualValue := GetEnvVariableOrDefault(testCase.input, testCase.defaultValue)
			if actualValue != testCase.expectedOutcome {
				t.Errorf("Expected: [%v], actual value: [%v]", testCase.expectedOutcome, actualValue)
			}

		})
	}

}
