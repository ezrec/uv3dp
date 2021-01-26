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
      -v, --version int              Specify the CTB version (2 or 3) (default 3)
    
    Options for '.cws':
    
    
    Options for '.fdg':
    
      -e, --encryption-seed uint32   Specify a specific encryption seed
      -v, --version int              Specify the CTB version (2 or 3) (default 2)
    
    Options for '.lgs':
    
    
    Options for '.lgs30':
    
    
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
    
      -m, --material-name string   config.init entry 'materialName' (default "3DM-ABS @")
    
    Options for '.uvj':
    
    
    Options for '.zcodex':
    
    
    Options for '.zip':
    
    
    Options for 'empty':
    
      -g, --gray uint8                 Grayscale color (0 for black, 255 for white)
      -l, --layers int                 Number of 0.05mm layers (default 1)
      -M, --machine string             Size preset by machine type (default "photon")
      -m, --millimeters float32Slice   Empty size, in millimeters (default [68.040001,120.959999])
      -p, --pixels ints                Empty size, in pixels (default [1440,2560])
    
    Known machines:
    
        e10-4k                 EPAX E10 mono 4K      Size: 2400x3840, 120x192 mm,	Format: .ctb --version=3
        e10-5k                 EPAX E10 mono 5K      Size: 2880x4920, 135x216 mm,	Format: .ctb --version=3
        e6                     EPAX E6 mono          Size: 1620x2560, 81x128 mm,	Format: .ctb --version=3
        elfin                Nova3D Elfin            Size: 1410x2550, 73x132 mm,	Format: .cws 
        inkspire            Zortrax Inkspire         Size: 1440x2560, 72x128 mm,	Format: .zcodex 
        ld-002r            Creality LD-002R          Size: 1440x2560, 68x121 mm,	Format: .ctb --version=2
        mars                 Elegoo Mars             Size: 1440x2560, 68x121 mm,	Format: .cbddlp 
        mars2-pro            Elegoo Mars 2 Pro       Size: 1620x2560, 82.6x131 mm,	Format: .ctb --version=3
        orange10             Longer Orange 10        Size: 480x854, 55.4x98.6 mm,	Format: .lgs 
        orange30             Longer Orange 30        Size: 1440x2560, 68x121 mm,	Format: .lgs30 
        photon             Anycubic Photon           Size: 1440x2560, 68x121 mm,	Format: .photon 
        photon0            Anycubic Photon Zero      Size: 480x854, 55.4x98.6 mm,	Format: .pw0 
        photons            Anycubic Photon S         Size: 1440x2560, 68x121 mm,	Format: .pws 
        polaris             Voxelab Polaris          Size: 1440x2560, 68x121 mm,	Format: .fdg 
        s400                 Kelant S400             Size: 2560x1600, 192x120 mm,	Format: .zip 
        shuffle             Phrozen Shuffle          Size: 1440x2560, 67.7x120 mm,	Format: .zip 
        sl1                   Prusa SL1              Size: 1440x2560, 68x121 mm,	Format: .sl1 
        sonic-mini          Phrozen Sonic Mini       Size: 1080x1920, 68x121 mm,	Format: .phz 
        sonic-mini-4k       Phrozen Sonic Mini 4K    Size: 3840x2160, 134x75.6 mm,	Format: .ctb --version=3
        x1                     EPAX X1               Size: 1440x2560, 68x121 mm,	Format: .cbddlp 
        x10                    EPAX X10              Size: 1600x2560, 135x216 mm,	Format: .cbddlp 
        x10n                   EPAX X10              Size: 1600x2560, 135x216 mm,	Format: .ctb --version=2
        x133                   EPAX X133             Size: 2160x3840, 165x293 mm,	Format: .cbddlp 
        x156                   EPAX X156             Size: 2160x3840, 194x345 mm,	Format: .cbddlp 
        x1k                    EPAX X1K              Size: 1440x2560, 68x121 mm,	Format: .ctb --version=2
        x1n                    EPAX X1N              Size: 1440x2560, 68x121 mm,	Format: .ctb --version=2
        x9                     EPAX X9               Size: 1600x2560, 120x192 mm,	Format: .cbddlp 
    
    Known resins: (from local user ChiTuBox config)
    
        Profile                                  bottom 5 layers, 50; nominal 15
        Voxelab Black for 0.05mm                 bottom 6 layers, 50; nominal 10
        Voxelab Green for 0.05mm                 bottom 6 layers, 50; nominal 10
        Voxelab Grey for 0.05mm                  bottom 6 layers, 50; nominal 8
        Voxelab Red for 0.05mm                   bottom 6 layers, 50; nominal 10
        Voxelab Transparent for 0.05mm           bottom 6 layers, 50; nominal 10
        Voxelab White for 0.05mm                 bottom 6 layers, 50; nominal 9
