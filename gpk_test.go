package gpk

import (
	"os"
	"path/filepath"
	"testing"
)

var wd string
var testpkg_t1 string
var testpkg_t2 string
var testpkg_t3 string

func init() {
	var err error
	wd, err = os.Getwd()
	if err != nil {
		panic(err)
	}
	testpkg_t1 = filepath.Join(wd, "testdata", "t1")
	testpkg_t2 = filepath.Join(wd, "testdata", "t2")
	testpkg_t3 = filepath.Join(wd, "testdata", "t3")
}

func TestImports(t *testing.T) {
	imps, err := Imports(testpkg_t1)
	if err != nil {
		t.Error(err)
	}

	if len(imps) != 3 {
		t.Errorf("len(imps) = %d // expected: %d", len(imps), 3)
	}

	// "fmt", "gopkg.in/go-on/builtin.v1", "runtime"

	if !inSlice(imps, "fmt") {
		t.Errorf("package %#v must be in %#v but is not", "fmt", imps)
	}

	if !inSlice(imps, "gopkg.in/go-on/builtin.v1") {
		t.Errorf("package %#v must be in %#v but is not", "gopkg.in/go-on/builtin.v1", imps)
	}

	if !inSlice(imps, "runtime") {
		t.Errorf("package %#v must be in %#v but is not", "runtime", imps)
	}

	// fmt.Println("GOROOT", runtime.GOROOT(), runtime.Version())
	// filepath.Join(runtime.GOROOT(), "src", p)
	// fmt.Printf("imports: %#v\n", imps)
}

func TestExtImports(t *testing.T) {
	imps, err := ExtImports(testpkg_t1)
	if err != nil {
		t.Error(err)
	}

	if len(imps) != 1 {
		t.Errorf("len(imps) = %d // expected: %d", len(imps), 1)
	}

	if !inSlice(imps, "gopkg.in/go-on/builtin.v1") {
		t.Errorf("package %#v must be in %#v but is not", "gopkg.in/go-on/builtin.v1", imps)
	}
}

func TestStdImports(t *testing.T) {
	imps, err := StdImports(testpkg_t1)
	if err != nil {
		t.Error(err)
	}

	if len(imps) != 2 {
		t.Errorf("len(imps) = %d // expected: %d", len(imps), 2)
	}

	if !inSlice(imps, "fmt") {
		t.Errorf("package %#v must be in %#v but is not", "fmt", imps)
	}

	if !inSlice(imps, "runtime") {
		t.Errorf("package %#v must be in %#v but is not", "runtime", imps)
	}
}

func TestDependents(t *testing.T) {
	deps, err := Dependents(wd, testpkg_t1)
	if err != nil {
		t.Error(err)
	}

	// fmt.Printf("%#v\n", deps)
	if len(deps) != 1 {
		t.Errorf("len(deps) = %d // expected: %d", len(deps), 1)
	}

	if !inSlice(deps, "github.com/go-on/gpk/testdata/t2") {
		t.Errorf("package %#v must be in %#v but is not", "github.com/go-on/gpk/testdata/t2", deps)
	}

}

func TestDependentsPrefix(t *testing.T) {
	deps, err := DependentsPrefix(wd, testpkg_t1)
	if err != nil {
		t.Error(err)
	}

	// fmt.Printf("%#v\n", deps)
	if len(deps) != 2 {
		t.Errorf("len(deps) = %d // expected: %d", len(deps), 1)
	}

	if !inSlice(deps, "github.com/go-on/gpk/testdata/t3") {
		t.Errorf("package %#v must be in %#v but is not", "github.com/go-on/gpk/testdata/t3", deps)
	}

	if !inSlice(deps, "github.com/go-on/gpk/testdata/t2") {
		t.Errorf("package %#v must be in %#v but is not", "github.com/go-on/gpk/testdata/t2", deps)
	}
}

func TestInGoPkgin(t *testing.T) {
	path := "gopkg.in/go-on/builtin.v1"
	is := InGoPkgin(path)

	if !is {
		t.Error("InGoPkgin(%#v) = false // expected true", path)
	}
}

func TestGoPkginVersion(t *testing.T) {

	tests := []struct {
		path    string
		version [3]int
		err     error
	}{
		{
			"gopkg.in/go-on/builtin.v1",
			[3]int{1, 0, 0},
			nil,
		},
		{
			"gopkg.in/go-on/builtin.v1.2",
			[3]int{1, 2, 0},
			nil,
		},
		{
			"gopkg.in/go-on/builtin.v1.2.3",
			[3]int{1, 2, 3},
			nil,
		},
		{
			"gopkg.in/go-on/builtin.1.2.3.4",
			[3]int{0, 0, 0},
			ErrInvalidGoPkginPath,
		},
		{
			"gopkg.in/go-on/builtin.v1.2.3/sql",
			[3]int{1, 2, 3},
			nil,
		},
		{
			"gopkg.in/go-on/builtin.v1/sql",
			[3]int{1, 0, 0},
			nil,
		},
	}

	for _, test := range tests {
		v, err := GoPkginVersion(test.path)
		if got, want := v, test.version; got != want || test.err != err {
			t.Errorf("GoPkginVersion(%#v) = %v, %#v; want %v, %#v", test.path, got, err, want, test.err)
		}
	}

}

func TestGoPkginPath(t *testing.T) {

	tests := []struct {
		path        string
		version     [3]int
		err         error
		gopkginpath string
	}{
		{"github.com/a/b", [3]int{1, 0, 0}, nil, "gopkg.in/a/b.v1"},
		{"github.com/a/b", [3]int{1, 2, 0}, nil, "gopkg.in/a/b.v1.2"},
		{"github.com/a/b", [3]int{1, 2, 3}, nil, "gopkg.in/a/b.v1.2.3"},
		{"github.com/a/b", [3]int{0, 2, 0}, nil, "gopkg.in/a/b.v0.2"},
		{"github.com/a/b", [3]int{0, 2, 1}, nil, "gopkg.in/a/b.v0.2.1"},
		{"github.com/a/b", [3]int{0, 0, 1}, nil, "gopkg.in/a/b.v0.0.1"},
		{"xgithub.com/a/b", [3]int{1, 2, 3}, ErrInvalidGithubPath, ""},
		{"google.com/a/b", [3]int{1, 2, 3}, ErrInvalidGithubPath, ""},
		{"github.com", [3]int{1, 2, 3}, ErrInvalidGithubPath, ""},
	}

	for _, test := range tests {
		p, err := GoPkginPath(test.path, test.version)
		if got, want := p, test.gopkginpath; got != want || err != test.err {
			t.Errorf("GoPkginPath(%#v, %v) = %#v, %v; want %#v, %v", test.path, test.version, got, err, want, test.err)
		}
	}

}

func TestGithubPath(t *testing.T) {

	tests := []struct {
		path     string
		err      error
		expected string
	}{
		{"gopkg.in/a/b.v1", nil, "github.com/a/b"},
		{"gopkg.in/a/b.v1.2.3", nil, "github.com/a/b"},
		{"gopkg.in/a/b.v1.2", nil, "github.com/a/b"},
		{"gopkg.in/a/b.v0.2", nil, "github.com/a/b"},
		{"xgopkg.in/a/b", ErrInvalidGoPkginPath, ""},
		{"google.com/a/b", ErrInvalidGoPkginPath, ""},
		{"gopkg.in", ErrInvalidGoPkginPath, ""},
	}

	for _, test := range tests {
		p, err := GithubPath(test.path)
		if got, want := p, test.expected; got != want || err != test.err {
			t.Errorf("GithubPath(%#v) = %#v, %v; want %#v, %v", test.path, got, err, want, test.err)
		}
	}

}

func TestinSlicePrefix(t *testing.T) {

	tests := []struct {
		slice    []string
		prefix   string
		expected bool
	}{
		{[]string{"a/b/c"}, "a/b", true},
		{[]string{"a/b/c"}, "b", false},
		{[]string{"a/b/c"}, "a/b/c", true},
	}

	for _, test := range tests {

		if got, want := inSlicePrefix(test.slice, test.prefix), test.expected; got != want {
			t.Errorf("inSlicePrefix(%#v, %#v) = %v; want %v", test.slice, test.prefix, got, want)
		}
	}

}

/*
func TestReplaceWithGithubPath(t *testing.T) {
	p := filepath.Join(wd, "..", "builtin")
	p, _ = filepath.Abs(p)
	err := ReplaceWithGithubPath(p)

	if err != nil {
		t.Fatal(err)
	}
}

func TestReplaceWithGopkginPath(t *testing.T) {
	p := filepath.Join(wd, "..", "builtin")
	p, _ = filepath.Abs(p)
	err := ReplaceWithGopkginPath(p, [3]int{3, 0, 0})

	if err != nil {
		t.Fatal(err)
	}
}

*/

func TestReplaceGopkgin(t *testing.T) {
	// replaceGopkgin(gopkgin string, in []byte) ([]byte, error)

	tests := []struct {
		file     string
		pkg      string
		expected string
	}{
		{
			"asbd \"gopkg.in/a/b/c.v1.2.3\"\n\"gopkg.in/a/b/c.v1.2.3\"",
			`github.com/a/b/c`,
			"asbd \"github.com/a/b/c\"\n\"github.com/a/b/c\"",
		},
		{
			"asbd \"gopkg.in/a/b.v1.2.3/c/d\"\n\"gopkg.in/a/b.v1.2.3/c/d\"",
			`github.com/a/b`,
			"asbd \"github.com/a/b/c/d\"\n\"github.com/a/b/c/d\"",
		},
		{
			"asbd \"gopkg.in/a/b.v1/c/d\"\n\"gopkg.in/a/b.v1.2.3/c/d\"",
			`github.com/a/b`,
			"asbd \"github.com/a/b/c/d\"\n\"github.com/a/b/c/d\"",
		},
		{
			"asbd \"gopkg.in/a/b.v1/c/d\"\n\"gopkg.in/a/b.v1.2/d\"",
			`github.com/a/b`,
			"asbd \"github.com/a/b/c/d\"\n\"github.com/a/b/d\"",
		},
	}

	for _, test := range tests {
		bare, err0 := bareGoPkginPath(test.pkg)
		if err0 != nil {
			t.Fatal(err0)
		}
		bt, err1 := replaceGopkgin(bare, test.pkg, []byte(test.file))
		if err1 != nil {
			t.Fatal(err1)
		}
		if got, want := string(bt), test.expected; got != want {
			t.Errorf("replaceGopkgin(%#v, %#v, %#v) = %#v; want %#v", bare, test.pkg, test.file, got, want)
		}
	}

}

func TestReplaceGithub(t *testing.T) {
	// replaceGopkgin(gopkgin string, in []byte) ([]byte, error)

	tests := []struct {
		file     string
		search   string
		target   string
		expected string
	}{
		{
			"asbd \"github.com/a/b/c\"\n\"github.com/a/b/c\"",
			`github.com/a/b/c`,
			"gopkg.in/a/b/c.v1.2.3",
			"asbd \"gopkg.in/a/b/c.v1.2.3\"\n\"gopkg.in/a/b/c.v1.2.3\"",
		},
		{
			"asbd \"github.com/a/b/c\"\n\"github.com/a/b/d\"",
			`github.com/a/b`,
			"gopkg.in/a/b.v1.2.3",
			"asbd \"gopkg.in/a/b.v1.2.3/c\"\n\"gopkg.in/a/b.v1.2.3/d\"",
		},
	}

	for _, test := range tests {
		bt, err1 := replaceGithub(test.search, test.target, []byte(test.file))
		if err1 != nil {
			t.Fatal(err1)
		}
		if got, want := string(bt), test.expected; got != want {
			t.Errorf("replaceGithub(%#v,%#v, %#v) = %#v; want %#v", test.search, test.target, test.file, got, want)
		}
	}

}

func TestlastVersion(t *testing.T) {

	tests := []struct {
		versions [][3]int
		last     [3]int
	}{
		{[][3]int{{1, 2, 0}, {1, 4, 0}}, [3]int{1, 4, 0}},
		{[][3]int{{3, 2, 0}, {1, 4, 0}}, [3]int{3, 2, 0}},
		{[][3]int{{1, 2, 1}, {1, 2, 0}}, [3]int{1, 2, 1}},
		{[][3]int{{1, 0, 0}, {2, 0, 0}}, [3]int{2, 0, 0}},
		{[][3]int{{1, 0, 0}, {2, 0, 0}, {1, 2, 0}}, [3]int{2, 0, 0}},
		{[][3]int{{3, 0, 0}, {2, 0, 0}, {3, 2, 0}}, [3]int{3, 2, 0}},
	}

	for _, test := range tests {

		if got, want := lastVersion(test.versions...), test.last; got != want {
			t.Errorf("lastVersion(%v...) = %v; want %v", test.versions, got, want)
		}
	}

}

// replaceGithub(pkgPath, target, out)

func TestLastVersion(t *testing.T) {

	tests := []struct {
		versions []string
		last     [3]int
	}{
		{[]string{"v1.4", "v1"}, [3]int{1, 4, 0}},
		{[]string{"v2", "v1.4", "v1"}, [3]int{2, 0, 0}},
		{[]string{"v2", "v1.4", "v1", "v2.0.1"}, [3]int{2, 0, 1}},
		{[]string{"v2", "v1.4", "v1", "v2.3"}, [3]int{2, 3, 0}},
	}

	for _, test := range tests {
		vers, err := LastVersion(test.versions...)

		if err != nil {
			t.Error(err)
		}

		if got, want := vers, test.last; got != want {
			t.Errorf("LastVersion(%#v...) = %v; want %v", test.versions, got, want)
		}
	}

}

func TestVersionString(t *testing.T) {
	tests := []struct {
		version  [3]int
		expected string
	}{
		{[3]int{1, 4, 0}, "v1.4"},
		{[3]int{1, 4, 2}, "v1.4.2"},
		{[3]int{3, 0, 0}, "v3"},
	}

	for _, test := range tests {

		if got, want := VersionString(test.version), test.expected; got != want {
			t.Errorf("VersionString(%v) = %#v; want %#v", test.version, got, want)
		}
	}
}

func TestSetNewMinor(t *testing.T) {
	p := filepath.Join(wd, "..", "builtin")
	p, _ = filepath.Abs(p)
	SetNewMinor(p, "just a test")
}
