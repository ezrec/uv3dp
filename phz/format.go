//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package phz

import (
	"fmt"
	"image"
	"io/ioutil"
	"sort"
	"time"

	"encoding/binary"

	"github.com/go-restruct/restruct"
	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

const (
	defaultHeaderMagic = uint32(0x9fda83ae)

	defaultBottomLiftHeight = 5.0
	defaultBottomLiftSpeed  = 300.0
	defaultLiftHeight       = 5.0
	defaultLiftSpeed        = 300.0
	defaultRetractSpeed     = 300.0
	defaultRetractHeight    = 6.0
	defaultBottomLightOff   = 1.0
	defaultLightOff         = 1.0

	forceBedSizeMM_3 = 155.0
)

type phzHeader struct {
	Magic          uint32    // 00:
	Version        uint32    // 04: Always '2'
	LayerHeight    float32   // 08
	LayerExposure  float32   // 0c: Layer exposure (in seconds)
	BottomExposure float32   // 10: Bottom layers exporsure (in seconds)
	BottomCount    uint32    // 14: Number of bottom layers
	ResolutionX    uint32    // 18:
	ResolutionY    uint32    // 1c:
	PreviewHigh    uint32    // 20: Offset of the high-res preview
	LayerDefs      uint32    // 24: Offset of the layer definitions
	LayerCount     uint32    // 28:
	PreviewLow     uint32    // 2c: Offset of the low-rew preview
	PrintTime      uint32    // 30: In seconds
	Projector      uint32    // 34: 0 = CAST, 1 = LCD_X_MIRROR
	AntiAliasLevel uint32    // 38: Always 1 for this format
	LightPWM       uint16    // 3C:
	BottomLightPWM uint16    // 3E:
	_              [2]uint32 // 40:

	HeightMM           float32    // 48:
	BedSizeMM          [3]float32 // 4C:
	EncryptionSeed     uint32     // 58: Compressed grayscale image encryption key
	BottomLightOffTime float32    // 5c:
	LayerOffTime       float32    // 60: Layer off time (in seconds)
	BottomLayerCount   uint32     // 64:
	_                  uint32     // 68:
	BottomLiftHeight   float32    // 6c:
	BottomLiftSpeed    float32    // 70:

	LiftHeight   float32 // 74:
	LiftSpeed    float32 // 78:
	RetractSpeed float32 // 7c:

	VolumeMilliliters float32 // 80:
	WeightGrams       float32 // 84:
	CostDollars       float32 // 88:
	_                 uint32  // 8c:
	MachineOffset     uint32  // 90: Machine name offset
	MachineSize       uint32  // 94: Machine name length
	_                 [6]uint32
	EncryptionMode    uint32  // b0: Always 0xf for CTB
	_                 float32 // b4
	_                 uint32  // b8
	ChiTuBoxVersion   [4]byte // bc: release, patch, minor, major
	_                 [6]uint32
}

type phzPreview struct {
	ResolutionX uint32    // 00:
	ResolutionY uint32    // 04:
	ImageOffset uint32    // 08:
	ImageLength uint32    // 0c:
	_           [4]uint32 // 10:
}

type phzLayerDef struct {
	LayerHeight   float32   // 00:
	LayerExposure float32   // 04:
	LayerOffTime  float32   // 08:
	ImageOffset   uint32    // 0c:
	ImageLength   uint32    // 10:
	_             [4]uint32 // 14:
}

type Print struct {
	uv3dp.Print
	layerDef []phzLayerDef

	rleMap map[uint32]([]byte)
}

type Formatter struct {
	*pflag.FlagSet

	EncryptionSeed uint32
}

func NewFormatter(suffix string) (pf *Formatter) {
	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)
	flagSet.SetInterspersed(false)

	pf = &Formatter{
		FlagSet: flagSet,
	}

	pf.Uint32VarP(&pf.EncryptionSeed, "encryption-seed", "e", 0, "Specify a specific encryption seed")

	return
}

// Save a uv3dp.Printable in CBD DLP format
func (pf *Formatter) Encode(writer uv3dp.Writer, printable uv3dp.Printable) (err error) {
	size := printable.Size()
	exp := printable.Exposure()
	bot := printable.Bottom()

	mach, ok := printable.Metadata("Machine")
	if !ok {
		mach = "default"
	}

	// First, compute the rle images
	type rleInfo struct {
		offset uint32
		rle    []byte
	}
	rleHash := map[uint64]rleInfo{}

	// Select an encryption seed
	// A zero encryption seed is permitted by the printer
	seed := pf.EncryptionSeed

	headerBase := uint32(0)
	header := phzHeader{
		Magic:          defaultHeaderMagic,
		Version:        2,
		EncryptionSeed: seed, // Force encryption off, so we can de-duplicate layers
	}
	headerSize, _ := restruct.SizeOf(&header)

	// Add the preview images
	var previewHuge phzPreview
	var previewTiny phzPreview
	previewSize, _ := restruct.SizeOf(&previewHuge)

	// Set up the RLE hash indexes
	rleHashList := []uint64{}

	savePreview := func(base uint32, preview *phzPreview, ptype uv3dp.PreviewType) uint32 {
		pic, found := printable.Preview(ptype)
		if !found {
			return base
		}

		base += uint32(previewSize)
		size := pic.Bounds().Size()
		if size == image.Pt(0, 0) {
			return base
		}

		// Collect preview images
		rle, hash := rleEncodeRGB15(pic)
		if len(rle) == 0 {
			return base
		}

		rleHash[hash] = rleInfo{offset: base, rle: rle}
		rleHashList = append(rleHashList, hash)

		preview.ResolutionX = uint32(size.X)
		preview.ResolutionY = uint32(size.Y)
		preview.ImageOffset = rleHash[hash].offset
		preview.ImageLength = uint32(len(rle))

		return base + uint32(len(rle))
	}

	previewHugeBase := headerBase + uint32(headerSize)

	previewTinyBase := savePreview(previewHugeBase, &previewHuge, uv3dp.PreviewTypeHuge)

	machineBase := savePreview(previewTinyBase, &previewTiny, uv3dp.PreviewTypeTiny)
	machine, _ := mach.(string)
	machineSize := len(machine)

	layerDefBase := machineBase + uint32(machineSize)
	layerDef := make([]phzLayerDef, size.Layers)
	layerDefSize, _ := restruct.SizeOf(&phzLayerDef{})

	// And then all the layer images
	layerPage := uint32(layerDefSize * size.Layers)
	imageBase := layerDefBase + layerPage
	totalOn := uint64(0)

	type layerInfo struct {
		Z        float32
		Exposure uv3dp.Exposure
		Rle      []byte
		Hash     uint64
		BitsOn   uint
	}

	doneMap := make([]chan layerInfo, size.Layers)
	for n := 0; n < size.Layers; n++ {
		doneMap[n] = make(chan layerInfo, 1)
	}

	uv3dp.WithAllLayers(printable, func(p uv3dp.Printable, n int) {
		rle, hash, bitsOn := rleEncodeGraymap(p.LayerImage(n))
		doneMap[n] <- layerInfo{
			Z:        p.LayerZ(n),
			Exposure: p.LayerExposure(n),
			Rle:      rle,
			Hash:     hash,
			BitsOn:   bitsOn,
		}
		close(doneMap[n])
	})

	for n := 0; n < size.Layers; n++ {
		info := <-doneMap[n]
		if header.EncryptionSeed != 0 {
			info.Hash = uint64(n)
			info.Rle = cipher(header.EncryptionSeed, uint32(n), info.Rle)
		}
		_, ok := rleHash[info.Hash]
		if !ok {
			rleHash[info.Hash] = rleInfo{offset: imageBase, rle: info.Rle}
			rleHashList = append(rleHashList, info.Hash)
			imageBase = imageBase + uint32(len(info.Rle))
		}

		layerDef[n] = phzLayerDef{
			LayerHeight:   info.Z,
			LayerExposure: info.Exposure.LightOnTime,
			LayerOffTime:  info.Exposure.LightOffTime,
			ImageOffset:   rleHash[info.Hash].offset,
			ImageLength:   uint32(len(info.Rle)),
		}

		totalOn += uint64(info.BitsOn)
	}

	// phzHeader
	header.BedSizeMM[0] = size.Millimeter.X
	header.BedSizeMM[1] = size.Millimeter.Y
	header.BedSizeMM[2] = forceBedSizeMM_3
	header.HeightMM = size.LayerHeight * float32(size.Layers)
	header.LayerHeight = size.LayerHeight
	header.LayerExposure = exp.LightOnTime
	header.BottomExposure = bot.Exposure.LightOnTime
	header.LayerOffTime = exp.LightOffTime
	header.BottomCount = uint32(bot.Count)
	header.ResolutionX = uint32(size.X)
	header.ResolutionY = uint32(size.Y)
	header.PreviewHigh = previewHugeBase
	header.LayerDefs = layerDefBase
	header.LayerCount = uint32(size.Layers)
	header.PreviewLow = previewTinyBase
	header.PrintTime = uint32(uv3dp.PrintDuration(printable) / time.Second)
	header.Projector = 1 // LCD_X_MIRROR

	header.AntiAliasLevel = 1

	if exp.LightPWM == 0 {
		exp.LightPWM = 255
	}

	if bot.Exposure.LightPWM == 0 {
		bot.Exposure.LightPWM = 255
	}

	header.LightPWM = uint16(exp.LightPWM)
	header.BottomLightPWM = uint16(bot.Exposure.LightPWM)

	header.BottomLayerCount = uint32(bot.Count)
	header.BottomLiftSpeed = bot.Exposure.LiftSpeed
	header.BottomLiftHeight = bot.Exposure.LiftHeight
	header.LiftHeight = exp.LiftHeight
	header.LiftSpeed = exp.LiftSpeed
	header.RetractSpeed = exp.RetractSpeed

	if header.BottomLiftSpeed < 0 {
		header.BottomLiftSpeed = defaultBottomLiftSpeed
	}
	if header.BottomLiftHeight < 0 {
		header.BottomLiftHeight = defaultBottomLiftHeight
	}
	if header.LiftHeight < 0 {
		header.LiftHeight = defaultLiftHeight
	}
	if header.LiftSpeed < 0 {
		header.LiftSpeed = defaultLiftSpeed
	}
	if header.RetractSpeed < 0 {
		header.RetractSpeed = defaultRetractSpeed
	}

	// phzSlicer
	header.MachineOffset = machineBase
	header.MachineSize = uint32(machineSize)
	header.EncryptionMode = 0x1c  // Magic!
	header.ChiTuBoxVersion[3] = 1 // Magic!
	header.ChiTuBoxVersion[2] = 6
	header.ChiTuBoxVersion[1] = 3

	// Compute total cubic millimeters (== milliliters) of all the on pixels
	bedArea := float64(header.BedSizeMM[0] * header.BedSizeMM[1])
	bedPixels := uint64(header.ResolutionX) * uint64(header.ResolutionY)
	pixelVolume := float64(header.LayerHeight) * bedArea / float64(bedPixels)
	header.VolumeMilliliters = float32(float64(totalOn) * pixelVolume / 1000.0)

	header.BottomLightOffTime = bot.Exposure.LightOffTime
	header.BottomLayerCount = header.BottomCount

	// Collect file data
	fileData := map[int][]byte{}

	fileData[int(headerBase)], _ = restruct.Pack(binary.LittleEndian, &header)

	fileData[int(machineBase)] = ([]byte)(machine)

	for n, layer := range layerDef {
		base := int(layerDefBase) + layerDefSize*n
		fileData[base], _ = restruct.Pack(binary.LittleEndian, &layer)
	}

	fileData[int(previewHugeBase)], _ = restruct.Pack(binary.LittleEndian, &previewHuge)
	fileData[int(previewTinyBase)], _ = restruct.Pack(binary.LittleEndian, &previewTiny)

	for _, hash := range rleHashList {
		info := rleHash[hash]
		fileData[int(info.offset)] = info.rle
	}

	// Sort the file data
	fileIndex := []int{}
	for key := range fileData {
		fileIndex = append(fileIndex, key)
	}

	sort.Ints(fileIndex)

	offset := 0
	for _, base := range fileIndex {
		// Pad as needed
		writer.Write(make([]byte, base-offset))

		// Write the data
		data := fileData[base]
		delete(fileData, base)

		writer.Write(data)

		// Set up next offset
		offset = base + len(data)
	}

	return
}

func cipher(seed uint32, slice uint32, in []byte) (out []byte) {
	if seed == 0 {
		out = in
	} else {
		kr := NewKeyring(seed, slice)

		for _, c := range in {
			out = append(out, c^kr.Next())
		}
	}

	return
}

func (pf *Formatter) Decode(file uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {
	// Collect file
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	prop := uv3dp.Properties{
		Preview:  make(map[uv3dp.PreviewType]image.Image),
		Metadata: make(map[string]interface{}),
	}

	header := phzHeader{}
	err = restruct.Unpack(data, binary.LittleEndian, &header)
	if err != nil {
		return
	}

	if header.Magic != defaultHeaderMagic {
		err = fmt.Errorf("Unknown header magic: 0x%08x", header.Magic)
		return
	}

	// Machine Name
	mach := string(data[header.MachineOffset : header.MachineOffset+header.MachineSize])
	if len(mach) > 0 {
		prop.Metadata["Machine"] = mach
	}

	// Collect previews
	previewTable := []struct {
		previewType   uv3dp.PreviewType
		previewOffset uint32
	}{
		{previewType: uv3dp.PreviewTypeTiny, previewOffset: header.PreviewLow},
		{previewType: uv3dp.PreviewTypeHuge, previewOffset: header.PreviewHigh},
	}

	for _, item := range previewTable {
		if item.previewOffset == 0 {
			continue
		}

		var preview phzPreview
		err = restruct.Unpack(data[item.previewOffset:], binary.LittleEndian, &preview)
		if err != nil {
			return
		}

		addr := preview.ImageOffset
		size := preview.ImageLength

		bounds := image.Rect(0, 0, int(preview.ResolutionX), int(preview.ResolutionY))
		var pic image.Image
		pic, err = rleDecodeRGB15(bounds, data[addr:addr+size])
		if err != nil {
			return
		}

		prop.Preview[item.previewType] = pic
	}

	seed := header.EncryptionSeed

	// Collect layers
	rleMap := make(map[uint32]([]byte))

	layerDef := make([]phzLayerDef, header.LayerCount)

	layerDefSize := uint32(9 * 4)
	for n := uint32(0); n < header.LayerCount; n++ {
		offset := header.LayerDefs + layerDefSize*n
		err = restruct.Unpack(data[offset:], binary.LittleEndian, &layerDef[n])
		if err != nil {
			return
		}

		addr := layerDef[n].ImageOffset
		size := layerDef[n].ImageLength

		rleMap[addr] = cipher(seed, n, data[addr:addr+size])
	}

	size := &prop.Size
	size.Millimeter.X = header.BedSizeMM[0]
	size.Millimeter.Y = header.BedSizeMM[1]

	size.X = int(header.ResolutionX)
	size.Y = int(header.ResolutionY)

	size.Layers = int(header.LayerCount)
	size.LayerHeight = header.LayerHeight

	exp := &prop.Exposure
	exp.LightOnTime = header.LayerExposure
	exp.LightOffTime = header.LayerOffTime
	exp.LightPWM = uint8(header.LightPWM)

	bot := &prop.Bottom
	bot.Count = int(header.BottomCount)
	bot.Exposure.LightOnTime = header.BottomExposure
	bot.Exposure.LightOffTime = header.LayerOffTime
	bot.Exposure.LightPWM = uint8(header.BottomLightPWM)

	bot.Count = int(header.BottomLayerCount)
	bot.Exposure.LiftHeight = header.BottomLiftHeight
	bot.Exposure.LiftSpeed = header.BottomLiftSpeed
	bot.Exposure.RetractSpeed = header.RetractSpeed
	bot.Exposure.RetractHeight = defaultRetractHeight

	exp.LiftHeight = header.LiftHeight
	exp.LiftSpeed = header.LiftSpeed
	exp.RetractSpeed = header.RetractSpeed
	exp.RetractHeight = defaultRetractHeight

	prop.Metadata["Machine"] = mach

	phz := &Print{
		Print:    uv3dp.Print{Properties: prop},
		layerDef: layerDef,
		rleMap:   rleMap,
	}

	printable = phz

	return
}

// Layer gets a layer - we decode from the RLE on-the fly
func (phz *Print) LayerImage(index int) (layerImage *image.Gray) {
	layerDef := phz.layerDef[index]

	// Update per-layer info
	layerImage, err := rleDecodeGraymap(phz.Bounds(), phz.rleMap[layerDef.ImageOffset])
	if err != nil {
		panic(err)
	}

	return
}

func (phz *Print) LayerExposure(index int) (exposure uv3dp.Exposure) {
	layerDef := phz.layerDef[index]

	exposure = phz.Print.LayerExposure(index)

	if layerDef.LayerExposure > 0.0 {
		exposure.LightOnTime = layerDef.LayerExposure
	}

	if layerDef.LayerOffTime > 0.0 {
		exposure.LightOffTime = layerDef.LayerOffTime
	}

	return
}

func (phz *Print) LayerZ(index int) (z float32) {
	z = phz.layerDef[index].LayerHeight
	return
}
