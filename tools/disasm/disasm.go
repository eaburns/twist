// Disasm disassembles the Go source file.
// Does not support source files with import statements.
package main

import (
	"log"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"go/ast"

	"github.com/eaburns/twist/loader"
	"golang.org/x/tools/go/ssa"
)

func main() {
	prog, err := loader.NewProgram(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	for _, pkg := range prog.SSAProgram.AllPackages() {
		var mems members
		for _, m := range pkg.Members {
			mems = append(mems, m)
		}
		sort.Sort(mems)
		for _, m := range mems {
			f, ok := m.(*ssa.Function)
			if !ok {
				continue
			}
			printFunc(prog, f)
		}
	}
}

type members []ssa.Member

func (m members) Len() int           { return len(m) }
func (m members) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m members) Less(i, j int) bool { return m[i].Name() < m[j].Name() }

func printFunc(prog *loader.Program, f *ssa.Function) {
	os.Stdout.WriteString(f.Name() + "\n")
	for _, blk := range f.DomPreorder() {
		os.Stdout.WriteString("\t" + strconv.Itoa(blk.Index) + ":\n")
		for _, p := range blk.Instrs {
			if _, ok := p.(*ssa.DebugRef); ok {
				continue
			}
			os.Stdout.WriteString("\t\t")
			if v, ok := p.(ssa.Value); ok {
				os.Stdout.WriteString(v.Name() + " = ")
			}
			t := reflect.ValueOf(p).Elem().Type().Name()
			os.Stdout.WriteString(p.String() + " (" + t + ")")

			if n, ok := prog.Node[p]; ok {
				t := reflect.ValueOf(n).Elem().Type().Name()
				s := prog.Position(n.Pos()).String()
				os.Stdout.WriteString(" {" + t + "} [" + s + "]")
				if c := comment(prog, n); c != "" {
					os.Stdout.WriteString("\n" + c)
				}
			}
			os.Stdout.WriteString("\n")
		}
	}
}

func comment(prog *loader.Program, n ast.Node) string {
	var s string
	for _, g := range prog.Comments[n] {
		s += "\t\t\t// " + g.Text()
	}
	return strings.TrimRight(s, "\n")
}
