//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package lgs handles input and output of Longer Orange 10 print files
package lgs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"io"

	"github.com/ezrec/uv3dp"
	"github.com/go-restruct/restruct"
	"github.com/spf13/pflag"
)

var (
	headerMagic = []byte{76, 111, 110, 103, 101, 114, 51, 68}
)

type lgsHeader struct {
	Name                  [8]uint8 // 0x00:
	Uint_08               uint32   // 0x08: 0xff000001 ?
	Uint_0c               uint32   // 0x0c: 1 ?
	Uint_10               uint32   // 0x10: 30 ?
	Uint_14               uint32   // 0x14: 0 ?
	Uint_18               uint32   // 0x18: 34 ?
	PixelPerMmY           float32  // 0x1c:
	PixelPerMmX           float32  // 0x20:
	ImageY                float32  // 0x24
	ImageX                float32  // 0x28
	LayerHeight           float32  // 0x2c
	ExposureTimeMs        float32  // 0x30
	BottomExposureTimeMs  float32  // 0x34
	Float_38              float32  // 0x38: 10
	LightOffDelayMs       float32  // 0x3c
	BottomLightOffDelayMs float32  // 0x40
	BottomHeight          float32  // 0x44
	Float_48              float32  // 0x48: 0.6
	BottomLiftHeight      float32  // 0x4c
	LiftHeight            float32  // 0x50
	LiftSpeed             float32  // 0x54
	LiftSpeed_            float32  // 0x58
	BottomLiftSpeed       float32  // 0x5c
	BottomLiftSpeed_      float32  // 0x60
	Float_64              float32  // 0x64: 5?
	Float_68              float32  // 0x68: 60?
	Float_6c              float32  // 0x6c: 10?
	Float_70              float32  // 0x70: 600?
	Float_74              float32  // 0x74: 600?
	Float_78              float32  // 0x78: 2?
	Float_7c              float32  // 0x7c: 0.2?
	Float_80              float32  // 0x80: 60?
	Float_84              float32  // 0x84: 1?
	Float_88              float32  // 0x88: 6?
	Float_8c              float32  // 0x8c: 150 ?
	Float_90              float32  // 0x90: 1001 ?
	Float_94              float32  // 0x94: 140 for Longer 10, 170 for Longer 30?
	Uint_98               uint32   // 0x98: 0 ?
	Uint_9c               uint32   // 0x9c: 0 ?
	Uint_a0               uint32   // 0xa0: 0 ?
	LayerCount            uint32   // 0xa4
	Uint_a8               uint32   // 0xa8: 4 ?
	PreviewSizeX          uint32   // 0xac
	PreviewSizeY          uint32   // 0xb0
}

type lgsImage struct {
	Size uint32 `struct:"sizeof=Rle"`
	Rle  []byte `struct:"sizefrom=Size"`
}

type Print struct {
	uv3dp.Print

	rleMap []([]byte)
}

type Formatter struct {
	*pflag.FlagSet

	model int
}

func NewFormatter(suffix string, model int) (f *Formatter) {
	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)
	flagSet.SetInterspersed(false)

	f = &Formatter{
		FlagSet: flagSet,
		model:   model,
	}

	return
}

func (f *Formatter) Encode(writer uv3dp.Writer, p uv3dp.Printable) (err error) {

	size := p.Size()
	exp := p.Exposure()
	bot := p.Bottom()

	fModel := float32(140)
	if f.model == 30 {
		fModel = 170
	}

	preview, ok := p.Preview(uv3dp.PreviewTypeTiny)
	previewSize := image.Pt(0, 0)
	if ok {
		previewSize = preview.Bounds().Size()
	}

	header := lgsHeader{
		Uint_08:               0xff000001,
		Uint_0c:               1,
		Uint_10:               uint32(f.model),
		Uint_14:               0,
		Uint_18:               34,
		PixelPerMmY:           float32(size.Y) / float32(size.Millimeter.Y),
		PixelPerMmX:           float32(size.X) / float32(size.Millimeter.X),
		ImageY:                float32(size.Y),
		ImageX:                float32(size.X),
		LayerHeight:           float32(size.LayerHeight),
		ExposureTimeMs:        exp.LightOnTime * 1000.0,
		BottomExposureTimeMs:  bot.Exposure.LightOnTime * 1000.0,
		Float_38:              10.0,
		LightOffDelayMs:       exp.LightOffTime * 1000.0,
		BottomLightOffDelayMs: bot.Exposure.LightOffTime * 1000.0,
		BottomHeight:          float32(bot.Count) * size.LayerHeight,
		Float_48:              0.6,
		BottomLiftHeight:      bot.Exposure.LiftHeight,
		LiftHeight:            exp.LiftHeight,
		LiftSpeed:             exp.LiftSpeed,
		LiftSpeed_:            exp.LiftSpeed,
		BottomLiftSpeed:       bot.Exposure.LiftSpeed,
		BottomLiftSpeed_:      bot.Exposure.LiftSpeed,
		Float_64:              5,
		Float_68:              60,
		Float_6c:              10,
		Float_70:              600,
		Float_74:              600,
		Float_78:              2,
		Float_7c:              0.2,
		Float_80:              60,
		Float_84:              1,
		Float_88:              6,
		Float_8c:              150,
		Float_90:              1001,
		Float_94:              fModel,
		LayerCount:            uint32(size.Layers),
		Uint_a8:               4,
		PreviewSizeX:          uint32(previewSize.X),
		PreviewSizeY:          uint32(previewSize.Y),
	}

	copy(header.Name[:], headerMagic)

	data, err := restruct.Pack(binary.LittleEndian, &header)
	if err != nil {
		return
	}

	_, err = writer.Write(data)
	if err != nil {
		return
	}

	if previewSize != image.Pt(0, 0) {
		data = RGB15Encode(preview)

		_, err = writer.Write(data)
		if err != nil {
			return
		}
	}

	layerChan := make([](chan []byte), size.Layers)
	for n := range layerChan {
		layerChan[n] = make(chan []byte, 1)
	}

	uv3dp.WithAllLayers(p, func(p uv3dp.Printable, n int) {
		var rle []byte
		rle, err = Rle4Encode(p.LayerImage(n))
		if err == nil {
			layerChan[n] <- rle
		}
		close(layerChan[n])
	})

	for _, done := range layerChan {
		rle := <-done

		layer := lgsImage{
			Size: uint32(len(rle)),
			Rle:  rle,
		}
		var out []byte
		out, err = restruct.Pack(binary.LittleEndian, &layer)
		_, err = writer.Write(out)
		if err != nil {
			return
		}
	}

	return
}

func (cf *Formatter) Decode(file uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {
	// Collect file
	data, err := io.ReadAll(file)
	if err != nil {
		return
	}

	if !bytes.Equal(data[:len(headerMagic)], headerMagic) {
		err = fmt.Errorf("unexpected header magic number")
		return
	}

	header := lgsHeader{}
	err = restruct.Unpack(data, binary.LittleEndian, &header)
	if err != nil {
		return
	}

	size := uv3dp.Size{}
	size.Layers = int(header.LayerCount)
	size.LayerHeight = header.LayerHeight
	size.X = int(header.ImageX)
	size.Y = int(header.ImageY)
	size.Millimeter.X = header.ImageX / header.PixelPerMmX
	size.Millimeter.Y = header.ImageY / header.PixelPerMmY

	exp := uv3dp.Exposure{}
	exp.LiftHeight = header.LiftHeight
	exp.LiftSpeed = header.LiftSpeed
	exp.LightOnTime = header.ExposureTimeMs / 1000.0
	exp.LightOffTime = header.LightOffDelayMs / 1000.0
	exp.LightPWM = 255

	bot := uv3dp.Bottom{}
	bot.Count = int(header.BottomHeight / header.LayerHeight)
	bot.Exposure.LiftHeight = header.BottomLiftHeight
	bot.Exposure.LiftSpeed = header.BottomLiftSpeed
	bot.Exposure.LightOnTime = header.BottomExposureTimeMs / 1000.0
	bot.Exposure.LightOffTime = header.BottomLightOffDelayMs / 1000.0
	bot.Exposure.LightPWM = 255

	offset := 0xb4
	sizeX := int(header.PreviewSizeX)
	sizeY := int(header.PreviewSizeY)
	previewSize := sizeX * sizeY * 2
	previewRaw := data[offset : offset+previewSize]

	preview := RGB15Decode(image.Rect(0, 0, sizeX, sizeY), previewRaw)
	offset += previewSize
	previewMap := map[uv3dp.PreviewType]image.Image{
		uv3dp.PreviewTypeTiny: preview,
	}

	rleMap := []([]byte){}

	for offset < len(data) {
		rleSize := uint32(data[offset]) |
			(uint32(data[offset+1]) << 8) |
			(uint32(data[offset+2]) << 16) |
			(uint32(data[offset+3]) << 24)
		offset += 4
		rleMap = append(rleMap, data[offset:offset+int(rleSize)])
		offset += int(rleSize)
	}

	lgs := &Print{
		Print: uv3dp.Print{Properties: uv3dp.Properties{
			Size:     size,
			Preview:  previewMap,
			Exposure: exp,
			Bottom:   bot,
		}},
		rleMap: rleMap,
	}

	printable = lgs

	return
}

func (p *Print) LayerImage(index int) (gi *image.Gray) {
	return Rle4Decode(p.rleMap[index], p.Bounds())
}
