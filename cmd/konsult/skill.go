package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/activout/konsult-cli/skills"
	"github.com/spf13/cobra"
)

// skillAgents maps an agent to the config dir that shows it is installed and its user skills dir (both under $HOME).
// Codex, Copilot, Cursor and Gemini CLI all read the shared ~/.agents/skills, so one copy serves them all;
// installing into their own dirs as well would make them load the skill twice.
var skillAgents = []struct{ name, detect, dir string }{
	{"claude", ".claude", ".claude/skills"},
	{"codex", ".codex", ".agents/skills"},
	{"copilot", ".copilot", ".agents/skills"},
	{"cursor", ".cursor", ".agents/skills"},
	{"gemini", ".gemini", ".agents/skills"},
}

func skillCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "skill", Short: "Manage the bundled agent skill"}
	var agents []string
	var dir string
	var force, dryRun bool
	names := make([]string, len(skillAgents))
	for i, a := range skillAgents {
		names[i] = a.name
	}
	install := &cobra.Command{
		Use:   "install",
		Short: "Install the konsult skill for your AI agents",
		Long: "Writes the bundled skill to <skills dir>/konsult for each agent found in your home directory\n" +
			"(" + strings.Join(names, ", ") + "), or only for --agent / --dir. Re-run after upgrading konsult;\n" +
			"a changed or symlinked install is replaced only with --force. The skill contains no secrets.",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			var targets []string
			if dir != "" {
				targets = append(targets, dir)
			}
			if dir == "" || len(agents) > 0 {
				home, err := os.UserHomeDir()
				if err != nil {
					return fail(exitGeneral, "%v", err)
				}
				for _, a := range agents {
					if !slices.Contains(names, a) {
						return fail(exitUsage, "unknown agent %q (known: %s)", a, strings.Join(names, ", "))
					}
				}
				for _, a := range skillAgents {
					if len(agents) > 0 && !slices.Contains(agents, a.name) {
						continue
					}
					if _, err := os.Stat(filepath.Join(home, a.detect)); len(agents) > 0 || err == nil {
						if t := filepath.Join(home, a.dir); !slices.Contains(targets, t) {
							targets = append(targets, t)
						}
					}
				}
				if len(targets) == 0 {
					return fail(exitUsage, "no agent found in %s; use --agent (%s) or --dir", home, strings.Join(names, ", "))
				}
			}
			var errs []error
			for _, t := range targets {
				if err := installSkill(filepath.Join(t, "konsult"), force, dryRun); err != nil {
					errs = append(errs, err)
				}
			}
			if len(errs) > 0 {
				return fail(exitUsage, "%v", errors.Join(errs...))
			}
			return nil
		},
	}
	fl := install.Flags()
	fl.StringSliceVar(&agents, "agent", nil, "install only for these agents: "+strings.Join(names, ", "))
	fl.StringVar(&dir, "dir", "", "install into this skills directory instead (the skill goes in DIR/konsult)")
	fl.BoolVar(&force, "force", false, "replace an existing install that differs or is a symlink")
	fl.BoolVar(&dryRun, "dry-run", false, "show where the skill would be installed without writing")
	cmd.AddCommand(install)
	return cmd
}

// installSkill writes the embedded skill to dest, built in a temp dir and renamed into place.
func installSkill(dest string, force, dryRun bool) error {
	src, _ := fs.Sub(skills.FS, "konsult")
	fi, err := os.Lstat(dest)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return err
	case fi.IsDir() && skillMatches(src, dest):
		fmt.Fprintf(errOut, "%s: up to date\n", dest)
		return nil
	case !force:
		return fmt.Errorf("%s exists and differs (or is not a directory); use --force to replace it", dest)
	}
	if dryRun {
		fmt.Fprintf(errOut, "would install the konsult skill to %s\n", dest)
		return nil
	}
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, ".konsult.*.part")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := os.Chmod(tmp, 0o755); err != nil {
		return err
	}
	if err := os.CopyFS(tmp, src); err != nil {
		return err
	}
	if fi != nil { // only reached with --force; Remove/RemoveAll never follow a symlink at dest
		if err := os.RemoveAll(dest); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, dest); err != nil {
		return err
	}
	fmt.Fprintf(errOut, "installed the konsult skill to %s\n", dest)
	return nil
}

// skillMatches reports whether every embedded file exists in dest with the same content.
func skillMatches(src fs.FS, dest string) bool {
	err := fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		want, _ := fs.ReadFile(src, p)
		got, err := os.ReadFile(filepath.Join(dest, p))
		if err != nil || !bytes.Equal(got, want) {
			return fs.ErrInvalid
		}
		return nil
	})
	return err == nil
}
