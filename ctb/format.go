//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package ctb

import (
	"fmt"
	"image"
	"io/ioutil"
	"math"
	"math/rand"
	"sort"
	"time"

	"encoding/binary"

	"github.com/go-restruct/restruct"
	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

const (
	defaultHeaderMagic = uint32(0x12fd0086)

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

type ctbHeader struct {
	Magic          uint32     // 00:
	Version        uint32     // 04: Always '2'
	BedSizeMM      [3]float32 // 08:
	_              [2]uint32  // 14:
	HeightMM       float32    // 1c:
	LayerHeight    float32    // 20
	LayerExposure  float32    // 24: Layer exposure (in seconds)
	BottomExposure float32    // 28: Bottom layers exporsure (in seconds)
	LayerOffTime   float32    // 2c: Layer off time (in seconds)
	BottomCount    uint32     // 30: Number of bottom layers
	ResolutionX    uint32     // 34:
	ResolutionY    uint32     // 38:
	PreviewHigh    uint32     // 3c: Offset of the high-res preview
	LayerDefs      uint32     // 40: Offset of the layer definitions
	LayerCount     uint32     // 44:
	PreviewLow     uint32     // 48: Offset of the low-rew preview
	PrintTime      uint32     // 4c: In seconds
	Projector      uint32     // 50: 0 = CAST, 1 = LCD_X_MIRROR
	ParamOffset    uint32     // 54:
	ParamSize      uint32     // 58:
	AntiAliasLevel uint32     // 5c: Always 1 for this format
	LightPWM       uint16     // 60:
	BottomLightPWM uint16     // 62:
	EncryptionSeed uint32     // 64: Compressed grayscale image encryption key
	SlicerOffset   uint32     // 68: Offset to the slicer parameters
	SlicerSize     uint32     // 6c: Size of the slicer parameters (0x4c)
}

type ctbParam struct {
	BottomLiftHeight float32 // 00:
	BottomLiftSpeed  float32 // 04:

	LiftHeight   float32 // 08:
	LiftSpeed    float32 // 0c:
	RetractSpeed float32 // 10:

	VolumeMilliliters float32 // 14:
	WeightGrams       float32 // 18:
	CostDollars       float32 // 1c:

	BottomLightOffTime float32 // 20:
	LightOffTime       float32 // 24:

	BottomLayerCount uint32 // 28:

	Unknown2C uint32  // 2c:
	Unknown30 float32 // 30:
	Unknown34 uint32  // 34:
	Unknown38 uint32  // 38:
}

type ctbSlicer struct {
	_               [7]uint32 // 00: 7 all-zeros
	MachineOffset   uint32    // 1c: Machine name offset
	MachineSize     uint32    // 20: Machine name length
	EncryptionMode  uint32    // 24: Always 0xf for CTB
	TimeSeconds     uint32    // 28:
	Unknown2C       uint32    // 2c: Always 1?
	ChiTuBoxVersion [4]byte   // 30: major, minor, patch, release
	Unknown34       uint32
	Unknown38       uint32
	Unknown3C       float32
	Unknown40       uint32
	Unknown44       uint32
	Unknown48       float32
}

type ctbPreview struct {
	ResolutionX uint32    // 00:
	ResolutionY uint32    // 04:
	ImageOffset uint32    // 08:
	ImageLength uint32    // 0c:
	_           [4]uint32 // 10:
}

type ctbLayerDef struct {
	LayerHeight   float32   // 00:
	LayerExposure float32   // 04:
	LayerOffTime  float32   // 08:
	ImageOffset   uint32    // 0c:
	ImageLength   uint32    // 10:
	_             [4]uint32 // 14:
}

type Ctb struct {
	properties uv3dp.Properties
	layerDef   []ctbLayerDef

	rleMap map[uint32]([]byte)
}

func align4(in uint32) (out uint32) {
	out = (in + 0x3) & 0xfffffffc
	return
}

func float32ToDuration(time_s float32) time.Duration {
	return time.Duration(math.Round(float64(time_s) * float64(time.Second)))
}

func durationToFloat32(time_ns time.Duration) float32 {
	return float32(float64(time_ns) / float64(time.Second))
}

type CtbFormatter struct {
	*pflag.FlagSet

	EncryptionSeed uint32
}

func NewCtbFormatter(suffix string) (cf *CtbFormatter) {
	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)
	flagSet.SetInterspersed(false)

	cf = &CtbFormatter{
		FlagSet: flagSet,
	}

	cf.Uint32VarP(&cf.EncryptionSeed, "encryption-seed", "e", 0, "Specify a specific encryption seed")

	return
}

// Save a uv3dp.Printable in CTB format
func (cf *CtbFormatter) Encode(writer uv3dp.Writer, p uv3dp.Printable) (err error) {
	properties := p.Properties()

	size := &properties.Size
	exp := &properties.Exposure
	bot := &properties.Bottom

	// First, compute the rle images
	type rleInfo struct {
		offset uint32
		rle    []byte
	}
	rleHash := map[uint64]rleInfo{}

	// Select an encryption seed
	// A zero encryption seed is rejected by the printer, so check for that
	seed := cf.EncryptionSeed
	for seed == 0 {
		seed = rand.Uint32()
	}

	headerBase := uint32(0)
	header := ctbHeader{
		Magic:          defaultHeaderMagic,
		Version:        2,
		EncryptionSeed: seed,
	}
	headerSize, _ := restruct.SizeOf(&header)

	// Add the preview images
	var previewHuge ctbPreview
	var previewTiny ctbPreview
	previewSize, _ := restruct.SizeOf(&previewHuge)

	// Set up the RLE hash indexes
	rleHashList := []uint64{}

	savePreview := func(base uint32, preview *ctbPreview, ptype uv3dp.PreviewType) uint32 {
		pic, found := properties.Preview[ptype]
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
	paramBase := savePreview(previewTinyBase, &previewTiny, uv3dp.PreviewTypeTiny)

	param := ctbParam{}
	paramSize, _ := restruct.SizeOf(&param)

	slicerBase := paramBase + uint32(paramSize)
	slicer := ctbSlicer{}
	slicerSize, _ := restruct.SizeOf(&slicer)

	machineBase := slicerBase + uint32(slicerSize)
	machine := "default"
	machineSize := len(machine)

	layerDefBase := machineBase + uint32(machineSize)
	layerDef := make([]ctbLayerDef, size.Layers)
	layerDefSize, _ := restruct.SizeOf(&layerDef[0])

	// And then all the layer images
	layerPage := uint32(layerDefSize * size.Layers)
	imageBase := layerDefBase + layerPage
	totalOn := uint64(0)

	type layerInfo struct {
		Z        float32
		Exposure *uv3dp.Exposure
		Rle      []byte
		Hash     uint64
		BitsOn   uint
	}

	doneMap := make([]chan layerInfo, size.Layers)
	for n := 0; n < size.Layers; n++ {
		doneMap[n] = make(chan layerInfo, 1)
	}

	uv3dp.WithAllLayers(p, func(n int, layer uv3dp.Layer) {
		rle, hash, bitsOn := rleEncodeGraymap(layer.Image)
		doneMap[n] <- layerInfo{
			Z:        layer.Z,
			Exposure: layer.Exposure,
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
			imageBase = align4(imageBase + uint32(len(info.Rle)))
		}

		layerDef[n] = ctbLayerDef{
			LayerHeight:   info.Z,
			LayerExposure: durationToFloat32(info.Exposure.LightOnTime),
			LayerOffTime:  durationToFloat32(info.Exposure.LightOffTime),
			ImageOffset:   rleHash[info.Hash].offset,
			ImageLength:   uint32(len(info.Rle)),
		}

		totalOn += uint64(info.BitsOn)
	}

	// ctbHeader
	header.BedSizeMM[0] = size.Millimeter.X
	header.BedSizeMM[1] = size.Millimeter.Y
	header.BedSizeMM[2] = forceBedSizeMM_3
	header.HeightMM = size.LayerHeight * float32(size.Layers)
	header.LayerHeight = size.LayerHeight
	header.LayerExposure = durationToFloat32(exp.LightOnTime)
	header.BottomExposure = durationToFloat32(bot.Exposure.LightOnTime)
	header.LayerOffTime = durationToFloat32(exp.LightOffTime)
	header.BottomCount = uint32(bot.Count)
	header.ResolutionX = uint32(size.X)
	header.ResolutionY = uint32(size.Y)
	header.PreviewHigh = previewHugeBase
	header.LayerDefs = layerDefBase
	header.LayerCount = uint32(size.Layers)
	header.PreviewLow = previewTinyBase
	header.PrintTime = uint32(properties.Duration() / time.Second)
	header.Projector = 1 // LCD_X_MIRROR

	header.ParamOffset = paramBase
	header.ParamSize = uint32(paramSize)
	header.AntiAliasLevel = 1
	header.LightPWM = 255
	header.BottomLightPWM = 255

	header.SlicerOffset = slicerBase
	header.SlicerSize = uint32(slicerSize)

	// ctbParam
	param.BottomLayerCount = uint32(bot.Count)
	param.BottomLiftSpeed = bot.Exposure.LiftSpeed
	param.BottomLiftHeight = bot.Exposure.LiftHeight
	param.LiftHeight = exp.LiftHeight
	param.LiftSpeed = exp.LiftSpeed
	param.RetractSpeed = exp.RetractSpeed

	if param.BottomLiftSpeed < 0 {
		param.BottomLiftSpeed = defaultBottomLiftSpeed
	}
	if param.BottomLiftHeight < 0 {
		param.BottomLiftHeight = defaultBottomLiftHeight
	}
	if param.LiftHeight < 0 {
		param.LiftHeight = defaultLiftHeight
	}
	if param.LiftSpeed < 0 {
		param.LiftSpeed = defaultLiftSpeed
	}
	if param.RetractSpeed < 0 {
		param.RetractSpeed = defaultRetractSpeed
	}
	param.Unknown38 = 0x1234

	// ctbSlicer
	slicer.MachineOffset = machineBase
	slicer.MachineSize = uint32(machineSize)
	slicer.EncryptionMode = 7 // Magic!
	slicer.TimeSeconds = 0x12345678
	slicer.ChiTuBoxVersion[0] = 1 // Magic!
	slicer.ChiTuBoxVersion[1] = 6
	slicer.ChiTuBoxVersion[2] = 3
	slicer.Unknown2C = 1 // Magic?
	slicer.Unknown34 = 0 // Magic?

	// Compute total cubic millimeters (== milliliters) of all the on pixels
	bedArea := float64(header.BedSizeMM[0] * header.BedSizeMM[1])
	bedPixels := uint64(header.ResolutionX) * uint64(header.ResolutionY)
	pixelVolume := float64(header.LayerHeight) * bedArea / float64(bedPixels)
	param.VolumeMilliliters = float32(float64(totalOn) * pixelVolume / 1000.0)

	param.BottomLightOffTime = durationToFloat32(bot.Exposure.LightOffTime)
	param.LightOffTime = durationToFloat32(exp.LightOffTime)
	param.BottomLayerCount = header.BottomCount

	// Collect file data
	fileData := map[int][]byte{}

	fileData[int(headerBase)], _ = restruct.Pack(binary.LittleEndian, &header)

	fileData[int(slicerBase)], _ = restruct.Pack(binary.LittleEndian, &slicer)

	fileData[int(machineBase)] = ([]byte)(machine)

	fileData[int(paramBase)], _ = restruct.Pack(binary.LittleEndian, &param)

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

func (cf *CtbFormatter) Decode(file uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {
	// Collect file
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	prop := uv3dp.Properties{
		Preview: make(map[uv3dp.PreviewType]image.Image),
	}

	header := ctbHeader{}
	err = restruct.Unpack(data, binary.LittleEndian, &header)
	if err != nil {
		return
	}

	if header.Magic != defaultHeaderMagic {
		err = fmt.Errorf("Unknown header magic: 0x%08x", header.Magic)
		return
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

		var preview ctbPreview
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

	layerDef := make([]ctbLayerDef, header.LayerCount)

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
	exp.LightOnTime = float32ToDuration(header.LayerExposure)
	exp.LightOffTime = float32ToDuration(header.LayerOffTime)

	bot := &prop.Bottom
	bot.Count = int(header.BottomCount)
	bot.Exposure.LightOnTime = float32ToDuration(header.BottomExposure)
	bot.Exposure.LightOffTime = float32ToDuration(header.LayerOffTime)

	if header.ParamSize > 0 && header.ParamOffset > 0 {
		var param ctbParam

		addr := int(header.ParamOffset)
		err = restruct.Unpack(data[addr:], binary.LittleEndian, &param)
		if err != nil {
			return
		}

		bot.Count = int(param.BottomLayerCount)
		bot.Exposure.LiftHeight = param.BottomLiftHeight
		bot.Exposure.LiftSpeed = param.BottomLiftSpeed
		bot.Exposure.LightOffTime = float32ToDuration(param.BottomLightOffTime)
		bot.Exposure.RetractSpeed = param.RetractSpeed
		bot.Exposure.RetractHeight = defaultRetractHeight

		exp.LiftHeight = param.LiftHeight
		exp.LiftSpeed = param.LiftSpeed
		exp.LightOffTime = float32ToDuration(param.LightOffTime)
		exp.RetractSpeed = param.RetractSpeed
		exp.RetractHeight = defaultRetractHeight
	} else {
		// Use reasonable defaults
		bot.Exposure.LiftHeight = defaultBottomLiftHeight
		bot.Exposure.LiftSpeed = defaultBottomLiftSpeed
		bot.Exposure.RetractSpeed = defaultRetractSpeed
		bot.Exposure.RetractHeight = defaultRetractHeight

		exp.LiftHeight = defaultLiftHeight
		exp.LiftSpeed = defaultLiftSpeed
		exp.RetractSpeed = defaultRetractSpeed
		exp.RetractHeight = defaultRetractHeight
	}

	cbd := &Ctb{
		properties: prop,
		layerDef:   layerDef,
		rleMap:     rleMap,
	}

	printable = cbd

	return
}

// Properties get the properties of the Ctb Printable
func (cbd *Ctb) Properties() (prop uv3dp.Properties) {
	prop = cbd.properties

	return
}

// Layer gets a layer - we decode from the RLE on-the fly
func (cbd *Ctb) Layer(index int) (layer uv3dp.Layer) {
	if index < 0 || index >= len(cbd.layerDef) {
		return
	}

	layerDef := cbd.layerDef[index]

	// Update per-layer info
	prop := &cbd.properties
	size := &prop.Size
	bounds := image.Rect(0, 0, size.X, size.Y)
	layerImage, err := rleDecodeGraymap(bounds, cbd.rleMap[layerDef.ImageOffset])
	if err != nil {
		panic(err)
	}

	var exposure uv3dp.Exposure
	if index < prop.Bottom.Count {
		exposure = prop.Bottom.Exposure
	} else {
		exposure = prop.Exposure
	}

	if layerDef.LayerExposure > 0.0 {
		exposure.LightOnTime = float32ToDuration(layerDef.LayerExposure)
	}

	if layerDef.LayerOffTime > 0.0 {
		exposure.LightOffTime = float32ToDuration(layerDef.LayerOffTime)
	}

	layer = uv3dp.Layer{
		Z:        layerDef.LayerHeight,
		Image:    layerImage,
		Exposure: &exposure,
	}

	return
}
