package suites

import (
	"context"
	"fmt"
	"sort"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
	"github.com/beranekio/anthropic-compatibility-tester/internal/suitespec"
)

// Suite exercises one area of the Anthropic API through the official Go SDK.
type Suite interface {
	Name() string
	Description() string
	Run(ctx context.Context, client anthropic.Client, cfg *config.Config) error
}

// DeprecatedSuite is implemented by suites backed by deprecated Anthropic APIs.
type DeprecatedSuite interface {
	Suite
	Deprecated() bool
}

// IsDeprecated reports whether a suite is marked deprecated.
func IsDeprecated(suite Suite) bool {
	deprecated, ok := suite.(DeprecatedSuite)
	return ok && deprecated.Deprecated()
}

// All returns every registered compatibility suite.
func All() []Suite {
	return []Suite{
		Models{},
		ModelsGet{},
		Messages{},
		MessagesStream{},
		MessagesTools{},
		MessagesToolsStream{},
		MessagesJSON{},
		MessagesMultiTurn{},
		MessagesCountTokens{},
		MessagesVision{},
		Completions{},
		CompletionsStream{},
		MessageBatchesCreate{},
		MessageBatchesGet{},
		MessageBatchesCancel{},
		MessageBatchesList{},
		BetaFiles{},
		BetaSkills{},
		BetaSkillVersions{},
		ErrorResponses{},
	}
}

// ByName maps suite names to implementations.
func ByName() map[string]Suite {
	registry := make(map[string]Suite, len(All()))
	for _, suite := range All() {
		registry[suite.Name()] = suite
	}
	return registry
}

// ValidateNames reports whether every name is a registered suite.
func ValidateNames(names []string) error {
	if err := suitespec.ValidateNames(names); err != nil {
		return fmt.Errorf("%w (use --list-suites to see options)", err)
	}
	return nil
}

// Names returns sorted suite names for display.
func Names() []string {
	all := All()
	names := make([]string, len(all))
	for i, suite := range all {
		names[i] = suite.Name()
	}
	sort.Strings(names)
	return names
}