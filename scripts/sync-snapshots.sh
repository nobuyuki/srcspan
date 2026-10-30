#!/usr/bin/env bash
#
# Copies the upstream `insta` snapshots of codespan-reporting's tests/term.rs
# to term/testdata/snapshots, replacing the existing ones.
#
# Usage:
#   scripts/sync-snapshots.sh <path to codespan-reporting>
#
# <path to codespan-reporting> is the crate directory that contains
# tests/snapshots, either in a clone of the upstream repository:
#
#   git clone --depth 1 --branch 'release/codespan-reporting=0.13.1' \
#       https://github.com/brendanzab/codespan.git
#   scripts/sync-snapshots.sh codespan/codespan-reporting
#
# or in the cargo registry, which is populated by building
# term/testdata/snapshots_extra/rust:
#
#   scripts/sync-snapshots.sh \
#       ~/.cargo/registry/src/index.crates.io-*/codespan-reporting-0.13.1

set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "usage: $0 <path to codespan-reporting>" >&2
	exit 2
fi

src="$1/tests/snapshots"
dst="$(cd "$(dirname "$0")/.." && pwd)/term/testdata/snapshots"

shopt -s nullglob
snaps=("$src"/term__*.snap)
if [[ ${#snaps[@]} -eq 0 ]]; then
	echo "no term__*.snap in $src" >&2
	exit 1
fi

rm -f "$dst"/term__*.snap
cp "${snaps[@]}" "$dst"/
echo "copied ${#snaps[@]} snapshots to $dst"
