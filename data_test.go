package radolan

import (
	"bufio"
	"bytes"
	"math"
	"strings"
	"testing"
)

// --- identifyAndParse / encoding dispatch ---

func TestIdentifyAndParseRunlength(t *testing.T) {
	// Composite with level table → triggers runlength path.
	// One row of 4 pixels: [linenum=0] [offset=16 → pos 0] [value=0x31 → 3×class1]
	// then pad with 0x11 to fill remaining pixel.
	//
	// Level table: [10.0] → class index 1 maps to 10.0.
	levels := []float32{10.0}

	// Build a minimal 4-pixel × 1-row runlength blob.
	// Line format: [linenum] [offset_byte] [value_bytes...]
	// offset=16 (=16-16=0 skip), value=0x41 (4 repetitions of class 1)
	line := []byte{0x00, 16, 0x41, '\n'}

	c := &Composite{
		Px: 4, Py: 1, Dx: 4, Dy: 1,
		level:         levels,
		precisionMult: 1.0,
		dataLength:    999, // doesn't match Px*Py or Px*Py*2
	}

	flat := make([]float32, 4)
	c.PlainData = [][]float32{flat}

	err := c.identifyAndParse(bufio.NewReader(bytes.NewReader(line)))
	if err != nil {
		t.Fatalf("runlength parse: %v", err)
	}
	for i := range 4 {
		if c.PlainData[0][i] != 10.0 {
			t.Errorf("PlainData[0][%d] = %v; want 10.0", i, c.PlainData[0][i])
		}
	}
}

func TestIdentifyAndParseLittleEndian(t *testing.T) {
	// dataLength == Px*Py*2 → little-endian.
	payload := []byte{65, 0x00, 100, 0x00} // 1 row, 2 pixels
	c := &Composite{Px: 2, Py: 1, Dx: 2, Dy: 1, DataUnit: Unit_dBZ, dataLength: 4, precisionMult: 1.0}
	flat := make([]float32, 2)
	c.PlainData = [][]float32{flat}

	err := c.identifyAndParse(bufio.NewReader(bytes.NewReader(payload)))
	if err != nil {
		t.Fatalf("littleEndian parse: %v", err)
	}
	// rvp6=65 → dBZ = 65/2 - 32.5 = 0.0
	if c.PlainData[0][0] != 0.0 {
		t.Errorf("PlainData[0][0] = %v; want 0.0", c.PlainData[0][0])
	}
}

func TestIdentifyAndParseSingleByte(t *testing.T) {
	// dataLength == Px*Py → single-byte.
	payload := []byte{65, 100}
	c := &Composite{Px: 2, Py: 1, Dx: 2, Dy: 1, DataUnit: Unit_dBZ, dataLength: 2, precisionMult: 1.0}
	flat := make([]float32, 2)
	c.PlainData = [][]float32{flat}

	err := c.identifyAndParse(bufio.NewReader(bytes.NewReader(payload)))
	if err != nil {
		t.Fatalf("singleByte parse: %v", err)
	}
	if c.PlainData[0][0] != 0.0 {
		t.Errorf("PlainData[0][0] = %v; want 0.0", c.PlainData[0][0])
	}
}

func TestIdentifyAndParseUnknown(t *testing.T) {
	// dataLength doesn't match any encoding → error.
	c := &Composite{Px: 2, Py: 1, Dx: 2, Dy: 1, dataLength: 7, precisionMult: 1.0}
	flat := make([]float32, 2)
	c.PlainData = [][]float32{flat}

	err := c.identifyAndParse(bufio.NewReader(strings.NewReader("")))
	if err == nil {
		t.Fatal("expected error for unknown encoding")
	}
	if !strings.Contains(err.Error(), "unknown encoding") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- parseData ---

func TestParseDataNoDimensions(t *testing.T) {
	c := &Composite{Px: 0, Py: 0}
	err := c.parseData(bufio.NewReader(strings.NewReader("")))
	if err == nil {
		t.Fatal("expected error for zero dimensions")
	}
}

func TestParseDataDefaultPrecisionMult(t *testing.T) {
	// precisionMult==0 → parseData sets it to 1.0.
	payload := []byte{65, 100}
	c := &Composite{Px: 2, Py: 1, Dx: 2, Dy: 1, DataUnit: Unit_dBZ, dataLength: 2}
	// precisionMult intentionally left at zero
	err := c.parseData(bufio.NewReader(bytes.NewReader(payload)))
	if err != nil {
		t.Fatalf("parseData: %v", err)
	}
	if c.precisionMult != 1.0 {
		t.Errorf("precisionMult = %v; want 1.0", c.precisionMult)
	}
}

// --- arrangeData ---

func TestArrangeDataSingleLayer(t *testing.T) {
	// Py==Dy → single layer, Py%Dy==0.
	c := &Composite{Dx: 3, Dy: 2, Px: 3, Py: 2}
	c.PlainData = [][]float32{{1, 2, 3}, {4, 5, 6}}
	c.arrangeData()

	if c.Dz != 1 {
		t.Errorf("Dz = %d; want 1", c.Dz)
	}
	if c.Data[0][0] != 1 || c.Data[1][2] != 6 {
		t.Error("Data alias incorrect")
	}
}

func TestArrangeDataMultipleLayers(t *testing.T) {
	// PZ-like: Py=2400, Dy=200 → 12 layers.
	c := &Composite{Dx: 2, Dy: 3, Px: 2, Py: 6}
	c.PlainData = make([][]float32, 6)
	for i := range c.PlainData {
		c.PlainData[i] = []float32{float32(i), float32(i + 10)}
	}
	c.arrangeData()

	if c.Dz != 2 {
		t.Errorf("Dz = %d; want 2", c.Dz)
	}
	// Layer 0 = PlainData[0:3], layer 1 = PlainData[3:6].
	if c.DataZ[0][0][0] != 0 || c.DataZ[1][0][0] != 3 {
		t.Error("multi-layer DataZ incorrect")
	}
	if c.Data[0][0] != 0 {
		t.Error("Data alias should point to layer 0")
	}
}

func TestArrangeDataStripElevation(t *testing.T) {
	// PX-like: Py=224, Dy=200 → 224%200 != 0 → strip top 24 rows.
	c := &Composite{Dx: 2, Dy: 200, Px: 2, Py: 224}
	c.PlainData = make([][]float32, 224)
	for i := range c.PlainData {
		c.PlainData[i] = []float32{float32(i), float32(i + 100)}
	}
	c.arrangeData()

	if c.Dz != 1 {
		t.Errorf("Dz = %d; want 1", c.Dz)
	}
	// Data should be the bottom 200 rows: PlainData[24:224].
	if c.Data[0][0] != 24 {
		t.Errorf("Data[0][0] = %v; want 24 (strip top 24 rows)", c.Data[0][0])
	}
	if len(c.Data) != 200 {
		t.Errorf("len(Data) = %d; want 200", len(c.Data))
	}
}

func TestArrangeDataDyZeroFallback(t *testing.T) {
	// Dy=0 → fallback to Dy=Py.
	c := &Composite{Dx: 2, Dy: 0, Px: 2, Py: 3}
	c.PlainData = [][]float32{{1, 2}, {3, 4}, {5, 6}}
	c.arrangeData()

	if c.Dy != 3 {
		t.Errorf("Dy = %d; want 3 (fallback from 0)", c.Dy)
	}
	if c.Dz != 1 {
		t.Errorf("Dz = %d; want 1", c.Dz)
	}
}

// --- Runlength decoding ---

func TestDecodeRunlengthChainedOffset(t *testing.T) {
	// Test 255-chaining for large offsets.
	// offset=255 → skip 239, then offset=20 → skip 4. Total = 243.
	// Then value 0x12 → 1 repetition of class 2.
	levels := []float32{5.0, 10.0, 15.0}
	c := &Composite{level: levels}

	dst := make([]float32, 250)
	line := []byte{
		0x00,       // line number (skipped)
		0xFF,       // offset: 255 → chain, adds 239
		20,         // offset: 20 → adds 4, total pos=243
		0x12,       // value: 1 rep of class 2 → levels[1]=10.0
	}

	if err := c.decodeRunlength(dst, line); err != nil {
		t.Fatalf("decodeRunlength: %v", err)
	}

	// Position 243 should be 10.0; everything else NaN.
	if dst[243] != 10.0 {
		t.Errorf("dst[243] = %v; want 10.0", dst[243])
	}
	if !IsNaN(dst[0]) {
		t.Errorf("dst[0] = %v; want NaN", dst[0])
	}
	if !IsNaN(dst[242]) {
		t.Errorf("dst[242] = %v; want NaN", dst[242])
	}
}

func TestDecodeRunlengthClass0IsNaN(t *testing.T) {
	// Class index 0 → NaN (via rvp6Runlength).
	levels := []float32{5.0}
	c := &Composite{level: levels}

	dst := make([]float32, 4)
	line := []byte{
		0x00, // line number
		16,   // offset: pos 0
		0x20, // value: 2 reps of class 0 → NaN
		0x21, // value: 2 reps of class 1 → 5.0
	}

	if err := c.decodeRunlength(dst, line); err != nil {
		t.Fatalf("decodeRunlength: %v", err)
	}

	if !IsNaN(dst[0]) || !IsNaN(dst[1]) {
		t.Errorf("class 0 should be NaN: dst[0]=%v dst[1]=%v", dst[0], dst[1])
	}
	if dst[2] != 5.0 || dst[3] != 5.0 {
		t.Errorf("class 1 should be 5.0: dst[2]=%v dst[3]=%v", dst[2], dst[3])
	}
}

func TestDecodeRunlengthInvalidOffset(t *testing.T) {
	// Offset byte < 16 → error.
	c := &Composite{level: []float32{1.0}}
	dst := make([]float32, 4)
	line := []byte{0x00, 10} // offset=10 < 16

	err := c.decodeRunlength(dst, line)
	if err == nil {
		t.Fatal("expected error for offset < 16")
	}
}

func TestDecodeRunlengthOverflow(t *testing.T) {
	// Runlength overflows destination.
	c := &Composite{level: []float32{1.0}}
	dst := make([]float32, 2)
	line := []byte{
		0x00, // line number
		16,   // offset: pos 0
		0xF1, // value: 15 reps of class 1 → overflows dst of size 2
	}

	err := c.decodeRunlength(dst, line)
	if err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestRvp6RunlengthOutOfBounds(t *testing.T) {
	// Class index beyond level table → NaN (border markings).
	c := &Composite{level: []float32{5.0}} // 1 level, valid index is 0 (value-1)
	result := c.rvp6Runlength(3)           // index 2, out of bounds
	if !IsNaN(result) {
		t.Errorf("rvp6Runlength(3) = %v; want NaN", result)
	}
}

// --- Little-endian edge cases ---

func TestDecodeLittleEndianNegativeValue(t *testing.T) {
	// Bit 6 of high byte = negative sign.
	c := &Composite{DataUnit: Unit_mm, precisionMult: 0.1}
	dst := make([]float32, 1)

	// value = (0x01 << 8) | 0x00 = 256; negative flag set → -256; × 0.1 = -25.6
	line := []byte{0x00, 0x41} // high=0x41: bits 0-3=1, bit6=1
	if err := c.decodeLittleEndian(dst, line); err != nil {
		t.Fatalf("decodeLittleEndian: %v", err)
	}
	want := float32(-256) * 0.1
	if math.Abs(float64(dst[0]-want)) > 1e-4 {
		t.Errorf("dst[0] = %v; want %v", dst[0], want)
	}
}

func TestDecodeLittleEndianNonDBZ(t *testing.T) {
	// Unit_mm → no dBZ conversion, just precision scaling.
	c := &Composite{DataUnit: Unit_mm, precisionMult: 0.1}
	dst := make([]float32, 1)

	// value = (0x00 << 8) | 100 = 100; × 0.1 = 10.0
	line := []byte{100, 0x00}
	if err := c.decodeLittleEndian(dst, line); err != nil {
		t.Fatalf("decodeLittleEndian: %v", err)
	}
	if math.Abs(float64(dst[0]-10.0)) > 1e-4 {
		t.Errorf("dst[0] = %v; want 10.0", dst[0])
	}
}

func TestDecodeLittleEndianSizeMismatch(t *testing.T) {
	c := &Composite{DataUnit: Unit_dBZ, precisionMult: 1.0}
	dst := make([]float32, 2)
	line := []byte{0x00, 0x00, 0x00} // odd length → error
	err := c.decodeLittleEndian(dst, line)
	if err == nil {
		t.Fatal("expected error for odd line length")
	}
}

// --- Single-byte edge cases ---

func TestRvp6SingleByteNonDBZ(t *testing.T) {
	// Unit_mm → no dBZ conversion, just precision scaling.
	c := &Composite{DataUnit: Unit_mm, precisionMult: 0.1}
	result := c.rvp6SingleByte(100)
	want := float32(100) * 0.1
	if math.Abs(float64(result-want)) > 1e-4 {
		t.Errorf("rvp6SingleByte(100) = %v; want %v", result, want)
	}
}

func TestRvp6SingleByteNaN(t *testing.T) {
	c := &Composite{DataUnit: Unit_dBZ, precisionMult: 1.0}
	result := c.rvp6SingleByte(250)
	if !IsNaN(result) {
		t.Errorf("rvp6SingleByte(250) = %v; want NaN", result)
	}
}

func TestDecodeSingleByteSizeMismatch(t *testing.T) {
	c := &Composite{DataUnit: Unit_dBZ, precisionMult: 1.0}
	dst := make([]float32, 3)
	line := []byte{0x00, 0x00} // len(line)=2 != len(dst)=3
	err := c.decodeSingleByte(dst, line)
	if err == nil {
		t.Fatal("expected error for size mismatch")
	}
}
