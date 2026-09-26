# `srcspan` -- A Go port of Rust's Codespan

Beautiful diagnostic reporting for text-based programming languages
and complex configuration files.

![Example](docs/images/readme.png)

Error messages from Go-related tools are as simple as those from
classic Unix tools.

While this may not be a problem for experienced software engineers, it
can sometimes make it difficult for many others to understand the
nature of the error.

This is a Go module that features functionality similar to Rust's
[codespan](https://github.com/brendanzab/codespan).  Actually,
`srcspan` is a literal one-to-one port of `codespan` and tries to have
a similar interface as `codespan`.


## Example
```go
package main

import (
	"os"

	"srcspan/srcspan"
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
	files := files.NewFiles()
	file_id := files.Add("FizzBuzz.fun", string(content))

	// We normally recommend creating a custom diagnostic data type for your
	// application, and then converting that to `codespan-reporting`'s diagnostic
	// type, but for the sake of this example we construct it directly.

	diagnostic := diagnostic.NewError().WithMessage("`case` clauses have incompatible types").WithCode("E0308").WithLabels([]diagnostic.Label{
		*diagnostic.NewPrimary(file_id, srcspan.NewSpan(328, 331)).WithMessage("expected `String`, found `Nat`"),
		*diagnostic.NewSecondary(file_id, srcspan.NewSpan(211, 331)).WithMessage("`case` clauses have incompatible types"),
		*diagnostic.NewSecondary(file_id, srcspan.NewSpan(258, 268)).WithMessage("this is found to be of type `String`"),
		*diagnostic.NewSecondary(file_id, srcspan.NewSpan(284, 290)).WithMessage("this is found to be of type `String`"),
		*diagnostic.NewSecondary(file_id, srcspan.NewSpan(306, 312)).WithMessage("this is found to be of type `String`"),
		*diagnostic.NewSecondary(file_id, srcspan.NewSpan(186, 192)).WithMessage("expected type `String` found here"),
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
([./examples/example/example.go](./examples/example/example.go))


## Running the CLI example

To get an idea of what the colored CLI output looks like, clone the
repository and run the following shell command:

```sh
go run ./examples/codespan/term/term.go

```
More examples of using `srcspan` can be found in the `examples` directory.


## Contributing

A guide to contributing to codespan-reporting [can be found here](/CONTRIBUTING.md).


## Code of Conduct

Please note that this project is released with a [Code of Conduct](./CODE_OF_CONDUCT.md).
By participating in this project you agree to abide by its terms.


## Limitations

### API

In Rust, there is a trait called `File` and a standard implementation
called `SimpleFile`, but in srcspan, a standard implementation called
`File` is provided. If possible, I would like to define an interface
modeled after Rust.

(EOF)
