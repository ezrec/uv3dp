# uv3dp
Tools for UV Resin based 3D Printers (in Go)

## Supported File Formats

This tool is for devices that use the Prusa SL1 (`*.sla`) and ChiTuBox DLP (`*.cbddlp`) format files.

Printers known to work with this tool:

| Printer      | File Formats | Issues                                            |
| ------------ | ------------ | --------------------------------------------------|
| EPAX X-1     | cbddlp       | None                                              |

## Installation

* Release package: [https://github.com/ezrec/uv3dp/releases](https://github.com/ezrec/uv3dp/releases)
* Go install: `go get github.com/ezrec/uv3dp/cmd/uv3dp; ${GOROOT}/bin/uv3dp`

## Command Line Tool (`uv3dp`)

The command line tool is designed to be used in a 'pipeline' style, for example:

    uv3dp foo.sl1 info                    # Shows information about the SL1 file
    uv3dp foo.sl1 decimate bar.cbddlp     # Convert and decimates a SL1 file to a CBDDLP file
    uv3dp foo.sl1 qux.cbddlp --version 1  # Convert a SL1 file to a Version 1CBDDLP file

### Command summary:
    Usage:
    
      uv3dp [options] INFILE [command [options] | OUTFILE]...
      uv3dp [options] @cmdfile.cmd
    
    Options:
    
      -v, --verbose count   Verbosity
    
    Commands:
    
      (none)               Translates input file to output file
      bed                  Adjust image for a different bed size/resolution
      decimate             Remove outmost pixels of all islands in each layer (reduces over-curing on edges)
      exposure             Alters exposure times
      info                 Dumps information about the printable
    
    Options for 'bed':
    
      -M, --machine string             Size preset by machine type (default "EPAX-X1")
      -m, --millimeters float32Slice   Bed size, in millimeters (default [68.040001,120.959999])
      -p, --pixels ints                Bed size, in pixels (default [1440,2560])
    
      Machines:
    
        Anycubic-Photon      1440x2560, 68x121 mm
        EPAX-X1              1440x2560, 68x121 mm
        EPAX-X10             1600x2560, 135x216 mm
        EPAX-X133            2160x3840, 165x293 mm
        EPAX-X156            2160x3840, 194x345 mm
        EPAX-X9              1600x2560, 120x192 mm
        Elogoo-Mars          1440x2560, 68x121 mm
    
    Options for 'decimate':
    
    
    Options for 'exposure':
    
      -c, --bottom-count uint          Bottom layer count
      -b, --bottom-exposure duration   Bottom layer light-on time
      -s, --bottom-style string        Bottom layer style - 'fade' or 'slow' (default "slow")
      -e, --exposure duration          Normal layer light-on time
    
    Options for 'info':
    
      -e, --exposure   Show summary of the exposure settings (default true)
      -l, --layer      Show layer detail
      -s, --size       Show size summary (default true)
    
    Options for '.cbddlp':
    
      -a, --anti-alias int   Override antialias level (1,2,4,8) (default 4)
      -v, --version int      Override header Version (default 2)
    
    Options for '.photon':
    
      -a, --anti-alias int   Override antialias level (1,2,4,8) (default 1)
      -v, --version int      Override header Version (default 1)
    
    Options for '.sl1':
    
      -m, --material-name string   config.init entry 'materialName' (default "3DM-ABS @")
    
    Options for 'empty':
    
      -M, --machine string             Size preset by machine type (default "EPAX-X1")
      -m, --millimeters float32Slice   Empty size, in millimeters (default [68.040001,120.959999])
      -p, --pixels ints                Empty size, in pixels (default [1440,2560])
    
      Machines:
    
        Anycubic-Photon      1440x2560, 68x121 mm
        EPAX-X1              1440x2560, 68x121 mm
        EPAX-X10             1600x2560, 135x216 mm
        EPAX-X133            2160x3840, 165x293 mm
        EPAX-X156            2160x3840, 194x345 mm
        EPAX-X9              1600x2560, 120x192 mm
        Elogoo-Mars          1440x2560, 68x121 mm
