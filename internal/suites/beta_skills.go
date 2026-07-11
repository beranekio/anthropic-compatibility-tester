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
		DisplayTitle: anthropic.String(uniqueSkillDisplayTitle()),
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
	found := false
	for _, item := range page.Data {
		if item.ID == skillID {
			found = true
			break
		}
	}
	if !found {
		return fail("beta_skills", "created skill missing from list response")
	}

	deletedResp, err := client.Beta.Skills.Delete(ctx, skillID, anthropic.BetaSkillDeleteParams{})
	if err != nil {
		return fmt.Errorf("beta skill delete failed: %w", err)
	}
	if err := validateBetaSkillDeleteResponse("beta_skills", deletedResp, skillID); err != nil {
		return err
	}
	deleted = true
	return nil
}

func uniqueSkillDisplayTitle() string {
	return fmt.Sprintf("Compatibility Test Skill %d", time.Now().UnixNano())
}

func validateBetaSkillDeleteResponse(suite string, deleted *anthropic.BetaSkillDeleteResponse, wantID string) error {
	if deleted == nil {
		return fail(suite, "delete response is nil")
	}
	if deleted.ID == "" {
		return fail(suite, "delete response missing id")
	}
	if deleted.ID != wantID {
		return fail(suite, fmt.Sprintf("delete id is %q, want %q", deleted.ID, wantID))
	}
	if deleted.Type != "skill_deleted" {
		return fail(suite, fmt.Sprintf("delete type is %q, want skill_deleted", deleted.Type))
	}
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