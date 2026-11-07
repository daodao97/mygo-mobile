package text

import (
	"encoding/binary"
	"testing"
)

func TestNativeColorGlyphBoundsAndPremultipliedPixels(t *testing.T) {
	data := make([]byte, 20)
	binary.LittleEndian.PutUint32(data, 1)
	binary.LittleEndian.PutUint32(data[4:], 1)
	binary.LittleEndian.PutUint32(data[8:], uint32(3))
	binary.LittleEndian.PutUint32(data[12:], uint32(0xfffffff8))
	copy(data[16:], []byte{64, 32, 16, 128})
	b := unpackNativeColorGlyph(data)
	if !b.color || b.left != 3 || b.top != -8 || b.w != 1 || b.h != 1 || b.pix[0] != 64 || b.pix[3] != 128 {
		t.Fatalf("invalid bitmap: %+v", b)
	}
	for _, bad := range [][]byte{nil, data[:19], append(append([]byte{}, data...), 0)} {
		if unpackNativeColorGlyph(bad).w != 0 {
			t.Fatal("accepted truncated or oversized raster")
		}
	}
	binary.LittleEndian.PutUint32(data, 0xffffffff)
	if unpackNativeColorGlyph(data).w != 0 {
		t.Fatal("accepted unbounded dimensions")
	}
}
