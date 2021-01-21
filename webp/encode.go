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
)

type SetENUM int

const (
	SET_DEFAULT SetENUM = C.WEBP_PRESET_DEFAULT
	// for digital picture, like portrait, inner shot
	SET_PICTURE SetENUM = C.WEBP_PRESET_PICTURE
	// for outdoor photograph, with natural lighting
	SET_PHOTO SetENUM = C.WEBP_PRESET_PHOTO
	// for hand or line drawing, with high-contrast details
	SET_DRAWING SetENUM = C.WEBP_PRESET_DRAWING
	// for small-sized colorful images
	SET_ICON SetENUM = C.WEBP_PRESET_ICON
	// for text-like
	SET_TEXT SetENUM = C.WEBP_PRESET_TEXT
)

type EncErrorENUM int

const (
	ERR_ENC_OK EncErrorENUM = iota
	ERR_ENC_OUT_OF_MEMORY
	ERR_ENC_BITSTREAM_OUT_OF_MEMORY
	ERR_ENC_NULL_PARAMETER
	ERR_ENC_INVALID_CONFIGURATION
	ERR_ENC_BAD_DIMENSION
	ERR_ENC_PARTITION0_OVERFLOW
	ERR_ENC_PARTITION_OVERFLOW
	ERR_ENC_BAD_WRITE
	ERR_ENC_FILE_TOO_BIG
	ERR_ENC_USER_ABORT
	ERR_ENC_ERROR_LAST
)

func (e EncErrorENUM) String() string {
	switch e {
	case ERR_ENC_OUT_OF_MEMORY:
		return "memory error allocating objects"
	case ERR_ENC_BITSTREAM_OUT_OF_MEMORY:
		return "memory error while flushing bits"
	case ERR_ENC_NULL_PARAMETER:
		return "a pointer parameter is NULL"
	case ERR_ENC_INVALID_CONFIGURATION:
		return "configuration is invalid"
	case ERR_ENC_BAD_DIMENSION:
		return "picture has invalid width/height"
	case ERR_ENC_PARTITION0_OVERFLOW:
		return "partition is bigger than 512k"
	case ERR_ENC_PARTITION_OVERFLOW:
		return "partition is bigger than 16M"
	case ERR_ENC_BAD_WRITE:
		return "error while flushing bytes"
	case ERR_ENC_FILE_TOO_BIG:
		return "file is bigger than 4G"
	case ERR_ENC_USER_ABORT:
		return "abort request by user"
	case ERR_ENC_ERROR_LAST:
		return "list terminator. always last."
	default:
		return "undefined error"
	}
}

// Option specifies WebP encoding configuration.
type Option struct {
	config        C.WebPConfig
	webp_preset   SetENUM
	webp_quality  int
	webp_lossless bool
	resizeScale   float32
	resizeWidth   int
	resizeHeight  int
}

func NewConfig(set SetENUM, quality int) *Option {
	opt := &Option{}
	opt.webp_preset = set
	opt.webp_quality = quality

	return opt
}

func (o *Option) intWebpConfig() error {
	if C.WebPConfigPreset(&o.config,
		C.WebPPreset(o.webp_preset), C.float(o.webp_quality)) == 0 {
		return errors.New("init WebPConfig failed")
	}

	// important: this keep color as same as source, but increases size.
	o.config.use_sharp_yuv = C.int(1)
	o.config.lossless = bool2int(o.webp_lossless)

	o.config.segments = C.int(4)
	o.config.filter_strength = C.int(100)
	o.config.filter_sharpness = C.int(7)
	o.config.filter_type = C.int(1) // strong

	if C.WebPValidateConfig(&o.config) == 0 {
		return errors.New("invalid WebPConfig")
	}

	return nil
}

func (o *Option) SetLossless(v bool) {
	o.webp_lossless = v
}

func (o *Option) SetResizeScale(v float32) {
	o.resizeScale = v
}

func (o *Option) SetResize(width, height int) {
	o.resizeWidth, o.resizeHeight = width, height
	o.resizeScale = 0
}

func Encode(w io.Writer, img image.Image, opt *Option) error {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	switch p := img.(type) {
	case *image.RGBA:
		return EncodeBytes(w, p.Pix, width, height, 4, opt)
	case *image.NRGBA:
		return EncodeBytes(w, p.Pix, width, height, 4, opt)
	case *image.Gray:
		pix := make([]byte, width*height*3)
		idx := 0
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c := p.GrayAt(x, y)
				pix[idx*3], pix[idx*3+1], pix[idx*3+2] = c.Y, c.Y, c.Y
				idx++
			}
		}
		return EncodeBytes(w, pix, width, height, 3, opt)
	case *image.YCbCr:
		pix := make([]byte, width*height*3)
		idx := 0
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c := p.YCbCrAt(x, y)
				pix[idx], pix[idx+1], pix[idx+2] = color.YCbCrToRGB(c.Y, c.Cb, c.Cr)
				idx += 3
			}
		}
		return EncodeBytes(w, pix, width, height, 3, opt)
	case *image.NYCbCrA:
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
		return EncodeBytes(w, pix, width, height, 4, opt)
	default:
		fmt.Printf("unsupported image type: %T, convert into NRGBA\n", p)
		im := image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				im.Set(x, y, img.At(x, y))
			}
		}
		return EncodeBytes(w, im.Pix, width, height, 4, opt)
	}
}

func EncodeBytes(w io.Writer, pix []byte, width, height, comps int, opt *Option) error {
	if e := opt.intWebpConfig(); e != nil {
		return e
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
		C.WebPPictureImportRGBA(pic, (*C.uint8_t)(&pix[0]), C.int(width*4))
	case 3:
		C.WebPPictureImportRGB(pic, (*C.uint8_t)(&pix[0]), C.int(width*3))
	case 2:
		pixCount := width * height
		p := make([]byte, pixCount*4)
		for i := 0; i < int(pixCount); i++ {
			p[i*4], p[i*4+1], p[i*4+2] = pix[i*2], pix[i*2], pix[i*2]
			p[i*4+3] = pix[i*2+1]
		}
		C.WebPPictureImportRGBA(pic, (*C.uint8_t)(unsafe.Pointer(&p[0])), C.int(width*4))
	case 1:
		pixCount := width * height
		p := make([]byte, pixCount*3)
		for i := 0; i < int(pixCount); i++ {
			p[i*3], p[i*3+1], p[i*3+2] = pix[i], pix[i], pix[i]
		}
		C.WebPPictureImportRGB(pic, (*C.uint8_t)(unsafe.Pointer(&p[0])), C.int(width*3))
	default:
		return errors.New("not support image type")
	}

	if opt.resizeScale != 0.0 && opt.resizeScale != 1.0 {
		opt.resizeWidth = int(float32(width) * opt.resizeScale)
		opt.resizeHeight = int((float32(height) * opt.resizeScale))
	}
	if opt.resizeWidth != 0 || opt.resizeHeight != 0 {
		if C.WebPPictureRescale(pic, C.int(opt.resizeWidth), C.int(opt.resizeHeight)) == 0 {
			return errors.New("resize failed")
		}
	}

	if C.WebPEncode(&opt.config, pic) == 0 {
		return fmt.Errorf("encoding webp failed, error: %s", EncErrorENUM(pic.error_code))
	}

	byts := C.GoBytes(unsafe.Pointer(mr.mem), C.int(mr.size))
	_, e := w.Write(byts)
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
