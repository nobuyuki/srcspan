package term_test

import (
	"os"

	"github.com/nobuyuki/srcspan/diagnostic"
	"github.com/nobuyuki/srcspan/files"
	"github.com/nobuyuki/srcspan/term"
)

func ExampleEmitToIoWrite() {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("FizzBuzz.fun", `module FizzBuzz where

fizz₁ : Nat → String
fizz₁ num = case (mod num 5) (mod num 3) of
    0 0 => "FizzBuzz"
    0 _ => "Fizz"
    _ 0 => "Buzz"
    _ _ => num
`)

	d := diagnostic.NewError().
		WithMessage("`case` clauses have incompatible types").
		WithCode("E0308").
		WithLabels([]diagnostic.Label{
			*diagnostic.NewPrimary(file_id, files.NewRange(163, 166)).WithMessage("expected `String`, found `Nat`"),
			*diagnostic.NewSecondary(file_id, files.NewRange(62, 166)).WithMessage("`case` clauses have incompatible types"),
			*diagnostic.NewSecondary(file_id, files.NewRange(41, 47)).WithMessage("expected type `String` found here"),
		}).
		WithNotes([]string{"expected type `String`\n   found type `Nat`"})

	config := term.DefaultConfig()

	// Use `term.NewStylesWriter` and `term.EmitToWriteStyle` for colored output.
	if err := term.EmitToIoWrite(os.Stdout, &config, &fs, d); err != nil {
		panic(err)
	}
}
