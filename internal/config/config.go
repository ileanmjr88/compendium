package config

// Top-level -- maps to the entire compendium.toml
type Config struct {
	Compendium Compendium        `toml:"compendium"`
	Registry   Registry          `toml:"registry"`
	Languages  Languages         `toml:"languages"`
	Tools      Tools             `toml:"tools"`
	Target     Target            `toml:"target"`
	Packages   Packages          `toml:"packages"`
	Hooks      Hooks             `toml:"hooks"`
	Editor     Editor            `toml:"editor"`
	Scripts    map[string]string `toml:"scripts"`
}

// [compendium]
type Compendium struct {
	Name          string `toml:"name"`
	Version       string `toml:"version"`
	MinCompendium string `toml:"min_compendium"`
}

// [registry]
type Registry struct {
	Source string `toml:"source"`
}

// [languages]
type Languages map[string]string

// [tools]
type Tools map[string]string

// [target]
type Target struct {
	Platform     string `toml:"platform"`
	Arch         string `toml:"arch"`
	FPU          string `toml:"fpu"`
	LinkerScript string `toml:"linker_script"`
	OS           string `toml:"os"`  // for cross-compile
	ABI          string `toml:"abi"` // e.g. gnueabihf
}

// [packages]
type Packages map[string]string

// [hooks] -- git hooks + options
type Hooks struct {
	PreCommit        []string    `toml:"pre-commit"`
	CommitMsg        []string    `toml:"commit-msg"`
	PrepareCommitMsg []string    `toml:"prepare-commit-msg"`
	PrePush          []string    `toml:"pre-push"`
	PostMerge        []string    `toml:"post-merge"`
	PostCheckout     []string    `toml:"post-checkout"`
	PreBuild         []string    `toml:"pre-build"`
	PreFlash         []string    `toml:"pre-flash"`
	Options          HookOptions `toml:"options"`
}

// [hooks.options]
type HookOptions struct {
	Parallel bool `toml:"parallel"`
	Timeout  int  `toml:"timeout"`
}

// [editor]
type Editor struct {
	VSCode       string        `toml:"vscode"`
	EditorConfig string        `toml:"editorconfig"`
	Idea         string        `toml:"idea"`
	Enforce      EditorEnforce `toml:"enforce"`
}

// [editor.enforce]
type EditorEnforce struct {
	Extensions bool `toml:"extensions"`
	Settings   bool `toml:"settings"`
}
