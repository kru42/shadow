// utils.go
package utils

import (
	"crypto/sha256"
	"image"
	"image/color"
	"image/png"
	"os"
)

func generateIdenticon(pub []byte, filename string) error {
	const size = 5
	const scale = 50

	hash := sha256.Sum256(pub)
	img := image.NewRGBA(image.Rect(0, 0, size*scale, size*scale))
	bg := color.RGBA{240, 240, 240, 255}
	fg := color.RGBA{hash[0], hash[1], hash[2], 255}

	for x := range size/2 + 1 {
		for y := range size {
			i := x*size + y
			on := hash[i%len(hash)]%2 == 0
			c := bg
			if on {
				c = fg
			}
			for dx := range scale {
				for dy := 0; dy < scale; dy++ {
					img.Set(x*scale+dx, y*scale+dy, c)
					img.Set((size-1-x)*scale+dx, y*scale+dy, c)
				}
			}
		}
	}
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}

func GenerateIdenticon(pub []byte, filename string) error {
	return generateIdenticon(pub, filename)
}
