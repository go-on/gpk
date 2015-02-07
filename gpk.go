package gpk

import (
	"errors"
	"fmt"
	"go/build"
	"gopkg.in/metakeule/gitlib.v1"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

var DEBUG bool

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
	// fmt.Println(runtime.Version())
	// var p string
	if strings.HasPrefix(runtime.Version(), "go1.4") {
		p = filepath.Join(runtime.GOROOT(), "src", p)
	} else {
		p = filepath.Join(runtime.GOROOT(), "src", "pkg", p)
	}

	// fmt.Print("is stdlib ", p, ": ")
	info, err := os.Stat(p)
	if os.IsNotExist(err) {
		// fmt.Println(false)
		return false, nil
	}
	// log.Printf("info: %#v\n", info)
	// fmt.Println("p", p)
	if err != nil {
		// fmt.Println(true)
		return true, err
	}

	if !info.IsDir() {
		// fmt.Println(false)
		return false, fmt.Errorf("must be dir: %#v", p)
	}
	// fmt.Println(true)
	return true, nil
}

func extImports(pkg *build.Package) ([]string, error) {
	imps := pkg.Imports

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

type dependentsWalker struct {
	relpath   string
	deps      []string
	inSliceFn func([]string, string) bool
}

func (d *dependentsWalker) Walk(f string, info os.FileInfo, err error) error {

	var (
		imports []string
		pkg     *build.Package
		pkgPath string
	)

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps

		case 0:
			// handle only directories
			if !info.IsDir() {
				break steps
			}
		case 1:
			// skip hidden directories
			if strings.HasPrefix(filepath.Base(f), ".") {
				err = filepath.SkipDir
			}
		case 2:
			// skip non packages
			pkg, err = Pkg(f)
			if err != nil {
				err = nil
				break steps
			}
		case 3:
			imports, err = extImports(pkg)
		case 4:
			// don't track non dependent packages
			if !d.inSliceFn(imports, d.relpath) {
				break steps
			}
		case 5:
			pkgPath, err = PkgPath(pkg)
		case 6:
			d.deps = append(d.deps, pkgPath)
		}
	}
	return err
}

// Dependents returns packages inside the given dir
// that are dependent of the given package, because they import it
// does not look into directories that begin with the dot
func Dependents(dir, p string) ([]string, error) {

	var (
		err    error
		pkg    *build.Package
		walker = &dependentsWalker{inSliceFn: inSlice}
	)

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps
		case 0:
			pkg, err = Pkg(p)
		case 1:
			walker.relpath, err = PkgPath(pkg)
		case 2:
			err = filepath.Walk(dir, walker.Walk)
		}
	}
	return walker.deps, err
}

// DependentsPrefix is like DependentsPrefix, but relPath is a package path, not a directory
func DependentsPrefix(dir, relPath string) ([]string, error) {
	walker := &dependentsWalker{inSliceFn: inSlicePrefix, relpath: relPath}
	err := filepath.Walk(dir, walker.Walk)
	return walker.deps, err
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
func GoPkginVersion(path string) (version [3]int, err error) {

	var (
		start int
		end   int
	)

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps

		case 0:
			if !InGoPkgin(path) {
				err = ErrNoGoPkginPath
			}
		case 1:
			start = strings.LastIndex(path, "v")
			if start == -1 || len(path) <= start+1 {
				err = ErrInvalidGoPkginPath
			}
		case 2:
			end = strings.LastIndex(path, "/")
			if end == -1 {
				err = ErrInvalidGoPkginPath
			}
		case 3:
			if end < start {
				end = len(path)
			}
			version, err = parseVersion(path[start:end])
		}
	}
	return
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
	// fmt.Printf("replacing %#v with %#v\n", gopkginBare, target)
	re, err := regexp.Compile(`"` + regexp.QuoteMeta(gopkginBare+".v") + `([0-9]+)(\.[0-9]+)*`)
	// fmt.Println(re)
	if err != nil {
		return nil, err
	}

	return re.ReplaceAll(in, []byte(`"`+target)), nil
}

func replaceGithub(github string, target string, in []byte) ([]byte, error) {
	// fmt.Printf("replacing %#v with %#v\n", github, target)
	re, err := regexp.Compile(`"` + regexp.QuoteMeta(github) + `("|/)`)
	// fmt.Println(re)
	if err != nil {
		return nil, err
	}

	return re.ReplaceAll(in, []byte(`"`+target+"$1")), nil
}

type replaceImport struct {
	filepath       string
	originalImport string
	targetImport   string
}

func (r replaceImport) replaceInFile(in []byte) ([]byte, error) {
	// fmt.Printf("replacing %#v with %#v\n", gopkginBare, target)
	re, err := regexp.Compile(`"` + regexp.QuoteMeta(r.originalImport) + `"`)
	// fmt.Println(re)
	if err != nil {
		return nil, err
	}

	return re.ReplaceAll(in, []byte(`"`+r.targetImport+`"`)), nil
}

func (r replaceImport) replace() error {
	var (
		err      error
		original []byte
		replaced []byte
	)

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps
		case 0:
			original, err = ioutil.ReadFile(r.filepath)
		case 1:
			replaced, err = r.replaceInFile(original)
		case 2:
			err = ioutil.WriteFile(r.filepath, replaced, 0644)
		}
	}
	return err
}

type replaceFile struct {
	filepath string
	gopkgin  string
	target   string
	pkgPath  string
}

func (r replaceFile) replace() error {

	var (
		err      error
		original []byte
		replaced []byte
	)

	// fmt.Printf("replace called for file: %#v %s => %s, %s => %s\n", r.filepath, r.gopkgin, r.target, r.pkgPath, r.target)

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps
		case 0:
			original, err = ioutil.ReadFile(r.filepath)
		case 1:
			replaced, err = replaceGopkgin(r.gopkgin, r.target, original)
		case 2:
			if r.pkgPath != "" {
				replaced, err = replaceGithub(r.pkgPath, r.target, replaced)
			}
		case 3:
			err = ioutil.WriteFile(r.filepath, replaced, 0644)
		}
	}
	return err
}

// ReplaceWithGithubPath takes a pkg pkgDir that is a GithubPath.
// It replaces inside every file inside every package beneath the given dir
// an string that references any  gopkg variant of the given package by the github variant
// It can be used for developement to be able to run the tests, switch back by calling
// ReplaceWithGopkginPath
func ReplaceWithGithubPath(pkgDir string) error {
	var (
		err     error
		pkg     *build.Package
		pkgPath string
		gopkgin string
		deps    []string
	)

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps
		case 0:
			pkg, err = Pkg(pkgDir)
		case 1:
			pkgPath, err = PkgPath(pkg)
		case 2:
			gopkgin, err = bareGoPkginPath(pkgPath)
		case 3:
			deps, err = DependentsPrefix(pkgDir, gopkgin)
		}
	}

	if err != nil {
		return err
	}

	repl := replaceFile{gopkgin: gopkgin, target: pkgPath}

	for _, dep := range deps {
		dpkg, err := build.Import(dep, pkg.SrcRoot, build.ImportMode(0))
		if err != nil {
			return err
		}

		files := append(dpkg.GoFiles, dpkg.TestGoFiles...)
		for _, file := range files {
			repl.filepath = filepath.Join(pkg.SrcRoot, dep, file)
			if err := repl.replace(); err != nil {
				return err
			}
		}
	}

	return nil
}

/*
type replaceImport struct {
	filepath       string
	originalImport string
	targetImport   string
}
*/

func ReplaceImport(pkgDir, original, target string) (err error) {
	var (
		pkg  *build.Package
		dpkg *build.Package
		deps []string
	)

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps
		case 0:
			pkg, err = Pkg(pkgDir)
		case 1:
			deps, err = Dependents(pkgDir, original)
		case 2:
			repl := replaceImport{
				originalImport: original,
				targetImport:   target,
			}

			for _, dep := range deps {
				dpkg, err = build.Import(dep, pkg.SrcRoot, build.ImportMode(0))
				if err != nil {
					return
				}

				files := append(dpkg.GoFiles, dpkg.TestGoFiles...)
				for _, file := range files {
					repl.filepath = filepath.Join(pkg.SrcRoot, dep, file)
					if err = repl.replace(); err != nil {
						return
					}
				}
			}
		}
	}

	return
}

// ReplaceWithGithubPath takes a pkg pkgDir that is a GithubPath.
// It replaces inside every file inside every package beneath the given dir
// an string that references any  github or gopkgin variant of the given package by the gopkg variant
// with the given version.
// It can be used to release a package after ReplaceWithGithubPath has been used or to update
// a version number
func ReplaceWithGopkginPath(pkgdir string, version [3]int) error {
	// fmt.Printf("ReplaceWithGopkginPath(%#v, %v)\n", pkgdir, version)
	// fmt.Printf("ReplaceWithGithubPath(%#v)\n", pkgDir)
	var (
		err     error
		pkg     *build.Package
		pkgPath string
		gopkgin string
		target  string
		deps    []string
	)

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps
		case 0:
			pkg, err = Pkg(pkgdir)
		case 1:
			pkgPath, err = PkgPath(pkg)
		case 2:
			gopkgin, err = bareGoPkginPath(pkgPath)
		case 3:
			target, err = GoPkginPath(pkgPath, version)
		case 4:
			deps, err = DependentsPrefix(pkgdir, pkgPath)
		case 5:
			var addDeps []string
			addDeps, err = DependentsPrefix(pkgdir, gopkgin)
			deps = append(deps, addDeps...)
		}
	}

	if err != nil {
		// fmt.Printf("err1: %s\n", err)
		return err
	}

	// fmt.Printf("deps: %#v\n", deps)

	repl := replaceFile{gopkgin: gopkgin, target: target, pkgPath: pkgPath}

	for _, dep := range deps {
		dpkg, err := build.Import(dep, pkg.SrcRoot, build.ImportMode(0))
		if err != nil {
			// fmt.Printf("err2: %s\n", err)
			return err
		}

		files := append(dpkg.GoFiles, dpkg.TestGoFiles...)

		for _, file := range files {
			repl.filepath = filepath.Join(pkg.SrcRoot, dep, file)
			// fmt.Printf("replace %s\n", repl.filepath)
			if err := repl.replace(); err != nil {
				return err
			}
		}
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

// VersionString returns something like v2.4 for [3]int{2,4,0}
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
	var v [][3]int

	for _, vers := range versions {
		vv, err := parseVersion(vers)

		if err != nil {
			continue
		}

		v = append(v, vv)
	}

	last := lastVersion(v...)

	if last[0] == 0 && last[1] == 0 && last[2] == 0 {
		return last, fmt.Errorf("can't find last version")
	}
	return last, nil
}

// gitTags returns the tags for the repo
func gitTags(tr *gitlib.Transaction) (tags []string, err error) {
	return tr.Tags()
}

// lastVersionFromTag returns the last version from the tag of the repository
func lastVersionFromTag(tr *gitlib.Transaction) ([3]int, error) {
	var v [3]int
	tags, err := gitTags(tr)

	if err != nil {
		return v, err
	}

	return LastVersion(tags...)
}

func setTag(tr *gitlib.Transaction, tag string) error {
	sha1, err := tr.GetSymbolicRef("HEAD")

	if err != nil {
		return err
	}

	return tr.Tag(tag, sha1, "")
}

func gitPushTags(tr *gitlib.Transaction) error {
	return tr.PushTags()
}

func SetNewMajor(dir string) ([3]int, error) {
	return setNewVersion(dir, 0)
}

func SetNewMinor(dir string) ([3]int, error) {
	return setNewVersion(dir, 1)
}

func SetNewPatch(dir string) ([3]int, error) {
	return setNewVersion(dir, 2)
}

func PushNewMajor(dir string) ([3]int, error) {
	return pushNewVersion(dir, 0)
}

func PushNewMinor(dir string) ([3]int, error) {
	return pushNewVersion(dir, 1)
}

func PushNewPatch(dir string) ([3]int, error) {
	return pushNewVersion(dir, 2)
}

type newVersion struct {
	version [3]int
	level   int
	dir     string
}

func (n *newVersion) push(tr *gitlib.Transaction) (err error) {
	// tr.Debug = true
	var (
		pkg         *build.Package
		pkgPath     string
		gopkginPath string
		getVersion  [3]int
	)

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps
		case 0:
			n.version, err = lastVersionFromTag(tr)
		case 1:
			n.setVersion()
			err = setTag(tr, VersionString(n.version))
		case 2:
			err = tr.PushTags()
		case 3:
			getVersion[0] = n.version[0]
			pkg, err = Pkg(n.dir)
		case 4:
			pkgPath, err = PkgPath(pkg)
		case 5:
			gopkginPath, err = GoPkginPath(pkgPath, getVersion)
		case 6:
			err = GoGetAndInstall(pkg.SrcRoot, gopkginPath)
		}
	}
	return
}

func (n *newVersion) setVersion() {
	switch n.level {
	case 0:
		n.version[0]++
		n.version[1] = 0
		n.version[2] = 0
	case 1:
		n.version[1]++
		n.version[2] = 0
	case 2:
		n.version[2]++
	}
}

func (n *newVersion) setVersionInFiles(tr *gitlib.Transaction) (err error) {

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps

		case 0:
			// fmt.Println("get last version")
			n.version, err = lastVersionFromTag(tr)
		case 1:
			// fmt.Println("ReplaceWithGopkginPath")
			n.setVersion()
			var replaceVersion [3]int
			replaceVersion[0] = n.version[0]
			err = ReplaceWithGopkginPath(n.dir, replaceVersion)
		}
	}
	return
}

// setNewVersion does the following:
// - gets the next version (level = 2 (patch) / 1 (minor)/ 0 (major))
// - replaces the references inside this package to this version
// - does a commit with the given message
// - tags this version
// - pushes the current branch to the default target, including the new tags
// - returns the new version and the first error
//
func setNewVersion(dir string, level int) ([3]int, error) {

	var (
		err error
		git *gitlib.Git
		n   = newVersion{level: level, dir: dir}
	)

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps
		case 0:
			if level != 0 && level != 1 && level != 2 {
				err = fmt.Errorf("invalid level: %d", level)
			}
		case 1:
			git, err = gitlib.NewGit(dir)
		case 2:
			err = git.Transaction(n.setVersionInFiles)
		}
	}
	return n.version, err
}

func pushNewVersion(dir string, level int) ([3]int, error) {

	var (
		err error
		git *gitlib.Git
		n   = newVersion{level: level, dir: dir}
	)

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps
		case 0:
			if level != 0 && level != 1 && level != 2 {
				err = fmt.Errorf("invalid level: %d", level)
			}
		case 1:
			git, err = gitlib.NewGit(dir)
			if DEBUG {
				git.Debug = true
			}
		case 2:
			err = git.Transaction(n.push)
		}
	}
	return n.version, err
}

func GoGetAndInstall(src, pkgPath string) error {
	os.RemoveAll(filepath.Join(src, pkgPath))

	cmd := exec.Command("go", "get", pkgPath)
	err := cmd.Run()

	if err != nil {
		return err
	}

	cmd = exec.Command("go", "install", pkgPath+"/...")
	return cmd.Run()
}
