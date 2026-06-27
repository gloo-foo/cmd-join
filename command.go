package command

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/destel/rill"
	"github.com/spf13/afero"

	gloo "github.com/gloo-foo/framework"
)

// JoinInput supplies the second input as raw lines, taking precedence over any
// second positional file path.
type JoinInput [][]byte

// lines is one decoded input: the sorted lines join reads.
type lines [][]byte

// row is one input line split on the separator into its join key (first field)
// and the remaining fields (already without the separator that followed the
// key). A line with no separator has an empty rest.
type row struct {
	key  []byte
	rest []byte
}

// Join compares two sorted line streams on their common first field and emits
// one output line per matching pair: the key, the rest of the line1 row, then
// the rest of the line2 row, separated by the field separator. Lines without a
// pair in the other input are omitted (GNU default).
//
// Opts:
//   - 1st positional file/Reader: input1 (overrides the upstream stream).
//   - 2nd positional file/Reader: input2.
//   - JoinInput: input2 as raw lines (highest precedence for input2).
//   - JoinSeparator (-t): the field separator (defaults to a single space).
//   - JoinFs: filesystem used to open File positionals (defaults to the OS).
func Join(opts ...any) gloo.Command[[]byte, []byte] {
	params := gloo.NewParameters[gloo.File, flags](opts...)
	f := params.Flags
	src := newSources(opts, params.Positional, f.fs.value())
	sep := f.separator.value()
	return gloo.FuncCommand[[]byte, []byte](func(ctx context.Context, in gloo.Stream[[]byte]) gloo.Stream[[]byte] {
		return gloo.GenerateFrom(ctx, in, func(_ context.Context, send func([]byte) bool, sendErr func(error)) {
			run(send, sendErr, src, in, sep)
		})
	})
}

// run loads both inputs and emits the joined pairs, forwarding any load error.
func run(send func([]byte) bool, sendErr func(error), src sources, in gloo.Stream[[]byte], sep separator) {
	input1, input2, err := src.load(in)
	if err != nil {
		sendErr(err)
		return
	}
	joiner{send: send, sep: sep}.merge(rowsOf(input1, sep), rowsOf(input2, sep))
}

// rowsOf splits every input line into a join key and its remaining fields.
func rowsOf(in lines, sep separator) []row {
	out := make([]row, len(in))
	for i, line := range in {
		out[i] = sep.split(line)
	}
	return out
}

// sources resolves the two join inputs from opts, positionals, and the upstream
// stream. It is an immutable value built once per Join call.
type sources struct {
	fs             afero.Fs
	positionals    []any
	explicitInput2 lines
	hasExplicit2   bool
}

// newSources classifies the opts into the resolved input sources.
func newSources(opts []any, positionals []any, fs afero.Fs) sources {
	explicit, ok := explicitInput2(opts)
	return sources{
		fs:             fs,
		positionals:    positionals,
		explicitInput2: explicit,
		hasExplicit2:   ok,
	}
}

// explicitInput2 returns the first JoinInput option, if any.
func explicitInput2(opts []any) (lines, bool) {
	for _, o := range opts {
		if v, ok := o.(JoinInput); ok {
			return lines(v), true
		}
	}
	return nil, false
}

// load resolves input1 then input2.
func (s sources) load(in gloo.Stream[[]byte]) (lines, lines, error) {
	input1, err := s.loadInput1(in)
	if err != nil {
		return nil, nil, err
	}
	input2, err := s.loadInput2()
	if err != nil {
		return nil, nil, err
	}
	return input1, input2, nil
}

// loadInput1 reads the first positional, falling back to the upstream stream.
func (s sources) loadInput1(in gloo.Stream[[]byte]) (lines, error) {
	if len(s.positionals) >= 1 {
		return s.readPositional(s.positionals[0])
	}
	got, err := rill.ToSlice(in.Chan())
	return lines(got), err
}

// loadInput2 prefers an explicit JoinInput, else the second positional, else
// nothing.
func (s sources) loadInput2() (lines, error) {
	switch {
	case s.hasExplicit2:
		return s.explicitInput2, nil
	case len(s.positionals) >= 2:
		return s.readPositional(s.positionals[1])
	default:
		return nil, nil
	}
}

// readPositional decodes one positional argument into lines. The framework
// guarantees every positional is a gloo.File path or an io.Reader (see
// gloo.NewParameters), so those two cases are exhaustive.
func (s sources) readPositional(positional any) (lines, error) {
	if name, ok := positional.(gloo.File); ok {
		return s.readFile(name)
	}
	return scanLines(positional.(io.Reader))
}

// readFile opens a File positional on the injected filesystem and scans it.
func (s sources) readFile(name gloo.File) (out lines, err error) {
	f, err := s.fs.Open(string(name))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	return scanLines(f)
}

// scanLines reads r into a slice of independently-owned line copies.
func scanLines(r io.Reader) (lines, error) {
	scanner := bufio.NewScanner(r)
	var out lines
	for scanner.Scan() {
		out = append(out, bytes.Clone(scanner.Bytes()))
	}
	return out, scanner.Err()
}

// joiner emits joined output rows through send using the field separator.
type joiner struct {
	send func([]byte) bool
	sep  separator
}

// merge walks both sorted inputs, emitting the cross product of every group of
// rows that share a key. Unpaired groups are skipped (GNU default).
func (j joiner) merge(rows1, rows2 []row) {
	i, k := 0, 0
	for i < len(rows1) && k < len(rows2) {
		i, k = j.step(rows1, rows2, i, k)
	}
}

// step compares the keys at i and k. On a mismatch it advances past the smaller
// key; on a match it emits the cross product of both equal-key groups and
// advances past both groups.
func (j joiner) step(rows1, rows2 []row, i, k int) (int, int) {
	switch cmp := bytes.Compare(rows1[i].key, rows2[k].key); {
	case cmp < 0:
		return i + 1, k
	case cmp > 0:
		return i, k + 1
	default:
		end1 := groupEnd(rows1, i)
		end2 := groupEnd(rows2, k)
		j.emitGroups(rows1[i:end1], rows2[k:end2])
		return end1, end2
	}
}

// emitGroups emits the cross product of two equal-key groups.
func (j joiner) emitGroups(group1, group2 []row) {
	for _, r1 := range group1 {
		for _, r2 := range group2 {
			j.send(j.sep.join(r1.key, r1.rest, r2.rest))
		}
	}
}

// groupEnd returns the index just past the run of rows sharing rows[start].key.
func groupEnd(rows []row, start int) int {
	end := start + 1
	for end < len(rows) && bytes.Equal(rows[end].key, rows[start].key) {
		end++
	}
	return end
}
