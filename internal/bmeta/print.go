package bmeta

import "fmt"

const defaultBuildMeta = "N/A" // Значение по умолчанию

// Print Распечатывает версию, дату и комит сборки.
func Print(version, date, commit string) {
	meta := struct {
		version string
		date    string
		commit  string
	}{
		version: defaultBuildMeta,
		date:    defaultBuildMeta,
		commit:  defaultBuildMeta,
	}
	if version != "" {
		meta.version = version
	}
	if date != "" {
		meta.date = date
	}
	if commit != "" {
		meta.commit = commit
	}

	fmt.Printf("Build version: %s\n", meta.version) //nolint:forbidigo
	fmt.Printf("Build date: %s\n", meta.date)       //nolint:forbidigo
	fmt.Printf("Build commit: %s\n", meta.commit)   //nolint:forbidigo
}
