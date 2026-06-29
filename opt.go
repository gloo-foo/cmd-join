package command

import (
	"bytes"

	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
)

// JoinSeparator is the -t field separator: the byte sequence that delimits the
// join key from the rest of each line and joins fields in the output.
type JoinSeparator string

func (s JoinSeparator) Configure(f *flags) { f.separator = s }

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

// joinFs injects the filesystem used to open File positionals, so tests can
// supply an in-memory filesystem. The zero value falls back to the OS.
type joinFs struct{ afero.Fs }

// JoinFs selects the filesystem join uses to open File positional arguments.
func JoinFs(fs afero.Fs) gloo.Switch[flags] { return joinFs{fs} }

func (f joinFs) Configure(flags *flags) { flags.fs = f }

// value returns the configured filesystem, defaulting to the OS filesystem.
func (f joinFs) value() afero.Fs {
	if f.Fs == nil {
		return afero.NewOsFs()
	}
	return f.Fs
}

type flags struct {
	fs        joinFs
	separator JoinSeparator
}
