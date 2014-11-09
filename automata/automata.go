// Package automata defines
// the representation of finite automata,
// which represent processes of a Twist model.
package automata

import (
	"bytes"
	"strconv"

	"golang.org/x/tools/go/ssa"
)

// An Automata is a finite state machine
// representing a process of the model.
type Automata struct {
	// States is a slice of all states in the automata.
	States []*State
}

// A State is a single state of the finite automata
// representing a process of the model.
type State struct {
	// N is a small integer unique to the given state.
	// It is the index of the state
	// in the finite automata's States slice.
	N int

	// Prog is a sequence of instructions
	// executed atomically by the given state.
	// The last instruction
	// is always a control flow instruction
	// (If or Jump)
	// specifying the next state.
	prog []Instruction
}

// An Instruction is an SSA instruction
// that computes a value
// or changes the state of the model.
//
// The Instruction type consists of ssa.Instructions,
// except that the control flow instructions,
// ssa.If and ssa.Jump,
// are wrapped with If and Jump types.
type Instruction interface {
	// String returns the string representation
	// of the instruction in disassembly form.
	String() string
}

// An If is an Instruction that wraps an ssa.If,
// adding the successor states of the automata.
type If struct {
	// If is the instruction wrapped by this If.
	*ssa.If
	// True and False are the successor states,
	// corresponding to each truth value
	// of the condition.
	True, False *State
}

func (n *If) String() string {
	t := strconv.Itoa(n.True.N)
	f := strconv.Itoa(n.False.N)
	return "if " + n.If.Cond.Name() + " " + t + " else " + f
}

// A Jump is an Instruction that wraps an ssa.Jump,
// adding the successor state of the automata.
type Jump struct {
	// Jump is the instruction wrapped by this Jump.
	*ssa.Jump
	// To is the successor state.
	To *State
}

func (n *Jump) String() string {
	return "jump " + strconv.Itoa(n.To.N)
}

// String returns a multi-line, SSA disassembly
// representation of the automata.
func (a *Automata) String() string {
	b := bytes.NewBuffer(nil)
	for _, s := range a.States {
		b.WriteString(strconv.Itoa(s.N) + ":\n")
		for _, p := range s.prog {
			b.WriteString("\t")
			if v, ok := p.(ssa.Value); ok {
				b.WriteString(v.Name() + " = ")
			}
			b.WriteString(p.String() + "\n")
		}
	}
	return b.String()
}
