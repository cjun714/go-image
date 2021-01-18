package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"

	"github.com/cjun714/go-image/webp"
)

func main() {
	bs, e := ioutil.ReadFile("z:/test.jpg")
	if e != nil {
		panic(e)
	}

	img, _, e := image.Decode(bytes.NewReader(bs))
	if e != nil {
		panic(e)
	}

	f, e := os.Create("z:/test.webp")
	if e != nil {
		panic(e)
	}
	defer f.Close()
	e = webp.Encode(f, img)
	if e != nil {
		panic(e)
	}

}
