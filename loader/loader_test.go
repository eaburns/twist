package loader

import "testing"

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
