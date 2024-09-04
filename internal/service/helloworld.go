package helloworld

import "fmt"

// Helloworld function
func hello(name string) string {
	return fmt.Sprintf("Formatted: [%s]", name)
}
