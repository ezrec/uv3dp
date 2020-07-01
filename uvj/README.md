# UVJ File Format

## Introduction

The 'UVJ' file format is a vendor-neutral file format for UV curing resin 3D printers,
that use a parallel pixel UV light source (ie LCD masking or DLP projector).

It is comprised of a configuration JSON file, and a number of MxN image slices, and
optional thumbnail preview files. The set of files is wrapped up as a Zip file archive.

## ZIP Directory Layout

| Path | Comments |
| -- | -- |
| `config.json` | Configuration JSON file |
| `slice/`  | Image slice directory (PNG file format, 8-bit greyscale) |
| `slice/00000000.png` | First slice of the object |
| `slice/????????.png` | Nth slice of the object (`????????`  is the `printf` format `%08d` of the slice index) |
| `preview/` | Thumbnail previews (PNG file format, 8bit/color RGB) |
| `preview/huge.png` | Large preview |
| `preview/tiny.png` | Small preview |

## Configuration file

The configuration file is named `config.json`, and is in the JSON syntax.

Top level groups are as follows (required fields are in **bold** ):

| Field | Comment |
| -- | -- |
| **Properties** | Group of properties for the configuration |
| Layers | Optional array of `LayerSetting` to override per-layer exposure settings |

### Properties

| Field | Comment |
| -- | -- |
| **Size** | Group of size definitions for the configuration |
| **Exposure** | Nominal exposure settings |
| **Bottom** | Bottom exposure layer count and default exposure settings |

### Size

| Field | Comment |
| -- | -- |
| **X** | Size of the X dimension, in pixels |
| **Y** | Size of the Y dimension, in pixels |
| **Millimeter.X** | Size of the X dimension, in millimeters |
| **Millimeter.Y** | Size of the Y dimension, in millimeters |
| **Layers** | Total number of slices to print |
| **LayerHeight** | Nominal height per-layer, in mm |

### Exposure

| Field | Comment |
| -- | -- |
| **LightOnTime** | Duration of the UV exposure, in seconds |
| LightOffTime | Amount of time for the light to be off before lift & retract, in seconds |
| LightPWM | PWM (intentity) setting, from 1..255. If not present, the default is 255 (full intensity) |
| LiftHeight | Height to raise for lift/peel move in Z, in millimeters |
| LiftSpeed | Speed for lift/peel move in millimeters/minute |
| RetractHeight | Retraction height, after lift/peel move, in millimeters |
| RetractSPeed | Retraction speed, in millimeters/minute |

### Bottom

| Field | Comment |
| -- | -- |
| **Count** | Number of bottom layers to apply the bottom exposure settings |
| **LightOnTime** | Duration of the UV exposure, in seconds |
| LightOffTime | Amount of time for the light to be off before lift & retract, in seconds |
| LightPWM | PWM (intentity) setting, from 1..255. If not present, the default is 255 (full intensity) |
| LiftHeight | Height to raise for lift/peel move in Z, in millimeters |
| LiftSpeed | Speed for lift/peel move in millimeters/minute |
| RetractHeight | Retraction height, after lift/peel move, in millimeters |
| RetractSPeed | Retraction speed, in millimeters/minute |


### LayerSetting

- This array of per-layer overrides must be either empty, or have _Properties.Size.Layers_ entries.
- All entries must be in increasing Z order.

| Field | Comment |
| -- | -- |
| **Z** | Height in Z to move for this layer |
| **Exposure** | Per-layer exposure override |

## Worked Example A

This example is for a 1440x2560 pixel LCD printer, printing 432 layers at 0.05mm/layer.

The bottom exposure is set for 60 seconds, at 4 layers.

The nominal exposure is set for 11.5 seconds.

No per-layer overrides are set.

### `config.json`

    {
      "Properties": {
        "Size": {
          "X": 1440,
          "Y": 2560,
          "Millimeter": {
            "X": 72.0,
            "Y": 128.0
          },
          "Layers": 432,
          "LayerHeight": 0.05
        },
        "Exposure": {
          "LightOnTime": 11.5,
          "LightOffTime": 3,
          "LiftHeight": 5.5,
          "LiftSpeed": 120,
          "RetractHeight": 4,
          "RetractSpeed": 200
        },
        "Bottom": {
          "LightOnTime": 60,
          "LightOffTime": 3,
          "LiftHeight": 6,
          "LiftSpeed": 50,
          "RetractHeight": 4,
          "RetractSpeed": 200,
          "Count": 4
        }
      }
    }

## Worked Example B (with per-layer overrides)

This example is for a 1080x1920 pixel LCD printer, printing 14 layers at 0.1mm/layer.

The bottom exposure is set for 25 seconds, at 4 layers.

The nominal exposure is set for 3.1 seconds.

Per-layer overrides are set, specifically Layer 2's exposure it set for 20s.

### `config.json`
    {
      "Properties": {
        "Size": {
          "X": 1080,
          "Y": 1920,
          "Millimeter": {
            "X": 68.04,
            "Y": 120.96
          },
          "Layers": 14,
          "LayerHeight": 0.1
        },
        "Exposure": {
          "LightOnTime": 3.1,
          "LightOffTime": 6,
          "LiftHeight": 5,
          "LiftSpeed": 100,
          "RetractHeight": 6,
          "RetractSpeed": 200
        },
        "Bottom": {
          "LightOnTime": 25,
          "LightOffTime": 6,
          "LiftHeight": 10,
          "LiftSpeed": 60,
          "RetractHeight": 6,
          "RetractSpeed": 200,
          "Count": 2,
        }
      },
      "Layers": [
        {
          "Z": 0,
          "Exposure": {
            "LightOnTime": 25,
          }
        },
        {
          "Z": 0.1,
          "Exposure": {
            "LightOnTime": 20,
          }
        },
        {
          "Z": 0.2,
          "Exposure": {
            "LightOnTime": 3.1,
          }
        },
        {
          "Z": 0.3,
          "Exposure": {
            "LightOnTime": 3.1,
          }
        },
        {
          "Z": 0.4,
          "Exposure": {
            "LightOnTime": 3.1,
          }
        },
        {
          "Z": 0.5,
          "Exposure": {
            "LightOnTime": 3.1,
          }
        },
        {
          "Z": 0.6,
          "Exposure": {
            "LightOnTime": 3.1,
          }
        },
        {
          "Z": 0.7,
          "Exposure": {
            "LightOnTime": 3.1,
            "LightOffTime": 6,
            "LightPWM": 255,
            "LiftHeight": 5,
            "LiftSpeed": 100,
            "RetractHeight": 6,
            "RetractSpeed": 200
          }
        },
        {
          "Z": 0.8,
          "Exposure": {
            "LightOnTime": 3.1,
            "LightOffTime": 6,
            "LightPWM": 255,
            "LiftHeight": 5,
            "LiftSpeed": 100,
            "RetractHeight": 6,
            "RetractSpeed": 200
          }
        },
        {
          "Z": 0.90000004,
          "Exposure": {
            "LightOnTime": 3.1,
            "LightOffTime": 6,
            "LightPWM": 255,
            "LiftHeight": 5,
            "LiftSpeed": 100,
            "RetractHeight": 6,
            "RetractSpeed": 200
          }
        },
        {
          "Z": 1,
          "Exposure": {
            "LightOnTime": 3.1,
            "LightOffTime": 6,
            "LightPWM": 255,
            "LiftHeight": 5,
            "LiftSpeed": 100,
            "RetractHeight": 6,
            "RetractSpeed": 200
          }
        },
        {
          "Z": 1.1,
          "Exposure": {
            "LightOnTime": 3.1,
            "LightOffTime": 6,
            "LightPWM": 255,
            "LiftHeight": 5,
            "LiftSpeed": 100,
            "RetractHeight": 6,
            "RetractSpeed": 200
          }
        },
        {
          "Z": 1.2,
          "Exposure": {
            "LightOnTime": 3.1,
            "LightOffTime": 6,
            "LightPWM": 255,
            "LiftHeight": 5,
            "LiftSpeed": 100,
            "RetractHeight": 6,
            "RetractSpeed": 200
          }
        },
        {
          "Z": 1.3000001,
          "Exposure": {
            "LightOnTime": 3.1,
            "LightOffTime": 6,
            "LightPWM": 255,
            "LiftHeight": 5,
            "LiftSpeed": 100,
            "RetractHeight": 6,
            "RetractSpeed": 200
          }
        }
      ]
    }
