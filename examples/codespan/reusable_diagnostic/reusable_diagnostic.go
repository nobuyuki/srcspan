package main

// An example of converting the errors of your program into diagnostics.
//
// To run this example, execute the following command from the top level of
// this repository:
//
// ```sh
// go run ./examples/codespan/reusable_diagnostic/reusable_diagnostic.go
// ```

import (
	"flag"
	"fmt"
	"os"

	"github.com/nobuyuki/srcspan/diagnostic"
	"github.com/nobuyuki/srcspan/files"
	"github.com/nobuyuki/srcspan/term"

	"github.com/nobuyuki/srcspan/internal/unindent"
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
	file := files.NewSimpleFile(
		"main.rs",
		unindent.Unindent(`
                fn main() {
                    let foo: i32 = "hello, world";
                    foo += 1;
                }
            `),
	)

	errors := []Error{
		MismatchType{
			Left:  NewItem(files.NewRange(20, 23), "i32"),
			Right: NewItem(files.NewRange(31, 45), `"hello, world"`),
		},
		MutatingImmutable{
			Original: NewItem(files.NewRange(20, 23), "foo"),
			Mutating: NewItem(files.NewRange(51, 59), "foo += 1"),
		},
	}

	color := parseArgs()

	styles := term.DefaultColor()
	styles.ColorOutput = color
	writer := term.NewStylesWriter(os.Stderr, styles)
	config := term.DefaultConfig()

	for _, e := range errors {
		err := term.EmitToWriteStyle(writer, &config, &file, e.Report())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

// An error enum that represent all possible errors within your program.
// (`enum Error` in Rust; each variant is a type implementing it.)
type Error interface {
	Report() *diagnostic.Diagnostic
}

// The file id is ignored by `files.SimpleFile`. (`()` in Rust)
const file_id = 0

type MismatchType struct {
	Left  Item
	Right Item
}

func (e MismatchType) Report() *diagnostic.Diagnostic {
	return diagnostic.NewError().
		WithCode("E0308").
		WithMessage("mismatch types").
		WithLabels([]diagnostic.Label{
			diagnostic.NewPrimary(file_id, e.Right.Range).WithMessage(fmt.Sprintf(
				"Expected `%s`, found: `%s`",
				e.Left.Content, e.Right.Content,
			)),
			diagnostic.NewSecondary(file_id, e.Left.Range).WithMessage("expected due to this"),
		})
}

type MutatingImmutable struct {
	Original Item
	Mutating Item
}

func (e MutatingImmutable) Report() *diagnostic.Diagnostic {
	return diagnostic.NewError().
		WithCode("E0384").
		WithMessage(fmt.Sprintf(
			"cannot mutate immutable variable `%s`",
			e.Original.Content,
		)).
		WithLabels([]diagnostic.Label{
			diagnostic.NewSecondary(file_id, e.Original.Range).WithMessage(unindent.Unindent(
				fmt.Sprintf(`
                                first assignment to `+"`%[1]s`"+`
                                help: make this binding mutable: `+"`mut %[1]s`"+`
                            `,
					e.Original.Content,
				),
			)),
			diagnostic.NewPrimary(file_id, e.Mutating.Range).
				WithMessage("cannot assign twice to immutable variable"),
		})
}

// An item in the source code to be used in the `Error` enum.
// In a more complex program it could also contain a file id to handle
// errors that occur inside multiple files.
type Item struct {
	Range   files.Range
	Content string
}

func NewItem(range_ files.Range, content string) Item {
	return Item{Range: range_, Content: content}
}
