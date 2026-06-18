package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"scripts/internal/config"
	"scripts/internal/executor"
	"scripts/internal/installer"
	"scripts/internal/tui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "install" {
		if err := installer.SelfInstall(); err != nil {
			fmt.Fprintf(os.Stderr, "Install failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Installed to %s.\n", installer.InstallPath())
		return
	}

	if len(os.Args) > 2 && os.Args[1] == "--notui" {
		runNoTUI(os.Args[2])
		return
	}

	if !installer.IsRunningFromInstall() && !installer.IsInstalled() {
		if installer.PromptAndInstall() {
			if err := installer.SelfInstall(); err != nil {
				fmt.Fprintf(os.Stderr, "Install failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Installed to %s.\n", installer.InstallPath())
		}
	}

	cfg, err := config.LoadOrPrompt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	dir := "."
	if len(os.Args) > 1 && os.Args[1] != "install" {
		dir = os.Args[1]
	}

	scripts, err := executor.FindScripts(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding scripts: %v\n", err)
		os.Exit(1)
	}

	model := tui.New(scripts, cfg.OpenRouterKey)
	program := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runNoTUI(file string) {
	absPath, err := filepath.Abs(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is a directory\n", file)
		os.Exit(1)
	}

	ext := filepath.Ext(absPath)
	name := filepath.Base(absPath)
	s := executor.Script{Name: name, Path: absPath}

	var cmd *exec.Cmd

	switch ext {
	case ".c", ".cc", ".cpp", ".cxx", ".go", ".rs":
		compileCmd := executor.CompileCommand(s)
		fmt.Printf("Compiling %s...\n", name)
		out, cerr := compileCmd.CombinedOutput()
		if len(out) > 0 {
			os.Stdout.Write(out)
		}
		if cerr != nil {
			fmt.Fprintf(os.Stderr, "Compile failed: %v\n", cerr)
			os.Exit(1)
		}
		defer executor.Cleanup(name)
		cmd = executor.RunCompiledCmd(s)
	default:
		cmd, err = executor.Command(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
