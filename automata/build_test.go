package automata

import (
	"testing"

	"github.com/eaburns/twist/loader"
)

// TestCounts verifies the number of states and instructions
// in the automata built from a main function.
//
// TODO(eaburns): This is a horrible way to test automata construction.
// It's impossible to verify without having the disassembly available.
// Also, it makes for a weak test; it just tests the counts of instructions,
// but it does not actually test that they are the correct instructions.
func TestCounts(t *testing.T) {
	tests := []struct {
		src   string
		sizes []int
	}{
		{`package main; func main() {}`, []int{1}},
		{
			`	package main
				var ch = make(chan bool)
				func main() { ch <- true }`,
			// UnOp,Jump
			// Send, Jump
			// Return
			[]int{2, 2, 1},
		},
		{`package main; func main() {}`, []int{1}},
		{
			`	package main
				var ch = make(chan bool)
				func main() { panic("Panic, World!") }`,
			// MakeInterface(from string), Panic
			[]int{2},
		},
		{
			`	package main
				var ch = make(chan bool)
				func main() { <-ch }`,
			// UnOp(*), Jump
			// Send, Jump
			// Return
			[]int{2, 2, 1},
		},
		{
			`	package main
				var ch = make(chan bool)
				func main() {
					ch <- true
					<-ch
				}`,
			// UnOp(*), Jump
			// Send, Jump
			// UnOp(*), Jump
			// Send, Jump
			// Return
			[]int{2, 2, 2, 2, 1},
		},
		{
			`	package main
				var ch = make(chan bool)
				func main() {
					if <-ch {
						ch <- true
					} else {
						panic("Panic, World!")
					}
				}`,
			// UnOp(*), Jump
			// UnOp(recv), Jump
			// If
			// UnOp(*), Jump
			// Send, Jump
			// Return
			// MakeInterface, Panic
			[]int{2, 2, 1, 2, 2, 1, 2},
		},
		{
			`	package main
				var ch = make(chan bool)
				func main() {
					for {
						if <-ch {
							ch <- true
						}
					}
				}`,
			// UnOp(*), Jump
			// UnOp(recv), Jump
			// If
			// UnOp(*), Jump
			// Send, Jump
			[]int{2, 2, 1, 2, 2},
		},
	}

	for _, test := range tests {
		p, err := loader.NewProgramString(test.src)
		if err != nil {
			panic(err)
		}
		f, err := p.Function("main", "main")
		if err != nil {
			panic(err)
		}
		a, err := New(f)
		if err != nil {
			t.Fatalf("New(%s): unexpected error: %v", test.src, err)
		}
		if n := len(a.States); n != len(test.sizes) {
			t.Fatalf("New(%s), got %d states, want %d:\n%s",
				test.src, n, len(test.sizes), a.String())
		}
		for i, m := range test.sizes {
			if n := len(a.States[i].prog); n != m {
				t.Errorf("New(%s)[%d}, got %d instruction, want %d:\n%s",
					test.src, i, n, m, a.String())
			}
		}
	}
}
