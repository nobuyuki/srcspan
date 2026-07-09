# `srcspan` -- Poor man's version of Rust's `codespan` for Golang

Error messages from Go-related tools are as simple as those from
classic Unix tools.

While this may not be a problem for experienced software engineers, it
can sometimes make it difficult for many others to understand the
nature of the error.

This is a Go module that features functionality similar to Rust's
codespan.

`srcspan` tries to have a similar interface as `codespan`.


## How to use
```go
package main

import (
	"os"

	"srcspan/srcspan"
	"srcspan/srcspan/term"
)

var content = []byte(`module FizzBuzz where

fizz₁ : Nat → String
fizz₁ num = case (mod num 5) (mod num 3) of
    0 0 => "FizzBuzz"
    0 _ => "Fizz"
    _ 0 => "Buzz"
    _ _ => num

fizz₂ : Nat → String
fizz₂ num =
    case (mod num 5) (mod num 3) of
        0 0 => "FizzBuzz"
        0 _ => "Fizz"
        _ 0 => "Buzz"
        _ _ => num
`)

func main() {
	files := srcspan.NewFiles()
	file_id := files.Add("FizzBuzz.fun", string(content))

	// We normally recommend creating a custom diagnostic data type for your
	// application, and then converting that to `codespan-reporting`'s diagnostic
	// type, but for the sake of this example we construct it directly.

	diagnostic := srcspan.NewError().WithMessage("`case` clauses have incompatible types").WithCode("E0308").WithLabels([]srcspan.Label{
		*srcspan.NewPrimary(file_id, srcspan.NewSpan(328, 331)).WithMessage("expected `String`, found `Nat`"),
		*srcspan.NewSecondary(file_id, srcspan.NewSpan(211, 331)).WithMessage("`case` clauses have incompatible types"),
		*srcspan.NewSecondary(file_id, srcspan.NewSpan(258, 268)).WithMessage("this is found to be of type `String`"),
		*srcspan.NewSecondary(file_id, srcspan.NewSpan(284, 290)).WithMessage("this is found to be of type `String`"),
		*srcspan.NewSecondary(file_id, srcspan.NewSpan(306, 312)).WithMessage("this is found to be of type `String`"),
		*srcspan.NewSecondary(file_id, srcspan.NewSpan(186, 192)).WithMessage("expected type `String` found here"),
	}).WithNotes([]string{
		`expected type 'String'
   found type 'Nat'`})

	//	writer := term.NewPlainWriter(os.Stdout)
	writer := term.NewStylesWriter(os.Stdout, term.DefaultColor())
	config := term.DefaultConfig()

	err := term.EmitToWriteStyle(writer, &config, &files, diagnostic)
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
```

## Limitations


(EOF)
