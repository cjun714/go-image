package webp

// #cgo LDFLAGS: -lwebp
// #include <stdlib.h>
// #include <webp/encode.h>
import "C"

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/ioutil"
	"reflect"
	"unsafe"

	"github.com/disintegration/imaging"
)

// type WebPConfig struct {
// 	lossless int32   // Lossless encoding (0=lossy(default), 1=lossless).
// 	quality  float32 // between 0 and 100. For lossy, 0 gives the smallest
// 	// size and 100 the largest. For lossless, this
// 	// parameter is the amount of effort put into the
// 	// compression: 0 is the fastest but gives larger
// 	// files compared to the slowest, but best, 100.
// 	method int32 // quality/speed trade-off (0=fast, 6=slower-better)

// 	image_hint int32 // Hint for image type (lossless only for now).

// 	target_size int32 // if non-zero, set the desired target size in bytes.
// 	// Takes precedence over the 'compression' parameter.
// 	target_PSNR float32 // if non-zero, specifies the minimal distortion to
// 	// try to achieve. Takes precedence over target_size.
// 	segments         int32 // maximum number of segments to use, in [1..4]
// 	sns_strength     int32 // Spatial Noise Shaping. 0=off, 100=maximum.
// 	filter_strength  int32 // range: [0 = off .. 100 = strongest]
// 	filter_sharpness int32 // range: [0 = off .. 7 = least sharp]
// 	filter_type      int32 // filtering type: 0 = simple, 1 = strong (only used
// 	// if filter_strength > 0 or autofilter > 0)
// 	autofilter        int32 // Auto adjust filter's strength [0 = off, 1 = on]
// 	alpha_compression int32 // Algorithm for encoding the alpha plane (0 = none,
// 	// 1 = compressed with WebP lossless). Default is 1.
// 	alpha_filtering int32 // Predictive filtering method for alpha plane.
// 	//  0: none, 1: fast, 2: best. Default if 1.
// 	alpha_quality int32 // Between 0 (smallest size) and 100 (lossless).
// 	// Default is 100.
// 	pass int32 // number of entropy-analysis passes (in [1..10]).

// 	show_compressed int32 // if true, export the compressed picture back.
// 	// In-loop filtering is not applied.
// 	preprocessing int32 // preprocessing filter:
// 	// 0=none, 1=segment-smooth, 2=pseudo-random dithering
// 	partitions int32 // log2(number of token partitions) in [0..3]. Default
// 	// is set to 0 for easier progressive decoding.
// 	partition_limit int32 // quality degradation allowed to fit the 512k limit
// 	// on prediction modes coding (0: no degradation,
// 	// 100: maximum possible degradation).
// 	emulate_jpeg_size int32 // If true, compression parameters will be remapped
// 	// to better match the expected output size from
// 	// JPEG compression. Generally, the output size will
// 	// be similar but the degradation will be lower.
// 	thread_level int32 // If non-zero, try and use multi-threaded encoding.
// 	low_memory   int32 // If set, reduce memory usage (but increase CPU use).

// 	near_lossless int32 // Near lossless encoding [0 = max loss .. 100 = off
// 	// (default)].
// 	exact int32 // if non-zero, preserve the exact RGB values under
// 	// transparent area. Otherwise, discard this invisible
// 	// RGB information for better compression. The default
// 	// value is 0.

// 	use_delta_palette int32 // reserved for future lossless feature
// 	use_sharp_yuv     int32 // if needed, use sharp (and slow) RGB->YUV conversion

// 	pad [2]uint32 // padding for later use
// }

type WebPPreset int

const (
	WEBP_PRESET_DEFAULT WebPPreset = iota // default preset.
	WEBP_PRESET_PICTURE                   // digital picture, like portrait, inner shot
	WEBP_PRESET_PHOTO                     // outdoor photograph, with natural lighting
	WEBP_PRESET_DRAWING                   // hand or line drawing, with high-contrast details
	WEBP_PRESET_ICON                      // small-sized colorful images
	WEBP_PRESET_TEXT                      // text-like
)

type Config C.struct_WebPConfig
type Pic C.struct_WebPPicture

func NewConfig(preset WebPPreset, quality int) (*Config, error) {
	config := Config{}

	ret := C.WebPConfigPreset((*C.struct_WebPConfig)(&config),
		C.WebPPreset(preset), C.float(float32(quality)))
	if ret == 0 {
		return nil, fmt.Errorf("init config failed")
	}

	return &config, nil
}

func Encode(wr io.WriteCloser, img image.Image) error {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	var pix []byte
	// pix := make([]byte, w*h*3)
	var stride int
	switch t := img.(type) {
	case *image.NRGBA:
		pix = t.Pix
		stride = w * 4
	case *image.RGBA:
		pix = t.Pix
		stride = w * 4
	case *image.Gray:
		pix = make([]byte, w*h*3)
		length := len(t.Pix)
		for i := 0; i < length; i++ {
			pix[i*3], pix[i*3+1], pix[i*3+2] = t.Pix[i], t.Pix[i], t.Pix[i]
		}
		stride = w * 3
	case *image.YCbCr:
		pix = make([]byte, w*h*3)
		idx := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				c := t.YCbCrAt(x, y)
				pix[idx], pix[idx+1], pix[idx+2] = color.YCbCrToRGB(c.Y, c.Cb, c.Cr)
				idx += 3
			}
		}
		stride = w * 3
	default:
		return fmt.Errorf("unsupported image type:%T", t)
	}

	config := &Config{}

	ret := C.WebPConfigPreset((*C.struct_WebPConfig)(unsafe.Pointer(config)),
		C.WebPPreset(WEBP_PRESET_DRAWING), C.float(30))
	if ret == 0 {
		return fmt.Errorf("init config failed")
	}
	// config.sns_strength = 90
	// config.filter_sharpness = 6
	err := C.WebPValidateConfig((*C.struct_WebPConfig)(unsafe.Pointer(config)))
	if err == 0 {
		return fmt.Errorf("config is invalid")
	}

	pic := &Pic{}
	if C.WebPPictureInit((*C.struct_WebPPicture)(unsafe.Pointer(pic))) == 0 {
		return fmt.Errorf("init WebPPicture failed")
	}

	((*C.WebPPicture)(unsafe.Pointer(pic))).argb = (*C.uint32_t)(unsafe.Pointer(&pix[0]))
	((*C.WebPPicture)(unsafe.Pointer(pic))).argb_stride = (C.int)(stride)
	((*C.WebPPicture)(unsafe.Pointer(pic))).use_argb = (C.int)(1)

	((*C.WebPPicture)(unsafe.Pointer(pic))).width = (C.int)(w)
	((*C.WebPPicture)(unsafe.Pointer(pic))).height = (C.int)(h)
	// if C.WebPPictureAlloc((*C.struct_WebPPicture)(unsafe.Pointer(&pic))) == 0 {
	// 	return fmt.Errorf("allocate memory failed")
	// }
	// defer C.WebPPictureFree((*C.struct_WebPPicture)(unsafe.Pointer(&pic)))

	var wrt C.WebPMemoryWriter
	C.WebPMemoryWriterInit((*C.struct_WebPMemoryWriter)(unsafe.Pointer(&wrt)))
	defer C.WebPMemoryWriterClear((*C.struct_WebPMemoryWriter)(unsafe.Pointer(&wrt)))

	((*C.WebPPicture)(unsafe.Pointer(pic))).custom_ptr = unsafe.Pointer(&wrt)
	((*C.WebPPicture)(unsafe.Pointer(pic))).writer = (C.WebPWriterFunction)(C.WebPMemoryWrite)

	// res := int(C.WebPPictureImportRGBA(
	// 	(*C.WebPPicture)(unsafe.Pointer(&pic)),
	// 	(*C.uint8_t)(unsafe.Pointer(&pix[0])),
	// 	(C.int)(stride),
	// ))
	// if res == 0 {
	// 	return fmt.Errorf("error: WebPPictureImportBGRA")
	// }

	if C.WebPEncode((*C.struct_WebPConfig)(unsafe.Pointer(config)),
		(*C.struct_WebPPicture)(unsafe.Pointer(pic))) == 0 {
		return fmt.Errorf("encode failed")
	}

	var output []byte
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&output)))
	sliceHeader.Cap = int(((*C.WebPMemoryWriter)(unsafe.Pointer(&wrt))).max_size)
	sliceHeader.Len = int(((*C.WebPMemoryWriter)(unsafe.Pointer(&wrt))).size)
	sliceHeader.Data = uintptr(unsafe.Pointer(((*C.WebPMemoryWriter)(unsafe.Pointer(&wrt))).mem))
	fmt.Println(sliceHeader.Cap)
	fmt.Println(sliceHeader.Len)

	_, e := wr.Write(output)

	return e
}

func encodeRGB(rgb []byte, width, height, stride int, quality float32) ([]byte, error) {
	var coutput *C.uint8_t
	outptr := (**C.uint8_t)(unsafe.Pointer(&coutput))

	length := C.WebPEncodeRGB((*C.uint8_t)(unsafe.Pointer(&rgb[0])), C.int(width), C.int(height),
		C.int(stride), C.float(quality), outptr)
	if length == 0 {
		return nil, fmt.Errorf("encodeRGB() failed")
	}

	var output []byte
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&output)))
	sliceHeader.Cap = int(length)
	sliceHeader.Len = int(length)
	sliceHeader.Data = uintptr(unsafe.Pointer(coutput))

	return output, nil
}

func encodeRGBA(rgb []byte, width, height, stride int, quality float32) ([]byte, error) {
	var coutput *C.uint8_t
	outptr := (**C.uint8_t)(unsafe.Pointer(&coutput))

	length := C.WebPEncodeRGBA((*C.uint8_t)(unsafe.Pointer(&rgb[0])), C.int(width), C.int(height),
		C.int(stride), C.float(quality), outptr)
	if length == 0 {
		return nil, fmt.Errorf("encodeRGBA() failed")
	}

	var output []byte
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&output)))
	sliceHeader.Cap = int(length)
	sliceHeader.Len = int(length)
	sliceHeader.Data = uintptr(unsafe.Pointer(coutput))

	return output, nil
}

func encodeLosslessRGB(rgb []byte, width, height, stride int) ([]byte, error) {
	var coutput *C.uint8_t
	outptr := (**C.uint8_t)(unsafe.Pointer(&coutput))

	length := C.WebPEncodeLosslessRGB((*C.uint8_t)(unsafe.Pointer(&rgb[0])), C.int(width),
		C.int(height), C.int(stride), outptr)
	if length == 0 {
		return nil, fmt.Errorf("encodeLosslessRGB() failed")
	}

	var output []byte
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&output)))
	sliceHeader.Cap = int(length)
	sliceHeader.Len = int(length)
	sliceHeader.Data = uintptr(unsafe.Pointer(coutput))

	return output, nil
}

func encodeLosslessRGBA(rgb []byte, width, height, stride int) ([]byte, error) {
	var coutput *C.uint8_t
	outptr := (**C.uint8_t)(unsafe.Pointer(&coutput))

	length := C.WebPEncodeLosslessRGBA((*C.uint8_t)(unsafe.Pointer(&rgb[0])), C.int(width),
		C.int(height), C.int(stride), outptr)

	if length == 0 {
		return nil, fmt.Errorf("encodeLosslessRGBA() failed")
	}

	var output []byte
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&output)))
	sliceHeader.Cap = int(length)
	sliceHeader.Len = int(length)
	sliceHeader.Data = uintptr(unsafe.Pointer(coutput))

	return output, nil
}

func Free(img []byte) {
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&img)))
	C.free(unsafe.Pointer(sliceHeader.Data))
}

func Encode2(img image.Image, quality int) ([]byte, error) {
	var byts []byte

	var e error
	w, h := img.Bounds().Size().X, img.Bounds().Size().Y
	switch t := img.(type) {
	case *image.NRGBA:
		if quality >= 100 {
			byts, e = encodeLosslessRGBA(t.Pix, w, h, t.Stride)
		} else {
			byts, e = encodeRGBA(t.Pix, w, h, t.Stride, float32(quality))
		}
	case *image.RGBA:
		if quality >= 100 {
			byts, e = encodeLosslessRGBA(t.Pix, w, h, t.Stride)
		} else {
			byts, e = encodeRGBA(t.Pix, w, h, t.Stride, float32(quality))
		}
	case *image.Gray:
		pix := make([]byte, w*h*3)
		length := len(t.Pix)
		for i := 0; i < length; i++ {
			pix[i*3], pix[i*3+1], pix[i*3+2] = t.Pix[i], t.Pix[i], t.Pix[i]
		}
		if quality >= 100 {
			byts, e = encodeLosslessRGB(pix, w, h, w*3)
		} else {
			byts, e = encodeRGB(pix, w, h, w*3, float32(quality))
		}
	case *image.YCbCr:
		pix := make([]byte, w*h*3)
		idx := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				c := t.YCbCrAt(x, y)
				pix[idx], pix[idx+1], pix[idx+2] = color.YCbCrToRGB(c.Y, c.Cb, c.Cr)
				idx += 3
			}
		}
		if quality >= 100 {
			byts, e = encodeLosslessRGB(pix, w, h, w*3)
		} else {
			byts, e = encodeRGB(pix, w, h, w*3, float32(quality))
		}
	default:
		return nil, fmt.Errorf("unsupported type:%s", reflect.TypeOf(img))
	}

	if e != nil {
		return nil, e
	}

	return byts, nil
}

func ToWEBP(src, target string, quality int, scale float32) error {
	input, e := ioutil.ReadFile(src)
	if e != nil {
		return e
	}

	img, _, e := image.Decode(bytes.NewReader(input))
	if e != nil {
		return e
	}

	w := img.Bounds().Size().X
	h := img.Bounds().Size().Y

	// if scale > 0 && scale != 1.0 {
	w = int(float32(w) * scale)
	h = int(float32(h) * scale)
	// img = resize.Resize(uint(w), uint(h), img, resize.NearestNeighbor)
	img = imaging.Resize(img, w, h, imaging.Lanczos)
	img = imaging.Sharpen(img, 0.8)
	// img = imaging.AdjustSaturation(img, 30)
	// }

	output, e := Encode2(img, quality)
	if e != nil {
		return e
	}

	e = ioutil.WriteFile(target, output, 0666)
	if e != nil {
		return e
	}

	Free(output)

	return nil
}
