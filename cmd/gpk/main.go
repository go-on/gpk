package main

import (
	"fmt"
	"github.com/go-on/gpk"
	"gopkg.in/metakeule/config.v1"
	"os"
	"path/filepath"
)

var (
	cfg = config.MustNew("gpk", "0.0.1", "gpk is a tool to manage go libraries")
	dir = cfg.NewString(
		"dir",
		"directory of the concerned package (github working copy)",
		config.Default("."),
		config.Required,
		config.Shortflag('d'),
	)
	develop = cfg.MustCommand("develop", "switch package to github repo in order to develop")
	release = cfg.MustCommand("release", "commit current changes and create a release tag")
	step    = release.NewString("step", "step that should be upped, available options are: minor|major|patch",
		config.Required,
		config.Default("patch"),
		config.Shortflag('s'),
	)
	message = release.NewString(
		"message",
		"commit message",
		config.Required,
		config.Shortflag('m'),
	)
)

func reportError(err error) {
	if err != nil {
		// panic(err)
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func getDir() string {
	d := dir.Get()
	a, err := filepath.Abs(d)
	reportError(err)
	return a
}

func main() {

	err := cfg.Run()
	reportError(err)

	switch cfg.ActiveCommand() {
	case develop:
		err = gpk.ReplaceWithGithubPath(getDir())
	case release:
		var version [3]int
		switch step.Get() {
		case "major":
			version, err = gpk.SetNewMajor(getDir(), message.Get())
		case "minor":
			version, err = gpk.SetNewMinor(getDir(), message.Get())
		case "patch":
			version, err = gpk.SetNewPatch(getDir(), message.Get())
		default:
			err = fmt.Errorf("unsupported step: %s", step.Get())
			// report error here
		}

		if err != nil {
			fmt.Fprintf(os.Stdout, "added tag: %s\n", gpk.VersionString(version))
		}
	}

	reportError(err)
}
