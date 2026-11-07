//go:build android && cgo

package text

import (
	"bytes"
	"github.com/Heyana/typesetting-go/font"
	xdraw "golang.org/x/image/draw"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
)

// SetAndroidGlyphRasterizer is installed by the JNI host before app startup.
// It draws offscreen on the calling thread, never waits on Android's UI loop.
func SetAndroidGlyphRasterizer(fn func(path string, index int, glyph uint32, size, dx float32) []byte) {
	androidColorRasterizer = fn
}

var androidColorRasterizer func(string, int, uint32, float32, float32) []byte

func (e *androidEngine) colorGlyph(af *androidFont, f *Font, id uint32, scale, dx float32) (bitmap, bool) {
	gid := font.GID(id)
	_, color := af.face.GlyphDataColor(gid)
	source, hasBitmap := af.face.GlyphDataBitmap(gid)
	if !color && !hasBitmap {
		return bitmap{}, false
	}
	if androidColorRasterizer != nil && af.path != "" {
		b := unpackNativeColorGlyph(androidColorRasterizer(af.path, af.collection, id, f.Size*scale, dx))
		if b.w > 0 {
			return b, true
		}
	}
	// Android 10/11 use CBDT bitmap emoji. They predate Canvas.drawGlyphs;
	// preserve the native strike's color while scaling its own glyph metrics.
	if hasBitmap && (source.Format == font.PNG || source.Format == font.JPG) {
		img, _, err := image.Decode(bytes.NewReader(source.Data))
		if err != nil {
			return bitmap{}, false
		}
		ext, ok := af.face.GlyphExtents(gid)
		if !ok {
			return bitmap{}, false
		}
		s := f.Size * scale / af.upem
		left, top := int(math.Floor(float64(ext.XBearing*s+dx))), int(math.Floor(float64(-ext.YBearing*s)))
		right, bottom := int(math.Ceil(float64((ext.XBearing+ext.Width)*s+dx))), int(math.Ceil(float64(-(ext.YBearing+ext.Height)*s)))
		w, h := right-left, bottom-top
		if w <= 0 || h <= 0 || w > 2048 || h > 2048 {
			return bitmap{}, false
		}
		rgba := image.NewRGBA(image.Rect(0, 0, w, h))
		xdraw.ApproxBiLinear.Scale(rgba, rgba.Bounds(), img, img.Bounds(), xdraw.Src, nil)
		return bitmap{left: left, top: top, w: w, h: h, pix: rgba.Pix, color: true}, true
	}
	return bitmap{}, false
}
