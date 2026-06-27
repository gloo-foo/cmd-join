// Package alias provides unprefixed names for the join command's public API.
//
//	import join "github.com/gloo-foo/cmd-join/alias"
//	join.Join(join.Input{...}, join.Separator(","))
package alias

import command "github.com/gloo-foo/cmd-join"

// Join re-exports the constructor.
var Join = command.Join

// Fs re-exports the filesystem selector for File positionals.
var Fs = command.JoinFs

// Input is the second input supplied as raw lines.
type Input = command.JoinInput

// Separator is the -t field separator.
type Separator = command.JoinSeparator
