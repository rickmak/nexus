package vending

func GetEnvVars(providers []string) map[string]string {
	env := make(map[string]string)

	env["NEXUS_VENDING_URL"] = "http://localhost:10790"

	for _, provider := range providers {
		switch provider {
		case "codex":
			env["CODEX_API_URL"] = "http://localhost:10790/proxy/codex"
		case "opencode":
			env["OPENCODE_API_URL"] = "http://localhost:10790/proxy/opencode"
		case "claude":
			env["CLAUDE_API_URL"] = "http://localhost:10790/proxy/claude"
		case "openai":
			env["OPENAI_BASE_URL"] = "http://localhost:10790/proxy/openai"
		}
	}

	return env
}
