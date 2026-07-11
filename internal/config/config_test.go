package config

import (
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv(EnvBaseURL, "https://api.anthropic.com")
	t.Setenv(EnvAPIKey, "test-key")
	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.BaseURL != "https://api.anthropic.com" {
		t.Fatalf("BaseURL = %q, want https://api.anthropic.com", cfg.BaseURL)
	}
	if cfg.APIKey != "test-key" {
		t.Fatalf("APIKey = %q, want test-key", cfg.APIKey)
	}
	if len(cfg.Suites) != len(DefaultSuites) {
		t.Fatalf("len(Suites) = %d, want %d", len(cfg.Suites), len(DefaultSuites))
	}
}

func TestLoadDefaultSuitesExcludeErrorResponses(t *testing.T) {
	t.Setenv(EnvBaseURL, "https://api.anthropic.com")
	t.Setenv(EnvAPIKey, "test-key")

	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for _, name := range cfg.Suites {
		if name == "error_responses" {
			t.Fatal("default suites must not include error_responses")
		}
	}
}

func TestLoadRejectsUnknownSuiteName(t *testing.T) {
	t.Setenv(EnvBaseURL, "https://api.anthropic.com")
	t.Setenv(EnvAPIKey, "test-key")

	_, err := Load([]string{"--suites", "not-a-suite"})
	if err == nil || !strings.Contains(err.Error(), "unknown test suite") {
		t.Fatalf("expected unknown suite error, got %v", err)
	}
}

func TestLoadListSuitesDoesNotRequireAPIKey(t *testing.T) {
	cfg, err := Load([]string{"--list-suites"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.ListSuites {
		t.Fatal("expected ListSuites=true")
	}
}

func TestLoadHelpExitsCleanly(t *testing.T) {
	_, err := Load([]string{"--help"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}