package config

import "fmt"

// Allowed values for plans.consumed.
const (
	ConsumedDelete  = "delete"
	ConsumedArchive = "archive"
)

// Allowed values for components[*].version.format.
const (
	FormatJSON = "json"
	FormatYAML = "yaml"
	FormatTOML = "toml"
	FormatText = "text"
)

// validate enforces the rules from the PRD acceptance criteria. It mutates cfg
// only to apply defaults (plans.consumed defaults to "delete" when blank).
func validate(cfg *Config) error {
	if cfg.Plans.Consumed == "" {
		cfg.Plans.Consumed = ConsumedDelete
	}
	switch cfg.Plans.Consumed {
	case ConsumedDelete, ConsumedArchive:
	default:
		return fmt.Errorf("plans.consumed: %q is not one of [%s, %s]",
			cfg.Plans.Consumed, ConsumedDelete, ConsumedArchive)
	}
	if cfg.Plans.Consumed == ConsumedArchive && cfg.Plans.ArchiveDir == "" {
		return fmt.Errorf("plans.archive_dir: required when plans.consumed is %q", ConsumedArchive)
	}
	if len(cfg.Components) == 0 {
		return fmt.Errorf("components: at least one component must be declared")
	}
	for name, comp := range cfg.Components {
		if len(comp.Paths) == 0 {
			return fmt.Errorf("components.%s.paths: at least one path glob is required", name)
		}
		if comp.Version.File == "" {
			return fmt.Errorf("components.%s.version.file: required", name)
		}
		if comp.Version.Format == "" {
			return fmt.Errorf("components.%s.version.format: required", name)
		}
		switch comp.Version.Format {
		case FormatJSON, FormatYAML, FormatTOML, FormatText:
		default:
			return fmt.Errorf("components.%s.version.format: %q is not one of [%s, %s, %s, %s]",
				name, comp.Version.Format, FormatJSON, FormatYAML, FormatTOML, FormatText)
		}
	}
	return nil
}
