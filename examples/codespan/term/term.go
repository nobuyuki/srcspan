package main

// To run this example, execute the following command from the top level of
// this repository:
//
// ```sh
// go run ./examples/codespan/term/term.go
// ```

import (
	"flag"
	"fmt"
	"os"

	"srcspan/srcspan/diagnostic"
	"srcspan/srcspan/files"
	term "srcspan/srcspan/term"
)

var colorChoice string

func init() {
	flag.StringVar(&colorChoice, "color", "auto", "none|auto|color")
	flag.Parse()
}

func parseArgs() term.ColorOutput {
	switch colorChoice {
	case "none":
		return term.ColorOutputNone
	case "auto":
		return term.ColorOutputAuto
	case "color":
		return term.ColorOutputForce
	default:
		fmt.Printf("--color value should be one of 'none', 'auto', 'color'.")
		os.Exit(1)
		return term.ColorOutputAuto
	}
}

func main() {
	colorOutput := parseArgs()

	fs := files.NewSimpleFiles()

	file_id1 := fs.Add(
		"Data/Nat.fun",
		`module Data.Nat where

data Nat : Type where
    zero : Nat
    succ : Nat → Nat

{-# BUILTIN NATRAL Nat #-}

infixl 6 _+_ _-_

_+_ : Nat → Nat → Nat
zero    + n₂ = n₂
succ n₁ + n₂ = succ (n₁ + n₂)

_-_ : Nat → Nat → Nat
n₁      - zero    = n₁
zero    - succ n₂ = zero
succ n₁ - succ n₂ = n₁ - n₂
`,
	)

	file_id2 := fs.Add(
		"Test.fun",
		`module Test where

_ : Nat
_ = 123 + "hello"
`,
	)

	file_id3 := fs.Add(
		"FizzBuzz.fun",
		`module FizzBuzz where

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
`,
	)

	diagnostics := []*diagnostic.Diagnostic{
		// Unknown builtin error
		diagnostic.NewError().WithMessage("unknown builtin: `NATRAL`").
			WithLabel(*diagnostic.NewPrimary(file_id1, files.NewRange(96, 102)).WithMessage("unknown builtin")).
			WithNotes([]string{"there is a builtin with a similar name: `NATURAL`"}),

		// Unused parameter warning
		diagnostic.NewWarning().
			WithMessage("unused parameter pattern: `n₂`").
			WithLabel(*diagnostic.NewPrimary(file_id1, files.NewRange(285, 289)).WithMessage("unused parameter")).
			WithNotes([]string{"consider using a wildcard pattern: `_`"}),

		// Unexpected type error
		diagnostic.NewError().
			WithMessage("unexpected type in application of `_+_`").
			WithCode("E0001").
			WithLabels([]diagnostic.Label{
				*diagnostic.NewPrimary(file_id2, files.NewRange(37, 44)).WithMessage("expected `Nat`, found `String`"),
				*diagnostic.NewSecondary(file_id1, files.NewRange(130, 155)).WithMessage("based on the definition of `_+_`"),
			}).
			WithNotes([]string{`expected type 'Nat'
   found type 'String'
`,
			}),

		// Incompatible match clause error
		diagnostic.NewError().
			WithMessage("`case` clauses have incompatible types").
			WithCode("E0308").
			WithLabels([]diagnostic.Label{
				*diagnostic.NewPrimary(file_id3, files.NewRange(163, 166)).WithMessage("expected `String`, found `Nat`"),
				*diagnostic.NewSecondary(file_id3, files.NewRange(62, 166)).WithMessage("`case` clauses have incompatible types"),
				*diagnostic.NewSecondary(file_id3, files.NewRange(41, 47)).WithMessage("expected type `String` found here"),
			}).
			WithNotes([]string{
				`expected type 'String'
   found type 'Nat'
                `,
			}),
		// Incompatible match clause error
		diagnostic.NewError().
			WithMessage("`case` clauses have incompatible types").
			WithCode("E0308").
			WithLabels([]diagnostic.Label{
				*diagnostic.NewPrimary(file_id3, files.NewRange(328, 331)).WithMessage("expected `String`, found `Nat`"),
				*diagnostic.NewSecondary(file_id3, files.NewRange(211, 331)).WithMessage("`case` clauses have incompatible types"),
				*diagnostic.NewSecondary(file_id3, files.NewRange(258, 268)).WithMessage("this is found to be of type `String`"),
				*diagnostic.NewSecondary(file_id3, files.NewRange(284, 290)).WithMessage("this is found to be of type `String`"),
				*diagnostic.NewSecondary(file_id3, files.NewRange(306, 312)).WithMessage("this is found to be of type `String`"),
				*diagnostic.NewSecondary(file_id3, files.NewRange(186, 192)).WithMessage("expected type `String` found here"),
			}).
			WithNotes([]string{
				`expected type 'String'
   found type 'Nat'
                `,
			}),
	}

	styles := term.DefaultColor()
	styles.ColorOutput = colorOutput
	writer := term.NewStylesWriter(os.Stderr, styles)
	config := term.DefaultConfig()

	for _, diagnostic := range diagnostics {
		err := term.EmitToWriteStyle(writer, &config, &fs, diagnostic)
		if err != nil {
			os.Exit(1)
		}
	}

	os.Exit(0)
}
