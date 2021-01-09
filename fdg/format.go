//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package fdg

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
	defaultHeaderMagic = uint32(0xbd3c7ac8)

	defaultBottomLiftHeight = 5.0
	defaultBottomLiftSpeed  = 300.0
	defaultLiftHeight       = 5.0
	defaultLiftSpeed        = 300.0
	defaultRetractSpeed     = 300.0
	defaultRetractHeight    = 6.0
	defaultBottomLightOff   = 1.0
	defaultLightOff         = 1.0
)

type fdgHeader struct {
	Magic              uint32     // 00:
	Version            uint32     // 04: Always '2'
	LayerCount         uint32     // 08:
	BottomCount        uint32     // 0c: Number of bottom layers
	Projector          uint32     // 10: 0 = CAST, 1 = LCD_X_MIRROR
	BottomLayerCount   uint32     // 14: Number of bottom layers
	ResolutionX        uint32     // 18:
	ResolutionY        uint32     // 1c:
	LayerHeight        float32    // 20:
	LayerExposure      float32    // 24: Layer exposure (in seconds)
	BottomExposure     float32    // 28: Bottom layers exporsure (in seconds)
	PreviewHigh        uint32     // 2c: Offset of the high-res preview
	PreviewLow         uint32     // 30: Offset of the high-res preview
	LayerDefs          uint32     // 34: Offset of the layer definitions
	PrintTime          uint32     // 38: Print time, in seconds
	AntiAliasLevel     uint32     // 3c: Always 1 for this format?
	LightPWM           uint16     // 40:
	BottomLightPWM     uint16     // 42:
	_                  [2]uint32  // 44:
	HeightMM           float32    // 4c:
	BedSizeMM          [3]float32 // 50:
	EncryptionSeed     uint32     // 5c: Encryption seed
	AntiAliasDepth     uint32     // 60: AntiAlias Level
	EncryptionMode     uint32     // 64: Possible encryption mode? (0x4c)
	VolumeMilliliters  float32    // 68:
	WeightGrams        float32    // 6c:
	CostDollars        float32    // 70:
	MachineOffset      uint32     // 74: Machine name offset
	MachineSize        uint32     // 78: Machine name length
	BottomLightOffTime float32    // 7c:
	LightOffTime       float32    // 80:
	_                  uint32     // 84:
	BottomLiftHeight   float32    // 88:
	BottomLiftSpeed    float32    // 8c:
	LiftHeight         float32    // 90:
	LiftSpeed          float32    // 94:
	RetractSpeed       float32    // 98:
	_                  [7]uint32  // 9c:
	Timestamp          uint32     // b8: Minutes since Jan 1, 1970 UTC
	ChiTuBoxVersion    [4]byte    // bc: major, minor, patch, release
	_                  [6]uint32  // c0:
}

type fdgPreview struct {
	ResolutionX uint32    // 00:
	ResolutionY uint32    // 04:
	ImageOffset uint32    // 08:
	ImageLength uint32    // 0c:
	_           [4]uint32 // 10:
}

type fdgLayerDef struct {
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

type fdgImageInfo struct {
	LayerDef     fdgLayerDef // 00:  Repeat of the LayerDef information
	TotalSize    uint32      // 24:  Total size of fdgImageInfo and Image data
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
	layerDef  []fdgLayerDef
	imageInfo [](*fdgImageInfo)

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
	cf.IntVarP(&cf.Version, "version", "v", 2, "Specify the CTB version (2 or 3)")

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
	header := fdgHeader{
		Magic:          defaultHeaderMagic,
		Version:        uint32(cf.Version),
		EncryptionSeed: seed,
		AntiAliasLevel: 1,
	}
	headerSize, _ := restruct.SizeOf(&header)

	// Add the preview images
	var previewHuge fdgPreview
	var previewTiny fdgPreview
	previewSize, _ := restruct.SizeOf(&previewHuge)

	// Set up the RLE hash indexes
	rleHashList := []uint64{}

	savePreview := func(base uint32, preview *fdgPreview, ptype uv3dp.PreviewType) uint32 {
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
	machine := "Voxelab Polaris"
	machineSize := len(machine)

	layerDefBase := machineBase + uint32(machineSize)

	layerDef := make([]fdgLayerDef, size.Layers)
	imageInfo := make([]fdgImageInfo, size.Layers)
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

	info_size, _ := restruct.SizeOf(&fdgImageInfo{})
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

		layerDef[n] = fdgLayerDef{
			LayerHeight:   info.Z,
			LayerExposure: info.Exposure.LightOnTime,
			LayerOffTime:  info.Exposure.LightOffTime,
			ImageOffset:   rleHash[info.Hash].offset,
			ImageLength:   uint32(len(info.Rle)),
			InfoSize:      imageInfoSize,
		}

		if imageInfoSize > 0 {
			imageInfo[n] = fdgImageInfo{
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

	// fdgHeader
	header.BedSizeMM[0] = size.Millimeter.X
	header.BedSizeMM[1] = size.Millimeter.Y
	header.BedSizeMM[2] = 155.0
	header.HeightMM = size.LayerHeight * float32(size.Layers)
	header.LayerHeight = size.LayerHeight
	header.LayerExposure = exp.LightOnTime
	header.BottomExposure = bot.Exposure.LightOnTime
	header.LightOffTime = exp.LightOffTime
	header.BottomCount = uint32(bot.Count)
	header.ResolutionX = uint32(size.X)
	header.ResolutionY = uint32(size.Y)
	header.PreviewHigh = previewHugeBase
	header.LayerDefs = layerDefBase
	header.LayerCount = uint32(size.Layers)
	header.PreviewLow = previewTinyBase
	header.PrintTime = uint32(uv3dp.PrintDuration(printable) / time.Second)
	header.Projector = 1 // LCD_X_MIRROR

	header.AntiAliasDepth = 4
	header.EncryptionMode = 0x4c
	header.MachineOffset = machineBase
	header.MachineSize = uint32(machineSize)
	header.ChiTuBoxVersion[0] = 0
	header.ChiTuBoxVersion[1] = 0
	header.ChiTuBoxVersion[2] = 7
	header.ChiTuBoxVersion[3] = 1

	if exp.LightPWM == 0 {
		exp.LightPWM = 255
	}

	if bot.Exposure.LightPWM == 0 {
		bot.Exposure.LightPWM = 255
	}

	header.LightPWM = uint16(exp.LightPWM)
	header.BottomLightPWM = uint16(bot.Exposure.LightPWM)

	// fdgParam
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

	// Compute total cubic millimeters (== milliliters) of all the on pixels
	bedArea := float64(header.BedSizeMM[0] * header.BedSizeMM[1])
	bedPixels := float64(header.ResolutionX) * float64(header.ResolutionY)
	pixelVolume := float64(header.LayerHeight) * bedArea / bedPixels * 200.0
	header.VolumeMilliliters = float32(float64(totalOn) * pixelVolume / 1000.0)
	header.WeightGrams = header.VolumeMilliliters * 1.1 // Just a guess on resin density
	header.CostDollars = header.WeightGrams * 0.1

	header.BottomLightOffTime = bot.Exposure.LightOffTime
	header.LightOffTime = exp.LightOffTime
	header.BottomLayerCount = header.BottomCount
	header.Timestamp = uint32(time.Now().Unix() / 60)

	// Collect file data
	fileData := map[int][]byte{}

	fileData[int(headerBase)], _ = restruct.Pack(binary.LittleEndian, &header)

	for n, layer := range layerDef {
		base := int(layerDefBase) + layerDefSize*n
		fileData[base], _ = restruct.Pack(binary.LittleEndian, &layer)
	}

	fileData[int(previewHugeBase)], _ = restruct.Pack(binary.LittleEndian, &previewHuge)
	fileData[int(previewTinyBase)], _ = restruct.Pack(binary.LittleEndian, &previewTiny)

	fileData[int(machineBase)] = ([]byte)(machine)

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
		Preview: make(map[uv3dp.PreviewType]image.Image),
	}

	header := fdgHeader{}
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

		var preview fdgPreview
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

	layerDef := make([]fdgLayerDef, header.LayerCount)

	imageInfo := make([](*fdgImageInfo), header.LayerCount)

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
			info := &fdgImageInfo{}
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
	exp.LightOffTime = header.LightOffTime
	exp.LightPWM = uint8(header.LightPWM)

	bot := &prop.Bottom
	bot.Count = int(header.BottomCount)
	bot.Exposure.LightOnTime = header.BottomExposure
	bot.Exposure.LightOffTime = header.LightOffTime
	bot.Exposure.LightPWM = uint8(header.BottomLightPWM)

	bot.Count = int(header.BottomLayerCount)
	bot.Exposure.LiftHeight = header.BottomLiftHeight
	bot.Exposure.LiftSpeed = header.BottomLiftSpeed
	bot.Exposure.LightOffTime = header.BottomLightOffTime
	bot.Exposure.RetractSpeed = header.RetractSpeed
	bot.Exposure.RetractHeight = defaultRetractHeight

	exp.LiftHeight = header.LiftHeight
	exp.LiftSpeed = header.LiftSpeed
	exp.LightOffTime = header.LightOffTime
	exp.RetractSpeed = header.RetractSpeed
	exp.RetractHeight = defaultRetractHeight

	fdg := &Print{
		Print:     uv3dp.Print{Properties: prop},
		layerDef:  layerDef,
		imageInfo: imageInfo,
		rleMap:    rleMap,
	}

	printable = fdg

	return
}

func (fdg *Print) LayerImage(index int) (layerImage *image.Gray) {
	layerDef := fdg.layerDef[index]

	// Update per-layer info
	layerImage, err := rleDecodeGraymap(fdg.Bounds(), fdg.rleMap[layerDef.ImageOffset])
	if err != nil {
		panic(err)
	}

	return
}

func (fdg *Print) LayerExposure(index int) (exposure uv3dp.Exposure) {
	layerDef := fdg.layerDef[index]

	if index < fdg.Bottom().Count {
		exposure = fdg.Bottom().Exposure
	} else {
		exposure = fdg.Exposure()
	}

	if layerDef.LayerExposure > 0.0 {
		exposure.LightOnTime = layerDef.LayerExposure
	}

	if layerDef.LayerOffTime > 0.0 {
		exposure.LightOffTime = layerDef.LayerOffTime
	}

	// See if we have per-layer overrides
	info := fdg.imageInfo[index]
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

func (fdg *Print) LayerZ(index int) (z float32) {
	z = fdg.layerDef[index].LayerHeight
	return
}
