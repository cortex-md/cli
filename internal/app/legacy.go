package app

import (
	"fmt"

	"github.com/cortex/cli/internal/project"
	"github.com/cortex/cli/internal/ux"
	"github.com/spf13/cobra"
)

func newLegacyCommands() []*cobra.Command {
	return []*cobra.Command{
		newLegacyKindCommand("plugin"),
		newLegacyKindCommand("theme"),
		newLegacyRegistryCommand(),
	}
}

func newLegacyKindCommand(kind string) *cobra.Command {
	group := &cobra.Command{Use: kind, Hidden: true}

	var strict bool
	validate := legacy(kind+" validate", "cortex check", func(cmd *cobra.Command, args []string) error {
		p, err := project.Read(projectDir(args))
		if err != nil {
			return err
		}
		return runChecks(p, false, false)
	})
	validate.Flags().BoolVar(&strict, "strict", false, "Ignored; warnings no longer fail legacy validation")

	var vault string
	link := legacy(kind+" link", "cortex dev --once", func(cmd *cobra.Command, args []string) error {
		return runDev(cmd.Context(), projectDir(args), devOptions{vault: vault, once: true})
	})
	link.Flags().StringVar(&vault, "vault", "", "Cortex vault path")

	var unlinkVault string
	unlink := legacy(kind+" unlink", "cortex dev --clean", func(cmd *cobra.Command, args []string) error {
		return runDev(cmd.Context(), projectDir(args), devOptions{vault: unlinkVault, clean: true})
	})
	unlink.Flags().StringVar(&unlinkVault, "vault", "", "Cortex vault path")

	var devVault string
	devCommand := legacy(kind+" dev", "cortex dev", func(cmd *cobra.Command, args []string) error {
		return runDev(cmd.Context(), projectDir(args), devOptions{vault: devVault})
	})
	devCommand.Flags().StringVar(&devVault, "vault", "", "Cortex vault path")

	group.AddCommand(
		legacy(kind+" create", "cortex create --"+kind, func(cmd *cobra.Command, args []string) error {
			opts := &createOptions{asPlugin: kind == "plugin", asTheme: kind == "theme"}
			if len(args) > 0 {
				opts.name = args[0]
			}
			return runCreate(opts)
		}),
		devCommand,
		validate,
		link,
		unlink,
		legacy(kind+" doctor", "cortex check", func(cmd *cobra.Command, args []string) error {
			p, err := project.Read(projectDir(args))
			if err != nil {
				return err
			}
			return runChecks(p, false, false)
		}),
		legacy(kind+" build", "cortex check", func(cmd *cobra.Command, args []string) error {
			p, err := project.Load(projectDir(args))
			if err != nil {
				return err
			}
			return buildProject(p)
		}),
		legacy(kind+" publish", "cortex pack", func(cmd *cobra.Command, args []string) error {
			p, err := project.Load(projectDir(args))
			if err != nil {
				return err
			}
			if err := runChecks(p, false, false); err != nil {
				return err
			}
			return packProject(p)
		}),
	)
	return group
}

func newLegacyRegistryCommand() *cobra.Command {
	group := &cobra.Command{Use: "registry", Hidden: true}
	group.AddCommand(legacy("registry submit", "cortex submit", func(cmd *cobra.Command, args []string) error {
		p, err := project.Load(".")
		if err != nil {
			return err
		}
		return runSubmit(cmd.Context(), p, submitOptions{})
	}))
	return group
}

func legacy(old string, replacement string, run func(cmd *cobra.Command, args []string) error) *cobra.Command {
	name := old
	for index := len(old) - 1; index >= 0; index-- {
		if old[index] == ' ' {
			name = old[index+1:]
			break
		}
	}
	return &cobra.Command{
		Use:    name + " [directory]",
		Short:  "Deprecated: use " + replacement,
		Hidden: true,
		Args:   cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(ux.Stderr, "%s `cortex %s` is deprecated; use `%s`\n", ux.Muted("!"), old, replacement)
			return run(cmd, args)
		},
	}
}
