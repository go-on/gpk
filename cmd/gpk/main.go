package main

import (
	"fmt"
	"github.com/go-on/gpk"
	"go/build"
	"gopkg.in/metakeule/config.v1"
	"os"
	"path/filepath"
	"strings"
)

/*
commands:

dependents // show packages inside the dir that depend on the given package
last_version // shows the last version by tags

*/

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

	imports = cfg.MustCommand("imports", "show imported packages excluding stdlib packages")
	deps    = cfg.MustCommand("deps", "show packages inside the given dir that depends packages of the repo")
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
	case imports:
		var imps []string
		imps, err = gpk.ExtImports(getDir())
		reportError(err)
		fmt.Fprintln(os.Stdout, strings.Join(imps, "\n"))
	case deps:
		var p *build.Package
		p, err = gpk.Pkg(getDir())
		reportError(err)
		var path string
		path, err = gpk.PkgPath(p)
		fmt.Println(path)
		reportError(err)
		var depends []string
		//depends, err = gpk.DependentsPrefix(getDir(), filepath.Join(p.SrcRoot, path))
		depends, err = gpk.DependentsPrefix(getDir(), path)
		reportError(err)
		fmt.Fprintln(os.Stdout, strings.Join(depends, "\n"))
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

		if err == nil {
			fmt.Fprintf(os.Stdout, "added tag: %s\n", gpk.VersionString(version))
		}
	}

	reportError(err)
}
