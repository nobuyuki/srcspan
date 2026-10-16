package main

import (
	"os"

	"github.com/nobuyuki/srcspan/diagnostic"
	"github.com/nobuyuki/srcspan/files"
	"github.com/nobuyuki/srcspan/term"
)

var data = []byte(`# This is a TOML document.

title = "TOML Example"
version = "1"

[owner]
name = "Tom Preston-Werner"
dob = 1979-05-27T07:32:00-08:00 # First class dates

[database]
server = "192.168.1.1"
ports = [ 8000, 8001, 8002 ]
connection_max = 5000
enabled = true

[servers]

  # Indentation (tabs and/or spaces) is allowed but not required
  [servers.alpha]
  ip = "10.0.0.1"
  dc = "eqdc10"

  [servers.beta]
  ip = "10.0.0.2"
  dc = "eqdc10"

[clients]
data = [ ["gamma", "delta"], [1, 2] ]

# Line breaks are OK when inside arrays
hosts = [
  "alpha",
  "omega"
]
`)

func main() {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("config.yaml", string(data))

	d0 := diagnostic.NewNote().WithMessage("This is a TOML comment").WithLabel(
		diagnostic.NewPrimary(file_id, files.NewRange(0, 26)),
	).WithNote("Comment starts from '#' to EOL")

	d1 := diagnostic.NewNote().WithMessage("Number or string").WithLabel(
		diagnostic.NewPrimary(file_id, files.NewRange(61, 64)).WithMessage("You can write just '1', rather thatn '\"1\"'."),
	).WithNote("Both string and number are accepted, but string should be able to interpret as a number.")

	d2 := diagnostic.NewError().WithMessage("invalid value").WithLabel(
		diagnostic.NewPrimary(file_id, files.NewRange(61, 64)).WithMessage("expected number"),
	)

	d3 := diagnostic.NewNote().WithMessage("First class date").WithLabel(
		diagnostic.NewPrimary(file_id, files.NewRange(108, 133)).WithMessage("Yon don't need double quotes."),
	).WithNote("If you put sinble/double qoutes, it's a string type rather than a date type.")

	d4 := diagnostic.NewNote().WithMessage("Indentation (tabs and/or spaces) is allowed but not required").WithLabels([]diagnostic.Label{
		diagnostic.NewPrimary(file_id, files.NewRange(332, 334)).WithMessage("Indentation."),
		diagnostic.NewPrimary(file_id, files.NewRange(350, 352)).WithMessage("Indentation."),
		diagnostic.NewPrimary(file_id, files.NewRange(368, 370)).WithMessage("Indentation."),

		diagnostic.NewPrimary(file_id, files.NewRange(385, 387)).WithMessage("Indentation."),
		diagnostic.NewPrimary(file_id, files.NewRange(402, 404)).WithMessage("Indentation."),
		diagnostic.NewPrimary(file_id, files.NewRange(420, 422)).WithMessage("Indentation."),

		diagnostic.NewSecondary(file_id, files.NewRange(270, 435)).WithMessage("A whole indented block."),
	})

	d5 := diagnostic.NewNote().WithMessage("Line breaks are OK when inside arrays").WithLabels([]diagnostic.Label{
		diagnostic.NewPrimary(file_id, files.NewRange(535, 535)).WithMessage("Line break here."),
		diagnostic.NewPrimary(file_id, files.NewRange(546, 546)).WithMessage("Line break here."),
		diagnostic.NewPrimary(file_id, files.NewRange(556, 556)).WithMessage("Line break here."),

		diagnostic.NewSecondary(file_id, files.NewRange(534, 568)).WithMessage("Whole array."),
	}).WithNote("You can also add the final ',' at the end of the last item.")

	diagnostics := []*diagnostic.Diagnostic{
		d0,
		d1,
		d2,
		d3,
		d4,
		d5,
	}

	// We now set up the writer and configuration, and then finally render the
	// diagnostic to standard error.

	writer := term.NewStylesWriter(os.Stdout, term.DefaultColor())
	config := term.DefaultConfig()

	for _, diagnostic := range diagnostics {
		err := term.EmitToWriteStyle(writer, &config, &fs, diagnostic)
		if err != nil {
			os.Exit(1)
		}
	}
	os.Exit(0)
}
