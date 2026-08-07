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

// BetaSkillVersionDownload verifies downloading skill version content.
type BetaSkillVersionDownload struct{}

func (BetaSkillVersionDownload) Name() string { return "beta_skill_version_download" }
func (BetaSkillVersionDownload) Description() string {
	return "Beta Skill version download (GET /v1/skills/{id}/versions/{version}/content?beta=true)"
}

func (BetaSkillVersionDownload) Run(ctx context.Context, client anthropic.Client, _ *config.Config) error {
	var skillID string
	defer func() {
		if skillID != "" {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cleanupBetaSkill(cleanupCtx, client, skillID)
		}
	}()

	created, err := client.Beta.Skills.New(ctx, anthropic.BetaSkillNewParams{
		DisplayTitle: anthropic.String(uniqueSkillDisplayTitle()),
		Files:        []io.Reader{testutil.SmallSkillFileReader()},
	})
	if err != nil {
		return fmt.Errorf("beta skill create failed: %w", err)
	}
	if err := validateBetaSkillResponse("beta_skill_version_download", created); err != nil {
		return err
	}
	skillID = created.ID

	version, err := client.Beta.Skills.Versions.New(ctx, skillID, anthropic.BetaSkillVersionNewParams{
		Files: []io.Reader{testutil.SkillVersionFileReader()},
	})
	if err != nil {
		return fmt.Errorf("beta skill version create failed: %w", err)
	}
	if err := validateBetaSkillVersionCreate("beta_skill_version_download", version, skillID); err != nil {
		return err
	}

	resp, err := client.Beta.Skills.Versions.Download(ctx, version.Version, anthropic.BetaSkillVersionDownloadParams{
		SkillID: skillID,
	})
	if err != nil {
		return fmt.Errorf("beta skill version download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fail("beta_skill_version_download", fmt.Sprintf("download status is %d, want 2xx", resp.StatusCode))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("beta skill version download read failed: %w", err)
	}
	if len(body) == 0 {
		return fail("beta_skill_version_download", "download body is empty")
	}
	return nil
}
