package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"gitwise/internal/git"
	"gitwise/internal/ui"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "doctor",
		Short: "Check your git installation and configuration",
		RunE:  func(cmd *cobra.Command, args []string) error { return runDoctor() },
	})
}

type checkLevel int

const (
	checkOK checkLevel = iota
	checkInfo
	checkWarn
	checkFail
)

type check struct {
	level  checkLevel
	name   string
	detail string
	fix    string
}

func (c check) render() string {
	var sym string
	switch c.level {
	case checkOK:
		sym = ui.Success.Render("✓")
	case checkInfo:
		sym = ui.Heading.Render("i")
	case checkWarn:
		sym = ui.Warn.Render("!")
	default:
		sym = ui.Danger.Render("✗")
	}
	s := fmt.Sprintf(" %s %s %s", sym, ui.Bold.Width(20).Render(c.name), c.detail)
	if c.fix != "" {
		s += "\n    " + strings.Repeat(" ", 20) + ui.Muted.Render("fix: ") + ui.Code.Render(c.fix)
	}
	return s
}

func runDoctor() error {
	if !git.Installed() {
		fmt.Println(check{checkFail, "git", "not found on PATH", "install git from https://git-scm.com/downloads"}.render())
		return nil
	}
	dir := cwd()
	fmt.Println(ui.Title.Render("gitwise doctor") + "\n")

	var checks []check
	ver, _ := git.Version()
	if versionAtLeast(ver, 2, 23) {
		checks = append(checks, check{checkOK, "git version", ver, ""})
	} else {
		checks = append(checks, check{checkWarn, "git version", ver + " is old — `git switch`/`git restore` need 2.23+", "upgrade git"})
	}

	name, email := git.Config(dir, "user.name"), git.Config(dir, "user.email")
	if name == "" || email == "" {
		if interactive() && askIdentity(&name, &email) {
			checks = append(checks, check{checkOK, "identity", name + " <" + email + "> (just configured)", ""})
		} else {
			checks = append(checks, check{checkFail, "identity", "user.name / user.email not set — you can't commit",
				`git config --global user.name "Your Name"`})
		}
	} else {
		checks = append(checks, check{checkOK, "identity", name + " <" + email + ">", ""})
	}

	if b := git.Config(dir, "init.defaultBranch"); b != "" {
		checks = append(checks, check{checkOK, "default branch", b, ""})
	} else {
		checks = append(checks, check{checkInfo, "default branch", "not set (new repos may start on master)", "git config --global init.defaultBranch main"})
	}

	switch {
	case os.Getenv("GIT_EDITOR") != "":
		checks = append(checks, check{checkOK, "editor", os.Getenv("GIT_EDITOR") + " (GIT_EDITOR)", ""})
	case git.Config(dir, "core.editor") != "":
		checks = append(checks, check{checkOK, "editor", git.Config(dir, "core.editor"), ""})
	case os.Getenv("VISUAL") != "" || os.Getenv("EDITOR") != "":
		checks = append(checks, check{checkOK, "editor", os.Getenv("VISUAL") + os.Getenv("EDITOR") + " (from environment)", ""})
	default:
		checks = append(checks, check{checkWarn, "editor", "none set — git falls back to vi, which confuses many beginners", `git config --global core.editor "nano"`})
	}

	if git.Config(dir, "pull.rebase") == "" && git.Config(dir, "pull.ff") == "" {
		checks = append(checks, check{checkInfo, "pull strategy", "not set — git pull may print a warning when branches diverge", "git config --global pull.rebase false"})
	} else {
		checks = append(checks, check{checkOK, "pull strategy", "configured", ""})
	}

	if h := git.Config(dir, "credential.helper"); h != "" {
		checks = append(checks, check{checkOK, "credentials", "helper: " + h, ""})
	} else {
		checks = append(checks, check{checkInfo, "credentials", "no credential helper — HTTPS remotes will ask for your password/token every time", "git config --global credential.helper store   (or use SSH)"})
	}

	checks = append(checks, sshCheck())

	if git.IsRepo(dir) {
		top, _ := git.TopLevel(dir)
		if _, err := os.Stat(filepath.Join(top, ".gitignore")); err != nil {
			checks = append(checks, check{checkWarn, ".gitignore", "this repository has none — build output and secrets may get committed", "see https://github.com/github/gitignore for templates"})
		} else {
			checks = append(checks, check{checkOK, ".gitignore", "present", ""})
		}
		if remotes, _ := git.Run(dir, "remote"); remotes == "" {
			checks = append(checks, check{checkInfo, "remote", "this repository has no remote — it only exists on your machine", "git remote add origin <url>"})
		} else {
			checks = append(checks, check{checkOK, "remote", strings.ReplaceAll(remotes, "\n", ", "), ""})
		}
	}

	problems := 0
	for _, c := range checks {
		fmt.Println(c.render())
		if c.level >= checkWarn {
			problems++
		}
	}
	fmt.Println()
	if problems == 0 {
		fmt.Println(ui.Success.Render("All good — you're ready to git!"))
	} else {
		fmt.Println(ui.Warn.Render(fmt.Sprintf("%d thing(s) worth fixing.", problems)))
	}
	return nil
}

func sshCheck() check {
	home, err := os.UserHomeDir()
	if err != nil {
		return check{checkInfo, "ssh key", "could not determine home folder", ""}
	}
	for _, k := range []string{"id_ed25519.pub", "id_ecdsa.pub", "id_rsa.pub"} {
		if _, err := os.Stat(filepath.Join(home, ".ssh", k)); err == nil {
			return check{checkOK, "ssh key", "~/.ssh/" + k, ""}
		}
	}
	return check{checkInfo, "ssh key", "none found — needed for SSH remotes (git@github.com:...)", `ssh-keygen -t ed25519 -C "you@example.com"`}
}

func askIdentity(name, email *string) bool {
	set := false
	if err := huh.NewConfirm().Title("Git doesn't know who you are yet. Set your name and email now?").Value(&set).Run(); err != nil || !set {
		return false
	}
	notEmpty := func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("required")
		}
		return nil
	}
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Your name").Description("shown on every commit").Value(name).Validate(notEmpty),
		huh.NewInput().Title("Your email").Description("use the one from your GitHub/GitLab account").Value(email).Validate(notEmpty),
	)).Run()
	if err != nil {
		return false
	}
	_, err1 := git.Run("", "config", "--global", "user.name", strings.TrimSpace(*name))
	_, err2 := git.Run("", "config", "--global", "user.email", strings.TrimSpace(*email))
	return err1 == nil && err2 == nil
}

func versionAtLeast(v string, major, minor int) bool {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return true
	}
	ma, _ := strconv.Atoi(parts[0])
	mi, _ := strconv.Atoi(parts[1])
	return ma > major || (ma == major && mi >= minor)
}
