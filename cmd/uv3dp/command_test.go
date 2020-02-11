package main

import (
	"testing"

	"bytes"
	"os"
)

func TestCommandExpand(t *testing.T) {
	table := map[string]struct {
		In    string
		Out   []string
		Error error
	}{
		"hello":  {`hello world`, []string{"hello", "world"}, nil},
		"setenv": {`hello ${MONKEY}`, []string{"hello", "monkey"}, nil},
		"oct":    {`\101`, []string{"A"}, nil},
		"esacpe": {`hello\ you\e[7m\z\e[m\r\n\101`, []string{"hello you\033[7mz\033[m\r\nA"}, nil},
		"quotes": {`"hello world" 'and you "too"'`, []string{"hello world", "and you \"too\""}, nil},
		"quoted": {`"hello 'nice' world" "you \'too"`, []string{"hello 'nice' world", "you 'too"}, nil},
		"multli": {`test.sl1
exposure --bottom-count 13 --bottom-exposure 120
foo.cbddlp
`, []string{"test.sl1", "exposure", "--bottom-count", "13", "--bottom-exposure", "120", "foo.cbddlp"}, nil},
	}

	os.Setenv("MONKEY", "monkey")

	for key, item := range table {
		reader := bytes.NewReader([]byte(item.In))
		args, err := CommandExpand(reader)
		if err != item.Error {
			t.Errorf("%v: expected %v, got %v", key, item.Error, err)
			continue
		}

		if err != nil {
			continue
		}

		if len(args) != len(item.Out) {
			t.Errorf("%v: expected len() %v, got %v", key, len(item.Out), len(args))
			continue
		}

		for n, arg := range args {
			if arg != item.Out[n] {
				t.Errorf("%v: expected [%v] %v, got %v", key, n, item.Out[n], arg)
				break
			}
		}
	}
}
