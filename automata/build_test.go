package automata

import (
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"go/ast"

	"github.com/eaburns/twist/loader"
)

func TestBuild(t *testing.T) {
	tests := []string{
		`package main; func main() { panic("Panic, World!") /* 0 */ }`,

		`package main
		var ch = make(chan bool)
		func main() {
			ch /* 0→1 */ <- true // 1→2
			return // 2 
		}`,

		`package main
		var ch = make(chan bool)
		func main() {
			/* 2→3,4 */ if /* 1→2 */ (<-ch /* 0→1 */) {
				panic("whoa!") // 3
			}
			return // 4
		}`,

		`package main
		var ch = make(chan bool)
		func main() {
			/* 2→3,6 */ if /* 1→2 */ (<-ch /* 0→1 */) {
				ch /* 3→4 */ <- true // 4→5
			} else {
				panic("Panic, World!") // 6
			}
			return // 5
		}`,

		`package main
		var ch = make(chan bool)
		func main() {
			for {
				/* 2→0,3 */ if /* 1→2 */ (<-ch /* 0→1 */) {
					break
				}
			}
			return // 3
		}`,
	}
	for _, test := range tests {
		p, err := loader.NewProgramString(test)
		if err != nil {
			panic(err)
		}
		f, err := p.Function("main", "main")
		if err != nil {
			panic(err)
		}
		a, err := New(f)
		if err != nil {
			panic(err)
		}
		if len(a.States) != len(p.Comments) {
			t.Errorf("New(%s), got %d states, want %d:\n%s", test, len(a.States), len(p.Comments), a.String())
		}
		for i, s := range a.States {
			c := annot(p, s)
			if c == "" {
				t.Errorf("New(%s), state %d has no annotation:\n%s", test, i, a.String())
				continue
			}
			// TODO(eaburns): This assumes ssa will never re-order basic blocks.
			n, ms := parseAnnot(c)
			if n != i {
				t.Errorf("New(%s), got state %d, want %d:\n%s", test, i, n, a.String())
			}
			next := succs(s)
			sort.Ints(next)
			sort.Ints(ms)
			if !reflect.DeepEqual(next, ms) {
				t.Errorf("New(%s), got succs %v, want %v:\n%s", test, next, ms, a.String())
			}
		}
	}
}

// Succs returns the successors of the state.
func succs(s *State) []int {
	switch p := s.prog[len(s.prog)-1].(type) {
	case *Jump:
		return []int{p.To.N}
	case *If:
		return []int{p.True.N, p.False.N}
	default:
		return []int{}
	}
}

// Annot returns
// the annotation for the state.
// The empty string is returned if
// no comment is found.
func annot(prog *loader.Program, s *State) string {
	var nd ast.Node
	for i := len(s.prog) - 1; i >= 0; i-- {
		p := s.prog[i]
		if f, ok := p.(*If); ok {
			p = f.If
		}
		var ok bool
		if nd, ok = prog.Node[p]; ok {
			break
		}
	}
	g, ok := prog.Comments[nd]
	if !ok {
		return ""
	}
	if len(g) != 1 {
		panic("comment groups must be singletons")
	}
	return g[0].Text()
}

// ParseAnnot parses a node annotation,
// returning the node number and its transitions.
// If the node has no transitions,
// the returned slice has len 0, but is non-nil.
// Annotations are of two forms:
// 	/* # */ — a terminal node
// 	/* #→#{,#} */ — a node with transitions
func parseAnnot(str string) (int, []int) {
	fs := strings.Split(str, "→")
	if len(fs) > 2 {
		panic("malformed node annotation: " + str)
	}

	nstr := strings.TrimSpace(fs[0])
	n, err := strconv.Atoi(nstr)
	if err != nil {
		panic("failed to parse node number: " + nstr)
	}
	if len(fs) == 1 {
		return n, make([]int, 0)
	}

	var ms []int
	for _, mstr := range strings.Split(fs[1], ",") {
		mstr = strings.TrimSpace(mstr)
		m, err := strconv.Atoi(mstr)
		if err != nil {
			panic("failed to parse node number: " + mstr)
		}
		ms = append(ms, m)
	}
	return n, ms
}
