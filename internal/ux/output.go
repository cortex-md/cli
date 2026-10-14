package ux

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const (
	BrandPrimary = "#FF3B38"
	BrandAccent  = "#E8A83C"
	BrandMuted   = "#8B8F9D"
	BrandSuccess = "#4A9B7A"
	BrandWarning = "#D9A33A"
	BrandError   = "#E5484D"
)

var (
	Stdout io.Writer = os.Stdout
	Stderr io.Writer = os.Stderr

	AssumeYes bool
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandSuccess)).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandError)).Bold(true)
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandWarning)).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandAccent))
	stepStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandPrimary)).Bold(true)
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandMuted))
	commandStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandAccent)).Bold(true)
	headingStyle = lipgloss.NewStyle().Bold(true)
)

const (
	SymbolSuccess = "✓"
	SymbolError   = "✗"
	SymbolWarning = "!"
	SymbolInfo    = "•"
	SymbolStep    = "›"
)

func Interactive() bool {
	if os.Getenv("CI") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func Success(format string, args ...interface{}) {
	fmt.Fprintf(Stdout, "%s %s\n", successStyle.Render(SymbolSuccess), fmt.Sprintf(format, args...))
}

func Error(format string, args ...interface{}) {
	fmt.Fprintf(Stderr, "%s %s\n", errorStyle.Render(SymbolError), fmt.Sprintf(format, args...))
}

func Warning(format string, args ...interface{}) {
	fmt.Fprintf(Stdout, "%s %s\n", warningStyle.Render(SymbolWarning), fmt.Sprintf(format, args...))
}

func Info(format string, args ...interface{}) {
	fmt.Fprintf(Stdout, "%s %s\n", infoStyle.Render(SymbolInfo), fmt.Sprintf(format, args...))
}

func Step(format string, args ...interface{}) {
	fmt.Fprintf(Stdout, "%s %s\n", stepStyle.Render(SymbolStep), fmt.Sprintf(format, args...))
}

func Muted(value string) string {
	return mutedStyle.Render(value)
}

func Command(value string) string {
	return commandStyle.Render(value)
}

func Path(value string) string {
	return mutedStyle.Render(value)
}

func Heading(value string) {
	fmt.Fprintln(Stdout)
	fmt.Fprintln(Stdout, headingStyle.Render(value))
}

func Blank() {
	fmt.Fprintln(Stdout)
}

type NextStep struct {
	Command string
	Hint    string
}

func NextSteps(steps ...NextStep) {
	if len(steps) == 0 {
		return
	}
	width := 0
	for _, step := range steps {
		if len(step.Command) > width {
			width = len(step.Command)
		}
	}

	Heading("Next")
	for _, step := range steps {
		line := "  " + Command(step.Command)
		if step.Hint != "" {
			line += strings.Repeat(" ", width-len(step.Command)) + "  " + Muted(step.Hint)
		}
		fmt.Fprintln(Stdout, line)
	}
}

type Problem struct {
	Title string
	Fix   string
	Err   error
}

func (p *Problem) Error() string {
	if p.Err != nil {
		return fmt.Sprintf("%s: %v", p.Title, p.Err)
	}
	return p.Title
}

func (p *Problem) Unwrap() error {
	return p.Err
}

func NewProblem(title string, fix string, err error) *Problem {
	return &Problem{Title: title, Fix: fix, Err: err}
}

var ErrReported = errors.New("already reported")

func PrintError(err error) {
	if err == nil || errors.Is(err, ErrReported) {
		return
	}

	var problem *Problem
	if errors.As(err, &problem) {
		Error("%s", problem.Title)
		if problem.Err != nil {
			fmt.Fprintf(Stderr, "  %s\n", Muted(problem.Err.Error()))
		}
		if problem.Fix != "" {
			fmt.Fprintf(Stderr, "  %s %s\n", infoStyle.Render("fix:"), problem.Fix)
		}
		return
	}

	Error("%v", err)
}
