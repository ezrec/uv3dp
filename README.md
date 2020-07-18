# uv3dp
Tools for UV Resin based 3D Printers (in Go)

## Supported File Formats

This tool is for devices that use sliced image files for UV resin 3D printers.

Printers known to work with this tool:

| Printer          | File Formats | Issues                                            |
| ---------------- | ------------ | --------------------------------------------------|
| -                | uvj          | Zip file with JSON and image slices               |
| EPAX X1/X10      | cbddlp       | None                                              |
| EPAX X1-N        | ctb          | None                                              |
| Anycubic Photon  | photon       | None                                              |
| Anycubic Zero    | pw0          | None                                              |
| Anycubic Photons | pws          | None                                              |
| Prusa SL1        | sl1          | None                                              |
| NOVA3D Elfin     | cws          | None                                              |
| Phrozen Sonic    | phz          | None                                              |
| Zortrax Inkspire | zcodex       | Read-only (for format conversion)                 |

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
    
      -p, --progress        Show progress during operations
      -v, --verbose count   Verbosity
      -V, --version         Show version
    
    Commands:
    
      (none)               Translates input file to output file
      bed                  Adjust image for a different bed size/resolution
      bottom               Alters bottom layer exposure
      decimate             Remove outmost pixels of all islands in each layer (reduces over-curing on edges)
      exposure             Alters exposure times
      info                 Dumps information about the printable
      lift                 Alters layer lift properties
      resin                Changes all properties to match a selected resin
      retract              Alters layer retract properties
      select               Select to print only a range of layers
    
    Options for 'bed':
    
      -M, --machine string             Size preset by machine type (default "EPAX-X1")
      -m, --millimeters float32Slice   Bed size, in millimeters (default [68.040001,120.959999])
      -p, --pixels ints                Bed size, in pixels (default [1440,2560])
      -r, --reflect                    Mirror image along the X axis
    
    Options for 'bottom':
    
      -c, --count int             Bottom layer count
      -h, --lift-height float32   Bottom layer lift height in mm
      -s, --lift-speed float32    Bottom layer lift speed in mm/min
      -f, --light-off float32     Bottom layer light-off time in seconds
      -o, --light-on float32      Bottom layer light-on time in seconds
      -p, --pwm uint8             Light PWM rate (0..255) (default 255)
      -y, --style string          Bottom layer style - 'fade' or 'slow' (default "slow")
    
    Options for 'decimate':
    
      -b, --bottom int   Number of bottom layer passes
      -n, --normal int   Number of normal layer passes (default 1)
    
    Options for 'exposure':
    
      -f, --light-off float32   Normal layer light-off time in seconds
      -o, --light-on float32    Normal layer light-on time in seconds
      -p, --pwm uint8           Light PWM rate (0..255) (default 255)
    
    Options for 'info':
    
      -e, --exposure   Show summary of the exposure settings (default true)
      -l, --layer      Show layer detail
      -s, --size       Show size summary (default true)
    
    Options for 'lift':
    
      -h, --height float32   Lift height in mm
      -s, --speed float32    Lift speed in mm/min
    
    Options for 'resin':
    
      -t, --type string   Resin type [see 'Known resins' in help]
    
    Options for 'retract':
    
      -h, --height float32   Retract height in mm
      -s, --speed float32    Retract speed in mm/min
    
    Options for 'select':
    
      -c, --count int   Count of layers to select (-1 for all layers after first) (default -1)
      -f, --first int   First layer to select
    
    Options for '.cbddlp':
    
      -a, --anti-alias int   Override antialias level (1..16) (default 1)
      -v, --version int      Override header Version (default 2)
    
    Options for '.ctb':
    
      -e, --encryption-seed uint32   Specify a specific encryption seed
    
    Options for '.cws':
    
    
    Options for '.photon':
    
      -a, --anti-alias int   Override antialias level (1..16) (default 1)
      -v, --version int      Override header Version (default 1)
    
    Options for '.phz':
    
      -e, --encryption-seed uint32   Specify a specific encryption seed
    
    Options for '.pw0':
    
      -a, --anti-alias int   Override antialias level (1,2,4,8) (default 1)
    
    Options for '.pws':
    
      -a, --anti-alias int   Override antialias level (1,2,4,8) (default 1)
    
    Options for '.sl1':
    
      -f, --bottom-fade            Fade bottom layers exposure time
      -m, --material-name string   config.init entry 'materialName' (default "3DM-ABS @")
    
    Options for '.uvj':
    
    
    Options for '.zcodex':
    
    
    Options for 'empty':
    
      -M, --machine string             Size preset by machine type (default "EPAX-X1")
      -m, --millimeters float32Slice   Empty size, in millimeters (default [68.040001,120.959999])
      -p, --pixels ints                Empty size, in pixels (default [1440,2560])
    
    Known machines:
    
        Anycubic-Photon      1440x2560, 68x121 mm
        Anycubic-Photon-S    1440x2560, 68x121 mm
        Anycubic-Photon-Zero 480x854, 55.4x98.6 mm
        EPAX-X1              1440x2560, 68x121 mm
        EPAX-X10             1600x2560, 135x216 mm
        EPAX-X133            2160x3840, 165x293 mm
        EPAX-X156            2160x3840, 194x345 mm
        EPAX-X9              1600x2560, 120x192 mm
        Elogoo-Mars          1440x2560, 68x121 mm
        Kelant-S400          2560x1600, 192x120 mm
        Nova3D-Elfin         1410x2550, 73x132 mm
        Phrozen-Sonic-Mini   1080x1920, 68x121 mm
        Zortrax-Inkspire     1440x2560, 72x128 mm
    
    Known resins: (from local user ChiTuBox config)
    
