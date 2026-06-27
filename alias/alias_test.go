package alias_test

import (
	"slices"
	"testing"

	join "github.com/gloo-foo/cmd-join/alias"
	"github.com/gloo-foo/testable"
)

// The alias package re-exports the constructor and flag types under unprefixed
// names. A mis-wired re-export (Join bound to the wrong function, or Separator
// bound to the wrong type) compiles cleanly, so only behavior can prove the
// wiring. Each test exercises one re-export through observable output.

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAlias_JoinPairsOnFirstField(t *testing.T) {
	// input1 is the upstream stream; input2 is supplied via the re-exported
	// Input type. Keys a and b pair; the default separator is a single space.
	input2 := join.Input{[]byte("a x"), []byte("b y")}
	lines, err := testable.TestLines(join.Join(input2), "a 1\nb 2\n")
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"a 1 x", "b 2 y"})
}

func TestAlias_SeparatorChangesField(t *testing.T) {
	// The re-exported Separator type must rebind the field delimiter to a comma
	// for both the key split and the output rendering.
	input2 := join.Input{[]byte("a,x"), []byte("b,y")}
	lines, err := testable.TestLines(join.Join(input2, join.Separator(",")), "a,1\nb,2\n")
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"a,1,x", "b,2,y"})
}
