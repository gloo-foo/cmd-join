# Join Command Compatibility

## Summary

Compatible with GNU `join` for the implemented subset: the default join on field 1 and the `-t` field separator. Output is byte-identical to GNU `join` for those cases (verified in the `yupsh/bin-join` Docker integration harness against Debian coreutils, under `LC_ALL=C`). Both inputs must be sorted on the join field, as with GNU `join`.

## Key Behaviors

```bash
# Default join on the first field, single-space separator.
# Only keys present in both inputs are emitted (GNU default).
$ join a.txt b.txt        # a.txt: "a 1 / b 2 / c 3", b.txt: "a x / b y / d z"
a 1 x
b 2 y

# Many-to-one: a key repeated in either input yields the cross product
# of the two equal-key groups.
$ join m1.txt m2.txt      # m1: "a 1 / a 2 / b 3", m2: "a x / b y / b z"
a 1 x
a 2 x
b 3 y
b 3 z

# A line with no second field contributes no trailing fields; the
# separator is not doubled.
$ join k1.txt k2.txt      # k1: "a / b 2", k2: "a / b y"
a
b 2 y

# -t sets the input and output field separator (tab shown here).
$ join -t '\t' t1.txt t2.txt
a\t1\tx
b\t2\ty

# -t with a comma separator.
$ join -t , c1.txt c2.txt
a,1,x
b,2,y
```

## Intentional Divergences

- cmd-join implements the default-join subset of GNU `join`: an inner join on field 1, with the `-t`/`--separator` field separator. The GNU flags `-1`/`-2` (alternate join fields), `-a` (print unpairable lines), `-v` (only unpairable lines), `-e` (replace missing fields), `-o` (output format), `-i` (case-insensitive), `--header`, and `--check-order` are not implemented. Within the implemented surface, output matches GNU `join` exactly.
- Both inputs must already be sorted on the join field (GNU requirement); cmd-join does not sort and does not warn on unsorted input.
