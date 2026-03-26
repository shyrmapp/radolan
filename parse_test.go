package radolan

import (
	"bufio"
	"bytes"
	"math"
	"testing"
)

// TestParseLittleEndian verifies correct value decoding, NaN propagation for the
// no-data flag, and vertical flip after the single-buffer allocation change.
//
// Encoding rules (precision=0, Unit_dBZ):
//   tuple = [low, high]; value = (high&0x0F)<<8 | low
//   bit5 of high = no-data → NaN
//   decoded dBZ = float32(value)/2.0 − 32.5
//
// The decoder reads binary row 0 first and writes it to PlainData[last] (flip),
// so binary row 0 → PlainData[2], row 1 → PlainData[1], row 2 → PlainData[0].
func TestParseLittleEndian(t *testing.T) {
	// 2×3 composite. dataLength = Px*Py*2 triggers littleEndian path.
	//
	// Binary row 0: rvp6=65 (0.0 dBZ),  rvp6=100 (17.5 dBZ)
	// Binary row 1: no-data (NaN),       rvp6=200 (67.5 dBZ)
	// Binary row 2: rvp6=65 (0.0 dBZ),  rvp6=65  (0.0 dBZ)
	payload := []byte{
		0x41, 0x00, 0x64, 0x00, // row 0: rvp6=65, rvp6=100
		0x00, 0x20, 0xC8, 0x00, // row 1: no-data (bit5 set), rvp6=200
		0x41, 0x00, 0x41, 0x00, // row 2: rvp6=65, rvp6=65
	}

	c := &Composite{Px: 2, Py: 3, Dx: 2, Dy: 3, DataUnit: Unit_dBZ, dataLength: 12}
	if err := c.parseData(bufio.NewReader(bytes.NewReader(payload))); err != nil {
		t.Fatalf("parseData: %v", err)
	}

	check := func(row, col int, wantNaN bool, want float32) {
		t.Helper()
		got := c.PlainData[row][col]
		if wantNaN {
			if !IsNaN(got) {
				t.Errorf("PlainData[%d][%d] = %v; want NaN", row, col, got)
			}
			return
		}
		if math.Abs(float64(got-want)) > 1e-4 {
			t.Errorf("PlainData[%d][%d] = %v; want %v", row, col, got, want)
		}
	}

	// Vertical flip: PlainData[0] ← binary row 2, PlainData[2] ← binary row 0.
	check(0, 0, false, 0.0)  // binary row 2, pixel 0
	check(0, 1, false, 0.0)  // binary row 2, pixel 1
	check(1, 0, true, 0)     // binary row 1, pixel 0: no-data → NaN
	check(1, 1, false, 67.5) // binary row 1, pixel 1
	check(2, 0, false, 0.0)  // binary row 0, pixel 0
	check(2, 1, false, 17.5) // binary row 0, pixel 1
}

// TestParseSingleByte verifies the same properties as TestParseLittleEndian for
// the single-byte encoding: value=250 is the no-data sentinel; all others decode
// as float32(value)/2.0 − 32.5 for Unit_dBZ.
func TestParseSingleByte(t *testing.T) {
	// 2×3 composite. dataLength = Px*Py triggers singleByte path.
	//
	// Binary row 0: value=65 (0.0 dBZ),  value=100 (17.5 dBZ)
	// Binary row 1: value=250 (NaN),     value=200 (67.5 dBZ)
	// Binary row 2: value=65 (0.0 dBZ),  value=65  (0.0 dBZ)
	payload := []byte{
		65, 100,  // row 0
		250, 200, // row 1: 250 = no-data sentinel
		65, 65,   // row 2
	}

	c := &Composite{Px: 2, Py: 3, Dx: 2, Dy: 3, DataUnit: Unit_dBZ, dataLength: 6}
	if err := c.parseData(bufio.NewReader(bytes.NewReader(payload))); err != nil {
		t.Fatalf("parseData: %v", err)
	}

	check := func(row, col int, wantNaN bool, want float32) {
		t.Helper()
		got := c.PlainData[row][col]
		if wantNaN {
			if !IsNaN(got) {
				t.Errorf("PlainData[%d][%d] = %v; want NaN", row, col, got)
			}
			return
		}
		if math.Abs(float64(got-want)) > 1e-4 {
			t.Errorf("PlainData[%d][%d] = %v; want %v", row, col, got, want)
		}
	}

	// Same flip layout as littleEndian.
	check(0, 0, false, 0.0)
	check(0, 1, false, 0.0)
	check(1, 0, true, 0)
	check(1, 1, false, 67.5)
	check(2, 0, false, 0.0)
	check(2, 1, false, 17.5)
}
