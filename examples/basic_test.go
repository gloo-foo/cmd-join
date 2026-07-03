package join_test

import (
	gloo "github.com/gloo-foo/framework/patterns"

	join "github.com/gloo-foo/cmd-join/alias"
)

// ExampleJoin_basic joins two sorted files on their common first field. Keys 1,
// 2 and 3 appear in both files and pair up; key 4 (names only) and key 5
// (scores only) are unpaired and omitted (GNU default).
func ExampleJoin_basic() {
	if err := gloo.Run(
		join.Join("testdata/names.txt", "testdata/scores.txt"),
	); err != nil {
		panic(err)
	}
	// Output:
	// 1 Alice 95
	// 2 Bob 87
	// 3 Charlie 92
}

// ExampleJoin_separator joins on a custom field separator (-t ','). The first
// comma-delimited field is the key; remaining fields are concatenated with the
// same separator on output.
func ExampleJoin_separator() {
	if err := gloo.Run(
		join.Join(
			join.Input{[]byte("1,Alice"), []byte("2,Bob")},
			join.Separator(","),
			"testdata/csv-left.txt",
		),
	); err != nil {
		panic(err)
	}
	// Output:
	// 1,95,Alice
	// 2,87,Bob
}
