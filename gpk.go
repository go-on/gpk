package gpk

import (
	"errors"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

func Pkg(dir string) (*build.Package, error) {
	return build.ImportDir(dir, build.ImportMode(0))
}

func PkgPath(pkg *build.Package) (string, error) {
	return filepath.Rel(pkg.SrcRoot, pkg.Dir)
}

// GOPATH returns the gopath of a package
func GOPATH(pkg *build.Package) string {
	return pkg.Root
}

// Imports returns the imports of a package
func Imports(dir string) ([]string, error) {
	pkg, err := build.ImportDir(dir, build.ImportMode(0))
	if err != nil {
		return nil, err
	}
	return pkg.Imports, nil
}

func isStdLib(p string) (bool, error) {
	// var p string
	switch runtime.Version() {
	case "go1.4":
		p = filepath.Join(runtime.GOROOT(), "src", p)
	default:
		p = filepath.Join(runtime.GOROOT(), "src", "pkg", p)
	}

	info, err := os.Stat(p)
	// log.Printf("info: %#v\n", info)
	// fmt.Println("p", p)
	if err != nil {
		return false, nil
	}

	if !info.IsDir() {
		return false, fmt.Errorf("must be dir: %#v", p)
	}
	return true, nil
	// return filepath.HasPrefix(p, runtime.GOROOT())
}

func extImports(pkg *build.Package) ([]string, error) {
	// imps, err := Imports(dir)
	imps := pkg.Imports
	/*
		if err != nil {
			return nil, err
		}
	*/

	e := []string{}

	for _, im := range imps {
		is, err := isStdLib(im)
		if err != nil {
			return nil, err
		}
		if !is {
			e = append(e, im)
		}
	}

	return e, nil
}

//  ExtImports returns the imports of a package that are not part of the standard library
func ExtImports(dir string) ([]string, error) {
	imps, err := Imports(dir)

	if err != nil {
		return nil, err
	}

	e := []string{}

	for _, im := range imps {
		is, err := isStdLib(im)
		if err != nil {
			return nil, err
		}
		if !is {
			e = append(e, im)
		}
	}

	return e, nil
}

//  StdImports returns the imports of a package that are  part of the standard library
func StdImports(dir string) ([]string, error) {
	imps, err := Imports(dir)

	if err != nil {
		return nil, err
	}

	e := []string{}

	for _, im := range imps {
		is, err := isStdLib(im)
		if err != nil {
			return nil, err
		}
		if is {
			e = append(e, im)
		}
	}

	return e, nil
}

func inSlice(s []string, what string) bool {

	for _, str := range s {
		if str == what {
			return true
		}
	}
	return false

}

func inSlicePrefix(s []string, prefix string) bool {
	for _, str := range s {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}
	return false
}

// Dependents returns packages inside the given dir
// that are dependent of the given package, because they import it
// does not look into directories that begin with the dot
func Dependents(dir, p string) ([]string, error) {
	d := []string{}
	pkg, errPkg := Pkg(p)

	if errPkg != nil {
		return nil, errPkg
	}

	relPath, errRel := PkgPath(pkg)
	if errRel != nil {
		return nil, errRel
	}

	err := filepath.Walk(dir, func(f string, info os.FileInfo, err error) error {
		// fmt.Printf("f: %#v\n", f)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(filepath.Base(f), ".") {
				return filepath.SkipDir
			}
			pk, errPkg := Pkg(f)
			if errPkg != nil {
				return nil
			}

			imps, errImps := extImports(pk)
			if errImps != nil {
				return errImps
			}

			if inSlice(imps, relPath) {
				relImp, errRelImp := PkgPath(pk)

				if errRelImp != nil {
					return errRelImp
				}

				d = append(d, relImp)
			}
		}
		return nil
	})

	return d, err
}

// dependentsPrefix is like DependentsPrefix, but p is a package path, not a directory
func dependentsPrefix(dir, relPath string) ([]string, error) {
	d := []string{}

	err := filepath.Walk(dir, func(f string, info os.FileInfo, err error) error {
		// fmt.Printf("f: %#v\n", f)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(filepath.Base(f), ".") {
				return filepath.SkipDir
			}
			pk, errPkg := Pkg(f)
			if errPkg != nil {
				return nil
			}

			imps, errImps := extImports(pk)
			if errImps != nil {
				return errImps
			}

			if inSlicePrefix(imps, relPath) {
				relImp, errRelImp := PkgPath(pk)

				if errRelImp != nil {
					return errRelImp
				}

				d = append(d, relImp)
			}
		}
		return nil
	})

	return d, err
}

// DependentsPrefix returns packages inside the given dir
// that are dependent of the given package because they import it or a subpackage of it
// does not look into directories that begin with the dot
func DependentsPrefix(dir, p string) ([]string, error) {
	// d := []string{}
	pkg, errPkg := Pkg(p)

	if errPkg != nil {
		return nil, errPkg
	}

	relPath, errRel := PkgPath(pkg)
	if errRel != nil {
		return nil, errRel
	}

	return dependentsPrefix(dir, relPath)
}

// InGoPkgIn checks if the given path is in gopkg.in
func InGoPkgin(path string) bool {
	return filepath.HasPrefix(path, "gopkg.in")
}

var ErrNoGoPkginPath = errors.New("no gopkg.in path")
var ErrInvalidGoPkginPath = errors.New("invalid gopkg.in path")
var ErrInvalidGoPkginVersion = errors.New("invalid gopkg.in version string")

// parseVersion parses the version out of strings like
// v1 v2.3 v4.0.3
func parseVersion(version string) ([3]int, error) {
	v := [3]int{0, 0, 0}
	if !strings.HasPrefix(version, "v") || len(version) < 2 {
		return v, ErrInvalidGoPkginVersion
	}
	vers := version[1:]
	a := strings.Split(vers, ".")

	max := len(a)
	if max > 3 {
		max = 3
	}

	for i := 0; i < max; i++ {
		n, err := strconv.Atoi(a[i])

		if err != nil {
			return v, err
		}
		v[i] = n
	}
	return v, nil
}

// GoPkginVersion parses major minor and patch version out of
// a gopkg.in package string
func GoPkginVersion(path string) ([3]int, error) {
	v := [3]int{0, 0, 0}

	if !InGoPkgin(path) {
		return v, ErrNoGoPkginPath
	}

	idx := strings.LastIndex(path, "v")
	if idx == -1 || len(path) <= idx+1 {
		return v, ErrInvalidGoPkginPath
	}

	end := strings.LastIndex(path, "/")
	if end == -1 {
		return v, ErrInvalidGoPkginPath
	}

	if end < idx {
		end = len(path)
	}

	return parseVersion(path[idx:end])
}

var ErrInvalidGithubPath = errors.New("invalid github path")

// bareGoPkginPath is like GoPkginPath but without a version
func bareGoPkginPath(p string) (string, error) {
	idx := strings.Index(p, "/")

	if idx == -1 || idx == 0 || p[:idx] != "github.com" {
		return "", ErrInvalidGithubPath
	}

	return fmt.Sprintf("gopkg.in%s", p[idx:]), nil
}

// GoPkginPath returns the versioned gopkg.in path for a package
func GoPkginPath(p string, version [3]int) (string, error) {
	idx := strings.Index(p, "/")

	if idx == -1 || idx == 0 || p[:idx] != "github.com" {
		return "", ErrInvalidGithubPath
	}

	vers := VersionString(version)

	return fmt.Sprintf("gopkg.in%s.%s", p[idx:], vers), nil
}

// GithubPath returns the github path for the gopkg.in path
func GithubPath(p string) (string, error) {
	start := strings.Index(p, "/")

	if start == -1 || start == 0 || p[:start] != "gopkg.in" {
		return "", ErrInvalidGoPkginPath
	}

	stop := strings.LastIndex(p, ".v")

	if stop == -1 || stop == 0 {
		return "", ErrInvalidGoPkginPath
	}

	return fmt.Sprintf("github.com%s", p[start:stop]), nil
}

func replaceGopkgin(gopkginBare string, target string, in []byte) ([]byte, error) {
	re, err := regexp.Compile(`"` + regexp.QuoteMeta(gopkginBare+".v") + `([0-9]+)(\.[0-9]+)*`)
	// fmt.Println(re)
	if err != nil {
		return nil, err
	}

	return re.ReplaceAll(in, []byte(`"`+target)), nil
}

func replaceGithub(github string, target string, in []byte) ([]byte, error) {
	re, err := regexp.Compile(`"` + regexp.QuoteMeta(github) + `("|/)`)
	// fmt.Println(re)
	if err != nil {
		return nil, err
	}

	return re.ReplaceAll(in, []byte(`"`+target+"$1")), nil
}

// ReplaceWithGithubPath takes a pkg pkgDir that is a GithubPath.
// It replaces inside every file inside every package beneath the given dir
// an string that references any  gopkg variant of the given package by the github variant
// It can be used for developement to be able to run the tests, switch back by calling
// ReplaceWithGopkginPath
func ReplaceWithGithubPath(pkgDir string) error {
	// fmt.Println("replacing inside", pkgDir)
	pkg, err := Pkg(pkgDir)
	if err != nil {
		return err
	}

	pkgPath, err2 := PkgPath(pkg)

	if err2 != nil {
		return err2
	}

	gopkgin, err3 := bareGoPkginPath(pkgPath)

	if err3 != nil {
		return err3
	}

	deps, err4 := dependentsPrefix(pkgDir, gopkgin)

	if err4 != nil {
		return err4
	}

	// fmt.Printf("deps: %#v\n", deps)

	for _, dep := range deps {
		dpkg, err := build.Import(dep, pkg.SrcRoot, build.ImportMode(0))
		if err != nil {
			return err
		}

		for _, file := range dpkg.GoFiles {
			// fmt.Println(filepath.Join(pkg.SrcRoot, dep, file))

			path := filepath.Join(pkg.SrcRoot, dep, file)

			bt, err := ioutil.ReadFile(path)

			if err != nil {
				return err
			}

			out, err2 := replaceGopkgin(gopkgin, pkgPath, bt)

			if err2 != nil {
				return err2
			}

			err3 := ioutil.WriteFile(path, out, 0644)

			if err3 != nil {
				return err3
			}

			// bytes.Replace(bt, pkgPath, new, n)
			/**/
		}

		for _, file := range dpkg.TestGoFiles {
			// fmt.Println(filepath.Join(pkg.SrcRoot, dep, file))

			path := filepath.Join(pkg.SrcRoot, dep, file)

			bt, err := ioutil.ReadFile(path)

			if err != nil {
				return err
			}

			out, err2 := replaceGopkgin(gopkgin, pkgPath, bt)

			if err2 != nil {
				return err2
			}

			err3 := ioutil.WriteFile(path, out, 0644)

			if err3 != nil {
				return err3
			}

		}

		// dpkg.GoFiles

		// dpkg.TestGoFiles
	}

	return nil
}

// ReplaceWithGithubPath takes a pkg pkgDir that is a GithubPath.
// It replaces inside every file inside every package beneath the given dir
// an string that references any  github or gopkgin variant of the given package by the gopkg variant
// with the given version.
// It can be used to release a package after ReplaceWithGithubPath has been used or to update
// a version number
func ReplaceWithGopkginPath(pkgdir string, version [3]int) error {
	// fmt.Println("replacing inside", pkgDir)
	pkg, err := Pkg(pkgdir)
	if err != nil {
		return err
	}

	pkgPath, err2 := PkgPath(pkg)

	if err2 != nil {
		return err2
	}

	gopkgin, err3 := bareGoPkginPath(pkgPath)

	if err3 != nil {
		return err3
	}

	target, errX := GoPkginPath(pkgPath, version)

	if errX != nil {
		return errX
	}

	deps, err4 := dependentsPrefix(pkgdir, pkgPath)

	if err4 != nil {
		return err4
	}

	// fmt.Printf("deps: %#v\n", deps)

	for _, dep := range deps {
		dpkg, err := build.Import(dep, pkg.SrcRoot, build.ImportMode(0))
		if err != nil {
			return err
		}

		for _, file := range dpkg.GoFiles {
			// fmt.Println(filepath.Join(pkg.SrcRoot, dep, file))

			path := filepath.Join(pkg.SrcRoot, dep, file)

			bt, err := ioutil.ReadFile(path)

			if err != nil {
				return err
			}

			out, err2 := replaceGopkgin(gopkgin, target, bt)

			if err2 != nil {
				return err2
			}

			out, err2 = replaceGithub(pkgPath, target, out)

			if err2 != nil {
				return err2
			}

			err3 := ioutil.WriteFile(path, out, 0644)

			if err3 != nil {
				return err3
			}

			// bytes.Replace(bt, pkgPath, new, n)
			/**/
		}

		for _, file := range dpkg.TestGoFiles {
			// fmt.Println(filepath.Join(pkg.SrcRoot, dep, file))

			path := filepath.Join(pkg.SrcRoot, dep, file)

			bt, err := ioutil.ReadFile(path)

			if err != nil {
				return err
			}

			out, err2 := replaceGopkgin(gopkgin, target, bt)

			if err2 != nil {
				return err2
			}

			out, err2 = replaceGithub(pkgPath, target, out)

			if err2 != nil {
				return err2
			}

			err3 := ioutil.WriteFile(path, out, 0644)

			if err3 != nil {
				return err3
			}

		}

		// dpkg.GoFiles

		// dpkg.TestGoFiles
	}

	return nil
}

type sortVersion [][3]int

func (v sortVersion) Len() int      { return len(v) }
func (v sortVersion) Swap(i, j int) { v[i], v[j] = v[j], v[i] }
func (v sortVersion) Less(i, j int) bool {
	if v[i][0] == v[j][0] {
		if v[i][1] == v[j][1] {
			return v[i][2] < v[j][2]
		}
		return v[i][1] < v[j][1]
	}
	return v[i][0] < v[j][0]
}

// lastVersion returns the last version of a version slice
func lastVersion(versions ...[3]int) [3]int {
	sv := sortVersion(versions)
	sort.Sort(sv)
	return [3]int(sv[len(versions)-1])
}

func VersionString(version [3]int) string {
	vers := fmt.Sprintf("v%d", version[0])
	if version[1] != 0 {
		vers += fmt.Sprintf(".%d", version[1])
	}
	if version[2] != 0 {
		if version[1] == 0 {
			vers += fmt.Sprintf(".0.%d", version[2])
		} else {
			vers += fmt.Sprintf(".%d", version[2])
		}
	}
	return vers
}

// LastVersion returns the last version of a version slice like this
// []string{"v1","v5", "v1.10"}
func LastVersion(versions ...string) ([3]int, error) {
	l := [3]int{0, 0, 0}
	var v [][3]int

	for _, vers := range versions {
		vv, err := parseVersion(vers)

		if err != nil {
			return l, err
		}

		v = append(v, vv)
	}

	return lastVersion(v...), nil
}
