package diagnostic

// Serialization of the diagnostic data structures, compatible with the
// `serialization` feature in Rust (`#[derive(Serialize, Deserialize)]`
// with serde), e.g. with serde_json:
//
//	{"severity":"Error","code":"E0001","message":"...",
//	 "labels":[{"style":"Primary","file_id":0,"range":{"start":9,"end":9},"message":"..."}],
//	 "notes":["..."]}

import (
	"encoding/json"
	"fmt"
)

// The variant names of `Severity` in Rust, in declaration order.
var severityNames = []struct {
	severity Severity
	name     string
}{
	{Help, "Help"},
	{Note, "Note"},
	{Warning, "Warning"},
	{Error, "Error"},
	{Bug, "Bug"},
}

// Encodes the severity as its variant name in Rust (e.g. "Error").
func (s Severity) MarshalText() ([]byte, error) {
	for _, n := range severityNames {
		if n.severity == s {
			return []byte(n.name), nil
		}
	}
	return nil, fmt.Errorf("unknown Severity: %d", int(s))
}

// Decodes a severity from its variant name in Rust (e.g. "Error").
func (s *Severity) UnmarshalText(text []byte) error {
	for _, n := range severityNames {
		if n.name == string(text) {
			*s = n.severity
			return nil
		}
	}
	return fmt.Errorf("unknown variant `%s`, expected one of `Help`, `Note`, `Warning`, `Error`, `Bug`", text)
}

// The variant names of `LabelStyle` in Rust, in declaration order.
var labelStyleNames = []struct {
	style LabelStyle
	name  string
}{
	{Primary, "Primary"},
	{Secondary, "Secondary"},
}

// Encodes the label style as its variant name in Rust (e.g. "Primary").
func (l LabelStyle) MarshalText() ([]byte, error) {
	for _, n := range labelStyleNames {
		if n.style == l {
			return []byte(n.name), nil
		}
	}
	return nil, fmt.Errorf("unknown LabelStyle: %d", int(l))
}

// Decodes a label style from its variant name in Rust (e.g. "Primary").
func (l *LabelStyle) UnmarshalText(text []byte) error {
	for _, n := range labelStyleNames {
		if n.name == string(text) {
			*l = n.style
			return nil
		}
	}
	return fmt.Errorf("unknown variant `%s`, expected `Primary` or `Secondary`", text)
}

// The JSON representation of `Diagnostic`. `Code` is `Option<String>` in
// Rust, so it is a pointer here to encode no code as `null`.
type diagnosticJSON struct {
	Severity Severity `json:"severity"`
	Code     *string  `json:"code"`
	Message  string   `json:"message"`
	Labels   []Label  `json:"labels"`
	Notes    []string `json:"notes"`
}

// Encodes the diagnostic like serde in Rust.
//
// An empty `Code` means no code (`None` in Rust), and is encoded as `null`.
// `nil` labels and notes are encoded as `[]`, since they are `Vec`s in Rust.
func (d Diagnostic) MarshalJSON() ([]byte, error) {
	j := diagnosticJSON{
		Severity: d.Severity,
		Message:  d.Message,
		Labels:   d.Labels,
		Notes:    d.Notes,
	}
	if d.Code != "" {
		j.Code = &d.Code
	}
	if j.Labels == nil {
		j.Labels = []Label{}
	}
	if j.Notes == nil {
		j.Notes = []string{}
	}
	return json.Marshal(j)
}

// Decodes a diagnostic encoded like serde in Rust.
//
// A `null` or missing code is decoded as an empty `Code`.
//
// NOTE: `Some("")` in Rust is also decoded as an empty `Code`, which means
// no code in Go. Both are rendered the same way.
func (d *Diagnostic) UnmarshalJSON(data []byte) error {
	var j diagnosticJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}

	*d = Diagnostic{
		Severity: j.Severity,
		Message:  j.Message,
		Labels:   j.Labels,
		Notes:    j.Notes,
	}
	if j.Code != nil {
		d.Code = *j.Code
	}
	if d.Labels == nil {
		d.Labels = []Label{}
	}
	if d.Notes == nil {
		d.Notes = []string{}
	}
	return nil
}
