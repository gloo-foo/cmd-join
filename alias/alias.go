// Package alias provides unprefixed names for the join command's public API.
//
//	import join "github.com/gloo-foo/cmd-join/alias"
//	join.Join(join.Input{...}, join.Separator(","))
package alias

import (
	gloo "github.com/gloo-foo/framework"

	command "github.com/gloo-foo/cmd-join"
)

// Join re-exports the constructor.
func Join(opts ...any) gloo.Command[[]byte, []byte] { return command.Join(opts...) }

// Fs re-exports the filesystem option for File positionals.
type Fs = command.JoinFs

// Input is the second input supplied as raw lines.
type Input = command.JoinInput

// Separator is the -t field separator.
type Separator = command.JoinSeparator
