//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Reader needs io.ReaderAt for archive/zip
type Reader interface {
	io.Reader
	io.ReaderAt
}

// Writer
type Writer interface {
	io.Writer
}

// Printable file format
type Formatter interface {
	Parse(args []string) (err error)
	Parsed() bool
	Args() (args []string)
	NArg() int
	PrintDefaults()

	Decode(reader Reader, size int64) (printable Printable, err error)
	Encode(writer Writer, printable Printable) (err error)
}

// Printable to file format
type NewFormatter func(suffix string) (formatter Formatter)

var formatterMap map[string]NewFormatter

func RegisterFormatter(suffix string, newFormatter NewFormatter) {
	if formatterMap == nil {
		formatterMap = make(map[string]NewFormatter)
	}

	formatterMap[suffix] = newFormatter
}

func FormatterUsage() {
	if formatterMap != nil {
		list := []string{}
		for suffix := range formatterMap {
			list = append(list, suffix)
		}
		sort.Strings(list)

		for _, suffix := range list {
			newFormatter := formatterMap[suffix]
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "Options for '%s':\n", suffix)
			fmt.Fprintln(os.Stderr)
			newFormatter(suffix).PrintDefaults()
		}
	}
}

type Format struct {
	Formatter
	Suffix   string
	Filename string
}

func NewFormat(filename string, args []string) (format *Format, err error) {
	var formatter Formatter
	var suffix string
	var newFormatter NewFormatter

	for suffix, newFormatter = range formatterMap {
		if strings.HasSuffix(filename, suffix) {

			// Get formatter, and parse arguments
			formatter = newFormatter(suffix)
			break
		}
	}

	if formatter == nil {
		err = fmt.Errorf("%s: File extension unknown", filename)
		return
	}

	err = formatter.Parse(args)
	if err != nil {
		return
	}

	format = &Format{
		Formatter: formatter,
		Suffix:    suffix,
		Filename:  filename,
	}
	return
}

func (format *Format) Printable() (printable Printable, err error) {
	var reader *os.File
	var filesize int64

	if format.Suffix != "empty" {
		reader, err = os.Open(format.Filename)
		if err != nil {
			return
		}
		defer func() { reader.Close() }()

		filesize, err = reader.Seek(0, io.SeekEnd)
		if err != nil {
			return
		}

		_, err = reader.Seek(0, io.SeekStart)
		if err != nil {
			return
		}
	}

	decoded, err := format.Decode(reader, filesize)
	if err != nil {
		return
	}

	printable = decoded
	return
}

// Write writes a printable to the file format
func (format *Format) SetPrintable(printable Printable) (err error) {
	writer, err := os.Create(format.Filename)
	if err != nil {
		return
	}
	defer func() { writer.Close() }()

	err = format.Encode(writer, printable)
	if err != nil {
		return
	}

	return
}
