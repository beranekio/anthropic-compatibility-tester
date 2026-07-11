package config

import (
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/beranekio/anthropic-compatibility-tester/internal/suitespec"
)

const (
	EnvBaseURL           = "ANTHROPIC_BASE_URL"
	EnvAPIKey            = "ANTHROPIC_API_KEY"
	EnvModel             = "ANTHROPIC_MODEL"
	EnvCompletionModel   = "ANTHROPIC_COMPLETION_MODEL"
	EnvVisionModel       = "ANTHROPIC_VISION_MODEL"
	EnvTestSuites        = "TEST_SUITES"
	EnvRequestTimeout    = "REQUEST_TIMEOUT"
	EnvAllowInsecureHTTP = "ALLOW_INSECURE_HTTP"

	// DefaultCompletionModel is used when the completions suite is selected without
	// an explicit completion model. Legacy /v1/complete expects text completion models.
	DefaultCompletionModel = "claude-2.1"
)

// DefaultSuites are run when TEST_SUITES is unset or set to "all" or "default".
var DefaultSuites = []string{
	"models",
	"models_get",
	"messages",
	"messages_stream",
}

// ExtendedSuites adds commonly optional inference suites to the default set.
var ExtendedSuites = []string{
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
	"error_responses",
}

// FullSuites lists every registered suite name. Keep in sync with suites.All().
var FullSuites = []string{
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

// Config holds runtime settings for compatibility testing.
type Config struct {
	BaseURL           string
	APIKey            string
	Model             string
	CompletionModel   string
	VisionModel       string
	Suites            []string
	RequestTimeout    time.Duration
	AllowInsecureHTTP bool
	ListSuites        bool
}

// Load parses configuration from environment variables and command-line flags.
func Load(args []string) (*Config, error) {
	fs := flag.NewFlagSet("anthropic-compatibility-tester", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	baseURL := fs.String("base-url", envOrDefault(EnvBaseURL, ""), "Anthropic-compatible API base URL")
	apiKey := fs.String("api-key", "", "API key for the endpoint (or set "+EnvAPIKey+")")
	model := fs.String("model", envOrDefault(EnvModel, "claude-sonnet-4-6"), "Model for messages and models_get suites")
	completionModel := fs.String("completion-model", envOrDefault(EnvCompletionModel, ""), "Model for legacy completions suite (defaults to "+DefaultCompletionModel+" when completions is selected)")
	visionModel := fs.String("vision-model", envOrDefault(EnvVisionModel, ""), "Model for vision message suites (defaults to --model)")
	allowInsecureHTTP := fs.Bool("allow-insecure-http", envBoolOrDefault(EnvAllowInsecureHTTP, false), "Allow plaintext HTTP to non-loopback hosts")
	suiteList := fs.String("suites", envOrDefault(EnvTestSuites, "all"), "Comma-separated suite names, or preset: all, default, extended, full")
	timeout := fs.Duration("timeout", 2*time.Minute, "Per-suite timeout")
	listSuites := fs.Bool("list-suites", false, "List available test suites and exit")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage of %s:\n", fs.Name())
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if len(fs.Args()) > 0 {
		return nil, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	cfg := &Config{
		BaseURL:           strings.TrimRight(strings.TrimSpace(*baseURL), "/"),
		APIKey:            strings.TrimSpace(*apiKey),
		Model:             strings.TrimSpace(*model),
		CompletionModel:   strings.TrimSpace(*completionModel),
		VisionModel:       strings.TrimSpace(*visionModel),
		RequestTimeout:    *timeout,
		AllowInsecureHTTP: *allowInsecureHTTP,
		ListSuites:        *listSuites,
	}

	if cfg.ListSuites {
		return cfg, nil
	}

	if explicit, empty := apiKeyFlagExplicit(args); explicit && empty {
		return nil, fmt.Errorf("%s or --api-key is required", EnvAPIKey)
	}
	if cfg.APIKey == "" {
		cfg.APIKey = envOrDefault(EnvAPIKey, "")
	}

	suites, err := resolveSuiteSelection(*suiteList)
	if err != nil {
		return nil, err
	}
	cfg.Suites = suites

	if explicit, empty := completionModelFlagExplicit(args); explicit && empty && suiteNeedsCompletion(cfg.Suites) {
		return nil, fmt.Errorf("%s or --completion-model is required for selected suites", EnvCompletionModel)
	}
	if cfg.CompletionModel == "" {
		if suiteNeedsCompletion(cfg.Suites) {
			cfg.CompletionModel = DefaultCompletionModel
		} else {
			cfg.CompletionModel = cfg.Model
		}
	}
	if cfg.VisionModel == "" {
		cfg.VisionModel = cfg.Model
	}

	if !timeoutFlagExplicit(args) {
		envTimeout, err := envDurationOrDefault(EnvRequestTimeout, cfg.RequestTimeout)
		if err != nil {
			return nil, err
		}
		cfg.RequestTimeout = envTimeout
	}

	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("%s or --base-url is required", EnvBaseURL)
	}
	if err := validateBaseURL(cfg.BaseURL, cfg.AllowInsecureHTTP); err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("%s or --api-key is required", EnvAPIKey)
	}
	if len(cfg.Suites) == 0 {
		return nil, fmt.Errorf("at least one test suite must be selected")
	}
	if err := suitespec.ValidateNames(cfg.Suites); err != nil {
		return nil, fmt.Errorf("%w (use --list-suites to see options)", err)
	}
	if cfg.RequestTimeout <= 0 {
		return nil, fmt.Errorf("request timeout must be greater than zero")
	}
	if err := validateModelsForSuites(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func resolveSuiteSelection(raw string) ([]string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "all", "default":
		return append([]string(nil), DefaultSuites...), nil
	case "extended":
		return append([]string(nil), ExtendedSuites...), nil
	case "full":
		return append([]string(nil), FullSuites...), nil
	}

	seen := make(map[string]struct{})
	var suites []string
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("duplicate test suite %q", name)
		}
		seen[name] = struct{}{}
		suites = append(suites, name)
	}
	return suites, nil
}

func suiteNeedsCompletion(names []string) bool {
	for _, name := range names {
		if name == "completions" || name == "completions_stream" {
			return true
		}
	}
	return false
}

func validateModelsForSuites(cfg *Config) error {
	var needsChat, needsCompletion, needsVision bool
	for _, name := range cfg.Suites {
		switch name {
		case "messages", "messages_stream", "messages_tools", "messages_tools_stream", "messages_json", "messages_multi_turn", "messages_count_tokens", "models_get", "message_batches_create", "message_batches_get", "message_batches_cancel", "message_batches_list":
			needsChat = true
		case "completions", "completions_stream":
			needsCompletion = true
		case "messages_vision":
			needsVision = true
		}
	}
	if needsChat && cfg.Model == "" {
		return fmt.Errorf("%s or --model is required for selected suites", EnvModel)
	}
	if needsCompletion && cfg.CompletionModel == "" {
		return fmt.Errorf("%s or --completion-model is required for selected suites", EnvCompletionModel)
	}
	if needsVision && cfg.VisionModel == "" {
		return fmt.Errorf("%s or --vision-model is required for selected suites", EnvVisionModel)
	}
	return nil
}

func validateBaseURL(raw string, allowInsecureHTTP bool) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s: invalid URL: %w", EnvBaseURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s: URL must use http or https scheme", EnvBaseURL)
	}
	if u.Host == "" {
		return fmt.Errorf("%s: URL must include a host", EnvBaseURL)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("%s: URL must include a hostname", EnvBaseURL)
	}
	if port := u.Port(); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil || p < 1 || p > 65535 {
			return fmt.Errorf("%s: URL port must be between 1 and 65535", EnvBaseURL)
		}
	}
	if u.RawQuery != "" {
		return fmt.Errorf("%s: query parameters in the base URL are not supported by the Anthropic Go SDK", EnvBaseURL)
	}
	if strings.Contains(strings.ToLower(raw), "%2f") {
		return fmt.Errorf("%s: encoded path separators (%%2F) are not supported by the Anthropic Go SDK", EnvBaseURL)
	}
	if u.Scheme == "http" && !allowInsecureHTTP && !isLoopbackHost(u.Hostname()) {
		return fmt.Errorf("%s: plaintext HTTP is only permitted for loopback hosts unless --allow-insecure-http is set", EnvBaseURL)
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func apiKeyFlagExplicit(args []string) (explicit bool, valueEmpty bool) {
	for i, arg := range args {
		switch {
		case arg == "--api-key", arg == "-api-key":
			explicit = true
			if i+1 >= len(args) {
				valueEmpty = true
			} else {
				valueEmpty = strings.TrimSpace(args[i+1]) == ""
			}
		case strings.HasPrefix(arg, "--api-key="):
			explicit = true
			valueEmpty = strings.TrimSpace(strings.TrimPrefix(arg, "--api-key=")) == ""
		case strings.HasPrefix(arg, "-api-key="):
			explicit = true
			valueEmpty = strings.TrimSpace(strings.TrimPrefix(arg, "-api-key=")) == ""
		}
	}
	return explicit, valueEmpty
}

func completionModelFlagExplicit(args []string) (explicit bool, valueEmpty bool) {
	for i, arg := range args {
		switch {
		case arg == "--completion-model", arg == "-completion-model":
			explicit = true
			if i+1 >= len(args) {
				valueEmpty = true
			} else {
				valueEmpty = strings.TrimSpace(args[i+1]) == ""
			}
		case strings.HasPrefix(arg, "--completion-model="):
			explicit = true
			valueEmpty = strings.TrimSpace(strings.TrimPrefix(arg, "--completion-model=")) == ""
		case strings.HasPrefix(arg, "-completion-model="):
			explicit = true
			valueEmpty = strings.TrimSpace(strings.TrimPrefix(arg, "-completion-model=")) == ""
		}
	}
	return explicit, valueEmpty
}

func timeoutFlagExplicit(args []string) bool {
	for _, arg := range args {
		switch {
		case arg == "--timeout", arg == "-timeout":
			return true
		case strings.HasPrefix(arg, "--timeout="), strings.HasPrefix(arg, "-timeout="):
			return true
		}
	}
	return false
}

func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func envBoolOrDefault(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envDurationOrDefault(key string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s: invalid duration %q: %w", key, value, err)
	}
	return d, nil
}