//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package cbddlp

import (
	"image"
	"io/ioutil"
	"time"

	"encoding/binary"

	"github.com/go-restruct/restruct"
	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

const (
	defaultHeaderMagic = uint32(0x12fd0019)
	defaultLayerCache  = 16

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
	Header         uint32     // 00
	Version        uint32     // 04
	BedSizeMM      [3]float32 // 08
	_              [3]uint32  // 14
	LayerHeight    float32    // 20
	LayerExposure  float32    // 24: Layer exposure (in seconds)
	Bottom         float32    // 28: Bottom layers exporsure (in seconds)
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

	rleMap map[uint32]([]byte)

	layerCache map[int]uv3dp.Layer
}

func align4(in uint32) (out uint32) {
	out = (in + 0x3) & 0xfffffffc
	return
}

type CbddlpFormatter struct {
	*pflag.FlagSet

	Version uint32 // Version of file to use, one of [1,2]
}

func NewCbddlpFormatter(suffix string) (cf *CbddlpFormatter) {
	var version uint32

	switch suffix {
	case ".cbddlp":
		version = 2
	case ".photon":
		version = 1
	default:
		version = 1
	}

	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)
	flagSet.SetInterspersed(false)

	cf = &CbddlpFormatter{
		FlagSet: flagSet,
		Version: version,
	}

	cf.Uint32Var(&cf.Version, "version", version, "Override header Version")

	return
}

// Save a uv3dp.Printable in CBD DLP format
func (cf *CbddlpFormatter) Encode(writer uv3dp.WriteAtSeeker, p uv3dp.Printable) (err error) {
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
	layerHash := make([]uint64, size.Layers)

	headerBase := uint32(0)
	header := cbddlpHeader{
		Version: cf.Version,
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

	layerDef := make([]cbddlpLayerDef, size.Layers)
	layerDefSize, _ := restruct.SizeOf(&layerDef[0])

	// And then all the layer images
	imageBase := layerDefBase + uint32(layerDefSize*size.Layers)
	totalOn := uint64(0)

	for n := 0; n < size.Layers; n++ {
		layer := p.Layer(n)
		rle, hash, bitsOn := rleEncodeBitmap(layer.Image)
		totalOn += uint64(bitsOn)
		_, ok := rleHash[hash]
		if !ok {
			rleHash[hash] = rleInfo{offset: imageBase, rle: rle}
			rleHashList = append(rleHashList, hash)
			imageBase = align4(imageBase + uint32(len(rle)))
		}

		layerExposure := layer.Exposure.LightExposure
		layerOffTime := layer.Exposure.LightOffTime

		layerHash[n] = hash
		layerDef[n] = cbddlpLayerDef{
			LayerHeight:   layer.Z,
			LayerExposure: float32(layerExposure) / float32(time.Second),
			LayerOffTime:  float32(layerOffTime) / float32(time.Second),
			ImageOffset:   rleHash[hash].offset,
			ImageLength:   uint32(len(rle)),
		}
	}

	// cbddlpHeader
	header.Header = defaultHeaderMagic
	header.Version = 2
	header.BedSizeMM[0] = size.Millimeter.X
	header.BedSizeMM[1] = size.Millimeter.Y
	header.BedSizeMM[2] = forceBedSizeMM_3
	header.LayerHeight = size.LayerHeight
	header.LayerExposure = float32(exp.LightExposure) / float32(time.Second)
	header.Bottom = float32(bot.Exposure.LightExposure) / float32(time.Second)
	header.LayerOffTime = float32(exp.LightOffTime) / float32(time.Second)
	header.BottomCount = uint32(bot.Count)
	header.ResolutionX = uint32(size.X)
	header.ResolutionY = uint32(size.Y)
	header.PreviewHigh = previewHugeBase
	header.LayerDefs = layerDefBase
	header.LayerCount = uint32(size.Layers)
	header.PreviewLow = previewTinyBase
	header.PrintTime = uint32(properties.Duration() / time.Second)
	header.Projector = 1 // LCD_X_MIRROR

	if header.Version >= 2 {
		header.ParamOffset = paramBase
		header.ParamSize = uint32(paramSize)
		header.AntiAliasLevel = 0
	}

	header.LightPWM = 255
	header.BottomLightPWM = 255

	if header.Version >= 2 {
		// cbddlpParam
		param.BottomLiftSpeed = bot.Exposure.LiftSpeed
		param.BottomLiftHeight = bot.Exposure.LiftHeight
		param.LiftHeight = exp.LiftHeight
		param.LiftSpeed = exp.LiftSpeed
		param.RetractSpeed = exp.RetractSpeed
	}

	// Compute total cubic millimeters (== milliliters) of all the on pixels
	bedArea := float64(header.BedSizeMM[0] * header.BedSizeMM[1])
	bedPixels := uint64(header.ResolutionX) * uint64(header.ResolutionY)
	pixelVolume := float64(header.LayerHeight) * bedArea / float64(bedPixels)
	param.VolumeMilliliters = float32(float64(totalOn) * pixelVolume / 1000.0)

	param.BottomLightOffTime = float32(bot.Exposure.LightOffTime) / float32(time.Second)
	param.LightOffTime = float32(exp.LightOffTime) / float32(time.Second)
	param.BottomLayerCount = header.BottomCount

	var data []byte

	data, _ = restruct.Pack(binary.LittleEndian, &header)
	_, err = writer.WriteAt(data, int64(headerBase))
	if err != nil {
		return
	}

	if header.Version >= 2 {
		data, _ = restruct.Pack(binary.LittleEndian, &param)
		_, err = writer.WriteAt(data, int64(paramBase))
		if err != nil {
			return
		}
	}

	for n, layer := range layerDef {
		data, _ = restruct.Pack(binary.LittleEndian, &layer)
		_, err = writer.WriteAt(data, int64(int(layerDefBase)+layerDefSize*n))
		if err != nil {
			return
		}
	}

	data, _ = restruct.Pack(binary.LittleEndian, &previewHuge)
	_, err = writer.WriteAt(data, int64(previewHugeBase))
	if err != nil {
		return
	}

	data, _ = restruct.Pack(binary.LittleEndian, &previewTiny)
	_, err = writer.WriteAt(data, int64(previewTinyBase))
	if err != nil {
		return
	}

	for _, hash := range rleHashList {
		info := rleHash[hash]
		_, err = writer.WriteAt(info.rle, int64(info.offset))
		if err != nil {
			return
		}
	}

	return
}

func (cf *CbddlpFormatter) Decode(file uv3dp.ReadAtSeeker, filesize int64) (printable uv3dp.Printable, err error) {
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
		println(item.previewType, item.previewOffset, addr, size)

		bounds := image.Rect(0, 0, int(preview.ResolutionX), int(preview.ResolutionY))
		var pic image.Image
		pic, err = rleDecodeRGB15(bounds, data[addr:addr+size])
		if err != nil {
			return
		}

		prop.Preview[item.previewType] = pic
	}

	// Collect layers
	rleMap := make(map[uint32]([]byte))

	layerDef := make([]cbddlpLayerDef, header.LayerCount)
	for n := uint32(0); n < header.LayerCount; n++ {
		offset := header.LayerDefs + (9*4)*n
		err = restruct.Unpack(data[offset:], binary.LittleEndian, &layerDef[n])
		if err != nil {
			return
		}

		addr := layerDef[n].ImageOffset
		size := layerDef[n].ImageLength

		rleMap[addr] = data[addr : addr+size]
	}

	size := &prop.Size
	size.Millimeter.X = header.BedSizeMM[0]
	size.Millimeter.Y = header.BedSizeMM[1]

	size.X = int(header.ResolutionX)
	size.Y = int(header.ResolutionY)

	size.Layers = int(header.LayerCount)
	size.LayerHeight = header.LayerHeight

	exp := &prop.Exposure
	exp.LightExposure = time.Duration(header.LayerExposure*1000) * time.Millisecond
	exp.LightOffTime = time.Duration(header.LayerOffTime*1000) * time.Millisecond

	bot := &prop.Bottom
	bot.Count = int(header.BottomCount)
	bot.Exposure.LightExposure = time.Duration(header.Bottom*1000) * time.Millisecond

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
		bot.Exposure.LightOffTime = time.Duration(param.BottomLightOffTime*1000) * time.Millisecond
		bot.Exposure.RetractSpeed = param.RetractSpeed
		bot.Exposure.RetractHeight = defaultRetractHeight

		exp.LiftHeight = param.LiftHeight
		exp.LiftSpeed = param.LiftSpeed
		exp.LightOffTime = time.Duration(param.LightOffTime*1000) * time.Millisecond
		exp.RetractSpeed = param.RetractSpeed
		exp.RetractHeight = defaultRetractHeight
	} else {
		// Use reasonable defaults
		bot.Exposure.LiftHeight = defaultBottomLiftHeight
		bot.Exposure.LiftSpeed = defaultBottomLiftSpeed
		bot.Exposure.LightOffTime = defaultBottomLightOff
		bot.Exposure.RetractSpeed = defaultRetractSpeed
		bot.Exposure.RetractHeight = defaultRetractHeight

		exp.LiftHeight = defaultLiftHeight
		exp.LiftSpeed = defaultLiftSpeed
		exp.LightOffTime = defaultLightOff
		exp.RetractSpeed = defaultRetractSpeed
		exp.RetractHeight = defaultRetractHeight
	}

	cbd := &CbdDlp{
		properties: prop,
		layerDef:   layerDef,
		rleMap:     rleMap,
		layerCache: make(map[int]uv3dp.Layer, defaultLayerCache),
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

	var exposure *uv3dp.Exposure
	if index < cbd.properties.Bottom.Count {
		exposure = &cbd.properties.Bottom.Exposure
	} else {
		exposure = &cbd.properties.Exposure
	}

	// Update per-layer info
	lightExposure := time.Duration(layerDef.LayerExposure*1000) * time.Millisecond
	lightOffTime := time.Duration(layerDef.LayerOffTime*1000) * time.Millisecond

	if lightExposure != exposure.LightExposure || lightOffTime != exposure.LightOffTime {
		exp := &uv3dp.Exposure{}
		*exp = *exposure
		exp.LightExposure = lightExposure
		exp.LightOffTime = lightOffTime
		exposure = exp
	}

	size := &cbd.properties.Size
	bounds := image.Rect(0, 0, size.X, size.Y)
	layerImage, err := rleDecodeBitmap(bounds, cbd.rleMap[layerDef.ImageOffset])
	if err != nil {
		panic(err)
	}

	layer = uv3dp.Layer{
		Z:        layerDef.LayerHeight,
		Image:    layerImage,
		Exposure: exposure,
	}

	return
}
