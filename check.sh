#!/bin/bash

# Build reference files
find . -name "*.uvd" -type d | \
    (while read i; do (cd ${i}; zip -r -D  -q -X ../$(basename $i .uvd).uvj .) done)

# Run tests
echo "=== Unit tests"
go test -v ./... -test.failfast || exit 1

if [ "$1" == "test" ]; then
    # Done!
    exit 0
fi

# Conform to formatting
echo "=== Code formatting"
gofmt -w -s -l . || exit 1

# Update README.md
cat <<'EOF' >README.md
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
EOF

go run github.com/ezrec/uv3dp/cmd/uv3dp 2>&1 | sed -e 's|^|    |' \
    -e 's|/home/jmcmullan/.config/ChiTuBox/machine/0.cfg|local user ChiTuBox config|' >>README.md
echo "=== README.md updated"

echo "=== CHECKED"
