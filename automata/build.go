package automata

import (
	"errors"
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// New returns a new, empty Automata.
func New(f *ssa.Function) (Automata, error) {
	b := newBuilder()
	if err := b.function(f); err != nil {
		return Automata{}, err
	}
	return b.build(), nil
}

type builder struct {
	states  []*State
	bblocks map[*ssa.BasicBlock]*State
}

func newBuilder() *builder {
	return &builder{bblocks: make(map[*ssa.BasicBlock]*State)}
}

func (b *builder) build() Automata {
	subst := make(map[*State]*State)

	var states []*State
	for _, s := range b.states {
		// Remove single-instruction jump states.
		if j, ok := s.prog[0].(*Jump); ok && len(s.prog) == 1 {
			subst[s] = j.To
			continue
		}
		states = append(states, s)
	}

	// Re-number states and substitute removed states.
	for i, s := range states {
		s.N = i
		for _, p := range s.prog {
			switch p := p.(type) {
			case *Jump:
				if t, ok := subst[p.To]; ok {
					p.To = t
				}
			case *If:
				if t, ok := subst[p.True]; ok {
					p.True = t
				}
				if t, ok := subst[p.False]; ok {
					p.False = t
				}
			}
		}
	}
	return Automata{states}
}

func (b *builder) function(f *ssa.Function) error {
	for _, blk := range f.DomPreorder() {
		_, err := b.basicBlock(blk)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) newState() *State {
	n := len(b.states)
	s := &State{N: n}
	b.states = append(b.states, s)
	return s
}

func (b *builder) basicBlock(blk *ssa.BasicBlock) (*State, error) {
	if s, ok := b.bblocks[blk]; ok {
		return s, nil
	}

	s0 := b.newState()
	b.bblocks[blk] = s0

	s := s0
	for _, p := range blk.Instrs {
		if s == nil {
			panic("instructions after return or unconditional jump")
		}
		var err error
		s, err = b.instruction(s, p)
		if err != nil {
			return nil, err
		}
	}
	return s0, nil
}

func (b *builder) instruction(s *State, p ssa.Instruction) (*State, error) {
	switch p := p.(type) {
	case *ssa.DebugRef:
		break

	case *ssa.If:
		ttrue, err := b.basicBlock(p.Block().Succs[0])
		if err != nil {
			return nil, err
		}
		tfalse, err := b.basicBlock(p.Block().Succs[1])
		if err != nil {
			return nil, err
		}
		s.prog = append(s.prog, &If{p, ttrue, tfalse})
		s = nil

	case *ssa.Jump:
		t, err := b.basicBlock(p.Block().Succs[0])
		if err != nil {
			return nil, err
		}
		s.prog = append(s.prog, &Jump{p, t})
		s = nil

	case *ssa.MakeInterface:
		s.prog = append(s.prog, p)

	case *ssa.Panic:
		s.prog = append(s.prog, p)
		s = nil

	case *ssa.Return:
		s.prog = append(s.prog, p)
		s = nil

	case *ssa.Send:
		snext := b.newState()
		s.prog = append(s.prog, p, &Jump{To: snext})
		s = snext

	case *ssa.UnOp:
		switch p.Op {
		case token.ARROW: // channel send
			snext := b.newState()
			s.prog = append(s.prog, p, &Jump{To: snext})
			s = snext
		case token.MUL:
			// Dereferencing is non-atomic.
			// Eventually some dereferences will be atomic.
			// For example, dereferencing globals that are never assigned.
			snext := b.newState()
			s.prog = append(s.prog, p, &Jump{To: snext})
			s = snext
		case token.NOT, token.SUB, token.XOR:
			s.prog = append(s.prog, p)
		default:
			panic("impossible UnOp")
		}

	default:
		return nil, errors.New("unsupported instruction: " + p.String())
	}

	return s, nil
}
