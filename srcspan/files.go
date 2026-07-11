package srcspan

type Files struct {
	files []File
}

func NewFiles() Files {
	return Files{
		files: []File{},
	}
}

func (f *Files) Add(name string, source Source) int {
	file_id := len(f.files)
	f.files = append(f.files, NewFile(name, source))
	return file_id
}

func (fs *Files) Update(file_id int, source Source) {
	f, _ := fs.Get(file_id)
	f.Update(source)
}

func (f *Files) Get(file_id int) (File, error) {
	return f.files[file_id], nil
}

func (f *Files) Name(id int) (string, error) {
	return f.files[id].Name(), nil
}

func (fs *Files) LineSpan(file_id, line_index int) (Span, error) {
	f, err := fs.Get(file_id)
	if err != nil {
		return Span{}, nil
	}
	return f.LineSpan(line_index)
}

func (fs *Files) LineRange(file_id, line_index int) (Range, error) {
	f, err := fs.Get(file_id)
	if err != nil {
		return Range{}, nil
	}
	span, err := f.LineSpan(line_index)
	if err != nil {
		return Range{}, nil
	}
	return NewRange(span.Start(), span.End()), nil
}

func (fs *Files) Source(id int) (*Source, error) {
	return fs.files[id].Source(), nil
}

func (fs *Files) SourceSpan(file_id int) Span {
	f, _ := fs.Get(file_id)
	span := f.SourceSpan()
	return span
}

func (fs *Files) SourceSlice(file_id int, span Span) (string, error) {
	f, _ := fs.Get(file_id)
	return f.SourceSlice(span)
}

func (fs *Files) LineIndex(id, byte_index int) (int, error) {
	// file_idのエラーがいるかもね〜
	return fs.files[id].LineIndex(byte_index), nil
}

func (*Files) LineNumber(_, line_index int) (int, error) {
	return line_index + 1, nil
}

func (fs *Files) ColumnNumber(id, line_index, byte_index int) (int, error) {
	f, err := fs.Get(id)
	if err != nil {
		return 0, err
	}

	line_range, err := fs.LineRange(id, line_index)
	if err != nil {
		return 0, err
	}

	source := f.Source()

	column_index := GetColumnIndex(*source, line_range, byte_index)

	return column_index + 1, nil
}

func (fs *Files) Location(id, byte_index int) (Location, error) {
	f, _ := fs.Get(id)
	return f.Location(byte_index)
}

func GetColumnIndex(source string, line_range Range, byte_index int) (count int) {
	end_index := min(byte_index, min(line_range.End, len(source)))
	for byte_index := line_range.Start; byte_index < end_index; byte_index++ {
		if IsRuneBoundary(source, byte_index+1) {
			count++
		}
	}
	return
}
