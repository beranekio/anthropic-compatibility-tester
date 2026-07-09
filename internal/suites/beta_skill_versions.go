package suites

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/beranekio/anthropic-compatibility-tester/internal/config"
	"github.com/beranekio/anthropic-compatibility-tester/internal/testutil"
)

// BetaSkillVersions verifies Beta Skills version APIs via client.Beta.Skills.Versions.*.
type BetaSkillVersions struct{}

func (BetaSkillVersions) Name() string { return "beta_skill_versions" }
func (BetaSkillVersions) Description() string {
	return "Beta Skill versions API (POST/GET/LIST /v1/skills/{id}/versions?beta=true)"
}

func (BetaSkillVersions) Run(ctx context.Context, client anthropic.Client, _ *config.Config) error {
	var skillID string
	defer func() {
		if skillID != "" {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = client.Beta.Skills.Delete(cleanupCtx, skillID, anthropic.BetaSkillDeleteParams{})
		}
	}()

	created, err := client.Beta.Skills.New(ctx, anthropic.BetaSkillNewParams{
		DisplayTitle: anthropic.String("Compatibility Test Skill"),
		Files:        []io.Reader{testutil.SmallSkillFileReader()},
	})
	if err != nil {
		return fmt.Errorf("beta skill create failed: %w", err)
	}
	skillID = created.ID

	version, err := client.Beta.Skills.Versions.New(ctx, skillID, anthropic.BetaSkillVersionNewParams{
		Files: []io.Reader{testutil.SkillVersionFileReader()},
	})
	if err != nil {
		return fmt.Errorf("beta skill version create failed: %w", err)
	}
	if version == nil || version.Version == "" {
		return fail("beta_skill_versions", "version response missing version")
	}

	got, err := client.Beta.Skills.Versions.Get(ctx, version.Version, anthropic.BetaSkillVersionGetParams{
		SkillID: skillID,
	})
	if err != nil {
		return fmt.Errorf("beta skill version get failed: %w", err)
	}
	if got == nil || got.Version == "" {
		return fail("beta_skill_versions", "get version response missing version")
	}

	page, err := client.Beta.Skills.Versions.List(ctx, skillID, anthropic.BetaSkillVersionListParams{})
	if err != nil {
		return fmt.Errorf("beta skill version list failed: %w", err)
	}
	if page == nil {
		return fail("beta_skill_versions", "list response is nil")
	}
	return nil
}