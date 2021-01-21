package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"time"

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

	w, e := os.Create("z:/test.webp")
	if e != nil {
		panic(e)
	}
	defer w.Close()

	cfg := webp.NewConfig(webp.SET_PHOTO, 95)
	cfg.SetResizeScale(0.5)

	start := time.Now()
	e = webp.Encode(w, img, cfg)
	if e != nil {
		panic(e)
	}
	fmt.Printf("done, cost: %s\n", time.Since(start))
}
