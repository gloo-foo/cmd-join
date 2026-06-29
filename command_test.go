package command_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/gloo-foo/testable"
	"github.com/spf13/afero"

	command "github.com/gloo-foo/cmd-join"
)

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestJoin_PairsOnFirstField joins input1 (the upstream stream) with input2
// (raw lines) on the shared first field, default single-space separator.
func TestJoin_PairsOnFirstField(t *testing.T) {
	input2 := command.JoinInput{[]byte("a x"), []byte("b y")}
	lines, err := testable.TestLines(command.Join(input2), "a 1\nb 2\n")
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"a 1 x", "b 2 y"})
}

// TestJoin_OmitsUnpairedLines drops keys present in only one input (GNU
// default): key "a" only in input1, key "d" only in input2.
func TestJoin_OmitsUnpairedLines(t *testing.T) {
	input2 := command.JoinInput{[]byte("b y"), []byte("d w")}
	lines, err := testable.TestLines(command.Join(input2), "a 1\nb 2\nc 3\n")
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"b 2 y"})
}

// TestJoin_CrossProductOnDuplicateKeys emits every pair when a key repeats in
// both inputs, matching GNU join's behavior on sorted duplicates.
func TestJoin_CrossProductOnDuplicateKeys(t *testing.T) {
	input2 := command.JoinInput{[]byte("a x"), []byte("a y")}
	lines, err := testable.TestLines(command.Join(input2), "a 1\na 2\n")
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"a 1 x", "a 1 y", "a 2 x", "a 2 y"})
}

// TestJoin_BareLineHasNoDoubledSeparator joins a key whose input1 line carries
// no remaining fields; the output must not contain an empty field.
func TestJoin_BareLineHasNoDoubledSeparator(t *testing.T) {
	input2 := command.JoinInput{[]byte("a x")}
	lines, err := testable.TestLines(command.Join(input2), "a\n")
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"a x"})
}

// TestJoin_CustomSeparator wires -t through to both the key split and the
// output rendering.
func TestJoin_CustomSeparator(t *testing.T) {
	input2 := command.JoinInput{[]byte("a,x"), []byte("b,y")}
	lines, err := testable.TestLines(command.Join(input2, command.JoinSeparator(",")), "a,1\nb,2\n")
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"a,1,x", "b,2,y"})
}

// TestJoin_EmptyInputs yields no output and no error.
func TestJoin_EmptyInputs(t *testing.T) {
	lines, err := testable.TestLines(command.Join(command.JoinInput{}), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Fatalf("got %d lines, want 0: %v", len(lines), lines)
	}
}

// TestJoin_TwoFilePositionals reads both inputs from File positionals on an
// injected in-memory filesystem. The upstream stream is ignored when a first
// positional is present.
func TestJoin_TwoFilePositionals(t *testing.T) {
	fs := afero.NewMemMapFs()
	write(t, fs, "left.txt", "a 1\nb 2\n")
	write(t, fs, "right.txt", "a x\nc z\n")
	lines, err := testable.TestLines(
		command.Join(command.JoinFs(fs), "left.txt", "right.txt"),
		"ignored\n",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"a 1 x"})
}

// TestJoin_ReaderPositionalForInput1 supplies input1 as an io.Reader positional
// (exercising the non-File positional branch), with input2 as raw lines.
func TestJoin_ReaderPositionalForInput1(t *testing.T) {
	input1 := strings.NewReader("a 1\nb 2\n")
	input2 := command.JoinInput{[]byte("b y")}
	lines, err := testable.TestLines(command.Join(input1, input2), "")
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"b 2 y"})
}

// TestJoin_FileNotFoundPropagates surfaces an open error from a missing File
// positional rather than silently emitting nothing.
func TestJoin_FileNotFoundPropagates(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, err := testable.TestLines(command.Join(command.JoinFs(fs), "missing.txt"), "")
	if err == nil {
		t.Fatal("expected an error for a missing file, got nil")
	}
}

// TestJoin_ScanErrorPropagates surfaces a read error from an input reader.
func TestJoin_ScanErrorPropagates(t *testing.T) {
	failing := failingReader{err: errors.New("boom")}
	input2 := command.JoinInput{[]byte("a x")}
	_, err := testable.TestLines(command.Join(failing, input2), "")
	if err == nil {
		t.Fatal("expected the reader error to propagate, got nil")
	}
}

// TestJoin_SecondPositionalNotFoundPropagates surfaces an open error for the
// second positional (input2) after input1 loaded successfully.
func TestJoin_SecondPositionalNotFoundPropagates(t *testing.T) {
	fs := afero.NewMemMapFs()
	write(t, fs, "left.txt", "a 1\n")
	_, err := testable.TestLines(
		command.Join(command.JoinFs(fs), "left.txt", "missing.txt"),
		"",
	)
	if err == nil {
		t.Fatal("expected an error for a missing second file, got nil")
	}
}

// TestJoin_AdvancesPastSmallerInput2Key exercises the branch where the input2
// key sorts before the input1 key: input2 "a" has no match and is skipped.
func TestJoin_AdvancesPastSmallerInput2Key(t *testing.T) {
	input2 := command.JoinInput{[]byte("a x"), []byte("b y")}
	lines, err := testable.TestLines(command.Join(input2), "b 2\n")
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines, []string{"b 2 y"})
}

// TestJoin_NoSecondInputYieldsNothing covers the case where only input1 is
// provided (one positional, no JoinInput): with no input2 to pair against,
// join emits nothing without error.
func TestJoin_NoSecondInputYieldsNothing(t *testing.T) {
	fs := afero.NewMemMapFs()
	write(t, fs, "left.txt", "a 1\nb 2\n")
	lines, err := testable.TestLines(command.Join(command.JoinFs(fs), "left.txt"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Fatalf("got %d lines, want 0: %v", len(lines), lines)
	}
}

func write(t *testing.T, fs afero.Fs, name, content string) {
	t.Helper()
	if err := afero.WriteFile(fs, name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// failingReader is an io.Reader that always fails, to exercise the scan-error
// path.
type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }
