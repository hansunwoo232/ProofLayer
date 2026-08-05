package scenario

import (
	"errors"
	"fmt"
	"sort"
)

var ErrNotAllowlisted = errors.New("scenario is not in the built-in allowlist")

type Definition struct {
	ID               string   `json:"scenario_id"`
	Version          string   `json:"scenario_version"`
	Handler          string   `json:"handler"`
	Platform         string   `json:"platform"`
	TimeoutSeconds   int      `json:"timeout_seconds"`
	CleanupHandler   string   `json:"cleanup_handler"`
	ExpectedProvider string   `json:"expected_provider"`
	ExpectedEventIDs []int    `json:"expected_event_ids"`
	RequiredFields   []string `json:"required_fields"`
}

type Catalog struct {
	definitions map[string]Definition
}

func BuiltInCatalog() Catalog {
	definition := Definition{
		ID:               "windows-process-marker",
		Version:          "0.1.0",
		Handler:          "builtin.emit_process_marker",
		Platform:         "windows",
		TimeoutSeconds:   30,
		CleanupHandler:   "builtin.verify_no_artifacts",
		ExpectedProvider: "sysmon",
		ExpectedEventIDs: []int{1},
		RequiredFields: []string{
			"host.name",
			"process.name",
			"process.command_line",
			"user.name",
		},
	}

	return Catalog{definitions: map[string]Definition{key(definition.ID, definition.Version): definition}}
}

func (c Catalog) Resolve(id, version string) (Definition, error) {
	definition, ok := c.definitions[key(id, version)]
	if !ok {
		return Definition{}, fmt.Errorf("%w: %s@%s", ErrNotAllowlisted, id, version)
	}
	return clone(definition), nil
}

func (c Catalog) List() []Definition {
	definitions := make([]Definition, 0, len(c.definitions))
	for _, definition := range c.definitions {
		definitions = append(definitions, clone(definition))
	}
	sort.Slice(definitions, func(i, j int) bool {
		if definitions[i].ID == definitions[j].ID {
			return definitions[i].Version < definitions[j].Version
		}
		return definitions[i].ID < definitions[j].ID
	})
	return definitions
}

func (c Catalog) Len() int {
	return len(c.definitions)
}

func key(id, version string) string {
	return id + "@" + version
}

func clone(definition Definition) Definition {
	definition.ExpectedEventIDs = append([]int(nil), definition.ExpectedEventIDs...)
	definition.RequiredFields = append([]string(nil), definition.RequiredFields...)
	return definition
}
