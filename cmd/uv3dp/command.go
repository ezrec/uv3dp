package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func ScanArgs(data []byte, atEOF bool) (advance int, token []byte, err error) {
	isSpace := func(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }
	skip := 0
	for ; skip < len(data) && isSpace(data[skip]); skip++ {
	}

	data = data[skip:]
	if len(data) == 0 {
		advance = skip
		return
	}

	var new_token []byte

	in_quote := false
	in_dquote := false
	in_escape := false
	oct := 0
	in_oct := 0

	for here, c := range data {
		if in_escape {
			switch c {
			case 'b':
				new_token = append(new_token, '\b')
			case 't':
				new_token = append(new_token, '\t')
			case 'n':
				new_token = append(new_token, '\n')
			case 'r':
				new_token = append(new_token, '\r')
			case '"':
				new_token = append(new_token, '"')
			case '\'':
				new_token = append(new_token, '\'')
			case ' ':
				new_token = append(new_token, ' ')
			case '\\':
				new_token = append(new_token, '\\')
			case 'e':
				new_token = append(new_token, '\033')
			case '0', '1', '2', '3', '4', '5', '6', '7':
				oct = (oct * 8) + int(c-'0')
				in_oct++
				if in_oct == 3 {
					new_token = append(new_token, byte(oct))
					oct = 0
					in_oct = 0
				}
			default:
				if in_oct > 0 {
					new_token = append(new_token, byte(oct))
					oct = 0
					in_oct = 0
				}
				new_token = append(new_token, c)
			}
			in_escape = (in_oct != 0)
		} else {
			switch c {
			case '"':
				if !in_quote {
					in_dquote = !in_dquote
				} else {
					new_token = append(new_token, c)
				}
			case '\'':
				if !in_dquote {
					in_quote = !in_quote
				} else {
					new_token = append(new_token, c)
				}
			case '\\':
				in_escape = true
			case ' ', '\t', '\r', '\n':
				if !in_dquote && !in_quote {
					advance = skip + here
					token = new_token
					return
				} else {
					new_token = append(new_token, c)
				}
			default:
				new_token = append(new_token, c)
			}
		}
	}

	if in_escape && in_oct > 0 {
		new_token = append(new_token, byte(oct))
		in_escape = false
	}

	if !in_dquote && !in_escape && !in_quote {
		advance = skip + len(data)
		if len(new_token) > 0 {
			token = new_token
		}
		return
	}

	if atEOF {
		err = fmt.Errorf("incomplete line: '%v' => '%v'", string(data), string(new_token))
	}

	return
}

func CommandExpand(reader io.Reader) (out []string, err error) {
	var lines []string
	scanner := bufio.NewScanner(reader)
	scanner.Split(ScanArgs)
	for scanner.Scan() {
		text := scanner.Text()
		expanded := os.ExpandEnv(text)
		lines = append(lines, expanded)
	}
	err = scanner.Err()
	if err != nil {
		return
	}

	out = lines

	return
}
