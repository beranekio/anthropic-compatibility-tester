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
			// Real Anthropic rejects skill delete while versions remain.
			cleanupBetaSkill(cleanupCtx, client, skillID)
		}
	}()

	created, err := client.Beta.Skills.New(ctx, anthropic.BetaSkillNewParams{
		DisplayName: anthropic.String(uniqueSkillDisplayName()),
		Files:       []io.Reader{testutil.SmallSkillFileReader()},
	})
	if err != nil {
		return fmt.Errorf("beta skill create failed: %w", err)
	}
	if err := validateBetaSkillResponse("beta_skill_versions", created); err != nil {
		return err
	}
	skillID = created.ID

	version, err := client.Beta.Skills.Versions.New(ctx, skillID, anthropic.BetaSkillVersionNewParams{
		Files: []io.Reader{testutil.SkillVersionFileReader()},
	})
	if err != nil {
		return fmt.Errorf("beta skill version create failed: %w", err)
	}
	if err := validateBetaSkillVersionCreate("beta_skill_versions", version, skillID); err != nil {
		return err
	}

	got, err := client.Beta.Skills.Versions.Get(ctx, version.ID, anthropic.BetaSkillVersionGetParams{
		SkillID: skillID,
	})
	if err != nil {
		return fmt.Errorf("beta skill version get failed: %w", err)
	}
	if err := validateBetaSkillVersionGet("beta_skill_versions", got, skillID, version.ID); err != nil {
		return err
	}

	page, err := client.Beta.Skills.Versions.List(ctx, skillID, anthropic.BetaSkillVersionListParams{})
	if err != nil {
		return fmt.Errorf("beta skill version list failed: %w", err)
	}
	if page == nil {
		return fail("beta_skill_versions", "list response is nil")
	}
	found := false
	for i := range page.Data {
		item := &page.Data[i]
		if item.ID == version.ID {
			if err := validateBetaSkillVersionListItem("beta_skill_versions", item, skillID, version.ID); err != nil {
				return err
			}
			found = true
			break
		}
	}
	if !found {
		return fail("beta_skill_versions", "created version missing from list response")
	}
	return nil
}

func validateBetaSkillVersionEnvelope(suite, id, typ, skillID string) error {
	if id == "" {
		return fail(suite, "skill version missing id")
	}
	if typ != "skill_version" {
		return fail(suite, fmt.Sprintf("skill version type is %q, want skill_version", typ))
	}
	if skillID == "" {
		return fail(suite, "skill version missing skill_id")
	}
	return nil
}

func validateBetaSkillVersionCreate(suite string, version *anthropic.BetaSkillVersion, wantSkillID string) error {
	if version == nil {
		return fail(suite, "version response is nil")
	}
	if err := validateBetaSkillVersionEnvelope(suite, version.ID, string(version.Type), version.SkillID); err != nil {
		return err
	}
	if version.SkillID != wantSkillID {
		return fail(suite, fmt.Sprintf("version skill_id is %q, want %q", version.SkillID, wantSkillID))
	}
	return nil
}

func validateBetaSkillVersionGet(suite string, version *anthropic.BetaSkillVersion, wantSkillID, wantID string) error {
	if version == nil {
		return fail(suite, "get version response is nil")
	}
	if err := validateBetaSkillVersionEnvelope(suite, version.ID, string(version.Type), version.SkillID); err != nil {
		return err
	}
	if version.SkillID != wantSkillID {
		return fail(suite, fmt.Sprintf("get skill_id is %q, want %q", version.SkillID, wantSkillID))
	}
	if version.ID != wantID {
		return fail(suite, fmt.Sprintf("get version id is %q, want %q", version.ID, wantID))
	}
	return nil
}

func validateBetaSkillVersionListItem(suite string, version *anthropic.BetaSkillVersion, wantSkillID, wantID string) error {
	if version == nil {
		return fail(suite, "list version item is nil")
	}
	if err := validateBetaSkillVersionEnvelope(suite, version.ID, string(version.Type), version.SkillID); err != nil {
		return err
	}
	if version.SkillID != wantSkillID {
		return fail(suite, fmt.Sprintf("list skill_id is %q, want %q", version.SkillID, wantSkillID))
	}
	if version.ID != wantID {
		return fail(suite, fmt.Sprintf("list version id is %q, want %q", version.ID, wantID))
	}
	return nil
}
