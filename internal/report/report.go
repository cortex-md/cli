package report

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/cortex/cli/internal/ux"
)

type Level int

const (
	LevelPass Level = iota
	LevelInfo
	LevelWarn
	LevelFail
)

type Check struct {
	Level   Level
	Message string
	Fix     string
}

type Section struct {
	Title  string
	Checks []Check
}

type Report struct {
	Sections []*Section
	current  *Section
}

func New() *Report {
	return &Report{}
}

func (r *Report) Section(title string) *Report {
	section := &Section{Title: title}
	r.Sections = append(r.Sections, section)
	r.current = section
	return r
}

func (r *Report) Pass(format string, args ...interface{}) {
	r.add(LevelPass, fmt.Sprintf(format, args...), "")
}

func (r *Report) Info(message string, fix string) {
	r.add(LevelInfo, message, fix)
}

func (r *Report) Warn(message string, fix string) {
	r.add(LevelWarn, message, fix)
}

func (r *Report) Fail(message string, fix string) {
	r.add(LevelFail, message, fix)
}

func (r *Report) add(level Level, message string, fix string) {
	if r.current == nil {
		r.Section("")
	}
	r.current.Checks = append(r.current.Checks, Check{Level: level, Message: message, Fix: fix})
}

func (r *Report) Count(level Level) int {
	count := 0
	for _, section := range r.Sections {
		for _, check := range section.Checks {
			if check.Level == level {
				count++
			}
		}
	}
	return count
}

func (r *Report) Passed(strict bool) bool {
	if r.Count(LevelFail) > 0 {
		return false
	}
	return !strict || r.Count(LevelWarn) == 0
}

var (
	passStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ux.BrandSuccess)).Bold(true)
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ux.BrandWarning)).Bold(true)
	failStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ux.BrandError)).Bold(true)
	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ux.BrandAccent))
)

func (r *Report) Print() {
	for _, section := range r.Sections {
		if len(section.Checks) == 0 {
			continue
		}
		if section.Title != "" {
			ux.Heading(section.Title)
		}
		for _, check := range section.Checks {
			fmt.Fprintf(ux.Stdout, "  %s %s\n", symbol(check.Level), check.Message)
			if check.Fix != "" {
				fmt.Fprintf(ux.Stdout, "    %s %s\n", infoStyle.Render("fix:"), ux.Muted(check.Fix))
			}
		}
	}
}

func (r *Report) PrintSummary(strict bool) {
	failures := r.Count(LevelFail)
	warnings := r.Count(LevelWarn)
	ux.Blank()
	switch {
	case failures > 0:
		ux.Error("%d %s, %d %s", failures, plural(failures, "error", "errors"), warnings, plural(warnings, "warning", "warnings"))
	case strict && warnings > 0:
		ux.Error("%d %s (strict mode treats warnings as errors)", warnings, plural(warnings, "warning", "warnings"))
	case warnings > 0:
		ux.Warning("Ready, with %d %s", warnings, plural(warnings, "warning", "warnings"))
	default:
		ux.Success("All checks passed")
	}
}

func symbol(level Level) string {
	switch level {
	case LevelFail:
		return failStyle.Render(ux.SymbolError)
	case LevelWarn:
		return warnStyle.Render(ux.SymbolWarning)
	case LevelInfo:
		return infoStyle.Render(ux.SymbolInfo)
	default:
		return passStyle.Render(ux.SymbolSuccess)
	}
}

func plural(count int, singular string, many string) string {
	if count == 1 {
		return singular
	}
	return many
}
