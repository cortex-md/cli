package app

import (
	"fmt"
	"strconv"
	"time"

	"github.com/cortex/cli/internal/project"
	"github.com/cortex/cli/internal/registry"
	"github.com/cortex/cli/internal/report"
	"github.com/cortex/cli/internal/ux"
	"github.com/spf13/cobra"
)

func newStatsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stats [directory]",
		Short: "Show Marketplace votes, downloads and review status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := project.Load(projectDir(args))
			if err != nil {
				return err
			}
			kind := kindLabel(p)
			client := optionalGitHub()

			var stats *registry.Stats
			var blocked []registry.BlockedEntry
			if err := ux.Task("Reading Marketplace data", func() error {
				var fetchErr error
				if stats, fetchErr = registry.FetchStats(cmd.Context(), client); fetchErr != nil {
					return fetchErr
				}
				blocked, fetchErr = registry.FetchBlocked(cmd.Context(), client)
				return fetchErr
			}); err != nil {
				return ux.NewProblem("Could not read Marketplace data", "check your connection and try again", err)
			}

			if stats.GeneratedAt == nil {
				ux.Info("Marketplace stats have not been published yet; they refresh every 30 minutes")
				return nil
			}
			item, listed := stats.Item(kind, p.ID())
			if !listed {
				ux.Info("%s is not listed on the Marketplace yet", p.DisplayName())
				ux.NextSteps(ux.NextStep{Command: "cortex submit", Hint: "list it after the first release"})
				return nil
			}

			r := report.New().Section(p.DisplayName())
			r.Pass("%s votes · %s downloads · %s stars", strconv.Itoa(item.Votes), strconv.Itoa(item.Downloads), strconv.Itoa(item.Stars))
			if item.Latest != "" {
				r.Pass("Latest release %s", item.Latest)
			}
			if item.Audit != nil {
				switch item.Audit.Status {
				case "pass":
					r.Pass("Automated review passed for %s", item.Audit.CheckedTag)
				case "warn":
					r.Warn(fmt.Sprintf("Automated review found warnings in %s", item.Audit.CheckedTag), joinIssues(item.Audit.Issues))
				default:
					r.Fail(fmt.Sprintf("Automated review flagged %s", item.Audit.CheckedTag), joinIssues(item.Audit.Issues))
				}
			}
			if block := registry.FindBlock(blocked, kind, p.ID()); block != nil {
				if block.Versions == nil {
					r.Fail("Removed from the Marketplace: "+block.Reason, "open an issue in cortex-md/registry to appeal")
				} else {
					r.Warn(fmt.Sprintf("Pulled versions %v: %s", block.Versions, block.Reason), "release a fixed version")
				}
			}
			if item.Discussion != nil {
				r.Info("Votes and comments: "+item.Discussion.URL, "")
			}
			r.Print()
			if stats.GeneratedAt != nil {
				ux.Blank()
				fmt.Fprintln(ux.Stdout, ux.Muted("Updated "+stats.GeneratedAt.Local().Format(time.RFC822)))
			}
			return nil
		},
	}
}

func joinIssues(issues []string) string {
	if len(issues) == 0 {
		return ""
	}
	result := issues[0]
	for _, issue := range issues[1:] {
		result += "; " + issue
	}
	return result
}
