//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uvj

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ezrec/uv3dp"
)

const ReferenceUVJ = "reference.uvj"

func TestUVJToUVJ(t *testing.T) {
	// Open reference file
	uvj, err := uv3dp.NewFormat(ReferenceUVJ, []string{})
	if err != nil {
		t.Fatalf("can't open %v: %v", ReferenceUVJ, err)
	}

	// Open output file
	file, err := ioutil.TempFile("", "*.uvj")
	name := file.Name()
	file.Close()

	out, err := uv3dp.NewFormat(name, []string{})
	if err != nil {
		t.Fatalf("can't open %v: %v", name, err)
	}

	// Convert
	uvj_print, err := uvj.Printable()
	if err != nil {
		t.Fatalf("can't decode %v: %v", ReferenceUVJ, err)
	}

	out.SetPrintable(uvj_print)
	defer os.Remove(name)

	// Verify files are equal
	out, err = uv3dp.NewFormat(name, []string{})
	if err != nil {
		t.Fatalf("can't read %v: %v", ReferenceUVJ, err)
	}

	out_print, err := out.Printable()
	if err != nil {
		t.Fatalf("can't read %v: %v", ReferenceUVJ, err)
	}

	if uvj_print.Size() != out_print.Size() {
		t.Errorf("printables are not the same Size")
	}
	if uvj_print.Bottom() != out_print.Bottom() {
		t.Errorf("printables are not the same Bottom exposure")
	}
	if uvj_print.Exposure() != out_print.Exposure() {
		t.Errorf("printables are not the same Exposure")
	}

	thumbnails := []uv3dp.PreviewType{
		uv3dp.PreviewTypeTiny,
		uv3dp.PreviewTypeHuge,
	}

	for _, code := range thumbnails {
		uvj_prev, uvj_ok := uvj_print.Preview(code)
		out_prev, out_ok := out_print.Preview(code)

		if uvj_ok != out_ok {
			t.Errorf("%+v: expected present %v, got %v", code, uvj_ok, out_ok)
			continue
		}

		if uvj_ok {
			if uvj_prev.Bounds() != out_prev.Bounds() {
				t.Errorf("%+v: expected bounds %+v, got %+v", code, uvj_prev.Bounds(), out_prev.Bounds())
				continue
			}
		} else {
			t.Logf("%+v: Not present", code)
		}
	}
}
