package main

import (
	"os"

	"srcspan/srcspan/diagnostic"
	"srcspan/srcspan/files"
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
	fs := files.NewSimpleFiles()
	file_id := fs.Add("FizzBuzz.fun", string(content))

	// We normally recommend creating a custom diagnostic data type for your
	// application, and then converting that to `codespan-reporting`'s diagnostic
	// type, but for the sake of this example we construct it directly.

	diagnostic := diagnostic.NewError().WithMessage("`case` clauses have incompatible types").WithCode("E0308").WithLabels([]diagnostic.Label{
		*diagnostic.NewPrimary(file_id, files.NewRange(328, 331)).WithMessage("expected `String`, found `Nat`"),
		*diagnostic.NewSecondary(file_id, files.NewRange(211, 331)).WithMessage("`case` clauses have incompatible types"),
		*diagnostic.NewSecondary(file_id, files.NewRange(258, 268)).WithMessage("this is found to be of type `String`"),
		*diagnostic.NewSecondary(file_id, files.NewRange(284, 290)).WithMessage("this is found to be of type `String`"),
		*diagnostic.NewSecondary(file_id, files.NewRange(306, 312)).WithMessage("this is found to be of type `String`"),
		*diagnostic.NewSecondary(file_id, files.NewRange(186, 192)).WithMessage("expected type `String` found here"),
	}).WithNotes([]string{
		`expected type 'String'
   found type 'Nat'`})

	//	writer := term.NewPlainWriter(os.Stdout)
	writer := term.NewStylesWriter(os.Stdout, term.DefaultColor())
	config := term.DefaultConfig()

	err := term.EmitToWriteStyle(writer, &config, &fs, diagnostic)
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
