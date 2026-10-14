package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/cortex/cli/internal/auth"
	"github.com/cortex/cli/internal/buildinfo"
	gh "github.com/cortex/cli/internal/github"
	"github.com/cortex/cli/internal/rules"
	"github.com/cortex/cli/internal/ux"
	"github.com/spf13/cobra"
)

func newLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Sign in with GitHub to submit to the Marketplace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd.Context())
		},
	}
}

func runLogin(ctx context.Context) error {
	if buildinfo.GitHubClientID == "" {
		return ux.NewProblem(
			"GitHub login is not available in this build",
			"run `gh auth login` (cortex uses the gh token automatically) or set CORTEX_TOKEN",
			nil,
		)
	}

	if token, err := auth.GetToken(); err == nil {
		if user, err := gh.NewClient(token).CurrentUser(ctx); err == nil {
			switchAccount, err := ux.Confirm("You are logged in as @"+user.Login+". Log in with another account?", false)
			if err != nil || !switchAccount {
				return err
			}
		}
	}

	client := auth.NewClient(buildinfo.GitHubClientID)
	device, err := client.StartDeviceFlow(ctx)
	if err != nil {
		return ux.NewProblem("Could not start the GitHub login", "check your connection and try again", err)
	}

	copied := auth.CopyToClipboard(device.UserCode) == nil
	printDeviceCode(device.UserCode, device.VerificationURI, copied)

	if ux.Interactive() {
		fmt.Fprint(ux.Stdout, ux.Muted("Press Enter to open the browser…"))
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		if err := auth.OpenBrowser(device.VerificationURI); err != nil {
			ux.Warning("Open %s in your browser", device.VerificationURI)
		}
	}

	expiry := time.Duration(device.ExpiresIn) * time.Second
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}
	pollContext, cancel := context.WithTimeout(ctx, expiry)
	defer cancel()

	var token string
	if err := ux.Task("Waiting for you to authorize Cortex CLI on GitHub", func() error {
		token, err = client.PollForToken(pollContext, device.DeviceCode, device.Interval)
		return err
	}); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, auth.ErrAuthExpired) {
			return ux.NewProblem("The login code expired", "run `cortex login` again", nil)
		}
		if errors.Is(err, auth.ErrAuthDenied) {
			return ux.NewProblem("Authorization was denied on GitHub", "run `cortex login` again and approve the request", nil)
		}
		return err
	}

	if err := auth.SaveToken(token); err != nil {
		return ux.NewProblem("Could not store the token in the system keychain", "export CORTEX_TOKEN with a GitHub token instead", err)
	}

	user, err := gh.NewClient(token).CurrentUser(ctx)
	if err != nil {
		return err
	}
	ux.Success("Logged in as %s", ux.Command("@"+user.Login))
	return nil
}

func printDeviceCode(code string, url string, copied bool) {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ux.BrandPrimary)).
		Padding(0, 3).
		Render(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ux.BrandAccent)).Render(code))

	ux.Blank()
	fmt.Fprintln(ux.Stdout, "Enter this code at "+ux.Command(url))
	fmt.Fprintln(ux.Stdout, box)
	if copied {
		fmt.Fprintln(ux.Stdout, ux.Muted("Copied to your clipboard."))
	}
}

func newLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored GitHub token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := auth.GetToken(); errors.Is(err, auth.ErrNoToken) {
				ux.Info("No stored login")
			} else {
				if err := auth.DeleteToken(); err != nil {
					return ux.NewProblem("Could not remove the token from the keychain", "", err)
				}
				ux.Success("Logged out")
			}

			for _, name := range []string{"CORTEX_TOKEN", "GITHUB_TOKEN"} {
				if os.Getenv(name) != "" {
					ux.Warning("%s is still set in your environment and will be used", name)
				}
			}
			return nil
		},
	}
}

func newWhoamiCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the GitHub account Cortex CLI uses",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := requireGitHub(cmd.Context())
			if err != nil {
				return err
			}

			ux.Success("Logged in as %s %s", ux.Command("@"+session.User.Login), ux.Muted("(token from "+string(session.Source)+")"))
			scopes := session.User.Scopes
			switch {
			case len(scopes) == 0:
				ux.Info("Fine-grained token: it must be allowed to fork repositories and open pull requests")
			case hasScope(scopes, "public_repo") || hasScope(scopes, "repo"):
				ux.Info("Scopes: %s", strings.Join(scopes, ", "))
			default:
				ux.Warning("Token scopes (%s) cannot open registry pull requests", strings.Join(scopes, ", "))
				ux.Info("Run %s to get a token with %s", ux.Command("cortex login"), auth.Scopes)
			}
			return nil
		},
	}
}

func hasScope(scopes []string, scope string) bool {
	for _, candidate := range scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

func isHTTPS(value string) bool {
	return rules.IsHTTPSURL(value)
}

func indent(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for index, line := range lines {
		lines[index] = "  " + line
	}
	return strings.Join(lines, "\n") + "\n"
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
