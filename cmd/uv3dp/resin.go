//
// Copyright (c) 2020  Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ezrec/uv3dp"
)

// Resin stores information about resin properties
type Resin struct {
	Name string
	uv3dp.Exposure
	uv3dp.Bottom
}

var ResinMap = map[string](*Resin){}

var ResinConfigPath string

func chituboxPath(suffix string) string {

	if runtime.GOOS == "windows" {
		return os.Getenv("LOCALAPPDATA") + strings.ReplaceAll("/ChiTuBox/"+suffix, "/", "\\")
	} else {
		return os.Getenv("HOME") + "/.config/ChiTuBox/" + suffix
	}
}

func setExposureFromDefault(exp *uv3dp.Exposure, def uv3dp.Exposure) {
	if exp.LightOnTime < 0 {
		exp.LightOnTime = def.LightOnTime
	}
	if exp.LightOffTime < 0 {
		exp.LightOffTime = def.LightOffTime
	}
	if exp.LiftHeight < 0 {
		exp.LiftHeight = def.LiftHeight
	}
	if exp.LiftSpeed < 0 {
		exp.LiftSpeed = def.LiftSpeed
	}
	if exp.RetractHeight < 0 {
		exp.RetractHeight = def.RetractHeight
	}
	if exp.RetractSpeed < 0 {
		exp.RetractSpeed = def.RetractSpeed
	}
}

// init initializes the resin map from the ChiTuBox config
func init() {
	ResinConfigPath = chituboxPath("machine/0.cfg")

	reader, err := os.Open(ResinConfigPath)
	if err != nil {
		// This is fine.
		return
	}
	defer reader.Close()

	defExposure := uv3dp.Exposure{
		LightOnTime:   -1,
		LightOffTime:  -1,
		LiftHeight:    -1,
		LiftSpeed:     -1,
		RetractHeight: -1,
		RetractSpeed:  -1}

	defResin := &Resin{
		Name:     "",
		Exposure: defExposure,
		Bottom:   uv3dp.Bottom{Count: -1, Exposure: defExposure, Style: uv3dp.BottomStyleSlow},
	}

	// Set reasonable defaults
	defResin.Exposure.LightOnTime = 6.0
	defResin.Exposure.LightOffTime = 0
	defResin.Bottom.Count = 8
	defResin.Bottom.Exposure.LightOnTime = 50.0
	defResin.Bottom.Exposure.LightOffTime = 0.0
	defResin.Bottom.Exposure.LiftSpeed = 65.0
	defResin.Bottom.Exposure.LiftHeight = 5.0
	defResin.Bottom.Exposure.RetractSpeed = 150.0
	defResin.Bottom.Exposure.RetractHeight = 5.0
	ResinMap[""] = defResin

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "@@") {
			l := strings.SplitN(line[2:], "@@", 2)
			name := l[0]
			av := strings.SplitN(l[1], ":", 2)
			attr := av[0]
			val := av[1]

			resin, ok := ResinMap[name]
			if !ok {
				resin = &Resin{
					Name:     name,
					Exposure: defExposure,
					Bottom:   uv3dp.Bottom{Count: -1, Exposure: defExposure, Style: uv3dp.BottomStyleSlow},
				}
			}

			iVal, _ := strconv.ParseInt(val, 10, 32)
			fVal, _ := strconv.ParseFloat(val, 32)
			tVal := time.Duration(fVal * float64(time.Second))

			switch attr {
			case "bottomLayCount":
				resin.Bottom.Count = int(iVal)
			case "bottomLayerCount":
				resin.Bottom.Count = int(iVal)
			case "bottomLayerLiftSpeed":
				resin.Bottom.Exposure.LiftSpeed = float32(fVal)
			case "bottomLayExposureTime":
				resin.Bottom.Exposure.LightOnTime = tVal
			case "bottomLightOffTime":
				resin.Bottom.Exposure.LightOffTime = tVal
			case "normalExposureTime":
				resin.Exposure.LightOnTime = tVal
			case "normalLayerLiftSpeed":
				resin.Exposure.LiftSpeed = float32(fVal)
			default:
				// Ignored
			}

			ResinMap[name] = resin
		}
	}

	defResin, ok := ResinMap[""]
	if ok {
		delete(ResinMap, "")
		setExposureFromDefault(&defResin.Exposure, defResin.Bottom.Exposure)
		for _, resin := range ResinMap {
			if resin.Bottom.Count < 0 {
				resin.Bottom.Count = defResin.Bottom.Count
			}
			setExposureFromDefault(&resin.Exposure, defResin.Exposure)
			setExposureFromDefault(&resin.Bottom.Exposure, defResin.Bottom.Exposure)
		}
	}
}

func PrintResins() {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Known resins: (from %v)\n", ResinConfigPath)
	fmt.Fprintln(os.Stderr)

	keys := []string{}
	for key := range ResinMap {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		item := ResinMap[key]
		fmt.Fprintf(os.Stderr, "    %-40s bottom %v %v layers, %v; nominal %v\n", key,
			item.Bottom.Count,
			item.Bottom.Style,
			item.Bottom.Exposure.LightOnTime,
			item.Exposure.LightOnTime)
	}
}
