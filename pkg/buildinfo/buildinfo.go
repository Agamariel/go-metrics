// Package buildinfo предоставляет функции для вывода информации о сборке приложения.
package buildinfo

import "fmt"

// orNA returns the string or "N/A" if empty
func orNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

// Print outputs build information to stdout
func Print(version, date, commit string) {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n",
		orNA(version), orNA(date), orNA(commit))
}
