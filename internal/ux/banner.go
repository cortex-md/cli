package ux

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	logoPrimaryStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(BrandPrimary)).
				Bold(true)

	logoAccentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(BrandAccent)).
			Bold(true)

	logoMutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(BrandMuted))

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(BrandAccent)).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(BrandMuted))

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(BrandMuted))
)

const cortexLogo = `                                          ++-------------
                                       +---------------------
                                     +-------------------------+
                                   +----------------------------+
                                 +++---------------..-------------+
                                ++++------------------------------++
                               +++++-----------------------------+++
                               ++++++----------------------------++++
                              ++++++++--------------------------++++++
                              ++++++++++-----------------------+++++++
                              ++++++++++++-------------------+++++++++
                              ++++++++++++++-------------+++++++++++++
                              +++++++++###+##++-------+###+##+++++++++
                               +++++++####.-##++++++++####.-##+++++++
                      ++++     +++++++########++++++++########+++++++     ++++
                   +++++++++    +++++++#######+++++++++#######++++++    ++++++++
                   ++++####+     +++++++####++++++++++++####+++++++     +###+++++
                  #++++#   #     ###+++++++++++++++++++++++++++++##     #   #++++#
                  ###+++###########++++++++++++++++++++++++++++++###########+++###
                   ###############+++++++++++++++++++++++++++++++++#######++#####
                    ##############+++++++++++++++++++++++++++++++++#############
                      ###+-----+######++###++++++##++++++###++++####+-----+###
                       +--+++--+############++++####+++#+###########+--+++--+
                      --+##################################################+--
                      --+########################  ########################+--
                       +++######################   #######################+++
                        ++++###################      ###################++++
                         ##+++###############         ################+++#
                             #############                ##############`

func PrintBanner(version string) {
	title := titleStyle.Render("Cortex CLI")
	ver := versionStyle.Render(fmt.Sprintf("v%s", version))

	fmt.Println(renderCortexLogo())
	fmt.Println()
	fmt.Printf("  %s %s\n", title, ver)
	fmt.Printf("  %s\n", subtitleStyle.Render("Plugin and theme developer tools"))
	fmt.Println()
}

func PrintBannerCompact(version string) {
	mark := renderCortexLogoCompact()
	title := titleStyle.Render("Cortex CLI")
	ver := versionStyle.Render(fmt.Sprintf("v%s", version))
	subtitle := subtitleStyle.Render("Plugin & theme tools")

	fmt.Printf("%s %s %s - %s\n\n", mark, title, ver, subtitle)
}

func PrintWelcome(version string) {
	fmt.Println(renderCortexLogo())
	fmt.Println()

	boxContent := fmt.Sprintf(
		"%s %s %s\n%s",
		renderCortexLogoCompact(),
		titleStyle.Render("Cortex CLI"),
		versionStyle.Render("v"+version),
		subtitleStyle.Render("Developer tools for plugins and themes"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.ASCIIBorder()).
		BorderForeground(lipgloss.Color(BrandAccent)).
		Padding(0, 2).
		Render(boxContent)
	fmt.Println(box)
	fmt.Println()
}

func renderCortexLogo() string {
	var builder strings.Builder
	for lineIndex, line := range strings.Split(cortexLogo, "\n") {
		if lineIndex > 0 {
			builder.WriteByte('\n')
		}
		for _, char := range line {
			builder.WriteString(renderLogoChar(char))
		}
	}
	return builder.String()
}

func renderCortexLogoCompact() string {
	return logoAccentStyle.Render("++") + logoPrimaryStyle.Render("##") + logoAccentStyle.Render("++")
}

func renderLogoChar(char rune) string {
	switch char {
	case '+':
		return logoAccentStyle.Render(string(char))
	case '#':
		return logoPrimaryStyle.Render(string(char))
	case '-', '.':
		return logoMutedStyle.Render(string(char))
	default:
		return string(char)
	}
}
