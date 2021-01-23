//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package ctb

import (
	"fmt"
	"image"
	"io/ioutil"
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
	EncryptionMode  uint32    // 24: Always 0xf for CTB v3, 0x07 for CTB v2
	TimeSeconds     uint32    // 28:
	Unknown2C       uint32    // 2c: Always 1?
	ChiTuBoxVersion [4]byte   // 30: major, minor, patch, release
	Unknown34       uint32
	Unknown38       uint32
	Unknown3C       float32 // 3c: TransitionLayerCount (?)
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
	LayerHeight   float32 // 00:
	LayerExposure float32 // 04:
	LayerOffTime  float32 // 08:
	ImageOffset   uint32  // 0c:
	ImageLength   uint32  // 10:
	Unknown14     uint32  // 14:
	InfoSize      uint32  // 18: Size of image info
	Unknown1c     uint32  // 1c:
	Unknown20     uint32  // 20:
}

type ctbImageInfo struct {
	LayerDef     ctbLayerDef // 00:  Repeat of the LayerDef information
	TotalSize    uint32      // 24:  Total size of ctbImageInfo and Image data
	LiftHeight   float32     // 28:
	LiftSpeed    float32     // 2c:
	Unknown30    uint32      // 30: Zero
	Unknown34    uint32      // 34: Zero
	RetractSpeed float32     // 38:
	Unknown3c    uint32      // 3c: Zero
	Unknown40    uint32      // 40: Zero
	Unknown44    uint32      // 44: Zero
	Unknown48    uint32      // 48: Zero
	Unknown4c    uint32      // 4c: ??
	LightPWM     float32     // 50:
}

type Print struct {
	uv3dp.Print
	layerDef  []ctbLayerDef
	imageInfo [](*ctbImageInfo)

	rleMap map[uint32]([]byte)
}

type Formatter struct {
	*pflag.FlagSet

	EncryptionSeed uint32
	Version        int
}

func NewFormatter(suffix string) (cf *Formatter) {
	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)
	flagSet.SetInterspersed(false)

	cf = &Formatter{
		FlagSet: flagSet,
	}

	cf.Uint32VarP(&cf.EncryptionSeed, "encryption-seed", "e", 0, "Specify a specific encryption seed")
	cf.IntVarP(&cf.Version, "version", "v", 3, "Specify the CTB version (2 or 3)")

	return
}

// Save a uv3dp.Printable in CTB format
func (cf *Formatter) Encode(writer uv3dp.Writer, printable uv3dp.Printable) (err error) {
	if cf.Version < 2 || cf.Version > 3 {
		err = fmt.Errorf("unsupported version %v", cf.Version)
		return
	}

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
	// A zero encryption seed is rejected by the printer, so check for that
	seed := cf.EncryptionSeed
	for seed == 0 {
		seed = rand.Uint32()
	}

	headerBase := uint32(0)
	header := ctbHeader{
		Magic:          defaultHeaderMagic,
		Version:        uint32(cf.Version),
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
	paramBase := savePreview(previewTinyBase, &previewTiny, uv3dp.PreviewTypeTiny)

	param := ctbParam{}
	paramSize, _ := restruct.SizeOf(&param)

	slicerBase := paramBase + uint32(paramSize)
	slicer := ctbSlicer{}
	slicerSize, _ := restruct.SizeOf(&slicer)

	machineBase := slicerBase + uint32(slicerSize)
	machine, _ := mach.(string)
	machineSize := len(machine)

	layerDefBase := machineBase + uint32(machineSize)
	layerDef := make([]ctbLayerDef, size.Layers)
	imageInfo := make([]ctbImageInfo, size.Layers)
	layerDefSize, _ := restruct.SizeOf(&layerDef[0])

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

	info_size, _ := restruct.SizeOf(&ctbImageInfo{})
	imageInfoSize := uint32(info_size)
	if cf.Version < 3 {
		imageInfoSize = 0
	}

	for n := 0; n < size.Layers; n++ {
		info := <-doneMap[n]
		if header.EncryptionSeed != 0 {
			info.Hash = uint64(n)
			info.Rle = cipher(header.EncryptionSeed, uint32(n), info.Rle)
		}
		_, ok := rleHash[info.Hash]
		if !ok {
			rleHash[info.Hash] = rleInfo{offset: imageBase + imageInfoSize, rle: info.Rle}
			rleHashList = append(rleHashList, info.Hash)
			imageBase = imageBase + imageInfoSize + uint32(len(info.Rle))
		}

		layerDef[n] = ctbLayerDef{
			LayerHeight:   info.Z,
			LayerExposure: info.Exposure.LightOnTime,
			LayerOffTime:  info.Exposure.LightOffTime,
			ImageOffset:   rleHash[info.Hash].offset,
			ImageLength:   uint32(len(info.Rle)),
			InfoSize:      imageInfoSize,
		}

		if imageInfoSize > 0 {
			imageInfo[n] = ctbImageInfo{
				LayerDef:     layerDef[n],
				TotalSize:    uint32(len(info.Rle)) + imageInfoSize,
				LiftHeight:   info.Exposure.LiftHeight,
				LiftSpeed:    info.Exposure.LiftSpeed,
				RetractSpeed: info.Exposure.RetractSpeed,
				LightPWM:     float32(info.Exposure.LightPWM),
			}
		}

		totalOn += uint64(info.BitsOn)
	}

	// ctbHeader
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

	header.ParamOffset = paramBase
	header.ParamSize = uint32(paramSize)

	header.AntiAliasLevel = 1

	if exp.LightPWM == 0 {
		exp.LightPWM = 255
	}

	if bot.Exposure.LightPWM == 0 {
		bot.Exposure.LightPWM = 255
	}

	header.LightPWM = uint16(exp.LightPWM)
	header.BottomLightPWM = uint16(bot.Exposure.LightPWM)

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
	param.Unknown38 = 0

	// ctbSlicer
	slicer.MachineOffset = machineBase
	slicer.MachineSize = uint32(machineSize)
	slicer.TimeSeconds = 0x12345678
	slicer.EncryptionMode = 0x7 // Magic!
	if cf.Version > 2 {
		slicer.EncryptionMode = 0xf // Magic!
	}
	slicer.ChiTuBoxVersion[0] = 0 // Magic!
	slicer.ChiTuBoxVersion[1] = 0
	slicer.ChiTuBoxVersion[2] = 7
	slicer.ChiTuBoxVersion[3] = 1
	slicer.Unknown2C = 1 // Magic?
	slicer.Unknown34 = 0 // Magic?

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

	if imageInfoSize > 0 {
		for _, info := range imageInfo {
			data, _ := restruct.Pack(binary.LittleEndian, &info)
			fileData[int(info.LayerDef.ImageOffset-imageInfoSize)] = data
		}
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

func (cf *Formatter) Decode(file uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {
	// Collect file
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	prop := uv3dp.Properties{
		Preview:  make(map[uv3dp.PreviewType]image.Image),
		Metadata: make(map[string]interface{}),
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

	// ctbSlicer info
	slicer := ctbSlicer{}
	if header.SlicerOffset > 0 {
		err = restruct.Unpack(data[header.SlicerOffset:], binary.LittleEndian, &slicer)
		if err != nil {
			return
		}
	}

	// Machine Name
	mach := string(data[slicer.MachineOffset : slicer.MachineOffset+slicer.MachineSize])
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

	imageInfo := make([](*ctbImageInfo), header.LayerCount)

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

		infoSize := layerDef[n].InfoSize
		if header.Version >= 3 && infoSize > 0 {
			info := &ctbImageInfo{}
			err = restruct.Unpack(data[addr-infoSize:addr], binary.LittleEndian, info)
			if err != nil {
				imageInfo[n] = info
			}
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

	ctb := &Print{
		Print:     uv3dp.Print{Properties: prop},
		layerDef:  layerDef,
		imageInfo: imageInfo,
		rleMap:    rleMap,
	}

	printable = ctb

	return
}

func (ctb *Print) LayerImage(index int) (layerImage *image.Gray) {
	layerDef := ctb.layerDef[index]

	// Update per-layer info
	layerImage, err := rleDecodeGraymap(ctb.Bounds(), ctb.rleMap[layerDef.ImageOffset])
	if err != nil {
		panic(err)
	}

	return
}

func (ctb *Print) LayerExposure(index int) (exposure uv3dp.Exposure) {
	layerDef := ctb.layerDef[index]

	if index < ctb.Bottom().Count {
		exposure = ctb.Bottom().Exposure
	} else {
		exposure = ctb.Exposure()
	}

	if layerDef.LayerExposure > 0.0 {
		exposure.LightOnTime = layerDef.LayerExposure
	}

	if layerDef.LayerOffTime > 0.0 {
		exposure.LightOffTime = layerDef.LayerOffTime
	}

	// See if we have per-layer overrides
	info := ctb.imageInfo[index]
	if info != nil {
		exposure.LightOnTime = info.LayerDef.LayerExposure
		exposure.LightOffTime = info.LayerDef.LayerOffTime
		exposure.LightPWM = uint8(info.LightPWM)
		exposure.LiftHeight = info.LiftHeight
		exposure.LiftSpeed = info.LiftSpeed
		exposure.RetractSpeed = info.RetractSpeed
	}

	return
}

func (ctb *Print) LayerZ(index int) (z float32) {
	z = ctb.layerDef[index].LayerHeight
	return
}
