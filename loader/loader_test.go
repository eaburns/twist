package loader

import (
	"go/ast"
	"reflect"
	"strings"
	"testing"

	"code.google.com/p/go.tools/go/ssa"
)

// TestNewProgramString just tests some simple cases
// for the presence of an error or not.
func TestNewProgramString(t *testing.T) {
	tests := []struct {
		src string
		ok  bool
	}{
		{`package main`, true},
		{`package main; import "fmt"; func main() { fmt.Println("Hi"); }`, true},
		{`package main; func main() { fmt.Println("Hi"); }`, false},
		{`package foo; var bar int = "string"`, false},
	}
	for _, test := range tests {
		_, err := NewProgramString(test.src)
		switch {
		case err != nil && test.ok:
			t.Errorf("NewProgramString(%s), unexpected error: %s", test.src, err)
		case err == nil && !test.ok:
			t.Errorf("NewProgramString(%s), expected an error", test.src)
		}
	}
}

// TestProgramFunction tests Program.Function
func TestProgramFunction(t *testing.T) {
	src := `
		package test
		func main() { }
		func foo(int) int { return 6; }`
	tests := []struct {
		pname, fname string
		ok           bool
	}{
		{"test", "main", true},
		{"test", "foo", true},
		{"main", "main", false},
		{"main", "foo", false},
		{"test", "bar", false},
	}
	for _, test := range tests {
		p, err := NewProgramString(src)
		if err != nil {
			panic(err)
		}
		_, err = p.Function(test.pname, test.fname)
		switch {
		case err != nil && test.ok:
			t.Errorf("Program.Function(%s, %s), unexpected error: %s", test.pname, test.fname, err)
		case err == nil && !test.ok:
			t.Errorf("Program.Function(%s, %s), expected an error", test.pname, test.fname)
		}
	}

}

// TestProgramNode tests that
// the Program.Node map associates
// ssa.Instructions with the desired ast.Nodes.
func TestProgramNode(t *testing.T) {
	// For each comment, the test ensures that
	// there is an instruction of the appropriate type
	// mapped to a node of the appropriate type
	// which is associated with the comment.
	// Annotations are type names of the form:
	// 	ssa.Instruction:ast.Node
	src := `
		package main
		var ch = make(chan bool)
		func main() {
			// If:ForStmt
			for i := 0; i < 5; i++ /* Jump:IncDecStmt */ {
				// If:IfStmt
				if /* UnOp:ParenExpr*/ (<- ch /* UnOp:Ident */)  {
					panic("Whoa!") // Panic:ExprStmt
				} else {
					/* Send:SendStmt */ ch /* UnOp:Ident */ <- true
				}
			}
		}`
	prog, err := NewProgramString(src)
	if err != nil {
		panic(err)
	}

	// Gather type annotations from comments.
	instrType := make(map[*ast.CommentGroup]string)
	nodeType := make(map[*ast.CommentGroup]string)
	for _, c := range prog.Comments {
		if len(c) != 1 {
			panic("annotations must only contain one comment")
		}
		fields := strings.Split(c[0].Text(), ":")
		instrType[c[0]] = strings.TrimSpace(fields[0])
		nodeType[c[0]] = strings.TrimSpace(fields[1])
	}

	f, err := prog.Function("main", "main")
	if err != nil {
		panic(err)
	}
	for _, blk := range f.Blocks {
		for _, p := range blk.Instrs {
			if _, ok := p.(*ssa.DebugRef); ok { // skip debug comments
				continue
			}
			n, ok := prog.Node[p]
			if !ok { // no node
				continue
			}
			cs, ok := prog.Comments[n]
			if !ok { // no comment
				continue
			}
			if len(cs) != 1 {
				panic("annotations must only contain one comment")
			}
			c := cs[0]

			// Found an instruction with a node and a comment.
			// Get the desired types.
			wantIt, ok := instrType[c]
			if !ok {
				t.Errorf("unexpected comment at: %s", prog.Position(n.Pos()))
				continue
			}
			wantNt, ok := nodeType[c]
			if !ok {
				panic("instrType and nodeType out of sync")
			}
			delete(instrType, c)
			delete(nodeType, c)

			// Check against the types we got.
			it := reflect.ValueOf(p).Elem().Type().Name()
			nt := reflect.ValueOf(n).Elem().Type().Name()
			if it != wantIt {
				t.Errorf("got instruction type %s, want %s", it, wantIt)
			}
			if nt != wantNt {
				t.Errorf("got node type %s, want %s", nt, wantNt)
			}
		}
	}
	for c := range instrType {
		t.Errorf("no instruction for %s", c.Text())
	}
}
