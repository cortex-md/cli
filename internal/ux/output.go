package ux

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

const (
	BrandPrimary = "#FB7185"
	BrandAccent  = "#F43F5E"
	BrandInk     = "#303342"
	BrandMuted   = "#8B8F9D"
	BrandSuccess = "#4A9B7A"
	BrandWarning = "#B89B3A"
	BrandError   = "#D45E6A"
	BrandSurface = "#FBFBFC"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandSuccess)).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandError)).Bold(true)
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandWarning)).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandAccent))
	stepStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandPrimary)).Bold(true)
	debugStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandMuted))
	commandStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandAccent)).Bold(true)
	pathStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandMuted))
)

func Success(format string, args ...interface{}) {
	printStatus(os.Stdout, successStyle, "ok", format, args...)
}

func Error(format string, args ...interface{}) {
	printStatus(os.Stderr, errorStyle, "x", format, args...)
}

func Warning(format string, args ...interface{}) {
	printStatus(os.Stdout, warningStyle, "!", format, args...)
}

func Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "%s %s\n", infoStyle.Render("->"), msg)
}

func Step(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "%s %s\n", stepStyle.Render(">>"), msg)
}

func Debug(format string, args ...interface{}) {
	printStatus(os.Stdout, debugStyle, "debug", format, args...)
}

func Command(value string) string {
	return commandStyle.Render(value)
}

func Path(value string) string {
	return pathStyle.Render(value)
}

func printStatus(file *os.File, style lipgloss.Style, label string, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(file, "%s %s\n", style.Render("["+label+"]"), msg)
}
