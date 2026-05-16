package config

type ValidateResult struct {
	Errors   []string
	Warnings []string
}

func (v *ValidateResult) HasErrors() bool {
	return len(v.Errors) > 0
}

func (v *ValidateResult) addError(msg string) {
	v.Errors = append(v.Errors, msg)
}

func (v *ValidateResult) HasWarnings() bool {
	return len(v.Warnings) > 0
}

func (v *ValidateResult) addWarning(msg string) {
	v.Warnings = append(v.Warnings, msg)
}

var knownPlatforms = map[string]bool{
	"linux": true, "darwin": true,
	"embedded": true, "wasm": true,
	// "windows": planned for future release
}

var knownArchitectures = map[string]bool{
	"x86_64": true, "arm64": true, "aarch64": true, "arm": true,
	"i386": true, "cortex-m0": true, "cortex-m3": true, "cortex-m4": true,
	"cortex-m7": true, "cortex-m33": true, "cortex-a53": true,
	"riscv32": true, "riscv64": true, "wasm32": true, "tricore": true,
}

func (c *Config) Validate() *ValidateResult {
	result := &ValidateResult{}

	// Error -- block execution
	if c.Compendium.Version == "" {
		result.addError("compendium.version is required")
	}

	if c.Compendium.MinCompendium == "" {
		result.addError("compendium.min_compendium is required")
	}

	if c.Registry.Source == "" {
		result.addError("registry.source is required")
	} else if c.Registry.Source != "public" {
		result.addError(`registry.source must be "public" (only the public registry is supported in v0.1)`)
	}

	// Warnings -- informational
	if c.Target.Platform != "" && !knownPlatforms[c.Target.Platform] {
		result.addWarning("unknown platform: " + c.Target.Platform)
	}

	if c.Target.Arch != "" && !knownArchitectures[c.Target.Arch] {
		result.addWarning("unknown architecture: " + c.Target.Arch)
	}

	if c.Target.Platform != "embedded" && (c.Target.FPU != "" || c.Target.LinkerScript != "") {
		result.addWarning("fpu/linker_script are typically only used with platform = \"embedded\"")
	}

	return result
}
