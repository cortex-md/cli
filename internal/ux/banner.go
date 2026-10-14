package ux

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	logoPrimaryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandPrimary)).Bold(true)
	logoAccentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandAccent)).Bold(true)
	logoMutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandMuted))
	titleStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color(BrandPrimary)).Bold(true)
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
	fmt.Fprintln(Stdout, renderCortexLogo())
	fmt.Fprintln(Stdout)
	PrintHeader(version)
}

func PrintHeader(version string) {
	fmt.Fprintf(Stdout, "%s %s %s  %s\n", renderCortexMark(), titleStyle.Render("Cortex CLI"), Muted("v"+version), Muted("plugin & theme tools"))
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

func renderCortexMark() string {
	return logoAccentStyle.Render("+") + logoPrimaryStyle.Render("##") + logoAccentStyle.Render("+")
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
