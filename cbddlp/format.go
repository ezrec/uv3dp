//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package cbddlp

import (
	"fmt"
	"image"
	"io/ioutil"
	"sort"

	"encoding/binary"

	"github.com/go-restruct/restruct"
	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

const (
	defaultHeaderMagic = uint32(0x12fd0019)

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

type cbddlpHeader struct {
	Magic          uint32     // 00
	Version        uint32     // 04
	BedSizeMM      [3]float32 // 08
	_              [3]uint32  // 14
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
	AntiAliasLevel uint32     // 5c:
	LightPWM       uint16     // 60:
	BottomLightPWM uint16     // 62:
	_              [3]uint32  // 64:
}

type cbddlpParam struct {
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

	_ [4]uint32 // 2c:
}

type cbddlpPreview struct {
	ResolutionX uint32    // 00:
	ResolutionY uint32    // 04:
	ImageOffset uint32    // 08:
	ImageLength uint32    // 0c:
	_           [4]uint32 // 10:
}

type cbddlpLayerDef struct {
	LayerHeight   float32   // 00:
	LayerExposure float32   // 04:
	LayerOffTime  float32   // 08:
	ImageOffset   uint32    // 0c:
	ImageLength   uint32    // 10:
	_             [4]uint32 // 14:
}

type CbdDlp struct {
	properties uv3dp.Properties
	layerDef   []cbddlpLayerDef

	rleMap map[uint32]([]([]byte))
}

func align4(in uint32) (out uint32) {
	out = (in + 0x3) & 0xfffffffc
	return
}

type CbddlpFormatter struct {
	*pflag.FlagSet

	Version   int // Version of file to use, one of [1,2]
	AntiAlias int // AntiAlias level, one of [1,2,4,8]
}

func NewCbddlpFormatter(suffix string) (cf *CbddlpFormatter) {
	var version int
	var antialias int

	switch suffix {
	case ".cbddlp":
		version = 2
		antialias = 1
	case ".photon":
		version = 1
		antialias = 1
	default:
		version = 1
		antialias = 1
	}

	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)
	flagSet.SetInterspersed(false)

	cf = &CbddlpFormatter{
		FlagSet:   flagSet,
		Version:   version,
		AntiAlias: antialias,
	}

	cf.IntVarP(&cf.Version, "version", "v", version, "Override header Version")
	cf.IntVarP(&cf.AntiAlias, "anti-alias", "a", antialias, "Override antialias level (1,2,4,8)")

	return
}

// Save a uv3dp.Printable in CBD DLP format
func (cf *CbddlpFormatter) Encode(writer uv3dp.Writer, p uv3dp.Printable) (err error) {
	switch cf.Version {
	case 1:
		if cf.AntiAlias != 1 {
			err = fmt.Errorf("illegal --anti-alias setting: must be '1' for Version 1 files")
			return
		}
	case 2:
		if cf.AntiAlias != 1 && cf.AntiAlias != 2 && cf.AntiAlias != 4 && cf.AntiAlias != 8 {
			err = fmt.Errorf("illegal --anti-alias setting: %v (must be one of 1,2,4, or 8 bits)", cf.AntiAlias)
			return
		}
	default:
		err = fmt.Errorf("illegal version: %v", cf.Version)
		return
	}

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

	headerBase := uint32(0)
	header := cbddlpHeader{
		Magic:   defaultHeaderMagic,
		Version: uint32(cf.Version),
	}
	headerSize, _ := restruct.SizeOf(&header)

	// Add the preview images
	var previewHuge cbddlpPreview
	var previewTiny cbddlpPreview
	previewSize, _ := restruct.SizeOf(&previewHuge)

	// Set up the RLE hash indexes
	rleHashList := []uint64{}

	savePreview := func(base uint32, preview *cbddlpPreview, ptype uv3dp.PreviewType) uint32 {
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

		return align4(base + uint32(len(rle)))
	}

	previewHugeBase := headerBase + uint32(headerSize)

	previewTinyBase := savePreview(previewHugeBase, &previewHuge, uv3dp.PreviewTypeHuge)
	paramBase := savePreview(previewTinyBase, &previewTiny, uv3dp.PreviewTypeTiny)

	param := cbddlpParam{}
	paramSize, _ := restruct.SizeOf(&param)

	layerDefBase := paramBase + uint32(paramSize)
	if header.Version < 2 {
		// Omit param items
		layerDefBase = paramBase
	}

	layerDef := make([]cbddlpLayerDef, size.Layers*cf.AntiAlias)
	layerDefSize, _ := restruct.SizeOf(&layerDef[0])

	// And then all the layer images
	layerPage := uint32(layerDefSize * size.Layers)
	imageBase := layerDefBase + layerPage*uint32(cf.AntiAlias)
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
		doneMap[n] = make(chan layerInfo, cf.AntiAlias)
	}

	uv3dp.WithAllLayers(p, func(n int, layer uv3dp.Layer) {
		for bit := 0; bit < cf.AntiAlias; bit++ {
			rle, hash, bitsOn := rleEncodeBitmap(layer.Image, bit, cf.AntiAlias)
			doneMap[n] <- layerInfo{
				Z:        layer.Z,
				Exposure: layer.Exposure,
				Rle:      rle,
				Hash:     hash,
				BitsOn:   bitsOn,
			}
		}
		close(doneMap[n])
	})

	for n := 0; n < size.Layers; n++ {
		for bit := 0; bit < cf.AntiAlias; bit++ {
			info := <-doneMap[n]
			_, ok := rleHash[info.Hash]
			if !ok {
				rleHash[info.Hash] = rleInfo{offset: imageBase, rle: info.Rle}
				rleHashList = append(rleHashList, info.Hash)
				imageBase = align4(imageBase + uint32(len(info.Rle)))
			}

			layerDef[n+bit*size.Layers] = cbddlpLayerDef{
				LayerHeight:   info.Z,
				LayerExposure: info.Exposure.LightOnTime,
				LayerOffTime:  info.Exposure.LightOffTime,
				ImageOffset:   rleHash[info.Hash].offset,
				ImageLength:   uint32(len(info.Rle)),
			}

			totalOn += uint64(info.BitsOn)
		}
	}

	// cbddlpHeader
	header.BedSizeMM[0] = size.Millimeter.X
	header.BedSizeMM[1] = size.Millimeter.Y
	header.BedSizeMM[2] = forceBedSizeMM_3
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
	header.PrintTime = uint32(properties.Duration())
	header.Projector = 1 // LCD_X_MIRROR

	if header.Version >= 2 {
		header.ParamOffset = paramBase
		header.ParamSize = uint32(paramSize)
		header.AntiAliasLevel = uint32(cf.AntiAlias)
		header.LightPWM = uint16(exp.LightPWM)
		header.BottomLightPWM = uint16(bot.Exposure.LightPWM)
	}

	if header.Version >= 2 {
		// cbddlpParam
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
	}

	// Compute total cubic millimeters (== milliliters) of all the on pixels
	bedArea := float64(header.BedSizeMM[0] * header.BedSizeMM[1])
	bedPixels := uint64(header.ResolutionX) * uint64(header.ResolutionY)
	pixelVolume := float64(header.LayerHeight) * bedArea / float64(bedPixels)
	param.VolumeMilliliters = float32(float64(totalOn) * pixelVolume / 1000.0)

	param.BottomLightOffTime = bot.Exposure.LightOffTime
	param.LightOffTime = exp.LightOffTime
	param.BottomLayerCount = header.BottomCount

	// Collect file data
	fileData := map[int][]byte{}

	fileData[int(headerBase)], _ = restruct.Pack(binary.LittleEndian, &header)

	if header.Version >= 2 {
		fileData[int(paramBase)], _ = restruct.Pack(binary.LittleEndian, &param)
	}

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

func (cf *CbddlpFormatter) Decode(file uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {
	// Collect file
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	prop := uv3dp.Properties{
		Preview: make(map[uv3dp.PreviewType]image.Image),
	}

	header := cbddlpHeader{}
	err = restruct.Unpack(data, binary.LittleEndian, &header)
	if err != nil {
		return
	}

	if header.Version < 2 {
		header.LightPWM = 255
		header.BottomLightPWM = 255
	}

	if header.AntiAliasLevel == 0 {
		header.AntiAliasLevel = 1
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

		var preview cbddlpPreview
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

	// Collect layers
	rleMap := make(map[uint32]([]([]byte)))

	layerDef := make([]cbddlpLayerDef, header.LayerCount)

	layerDefSize := uint32(9 * 4)
	layerDefPage := layerDefSize * header.LayerCount
	for n := uint32(0); n < header.LayerCount; n++ {
		offset := header.LayerDefs + layerDefSize*n
		err = restruct.Unpack(data[offset:], binary.LittleEndian, &layerDef[n])
		if err != nil {
			return
		}

		addr := layerDef[n].ImageOffset
		size := layerDef[n].ImageLength

		rleMap[addr] = []([]byte){data[addr : addr+size]}

		// Collect the remaining anti-alias layer RLEs
		for i := 1; i < int(header.AntiAliasLevel); i++ {
			offset += layerDefPage
			var layerTmp cbddlpLayerDef
			err = restruct.Unpack(data[offset:], binary.LittleEndian, &layerTmp)
			if err != nil {
				return
			}

			naddr := layerTmp.ImageOffset
			nsize := layerTmp.ImageLength

			rleMap[addr] = append(rleMap[addr], data[naddr:naddr+nsize])
		}
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

	if header.Version > 1 && header.ParamSize > 0 && header.ParamOffset > 0 {
		var param cbddlpParam

		addr := int(header.ParamOffset)
		err = restruct.Unpack(data[addr:], binary.LittleEndian, &param)
		if err != nil {
			return
		}

		bot.Count = int(param.BottomLayerCount)
		bot.Exposure.LiftHeight = param.BottomLiftHeight
		bot.Exposure.LiftSpeed = param.BottomLiftSpeed
		bot.Exposure.LightOffTime = param.BottomLightOffTime
		bot.Exposure.RetractSpeed = param.RetractSpeed
		bot.Exposure.RetractHeight = defaultRetractHeight

		exp.LiftHeight = param.LiftHeight
		exp.LiftSpeed = param.LiftSpeed
		exp.LightOffTime = param.LightOffTime
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

	cbd := &CbdDlp{
		properties: prop,
		layerDef:   layerDef,
		rleMap:     rleMap,
	}

	printable = cbd

	return
}

// Properties get the properties of the CbdDlp Printable
func (cbd *CbdDlp) Properties() (prop uv3dp.Properties) {
	prop = cbd.properties

	return
}

// Layer gets a layer - we decode from the RLE on-the fly
func (cbd *CbdDlp) Layer(index int) (layer uv3dp.Layer) {
	if index < 0 || index >= len(cbd.layerDef) {
		return
	}

	layerDef := cbd.layerDef[index]

	// Update per-layer info
	prop := &cbd.properties
	size := &prop.Size
	bounds := image.Rect(0, 0, size.X, size.Y)
	layerImage, err := rleDecodeBitmaps(bounds, cbd.rleMap[layerDef.ImageOffset])
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
		exposure.LightOnTime = layerDef.LayerExposure
	}

	if layerDef.LayerOffTime > 0.0 {
		exposure.LightOffTime = layerDef.LayerOffTime
	}

	layer = uv3dp.Layer{
		Z:        layerDef.LayerHeight,
		Image:    layerImage,
		Exposure: exposure,
	}

	return
}
