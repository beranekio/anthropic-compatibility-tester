package suitespec

func init() {
	for _, name := range allNames {
		register(name)
	}
}

// allNames lists every registered suite. Keep in sync with suites.All() and config.FullSuites.
var allNames = []string{
	"models",
	"models_get",
	"messages",
	"messages_stream",
	"messages_tools",
	"messages_tools_stream",
	"messages_json",
	"messages_multi_turn",
	"messages_count_tokens",
	"messages_vision",
	"completions",
	"completions_stream",
	"message_batches_create",
	"message_batches_get",
	"message_batches_cancel",
	"message_batches_list",
	"beta_files",
	"beta_skills",
	"beta_skill_versions",
	"error_responses",
}