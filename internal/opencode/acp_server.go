package opencode

func BuildACPArgs(repoRoot string) []string {
	return BuildACPArgsWithModel(repoRoot, "")
}

func BuildACPArgsWithModel(repoRoot string, model string) []string {
	args := []string{"opencode", "acp", "--print-logs", "--log-level", "DEBUG", "--cwd", repoRoot}
	return args
}
