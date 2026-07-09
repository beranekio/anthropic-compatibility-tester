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

// BetaSkills verifies the Beta Skills API lifecycle via client.Beta.Skills.*.
type BetaSkills struct{}

func (BetaSkills) Name() string { return "beta_skills" }
func (BetaSkills) Description() string {
	return "Beta Skills API lifecycle (POST/GET/LIST/DELETE /v1/skills?beta=true)"
}

func (BetaSkills) Run(ctx context.Context, client anthropic.Client, _ *config.Config) error {
	deleted := false
	var skillID string
	defer func() {
		if skillID != "" && !deleted {
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
	if err := validateBetaSkillResponse("beta_skills", created); err != nil {
		return err
	}
	skillID = created.ID

	got, err := client.Beta.Skills.Get(ctx, skillID, anthropic.BetaSkillGetParams{})
	if err != nil {
		return fmt.Errorf("beta skill get failed: %w", err)
	}
	if err := validateBetaSkillGetResponse("beta_skills", got); err != nil {
		return err
	}
	if got.ID != skillID {
		return fail("beta_skills", fmt.Sprintf("get id is %q, want %q", got.ID, skillID))
	}

	page, err := client.Beta.Skills.List(ctx, anthropic.BetaSkillListParams{})
	if err != nil {
		return fmt.Errorf("beta skill list failed: %w", err)
	}
	if page == nil {
		return fail("beta_skills", "list response is nil")
	}

	if _, err := client.Beta.Skills.Delete(ctx, skillID, anthropic.BetaSkillDeleteParams{}); err != nil {
		return fmt.Errorf("beta skill delete failed: %w", err)
	}
	deleted = true
	return nil
}

func validateBetaSkillResponse(suite string, skill *anthropic.BetaSkillNewResponse) error {
	if skill == nil {
		return fail(suite, "skill response is nil")
	}
	if skill.ID == "" {
		return fail(suite, "skill missing id")
	}
	if skill.LatestVersion == "" {
		return fail(suite, "skill missing latest_version")
	}
	if skill.Source == "" {
		return fail(suite, "skill missing source")
	}
	return nil
}

func validateBetaSkillGetResponse(suite string, skill *anthropic.BetaSkillGetResponse) error {
	if skill == nil {
		return fail(suite, "skill response is nil")
	}
	if skill.ID == "" {
		return fail(suite, "skill missing id")
	}
	if skill.LatestVersion == "" {
		return fail(suite, "skill missing latest_version")
	}
	return nil
}