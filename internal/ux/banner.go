package ux

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA")).
			Italic(true)

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	bannerBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(0, 2)
)

const logo = `
  ██████╗ ██████╗ ██████╗ ████████╗███████╗██╗  ██╗
 ██╔════╝██╔═══██╗██╔══██╗╚══██╔══╝██╔════╝╚██╗██╔╝
 ██║     ██║   ██║██████╔╝   ██║   █████╗   ╚███╔╝ 
 ██║     ██║   ██║██╔══██╗   ██║   ██╔══╝   ██╔██╗ 
 ╚██████╗╚██████╔╝██║  ██║   ██║   ███████╗██╔╝ ██╗
  ╚═════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝`

const logoSmall = `
 ▄████▄  ▒█████   ██▀███  ▄▄▄█████▓▓█████ ▒██   ██▒
▒██▀ ▀█ ▒██▒  ██▒▓██ ▒ ██▒▓  ██▒ ▓▒▓█   ▀ ▒▒ █ █ ▒░
▒▓█    ▄▒██░  ██▒▓██ ░▄█ ▒▒ ▓██░ ▒░▒███   ░░  █   ░
▒▓▓▄ ▄██▒██   ██░▒██▀▀█▄  ░ ▓██▓ ░ ▒▓█  ▄  ░ █ █ ▒ 
▒ ▓███▀ ░ ████▓▒░░██▓ ▒██▒  ▒██▒ ░ ░▒████▒▒██▒ ▒██▒
░ ░▒ ▒  ░ ▒░▒░▒░ ░ ▒▓ ░▒▓░  ▒ ░░   ░░ ▒░ ░▒▒ ░ ░▓ ░
  ░  ▒    ░ ▒ ▒░   ░▒ ░ ▒░    ░     ░ ░  ░░░   ░▒ ░
░       ░ ░ ░ ▒    ░░   ░   ░         ░    ░    ░  
░ ░         ░ ░     ░                 ░  ░ ░    ░  `

func PrintBanner(version string) {
	banner := titleStyle.Render(logo)
	subtitle := subtitleStyle.Render("CLI for plugin and theme developers")
	ver := versionStyle.Render(fmt.Sprintf("v%s", version))

	fmt.Println(banner)
	fmt.Println()
	fmt.Printf("  %s  %s\n", subtitle, ver)
	fmt.Println()
}

func PrintBannerCompact(version string) {
	title := titleStyle.Bold(true).Render("Cortex CLI")
	ver := versionStyle.Render(fmt.Sprintf("v%s", version))
	subtitle := subtitleStyle.Render("Plugin & Theme Developer Tools")

	fmt.Printf("%s %s - %s\n\n", title, ver, subtitle)
}

func PrintWelcome(version string) {
	boxContent := fmt.Sprintf(
		"%s %s\n%s",
		titleStyle.Render("Cortex CLI"),
		versionStyle.Render("v"+version),
		subtitleStyle.Render("Developer tools for plugins and themes"),
	)

	box := bannerBorder.Render(boxContent)
	fmt.Println(box)
	fmt.Println()
}
