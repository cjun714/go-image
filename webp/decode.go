package webp

// #cgo LDFLAGS: -lwebp
// #include <stdlib.h>
// #include <webp/decode.h>
import "C"

import (
	"fmt"
	"image"
	"unsafe"
)

func getWebpInfo(data []byte) (int, int, error) {
	var width, height int32
	ret := C.WebPGetInfo(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.ulonglong(len(data)),
		(*C.int)(unsafe.Pointer(&width)),
		(*C.int)(unsafe.Pointer(&height)))
	if ret == 0 {
		return 0, 0, fmt.Errorf("read .webp info faild")
	}
	return int(width), int(height), nil
}

func decodeRGBAInto(data, output []byte, stride int) ([]byte, error) {
	out := (*uint8)(C.WebPDecodeRGBAInto(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.ulonglong(len(data)),
		(*C.uint8_t)(unsafe.Pointer(&output[0])), C.ulonglong(len(output)), C.int(stride)))
	if out == nil {
		return nil, fmt.Errorf("decode .webp faild")
	}

	return output, nil
}

func Decode(data []byte) (image.Image, error) {
	w, h, e := getWebpInfo(data)
	fmt.Println(w, h)
	if e != nil {
		return nil, e
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	_, e = decodeRGBAInto(data, img.Pix, w*4)
	if e != nil {
		return nil, e
	}

	return img, nil
}
