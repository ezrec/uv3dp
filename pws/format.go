//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package pws

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/go-restruct/restruct"
	"image"
	"io/ioutil"

	"github.com/ezrec/uv3dp"
	"github.com/spf13/pflag"
	"golang.org/x/image/draw"
)

var sectionMarkFilemark = [12]byte{'A', 'N', 'Y', 'C', 'U', 'B', 'I', 'C'}

type Filemark struct {
	Mark           [12]byte // Forced to 'ANYCUBIC'
	Version        uint32   // Forced to 1
	AreaNum        uint32   // Forced to 4
	HeaderAddr     uint32
	_              uint32
	PreviewAddr    uint32
	_              uint32
	LayerDefAddr   uint32
	_              uint32
	LayerImageAddr uint32
}

type Section struct {
	Mark   [12]byte // Section mark
	Length uint32   // Length of this section
}

func (sec *Section) Marshal(from interface{}, raw []byte) (data []byte, err error) {
	from_data, err := restruct.Pack(binary.LittleEndian, from)
	if err != nil {
		return
	}

	sec.Length = uint32(len(from_data) + len(raw))
	sec_data, err := restruct.Pack(binary.LittleEndian, sec)
	if err != nil {
		return
	}

	data = append(sec_data, from_data...)
	data = append(data, raw...)

	return
}

func (sec *Section) Unmarshal(raw []byte, into interface{}) (data []byte, err error) {
	var newSection Section

	err = restruct.Unpack(raw, binary.LittleEndian, &newSection)
	if err != nil {
		return
	}

	if !bytes.Equal(sec.Mark[:], newSection.Mark[:]) {
		err = fmt.Errorf("section mark expected %+v, got %+v", sec.Mark, newSection.Mark)
		return
	}

	sec.Length = newSection.Length

	sec_size, _ := restruct.SizeOf(sec)

	raw = raw[sec_size : sec_size+int(sec.Length)]

	err = restruct.Unpack(raw, binary.LittleEndian, into)
	if err != nil {
		return
	}

	into_size, err := restruct.SizeOf(into)
	if err != nil {
		return
	}

	// Return 'leftover' data
	data = raw[into_size:]

	return
}

var sectionMarkHeader = [12]byte{'H', 'E', 'A', 'D', 'E', 'R'}

type Header struct {
	PixelSize         float32
	LayerHeight       float32
	LightOnTime       float32
	LightOffTime      float32
	BottomLightOnTime float32
	BottomLayers      float32
	LiftHeight        float32
	LiftSpeed         float32 // In mm/second
	RetractSpeed      float32 // In mm/second
	Volume            float32
	AntiAlias         uint32
	ResolutionX       uint32
	ResolutionY       uint32
	Weight            float32
	Price             float32
	ResinType         uint32 // 0x24 ?
	PerLayerOverride  uint32 // bool
	_                 [3]uint32
}

func (header *Header) Marshal(offset uint32) (data []byte, err error) {
	data, err = (&Section{Mark: sectionMarkHeader}).Marshal(header, []byte{})
	return
}

func (header *Header) Unmarshal(raw []byte) (err error) {
	_, err = (&Section{Mark: sectionMarkHeader}).Unmarshal(raw, header)
	return
}

var sectionMarkPreview = [12]byte{'P', 'R', 'E', 'V', 'I', 'E', 'W'}

const (
	defaultPreviewWidth  = 224
	defaultPreviewHeight = 168
)

type Preview struct {
	Width      uint32 // Image width
	Resolution uint32
	Height     uint32 // Image height

	// little-endian 16bit colors, RGB 565 encoded.
	imageData []byte
}

func (preview *Preview) Marshal(offset uint32) (data []byte, err error) {
	data, err = (&Section{Mark: sectionMarkPreview}).Marshal(preview, preview.imageData)
	return
}

func (preview *Preview) Unmarshal(raw []byte) (err error) {
	data, err := (&Section{Mark: sectionMarkPreview}).Unmarshal(raw, preview)

	if len(data) != int(2*(preview.Width*preview.Height)) {
		err = fmt.Errorf("preview image %vx%v: Expected %d bytes of image, got %v",
			preview.Width, preview.Height,
			2*(preview.Width*preview.Height),
			len(data))
		return
	}

	preview.imageData = data

	return
}

func (preview *Preview) GetImage() (preview_image *image.RGBA, err error) {
	bounds := image.Rect(0, 0, int(preview.Width), int(preview.Height))
	preview_image = image.NewRGBA(bounds)
	pix := preview_image.Pix
	data := preview.imageData

	if len(pix)/4 != len(data)/2 {
		panic(fmt.Sprintf("expected %v pixels, got %v", len(pix)/4, len(data)/2))
	}

	for n := 0; n < len(data); n += 2 {
		color16 := binary.LittleEndian.Uint16(data[n : n+2])
		r5 := uint8((color16 >> 11) & 0x1f)
		g5 := uint8((color16 >> 5) & 0x3f)
		b5 := uint8((color16 >> 0) & 0x1f)
		pix[n*2+0] = (r5 << 3) | (r5 & 0x7)
		pix[n*2+1] = (g5 << 2) | (g5 & 0x3)
		pix[n*2+2] = (b5 << 3) | (b5 & 0x7)
		pix[n*2+3] = 0xff
	}

	return
}

func (preview *Preview) SetImage(preview_image image.Image) {
	preview.Width = defaultPreviewWidth
	preview.Height = defaultPreviewHeight
	preview.Resolution = 42 // dpi?

	// Rescale to the expected preview size
	newRect := image.Rect(0, 0, defaultPreviewWidth, defaultPreviewHeight)
	newImage := image.NewRGBA(newRect)
	draw.NearestNeighbor.Scale(newImage, newRect, preview_image, preview_image.Bounds(), draw.Src, nil)

	pix := newImage.Pix
	data := make([]byte, newRect.Size().X*newRect.Size().Y*2)
	for n := 0; n < len(pix); n += 4 {
		r := pix[n+0] >> 3
		g := pix[n+1] >> 2
		b := pix[n+2] >> 3
		color := (uint16(r) << 11) | (uint16(g) << 5) | (uint16(b) << 0)
		binary.LittleEndian.PutUint16(data[n/2:], color)
	}

	preview.imageData = data
}

type SliceFormat int

const (
	SliceFormatPWS = SliceFormat(iota)
	SliceFormatPW0
)

type Slice struct {
	AntiAlias int
	Format    SliceFormat
	Bounds    image.Rectangle
	Data      []byte
}

func (slice *Slice) GetImage() (gray *image.Gray, err error) {
	switch slice.Format {
	case SliceFormatPWS:
		gray, err = rle1DecodeBitmaps(slice.Bounds, slice.Data, slice.AntiAlias)
	case SliceFormatPW0:
		gray, err = rle4DecodeBitmaps(slice.Bounds, slice.Data, slice.AntiAlias)
	}

	return
}

func (slice *Slice) SetImage(gray *image.Gray) (err error) {
	var data []byte
	switch slice.Format {
	case SliceFormatPWS:
		for bit := 0; bit < slice.AntiAlias; bit++ {
			rle, _, _ := rle1EncodeBitmap(gray, bit, slice.AntiAlias)
			data = append(data, rle...)
		}
	case SliceFormatPW0:
		data, err = rle4EncodeBitmaps(gray, slice.AntiAlias)
		if err != nil {
			return
		}
	}

	slice.Bounds = gray.Bounds()
	slice.Data = data

	return
}

type Layer struct {
	ImageAddr   uint32
	ImageLength uint32
	LiftHeight  float32
	LiftSpeed   float32
	LightOnTime float32
	LayerHeight float32
	_           [2]float32

	slice Slice
}

var sectionMarkLayerDef = [12]byte{'L', 'A', 'Y', 'E', 'R', 'D', 'E', 'F'}

type LayerDef struct {
	Layers uint32  `struct:"sizeof=Layer"`
	Layer  []Layer `struct:"sizefrom=Layers"`
}

func (layerdef *LayerDef) Marshal(offset uint32) (data []byte, err error) {
	data, err = (&Section{Mark: sectionMarkLayerDef}).Marshal(layerdef, []byte{})
	return
}

func (layerdef *LayerDef) Unmarshal(data []byte) (err error) {
	_, err = (&Section{Mark: sectionMarkLayerDef}).Unmarshal(data, layerdef)
	return
}

type Print struct {
	properties       uv3dp.Properties
	perLayerOverride bool
	layers           []Layer
}

type Format struct {
	*pflag.FlagSet

	AntiAlias   int // AntiAlias level, one of [1,2,4,8]
	sliceFormat SliceFormat
}

func NewFormatter(suffix string) (sf *Format) {
	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)

	sf = &Format{
		FlagSet: flagSet,
	}

	switch suffix {
	case ".pws":
		sf.sliceFormat = SliceFormatPWS
	case ".pw0":
		sf.sliceFormat = SliceFormatPW0
	}

	sf.IntVarP(&sf.AntiAlias, "anti-alias", "a", 1, "Override antialias level (1,2,4,8)")

	sf.SetInterspersed(false)

	return
}

func (sf *Format) Encode(writer uv3dp.Writer, printable uv3dp.Printable) (err error) {
	prop := printable.Properties()

	filemark := Filemark{
		Mark:    sectionMarkFilemark,
		Version: 1,
		AreaNum: 4,
	}

	header := Header{
		// TODO: Check for 'squareness' of pixels?
		PixelSize:         prop.Size.Millimeter.X / float32(prop.Size.X) * 1000.0,
		LayerHeight:       prop.Size.LayerHeight,
		LightOnTime:       prop.Exposure.LightOnTime,
		LightOffTime:      prop.Exposure.LightOffTime,
		BottomLightOnTime: prop.Bottom.Exposure.LightOnTime,
		BottomLayers:      float32(prop.Bottom.Count),
		LiftHeight:        prop.Exposure.LiftHeight,
		LiftSpeed:         prop.Exposure.LiftSpeed / 60.0,
		RetractSpeed:      prop.Exposure.RetractSpeed / 60.0,
		AntiAlias:         uint32(sf.AntiAlias),
		ResolutionX:       uint32(prop.Size.X),
		ResolutionY:       uint32(prop.Size.Y),
		PerLayerOverride:  1, // true
	}

	var preview Preview

	previewImage, ok := prop.Preview[uv3dp.PreviewTypeTiny]
	if ok {
		preview.SetImage(previewImage)
	}

	layers := make([]Layer, prop.Size.Layers)

	uv3dp.WithAllLayers(printable, func(n int, layer uv3dp.Layer) {
		l := Layer{
			LiftHeight:  layer.Exposure.LiftHeight,
			LiftSpeed:   layer.Exposure.LiftSpeed,
			LightOnTime: layer.Exposure.LightOnTime,
			LayerHeight: header.LayerHeight,
		}

		l.slice.AntiAlias = sf.AntiAlias
		l.slice.Format = sf.sliceFormat
		l.slice.SetImage(layer.Image)

		layers[n] = l
	})

	layerdef := LayerDef{
		Layers: uint32(len(layers)),
		Layer:  layers,
	}

	filemarkSize, err := restruct.SizeOf(&filemark)
	if err != nil {
		return
	}

	filemark.HeaderAddr = uint32(filemarkSize)

	headerData, err := header.Marshal(filemark.HeaderAddr)
	if err != nil {
		return
	}

	filemark.PreviewAddr = filemark.HeaderAddr + uint32(len(headerData))

	previewData, err := preview.Marshal(filemark.PreviewAddr)
	if err != nil {
		return
	}

	filemark.LayerDefAddr = filemark.PreviewAddr + uint32(len(previewData))

	layerdefData, err := layerdef.Marshal(filemark.LayerDefAddr)
	if err != nil {
		return
	}

	filemark.LayerImageAddr = filemark.LayerDefAddr + uint32(len(layerdefData))

	// Compute the layer offset
	offset := filemark.LayerImageAddr
	for n := 0; n < len(layers); n++ {
		size := uint32(len(layers[n].slice.Data))
		layers[n].ImageAddr = offset
		layers[n].ImageLength = size
		offset += size
	}
	layerdef.Layer = layers

	// Rebuild the layerdef data, now that we have all the image addresses filled in
	layerdefData, err = layerdef.Marshal(filemark.LayerDefAddr)
	if err != nil {
		return
	}

	filemarkData, err := restruct.Pack(binary.LittleEndian, &filemark)
	if err != nil {
		return
	}

	// Write out filemark
	_, err = writer.Write(filemarkData)
	if err != nil {
		return
	}

	// Write out header
	_, err = writer.Write(headerData)
	if err != nil {
		return
	}

	// Write out preview
	_, err = writer.Write(previewData)
	if err != nil {
		return
	}

	// Write out layerdef
	_, err = writer.Write(layerdefData)
	if err != nil {
		return
	}

	// Write out layer images
	for _, layer := range layerdef.Layer {
		_, err = writer.Write(layer.slice.Data)
	}

	return
}

func (sf *Format) Decode(reader uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {

	raw, err := ioutil.ReadAll(reader)
	if err != nil {
		return
	}

	var filemark Filemark

	err = restruct.Unpack(raw, binary.LittleEndian, &filemark)
	if err != nil {
		return
	}

	if !bytes.Equal(filemark.Mark[:], sectionMarkFilemark[:]) {
		err = fmt.Errorf("invalid Filemark mark %+v, expected %+v", filemark.Mark, sectionMarkFilemark)
		return
	}

	if filemark.Version != 1 {
		err = fmt.Errorf("invalid Version %v, exepcted %v", filemark.Version, 1)
		return
	}

	// Extract header
	var header Header

	err = header.Unmarshal(raw[int(filemark.HeaderAddr):])
	if err != nil {
		return
	}

	// Extract preview
	var preview Preview

	err = preview.Unmarshal(raw[int(filemark.PreviewAddr):])
	if err != nil {
		return
	}

	// Convert from RGB15 to RGBA
	previewImage, err := preview.GetImage()
	if err != nil {
		return
	}

	// Extract layerdef
	var layerdef LayerDef

	err = layerdef.Unmarshal(raw[int(filemark.LayerDefAddr):])
	if err != nil {
		return
	}

	bounds := image.Rect(0, 0, int(header.ResolutionX), int(header.ResolutionY))
	for n, layer := range layerdef.Layer {
		layerdef.Layer[n].slice = Slice{
			Data:      raw[int(layer.ImageAddr) : int(layer.ImageAddr)+int(layer.ImageLength)],
			Bounds:    bounds,
			Format:    sf.sliceFormat,
			AntiAlias: int(header.AntiAlias),
		}
	}

	exposure := uv3dp.Exposure{
		LightOnTime:  header.LightOnTime,
		LightOffTime: header.LightOffTime,
		LightPWM:     255,
		LiftHeight:   header.LiftHeight,
		LiftSpeed:    header.LiftSpeed * 60.0,
		RetractSpeed: header.RetractSpeed * 60.0,
	}

	bottom := uv3dp.Bottom{
		Count: int(header.BottomLayers),
		Exposure: uv3dp.Exposure{
			LightOnTime:  header.BottomLightOnTime,
			LightOffTime: header.LightOffTime,
			LightPWM:     255,
			LiftHeight:   header.LiftHeight,
			LiftSpeed:    header.LiftSpeed * 60.0,
			RetractSpeed: header.RetractSpeed * 60.0,
		},
	}

	prop := uv3dp.Properties{
		Size: uv3dp.Size{
			X: int(header.ResolutionX),
			Y: int(header.ResolutionY),
			Millimeter: uv3dp.SizeMillimeter{
				X: float32(header.ResolutionX) * header.PixelSize / 1000.0,
				Y: float32(header.ResolutionY) * header.PixelSize / 1000.0,
			},
			Layers:      len(layerdef.Layer),
			LayerHeight: header.LayerHeight,
		},
		Exposure: exposure,
		Bottom:   bottom,
		Preview: map[uv3dp.PreviewType]image.Image{
			uv3dp.PreviewTypeTiny: previewImage,
		},
	}

	printable = &Print{
		properties:       prop,
		layers:           layerdef.Layer,
		perLayerOverride: header.PerLayerOverride != 0,
	}

	return
}

func (sf *Print) Close() {
}

func (pws *Print) Properties() (prop uv3dp.Properties) {
	prop = pws.properties
	return
}

func (pws *Print) Layer(index int) (layer uv3dp.Layer) {
	l := pws.layers[index]

	prop := &pws.properties
	layer.Exposure = prop.LayerExposure(index)

	if pws.perLayerOverride {
		layer.Exposure.LightOnTime = l.LightOnTime
		layer.Exposure.LiftHeight = l.LiftHeight
		layer.Exposure.LiftSpeed = l.LiftSpeed * 60
	}

	slice, err := pws.layers[index].slice.GetImage()
	if err != nil {
		panic(fmt.Sprintf("pws: layer %v/%v: %s", index+1, prop.Size.Layers, err))
	}

	layer.Z = float32(index) * prop.Size.LayerHeight
	layer.Image = slice
	return
}
