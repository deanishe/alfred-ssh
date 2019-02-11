// Copyright (c) 2019 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

// +build mage

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/disintegration/imaging"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
)

var (
	blue   = "5485F3"
	green  = "03AE03"
	purple = "AF49FE"
	red    = "B00000"
	yellow = "F8AC30"

	icons = []struct {
		Filename string
		Font     string
		Colour   string
		Name     string
	}{
		{"update-available", "material", yellow, "cloud-download"},
		{"update-ok", "material", green, "cloud-done"},

		{"help", "material", green, "help"},
		{"issue", "fontawesome", yellow, "bug"},
		{"log", "fontawesome", blue, "history"},
		{"url", "fontawesome", blue, "globe"},

		{"off", "fontawesome", red, "circle-o"},
		{"on", "fontawesome", green, "check-circle-o"},
		{"reload", "fontawesome", yellow, "refresh"},
	}
)

func rotateIcon(path string, angles []int) error {

	var (
		dir  = filepath.Dir(path)
		ext  = filepath.Ext(path)
		base = filepath.Base(path)
		name = base[0 : len(base)-len(ext)]
		src  image.Image
		f    *os.File
		err  error
	)

	if f, err = os.Open(path); err != nil {
		return err
	}
	defer f.Close()

	if src, _, err = image.Decode(f); err != nil {
		return err
	}

	for i, n := range angles {

		p := filepath.Join(dir, fmt.Sprintf("%s-%d%s", name, n, ext))

		if exists(p) {
			fmt.Printf("[%d/%d] skipped existing: %s\n", i+1, len(angles), p)
			continue
		}

		dst := imaging.Rotate(src, 360-float64(n), image.Transparent)
		dst = imaging.CropCenter(dst, src.Bounds().Dx(), src.Bounds().Dy())

		if f, err = os.Create(p); err != nil {
			return err
		}
		defer f.Close()

		if err = png.Encode(f, dst); err != nil {
			return err
		}

		fmt.Printf("wrote %s\n", p)
	}

	return nil
}

// Copy image src to dest. If colour is non-empty, src is re-coloured.
// Colour should be an HTML hex, e.g. "ff33ba".
func copyImage(src, dest, colour string) error {

	if colour == "" {
		return sh.Copy(src, dest)
	}

	var (
		c    *color.RGBA
		f    *os.File
		err  error
		mask image.Image
	)

	if c, err = parseHex(colour); err != nil {
		return err
	}

	if f, err = os.Open(src); err != nil {
		return errors.Wrap(err, "read image")
	}
	defer f.Close()

	if mask, _, err = image.Decode(f); err != nil {
		return errors.Wrap(err, "decode image")
	}

	img := image.NewRGBA(mask.Bounds())
	draw.DrawMask(img, img.Bounds(), &image.Uniform{c}, image.ZP, mask, image.ZP, draw.Src)

	if f, err = os.Create(dest); err != nil {
		return errors.Wrap(err, "open new image")
	}
	defer f.Close()

	if err = png.Encode(f, img); err != nil {
		return errors.Wrap(err, "write PNG data")
	}

	fmt.Printf("copied %s to %s using colour #%s\n", src, dest, colour)

	return nil
}

var rxHexColour = regexp.MustCompile(`[a-fA-F0-9]{6}`)

func parseHex(s string) (*color.RGBA, error) {

	if !rxHexColour.MatchString(s) {
		return nil, fmt.Errorf("invalid hex colour: %s", s)
	}

	var (
		r, g, b uint64
		err     error
	)

	if r, err = strconv.ParseUint(s[0:2], 16, 8); err != nil {
		return nil, fmt.Errorf("invalid value for red (%s): %v", s[0:2], err)
	}
	if g, err = strconv.ParseUint(s[2:4], 16, 8); err != nil {
		return nil, fmt.Errorf("invalid value for green (%s): %v", s[2:4], err)
	}
	if b, err = strconv.ParseUint(s[4:6], 16, 8); err != nil {
		return nil, fmt.Errorf("invalid value for blue (%s): %v", s[4:6], err)
	}

	c := &color.RGBA{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
		A: 0xff,
	}

	return c, nil
}
