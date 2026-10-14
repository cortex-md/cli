package ux

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func Task(label string, run func() error) error {
	started := time.Now()

	if !Interactive() {
		Step("%s", label)
		err := run()
		finishTask(label, started, err)
		return err
	}

	done := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for frame := 0; ; frame++ {
			fmt.Fprintf(Stdout, "\r%s %s", stepStyle.Render(spinnerFrames[frame%len(spinnerFrames)]), label)
			select {
			case <-done:
				fmt.Fprint(Stdout, "\r\033[2K")
				return
			case <-ticker.C:
			}
		}
	}()

	err := run()
	close(done)
	wait.Wait()
	finishTask(label, started, err)
	return err
}

func finishTask(label string, started time.Time, err error) {
	elapsed := Muted(formatDuration(time.Since(started)))
	if err != nil {
		fmt.Fprintf(Stdout, "%s %s %s\n", errorStyle.Render(SymbolError), label, elapsed)
		return
	}
	fmt.Fprintf(Stdout, "%s %s %s\n", successStyle.Render(SymbolSuccess), label, elapsed)
}

func formatDuration(duration time.Duration) string {
	if duration < time.Second {
		return fmt.Sprintf("%dms", duration.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", duration.Seconds())
}

var ErrNonInteractive = errors.New("interactive input required")

func RequireInteractive(missing string) error {
	if Interactive() {
		return nil
	}
	return NewProblem("Missing "+missing, "pass it as a flag; prompts are disabled without a terminal or when CI is set", ErrNonInteractive)
}

func Confirm(question string, defaultValue bool) (bool, error) {
	if AssumeYes {
		return true, nil
	}
	if !Interactive() {
		return false, NewProblem("Confirmation required: "+question, "re-run with --yes to confirm in non-interactive mode", ErrNonInteractive)
	}

	answer := defaultValue
	err := huh.NewConfirm().Title(question).Affirmative("Yes").Negative("No").Value(&answer).WithTheme(FormTheme()).Run()
	return answer, err
}

func RunForm(groups ...*huh.Group) error {
	return huh.NewForm(groups...).WithTheme(FormTheme()).Run()
}

func FormTheme() *huh.Theme {
	theme := huh.ThemeBase()
	primary := lipgloss.Color(BrandPrimary)
	accent := lipgloss.Color(BrandAccent)
	muted := lipgloss.Color(BrandMuted)
	failure := lipgloss.Color(BrandError)

	theme.Focused.Base = theme.Focused.Base.BorderForeground(primary)
	theme.Focused.Title = theme.Focused.Title.Foreground(primary).Bold(true)
	theme.Focused.Description = theme.Focused.Description.Foreground(muted)
	theme.Focused.SelectSelector = theme.Focused.SelectSelector.Foreground(accent)
	theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(accent)
	theme.Focused.MultiSelectSelector = theme.Focused.MultiSelectSelector.Foreground(accent)
	theme.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(accent).SetString("[✓] ")
	theme.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(muted).SetString("[ ] ")
	theme.Focused.FocusedButton = theme.Focused.FocusedButton.Background(primary).Foreground(lipgloss.Color("#FFFFFF"))
	theme.Focused.TextInput.Cursor = theme.Focused.TextInput.Cursor.Foreground(accent)
	theme.Focused.TextInput.Prompt = theme.Focused.TextInput.Prompt.Foreground(accent)
	theme.Focused.ErrorIndicator = theme.Focused.ErrorIndicator.Foreground(failure)
	theme.Focused.ErrorMessage = theme.Focused.ErrorMessage.Foreground(failure)
	theme.Blurred = theme.Focused
	theme.Blurred.Base = theme.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	return theme
}
