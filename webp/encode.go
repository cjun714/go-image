package webp

/*
#cgo LDFLAGS: -lwebp
#include <stdlib.h>
#include <webp/encode.h>

static WebPPicture *calloc_WebPPicture(void) {
	return calloc(sizeof(WebPPicture), 1);
}

static void free_WebPPicture(WebPPicture* webpPicture) {
	free(webpPicture);
}
*/
import "C"
import (
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"unsafe"

	"github.com/cjun714/go-image-stb/stb"
)

// Config specifies WebP encoding configuration.
type Option struct {
	config       C.WebPConfig
	resizeScale  float32
	resizeWidth  int
	resizeHeight int
}

type PresetENUM int

const (
	PRESET_DEFAULT PresetENUM = C.WEBP_PRESET_DEFAULT
	// for digital picture, like portrait, inner shot
	PRESET_PICTURE PresetENUM = C.WEBP_PRESET_PICTURE
	// for outdoor photograph, with natural lighting
	PRESET_PHOTO PresetENUM = C.WEBP_PRESET_PHOTO
	// for hand or line drawing, with high-contrast details
	PRESET_DRAWING PresetENUM = C.WEBP_PRESET_DRAWING
	// for small-sized colorful images
	PRESET_ICON PresetENUM = C.WEBP_PRESET_ICON
	// for text-like
	PRESET_TEXT PresetENUM = C.WEBP_PRESET_TEXT
)

// SetMultiThreadLevel set multi-thread level. 0(off), 1(on)
func (o *Option) SetMultiThreadLevel(v int) {
	o.config.thread_level = C.int(v)
}

// SetSNSStrength set Spatial Noise Shaping.  0(off) - 100(max).
func (o *Option) SetSNSStrength(v int) {
	o.config.sns_strength = C.int(v)
}

// SetLossless set Loossless
func (o *Option) SetLossless(v bool) {
	o.config.lossless = bool2int(v)
}

// SetFilterStrength set filter strength. 0(off) - 100(strongest)
func (o *Option) SetFilterStrength(v int) {
	o.config.filter_strength = C.int(v)
}

// SetFilterSharpness set filter sharpness. 0(off) - 7(max)
func (o *Option) SetFilterSharpness(v int) {
	o.config.filter_sharpness = C.int(v)
}

func (o *Option) SetResizeScale(v float32) {
	o.resizeScale = v
}

func (o *Option) SetResizeHeight(v int) {
	o.resizeHeight = v
}

func (o *Option) SetResizeWidth(v int) {
	o.resizeWidth = v
}

func ConfigPreset(preset PresetENUM, quality int) (*Option, error) {
	opt := &Option{}
	if C.WebPConfigPreset(&opt.config, C.WebPPreset(preset), C.float(quality)) == 0 {
		return nil, errors.New("init WebPConfig failed")
	}

	// important: enable this to keep color as original as possiable, also
	// increases encoding size.
	opt.config.use_sharp_yuv = C.int(1)

	return opt, nil
}

func Encode(w io.Writer, img image.Image, opt *Option) error {
	if C.WebPValidateConfig(&opt.config) == 0 {
		return errors.New("invalid WebPConfig")
	}

	// alloc pic in c memory to avoid 'cgo argument has Go pointer to Go pointer'
	pic := C.calloc_WebPPicture()
	if pic == nil {
		return errors.New("alloc WebPPicture failed")
	}
	defer C.free_WebPPicture(pic)

	if C.WebPPictureInit(pic) == 0 {
		return errors.New("init WebPPicture failed")
	}
	defer C.WebPPictureFree(pic)

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	pic.use_argb = 1
	pic.width = C.int(width)
	pic.height = C.int(height)

	mr := &C.WebPMemoryWriter{}
	C.WebPMemoryWriterInit(mr)
	defer C.WebPMemoryWriterClear(mr)

	pic.custom_ptr = unsafe.Pointer(mr)
	pic.writer = C.WebPWriterFunction(C.WebPMemoryWrite)

	switch p := img.(type) {
	case *image.RGBA:
		C.WebPPictureImportRGBA(pic, (*C.uint8_t)(&p.Pix[0]), C.int(p.Stride))
	case *image.NRGBA:
		C.WebPPictureImportRGBA(pic, (*C.uint8_t)(&p.Pix[0]), C.int(p.Stride))
	case *image.Gray:
		pix := make([]byte, width*height*3)
		idx := 0
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c := p.GrayAt(x, y)
				pix[idx*3] = c.Y
				pix[idx*3+1] = c.Y
				pix[idx*3+2] = c.Y
				idx++
			}
		}
		C.WebPPictureImportRGB(pic, (*C.uint8_t)(unsafe.Pointer(&pix[0])), C.int(width*3))
	case *image.YCbCr:
		if p.SubsampleRatio == image.YCbCrSubsampleRatio420 {
			pic.use_argb = 0
			pic.colorspace = C.WEBP_YUV420
			pic.y, pic.u, pic.v = (*C.uint8_t)(&p.Y[0]), (*C.uint8_t)(&p.Cb[0]), (*C.uint8_t)(&p.Cr[0])
			pic.y_stride, pic.uv_stride = C.int(p.YStride), C.int(p.CStride)
		} else {
			pix := make([]byte, width*height*3)
			idx := 0
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					c := p.YCbCrAt(x, y)
					pix[idx], pix[idx+1], pix[idx+2] = color.YCbCrToRGB(c.Y, c.Cb, c.Cr)
					idx += 3
				}
			}
			C.WebPPictureImportRGB(pic, (*C.uint8_t)(&pix[0]), C.int(width*3))
		}
	case *image.NYCbCrA:
		if p.SubsampleRatio == image.YCbCrSubsampleRatio420 {
			pic.use_argb = 0
			pic.colorspace = C.WEBP_YUV420A
			pic.y, pic.u, pic.v = (*C.uint8_t)(&p.Y[0]), (*C.uint8_t)(&p.Cb[0]), (*C.uint8_t)(&p.Cr[0])
			pic.a = (*C.uint8_t)(&p.A[0])
			pic.y_stride, pic.uv_stride = C.int(p.YStride), C.int(p.CStride)
			pic.a_stride = C.int(p.AStride)
		} else {
			pix := make([]byte, width*height*4)
			idx := 0
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					c := p.NYCbCrAAt(x, y)
					pix[idx], pix[idx+1], pix[idx+2] = color.YCbCrToRGB(c.Y, c.Cb, c.Cr)
					pix[idx] = c.A
					idx += 4
				}
			}
			C.WebPPictureImportRGBA(pic, (*C.uint8_t)(&pix[0]), C.int(width*4))
		}
	default:
		fmt.Printf("unsupported image type: %T\n", p)
		im := image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				im.Set(x, y, img.At(x, y))
			}
		}
		C.WebPPictureImportRGBA(pic, (*C.uint8_t)(&im.Pix[0]), C.int(width*4))
	}

	if opt.resizeScale != 0.0 && opt.resizeScale != 1.0 &&
		opt.resizeWidth == 0 && opt.resizeHeight == 0 {
		opt.resizeWidth = int(float32(width) * opt.resizeScale)
		opt.resizeHeight = int(float32(height) * opt.resizeScale)
	}
	if opt.resizeWidth != 0 || opt.resizeHeight != 0 {
		if C.WebPPictureRescale(pic, C.int(opt.resizeWidth), C.int(opt.resizeHeight)) == 0 {
			return errors.New("resize failed")
		}
	}

	if C.WebPEncode(&opt.config, pic) == 0 {
		return fmt.Errorf("encoding web failed, error code: %d", pic.error_code)
	}

	byts := C.GoBytes(unsafe.Pointer(mr.mem), C.int(mr.size))

	_, e := w.Write(byts)
	if e != nil {
		return e
	}

	return nil
}

func EncodeBytes(w io.Writer, data []byte, opt *Option) error {
	pix, width, height, comps, e := stb.LoadBytes(data)
	if e != nil {
		return e
	}
	defer stb.Free(pix)

	if C.WebPValidateConfig(&opt.config) == 0 {
		return errors.New("invalid WebPConfig")
	}

	// alloc pic in c memory to avoid 'cgo argument has Go pointer to Go pointer'
	pic := C.calloc_WebPPicture()
	if pic == nil {
		return errors.New("alloc WebPPicture failed")
	}
	defer C.free_WebPPicture(pic)

	if C.WebPPictureInit(pic) == 0 {
		return errors.New("init WebPPicture failed")
	}
	defer C.WebPPictureFree(pic)

	pic.use_argb = 1
	pic.width = C.int(width)
	pic.height = C.int(height)

	mr := &C.WebPMemoryWriter{}
	C.WebPMemoryWriterInit(mr)
	defer C.WebPMemoryWriterClear(mr)

	pic.custom_ptr = unsafe.Pointer(mr)
	pic.writer = C.WebPWriterFunction(C.WebPMemoryWrite)

	switch comps {
	case 4:
		C.WebPPictureImportRGBA(pic, (*C.uint8_t)(pix), C.int(width*4))
	case 3:
		C.WebPPictureImportRGB(pic, (*C.uint8_t)(pix), C.int(width*3))
	case 1:
		pixCount := width * height
		p := make([]byte, pixCount*3)
		byts := C.GoBytes(unsafe.Pointer(pix), C.int(pixCount))

		for i := 0; i < int(pixCount); i++ {
			p[i*3], p[i*3+1], p[i*3+2] = byts[i], byts[i], byts[i]
		}
		C.WebPPictureImportRGB(pic, (*C.uint8_t)(unsafe.Pointer(&p[0])), C.int(width*3))
	default:
		return errors.New("not support image type")
	}

	if opt.resizeScale != 0.0 && opt.resizeScale != 1.0 &&
		opt.resizeWidth != 0 && opt.resizeHeight != 0 {
		opt.resizeWidth = int(float32(width) * opt.resizeScale)
		opt.resizeHeight = int((float32(height) * opt.resizeScale))
	}
	if opt.resizeWidth != 0 || opt.resizeHeight != 0 {
		if C.WebPPictureRescale(pic, C.int(opt.resizeWidth), C.int(opt.resizeHeight)) == 0 {
			return errors.New("resize failed")
		}
	}

	if C.WebPEncode(&opt.config, pic) == 0 {
		return fmt.Errorf("encoding webp failed, error code: %d", pic.error_code)
	}

	byts := C.GoBytes(unsafe.Pointer(mr.mem), C.int(mr.size))

	_, e = w.Write(byts)
	if e != nil {
		return e
	}

	return nil
}

func bool2int(v bool) C.int {
	if v {
		return 1
	}
	return 0
}
