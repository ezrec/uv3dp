//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/ezrec/uv3dp"
	_ "github.com/ezrec/uv3dp/cbddlp"
	_ "github.com/ezrec/uv3dp/sl1"
	_ "github.com/ezrec/uv3dp/zcodex"

	"github.com/spf13/pflag"
)

type Verbosity int

const (
	VerbosityWarning = Verbosity(iota)
	VerbosityNotice
	VerbosityInfo
	VerbosityDebug
)

var param struct {
	Verbose int // Verbose counts the number of '-v' flags
}

func TraceVerbosef(level Verbosity, format string, args ...interface{}) {
	if param.Verbose >= int(level) {
		fmt.Printf("<%v>", level)
		fmt.Printf(format+"\n", args...)
	}
}

type Commander interface {
	Parse(args []string) error
	Args() []string
	NArg() int
	PrintDefaults()
	Filter(input uv3dp.Printable) (output uv3dp.Printable, err error)
}

var commandMap = map[string]struct {
	NewCommander func() (cmd Commander)
	Description  string
}{
	"info": {
		NewCommander: func() Commander { return NewInfoCommand() },
		Description:  "Dumps information about the printable",
	},
	"bed": {
		NewCommander: func() Commander { return NewBedCommand() },
		Description:  "Adjust image for a different bed size/resolution",
	},
	"decimate": {
		NewCommander: func() Commander { return NewDecimateCommand() },
		Description:  "Remove outmost pixels of all islands in each layer (reduces over-curing on edges)",
	},
	"exposure": {
		NewCommander: func() Commander { return NewExposureCommand() },
		Description:  "Alters exposure times",
	},
	"bottom": {
		NewCommander: func() Commander { return NewBottomCommand() },
		Description:  "Alters bottom layer exposure",
	},
	"lift": {
		NewCommander: func() Commander { return NewLiftCommand() },
		Description:  "Alters layer lift properties",
	},
	"retract": {
		NewCommander: func() Commander { return NewRetractCommand() },
		Description:  "Alters layer retract properties",
	},
	"resin": {
		NewCommander: func() Commander { return NewResinCommand() },
		Description:  "Changes all properties to match a selected resin",
	},
}

func Usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  uv3dp [options] INFILE [command [options] | OUTFILE]...")
	fmt.Fprintln(os.Stderr, "  uv3dp [options] @cmdfile.cmd")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr)
	pflag.PrintDefaults()
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  %-20s %s\n", "(none)", "Translates input file to output file")

	commands := make(sort.StringSlice, len(commandMap))
	n := 0
	for key := range commandMap {
		commands[n] = key
		n++
	}

	commands.Sort()

	for _, key := range commands {
		item := commandMap[key]
		fmt.Fprintf(os.Stderr, "  %-20s %s\n", key, item.Description)
	}

	for _, key := range commands {
		item := commandMap[key]
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Options for '%s':\n", key)
		fmt.Fprintln(os.Stderr)
		item.NewCommander().PrintDefaults()
	}

	uv3dp.FormatterUsage()

	PrintMachines()
	PrintResins()
}

func init() {
	pflag.CountVarP(&param.Verbose, "verbose", "v", "Verbosity")
	pflag.SetInterspersed(false)
}

func evaluate(args []string) (err error) {
	if len(args) == 0 {
		Usage()
		return
	}

	var input uv3dp.Printable
	var format *uv3dp.Format

	for len(args) > 0 {
		if args[0] == "help" {
			Usage()
			return
		}

		item, found := commandMap[args[0]]
		if !found {
			format, err = uv3dp.NewFormat(args[0], args[1:])
			if err != nil {
				return err
			}
			err = format.Parse(args[1:])
			if err != nil {
				return err
			}
			TraceVerbosef(VerbosityNotice, "%v", args)
			args = format.Args()

			if input == nil {
				// If we have no input, get it from this file
				input, err = format.Printable()
				TraceVerbosef(VerbosityDebug, "%v: Input (err: %v)", format.Filename, err)
				if err != nil {
					return
				}
			} else {
				// Otherwise save the file
				err = format.SetPrintable(input)
				TraceVerbosef(VerbosityDebug, "%v: Output (err: %v)", format.Filename, err)
				if err != nil {
					return
				}
			}
		} else {
			cmd := item.NewCommander()
			err = cmd.Parse(args[1:])
			if err != nil {
				return
			}
			TraceVerbosef(VerbosityNotice, "%v", args)
			args = cmd.Args()

			input, err = cmd.Filter(input)
			if err != nil {
				return
			}
		}
	}

	return
}

func argExpand(in []string) (out []string, err error) {
	for _, arg := range in {
		if len(arg) > 1 && arg[0] == '@' {
			var reader *os.File
			reader, err = os.Open(arg[1:])
			if err != nil {
				return
			}
			defer reader.Close()

			var more []string
			more, err = CommandExpand(reader)
			if err != nil {
				return
			}
			out = append(out, more...)
		} else {
			out = append(out, arg)
		}
	}

	return
}

func main() {
	var err error
	os.Args, err = argExpand(os.Args)
	if err != nil {
		panic(err)
	}

	pflag.Parse()

	err = evaluate(pflag.Args())
	if err != nil {
		panic(err)
	}
}
