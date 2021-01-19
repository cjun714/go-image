package main

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"

	"github.com/cjun714/go-image/webp"
)

func main() {
	f, e := os.Open("z:/test.jpg")
	if e != nil {
		panic(e)
	}
	defer f.Close()
	img, _, e := image.Decode(f)
	if e != nil {
		panic(e)
	}

	config, e := webp.ConfigPreset(webp.PRESET_DRAWING, 85)
	if e != nil {
		panic(e)
	}
	// config.SetLossless(true)
	// config.SetSNSStrength(100)
	// config.SetFilterStrength(100)
	// config.SetFilterSharpness(7)
	// config.SetResizeScale(0.5)
	config.SetResizeHeight(1080)

	w, e := os.Create("z:/test.webp")
	if e != nil {
		panic(e)
	}
	defer w.Close()
	e = webp.Encode(w, img, config)
	if e != nil {
		panic(e)
	}

	bs, e := ioutil.ReadFile("z:/test.jpg")
	if e != nil {
		panic(e)
	}

	f, e = os.Create("z:/ttt.webp")
	if e != nil {
		panic(e)
	}
	defer f.Close()

	opt, e := webp.ConfigPreset(webp.PRESET_DEFAULT, 85)
	if e != nil {
		panic(e)
	}
	// config.SetLossless(true)
	// config.SetSNSStrength(100)
	// config.SetFilterStrength(100)
	// config.SetFilterSharpness(7)
	e = webp.EncodeBytes(f, bs, opt)
	if e != nil {
		panic(e)
	}

}
