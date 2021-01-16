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
	"io/ioutil"
	"reflect"
	"unsafe"

	"github.com/disintegration/imaging"
)

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

func Encode(img image.Image, quality int) ([]byte, error) {
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

	output, e := Encode(img, quality)
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
