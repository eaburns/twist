// Package loader contains functions for loading Go programs
// and wrapper types for go.tools/go/loader and go.tools/go/ssa types.
package loader

import (
	"errors"
	"go/ast"
	"go/parser"
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
	SSAProgram *ssa.Program
	// Node maps each ssa.Value and ssa.Instruction to an ast.Node.
	Node map[interface{}]ast.Node
	// Comments maps ast.Nodes to their associated comments.
	Comments ast.CommentMap
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
	files := token.NewFileSet()
	file := files.AddFile(path, files.Base(), len(src))
	file.SetLinesForContent([]byte(src))

	var err error
	cfg := goloader.Config{
		Fset:          files,
		ParserMode:    parser.ParseComments,
		SourceImports: false,
	}
	root, err := cfg.ParseFile(path, src)
	if err != nil {
		return nil, err
	}
	cfg.CreatePkgs = []goloader.CreatePkg{{Files: []*ast.File{root}}}
	lprog, err := cfg.Load()
	if err != nil {
		return nil, err
	}

	p.SSAProgram = ssa.Create(lprog, 0)
	for _, pkg := range p.SSAProgram.AllPackages() {
		pkg.SetDebugMode(true)
	}
	p.SSAProgram.BuildAll()

	buildNodeMap(&p, root)
	p.Comments = ast.NewCommentMap(files, root, root.Comments)

	return &p, nil
}

func buildNodeMap(prog *Program, root ast.Node) {
	prog.Node = make(map[interface{}]ast.Node)
	var ss []ast.Stmt

	// Walk populates prog.Node for ssa.Values and ss.
	ast.Walk(visitor{prog, &ss, root}, root)

	for _, f := range prog.functions() {
		for _, b := range f.Blocks {
			for i, p := range b.Instrs {
				// ast.Walk already found nodes for ssa.Values.
				if _, ok := p.(ssa.Value); ok {
					continue
				}

				// For Jumps, use the node of the previous instruction.
				if _, ok := p.(*ssa.Jump); ok && i > 0 {
					if n := prog.Node[b.Instrs[i-1]]; n != nil {
						prog.Node[p] = n
						continue
					}
				}

				// Find the latest-starting, containing statement.
				pos := p.Pos()
				if f, ok := p.(*ssa.If); ok {
					// Ifs don't have a position, use their condition's.
					pos = f.Cond.Pos()
				}
				n := ast.Node(findStmt(ss, pos))
				if n != nil {
					prog.Node[p] = n
					continue
				}
			}
		}
	}
}

// FindStmt returns the latest-starting statement
// containing the given position.
func findStmt(ss []ast.Stmt, p token.Pos) ast.Stmt {
	n := -1
	for i, s := range ss {
		if s.Pos() > p || s.End() <= p {
			continue
		}
		if n < 0 || s.Pos() > ss[n].Pos() {
			n = i
		}
	}
	if n < 0 {
		return nil
	}
	return ss[n]
}

type visitor struct {
	*Program
	stmts *[]ast.Stmt
	node  ast.Node
}

// Visit implements the ast.Visitor interface.
// Builds Program.Node by to each ssa.Value
// the shallowest ast.Expr associated with the value.
// It also populates stmts with all ast.Nodes
// that are also ast.Stmts.
func (v visitor) Visit(node ast.Node) ast.Visitor {
	if node != nil {
		return visitor{v.Program, v.stmts, node}
	}

	node = v.node
	if expr, ok := node.(ast.Expr); ok {
		if val := v.valueForExpr(expr); val != nil {
			v.Program.Node[val] = node
		}
	}
	if s, ok := node.(ast.Stmt); ok {
		*v.stmts = append(*v.stmts, s)
	}
	return nil
}

func (v visitor) valueForExpr(expr ast.Expr) ssa.Value {
	for _, f := range v.Program.functions() {
		if v, _ := f.ValueForExpr(expr); v != nil {
			return v
		}
	}
	return nil
}

func (p *Program) functions() []*ssa.Function {
	var fs []*ssa.Function
	for _, pkg := range p.SSAProgram.AllPackages() {
		for _, mem := range pkg.Members {
			if f, ok := mem.(*ssa.Function); ok {
				fs = append(fs, f)
			}
		}
	}
	return fs
}

// Position returns the token.Position for a token.Pos in the loaded program.
func (p *Program) Position(pos token.Pos) token.Position {
	return p.SSAProgram.Fset.Position(pos)
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
