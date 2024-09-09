package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func AnyEmpty(strings ...string) bool {
	for _, s := range strings {
		if s == "" {
			return true
		}
	}
	return false
}

func ReplaceEnvPropertiesInString(value string) (*string, error) {
	regex := regexp.MustCompile(`\$\{(.*?)\}`)
	matches := regex.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return &value, nil
	} else {
		for _, match := range matches {
			if len(match) == 2 {
				key := match[1]
				envValue := os.Getenv(key)
				if envValue == "" {
					return nil, fmt.Errorf("error: no system property found for key: [%s]", key)
				}
				replaced := strings.Replace(value, match[0], envValue, -1)
				return ReplaceEnvPropertiesInString(replaced)
			} else {
				return nil, fmt.Errorf("error: Invalid matched substring length: %d for match: [%s]", len(match), match)
			}

		}
		return &value, nil
	}
}

func GetEnvVariable(key string) (*string, error) {
	value, found := os.LookupEnv(key)
	if !found {
		return nil, fmt.Errorf("error: System property not found for key: [%s]", key)
	}
	return &value, nil
}

func GetEnvVariableOrDefault(key string, defaultValue string) string {
	value, err := GetEnvVariable(key)
	if err != nil {
		return defaultValue
	} else {
		return *value
	}
}
