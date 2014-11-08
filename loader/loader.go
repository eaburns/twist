// Package loader contains functions for loading Go programs
// and wrapper types for go.tools/go/loader and go.tools/go/ssa types.
package loader

import (
	"errors"
	"go/ast"
	"go/token"
	"io/ioutil"
	"os"

	goloader "code.google.com/p/go.tools/go/loader"
	"code.google.com/p/go.tools/go/ssa"
)

// A Program represents a loaded program.
// It contains:
// token locations, the AST, types, and an SSA representation.
type Program struct {
	FileSet       *token.FileSet
	File          *ast.File
	LoaderProgram *goloader.Program
	SSAProgram    *ssa.Program
}

// NewProgram loads and returns a program
// by reading Go source from the file at the given path.
func NewProgram(path string) (*Program, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	src, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return newProgram(path, string(src))
}

// NewProgramString loads and returns a program from a Go source string.
func NewProgramString(src string) (*Program, error) {
	return newProgram("", src)
}

func newProgram(path, src string) (*Program, error) {
	var p Program
	p.FileSet = token.NewFileSet()
	file := p.FileSet.AddFile(path, p.FileSet.Base(), len(src))
	file.SetLinesForContent([]byte(src))

	var err error
	cfg := goloader.Config{Fset: p.FileSet, SourceImports: false}
	if p.File, err = cfg.ParseFile(path, src); err != nil {
		return nil, err
	}
	cfg.CreatePkgs = []goloader.CreatePkg{{Files: []*ast.File{p.File}}}
	if p.LoaderProgram, err = cfg.Load(); err != nil {
		return nil, err
	}

	p.SSAProgram = ssa.Create(p.LoaderProgram, 0)
	p.SSAProgram.BuildAll()
	return &p, nil
}

// Function retuns the *ssa.Function for a fully-qualified function,
// specified by its package name and function name.
func (p *Program) Function(pname, fname string) (*ssa.Function, error) {
	pkg, err := p.Package(pname)
	if err != nil {
		return nil, err
	}
	mem, ok := pkg.Members[fname]
	if !ok {
		return nil, errors.New("function " + fname + " not found")
	}
	f, ok := mem.(*ssa.Function)
	if !ok {
		return nil, errors.New("member " + fname + " is not a function")
	}
	return f, nil
}

// Package retuns the named *ssa.Package,
func (p *Program) Package(pname string) (*ssa.Package, error) {
	for _, pkg := range p.SSAProgram.AllPackages() {
		if pkg.Object.Name() == pname {
			return pkg, nil
		}
	}
	return nil, errors.New("package " + pname + " not found")
}
