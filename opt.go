package command

import (
	"bytes"

	"github.com/spf13/afero"
)

// JoinSeparator is the -t field separator: the byte sequence that delimits the
// join key from the rest of each line and joins fields in the output.
type JoinSeparator string

// value returns the configured separator, defaulting to a single space.
func (s JoinSeparator) value() separator {
	if s == "" {
		return separator(" ")
	}
	return separator(s)
}

// separator is the resolved field separator used to split and join lines.
type separator []byte

// split divides a line into its join key (first field) and the remaining
// fields. A line with no separator yields an empty rest.
func (sep separator) split(line []byte) row {
	idx := bytes.Index(line, sep)
	if idx < 0 {
		return row{key: line}
	}
	return row{key: line[:idx], rest: line[idx+len(sep):]}
}

// join renders an output line: the key, then each non-empty remainder, joined
// by sep. A row with no remaining fields contributes nothing, so a key that
// matched a bare line does not produce a doubled separator.
func (sep separator) join(key, rest1, rest2 []byte) []byte {
	fields := [][]byte{key}
	fields = appendNonEmpty(fields, rest1)
	fields = appendNonEmpty(fields, rest2)
	return bytes.Join(fields, sep)
}

// appendNonEmpty appends field to fields only when it carries content.
func appendNonEmpty(fields [][]byte, field []byte) [][]byte {
	if len(field) == 0 {
		return fields
	}
	return append(fields, field)
}

// JoinFs injects the filesystem used to open File positionals, so tests can
// supply an in-memory filesystem (afero.NewMemMapFs()). The zero value falls
// back to the OS filesystem.
type JoinFs struct{ afero.Fs }

// value returns the configured filesystem, defaulting to the OS filesystem.
func (f JoinFs) value() afero.Fs {
	if f.Fs == nil {
		return afero.NewOsFs()
	}
	return f.Fs
}

// flags is the option set folded from a Join call's option values.
type flags struct {
	fs        JoinFs
	separator JoinSeparator
	input2    lines
	hasInput2 bool
}

// with folds one option value into the flag set, reporting whether o was a
// recognized join option.
func (f flags) with(o any) (flags, bool) {
	switch v := o.(type) {
	case JoinSeparator:
		f.separator = v
	case JoinFs:
		f.fs = v
	case JoinInput:
		f.input2, f.hasInput2 = lines(v), true
	default:
		return f, false
	}
	return f, true
}

// foldOptions applies every recognized join option to a zero flags value and
// returns the leftover arguments for the framework's positional classification.
func foldOptions(opts []any) (flags, []any) {
	var f flags
	var rest []any
	for _, o := range opts {
		next, isOption := f.with(o)
		if !isOption {
			rest = append(rest, o)
			continue
		}
		f = next
	}
	return f, rest
}
