package app

import (
	"context"
	"path/filepath"

	"github.com/cortex/cli/internal/config"
	"github.com/cortex/cli/internal/dev"
	"github.com/cortex/cli/internal/project"
	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
	"github.com/spf13/cobra"
)

type devOptions struct {
	vault string
	once  bool
	clean bool
}

func newDevCommand() *cobra.Command {
	var opts devOptions

	cmd := &cobra.Command{
		Use:   "dev [directory]",
		Short: "Install into a vault and reload on every change",
		Long: `Builds the project, copies it into <vault>/.cortex/plugins or <vault>/.cortex/themes,
and keeps it updated while you edit. Cortex reloads it automatically.

The vault is taken from --vault, CORTEX_VAULT, the folder you are in, or the last vault used.`,
		Example: `  cortex dev --vault ~/Notes
  cortex dev --once
  cortex dev --clean`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDev(cmd.Context(), projectDir(args), opts)
		},
	}

	cmd.Flags().StringVar(&opts.vault, "vault", "", "Cortex vault to install into (remembered)")
	cmd.Flags().BoolVar(&opts.once, "once", false, "Install once and exit")
	cmd.Flags().BoolVar(&opts.clean, "clean", false, "Remove the development copy from the vault")
	cmd.MarkFlagsMutuallyExclusive("once", "clean")

	return cmd
}

func runDev(ctx context.Context, dir string, opts devOptions) error {
	p, err := project.Load(dir)
	if err != nil {
		return err
	}

	vault, err := dev.ResolveVaultPath(p.Dir, opts.vault)
	if err != nil {
		return err
	}
	_ = config.RememberVault(vault)

	installDir := p.InstallDir(vault)
	if opts.clean {
		if err := dev.Uninstall(installDir); err != nil {
			return ux.NewProblem("Refusing to remove "+installDir, "it was installed from the Marketplace; uninstall it in Cortex instead", err)
		}
		ux.Success("Removed %s from %s", p.DisplayName(), ux.Path(vault))
		return nil
	}

	if dev.IsForeignInstall(installDir) {
		replace, err := ux.Confirm(p.DisplayName()+" is already installed in this vault from the Marketplace. Replace it with your local build?", false)
		if err != nil {
			return err
		}
		if !replace {
			return ux.ErrReported
		}
	}

	firstInstall := !exists(installDir)
	ux.Step("Developing %s %s in %s", kindLabel(p), ux.Command(p.DisplayName()), ux.Path(vault))
	if err := syncToVault(p, installDir); err != nil {
		return err
	}

	printDevHints(p, firstInstall)
	if opts.once {
		return nil
	}

	ux.Blank()
	ux.Info("Watching for changes. Press Ctrl+C to stop.")
	err = dev.Watch(ctx, p.Dir, relevantChange(p), func() {
		if reloaded, err := project.Load(p.Dir); err == nil {
			if reloaded.ID() != p.ID() {
				ux.Warning("The id changed to %q; restart `cortex dev` to install under the new id", reloaded.ID())
				return
			}
			p = reloaded
		}
		if err := syncToVault(p, installDir); err != nil {
			ux.PrintError(err)
		}
	})

	ux.Blank()
	ux.Info("Stopped. %s stays installed; remove it with %s", p.DisplayName(), ux.Command("cortex dev --clean"))
	return err
}

func syncToVault(p *project.Project, installDir string) error {
	if err := buildProject(p); err != nil {
		return err
	}
	return ux.Task("Installing into the vault", func() error {
		files, err := installFiles(p)
		if err != nil {
			return err
		}
		return dev.Install(installDir, files)
	})
}

func printDevHints(p *project.Project, firstInstall bool) {
	if !firstInstall {
		return
	}
	if p.Kind == manifest.KindTheme {
		ux.Info("Select %s in Cortex Settings to preview it", ux.Command(p.DisplayName()))
		return
	}
	ux.Info("In Cortex, turn on community plugins in Settings, then enable %s", ux.Command(p.DisplayName()))
}

func relevantChange(p *project.Project) func(string) bool {
	extensions := map[string]bool{".css": true, ".json": true}
	if p.Kind == manifest.KindPlugin {
		for _, extension := range []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"} {
			extensions[extension] = true
		}
	}
	return func(path string) bool {
		name := filepath.Base(path)
		if name == "package-lock.json" || name == "bun.lock" {
			return false
		}
		return extensions[filepath.Ext(path)]
	}
}
