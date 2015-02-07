package main

import (
	"fmt"
	"go/build"
	"gopkg.in/go-on/gpk.v1"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/metakeule/config.v1"
)

/*
commands:

dependents // show packages inside the dir that depend on the given package
last_version // shows the last version by tags

*/

var (
	cfg = config.MustNew("gpk", "1.1", "gpk is a tool to manage go libraries")

	dir = cfg.NewString(
		"dir",
		"directory of the concerned package (github working copy)",
		config.Default("."),
		config.Required,
		config.Shortflag('d'),
	)

	verbose = cfg.NewBool("verbose", "verbose messages", config.Shortflag('v'))

	develop = cfg.MustCommand("develop", "switch package to github repo in order to develop")

	replace       = cfg.MustCommand("replace", "replace an import with another")
	replaceSrc    = replace.NewString("src", "the import that should be replaced", config.Required)
	replaceTarget = replace.NewString("target", "the replacement for the import", config.Required)

	release     = cfg.MustCommand("release", "change pkg import paths to release tag")
	releaseStep = release.NewString("step", "step that should be upped, available options are: minor|major|patch",
		config.Required,
		config.Default("patch"),
		config.Shortflag('s'),
	)
	push     = cfg.MustCommand("push", "tag the version and push it")
	pushStep = push.NewString("step", "step that should be upped, available options are: minor|major|patch",
		config.Required,
		config.Default("patch"),
		config.Shortflag('s'),
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

	if verbose.Get() {
		gpk.DEBUG = true
	}

	switch cfg.ActiveCommand() {
	case replace:
		err = gpk.ReplaceImport(getDir(), replaceSrc.Get(), replaceTarget.Get())
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
		// fmt.Println(path)
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
		switch releaseStep.Get() {
		case "major":
			version, err = gpk.SetNewMajor(getDir())
		case "minor":
			version, err = gpk.SetNewMinor(getDir())
		case "patch":
			version, err = gpk.SetNewPatch(getDir())
		default:
			err = fmt.Errorf("unsupported step: %s", releaseStep.Get())
			// report error here
		}

		var changedVersion [3]int
		changedVersion[0] = version[0]
		if err == nil {
			fmt.Fprintf(
				os.Stdout,
				"changed pkg imports to: %s (for %s)\nDon't forget to run gpk push --step=%s\n",
				gpk.VersionString(changedVersion),
				gpk.VersionString(version),
				releaseStep.Get(),
			)
		}
	case push:
		var version [3]int
		switch pushStep.Get() {
		case "major":
			version, err = gpk.PushNewMajor(getDir())
		case "minor":
			version, err = gpk.PushNewMinor(getDir())
		case "patch":
			version, err = gpk.PushNewPatch(getDir())
		default:
			err = fmt.Errorf("unsupported step: %s", pushStep.Get())
			// report error here
		}

		var installedVersion [3]int
		installedVersion[0] = version[0]
		if err == nil {
			fmt.Fprintf(
				os.Stdout,
				"tagged and pushed: %s\ninstalled %s\n",
				gpk.VersionString(version),
				gpk.VersionString(installedVersion),
			)
		}
	default:
		fmt.Fprintln(os.Stderr, cfg.Usage())
		os.Exit(1)

	}

	reportError(err)
}
