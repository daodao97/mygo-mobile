package text

import "encoding/binary"

// Native color glyphs are a bounded little-endian geometry header followed by
// premultiplied RGBA pixels. Validate before admitting pixels to the atlas.
func unpackNativeColorGlyph(data []byte) bitmap {
	if len(data) < 16 {
		return bitmap{}
	}
	w, h := int(binary.LittleEndian.Uint32(data)), int(binary.LittleEndian.Uint32(data[4:]))
	if w <= 0 || h <= 0 || w > 2048 || h > 2048 || len(data) != 16+4*w*h {
		return bitmap{}
	}
	left, top := int(int32(binary.LittleEndian.Uint32(data[8:]))), int(int32(binary.LittleEndian.Uint32(data[12:])))
	return bitmap{left: left, top: top, w: w, h: h, pix: data[16:], color: true}
}
