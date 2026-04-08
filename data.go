package radolan

import (
	"bufio"
)

// identifyAndParse detects the encoding from header fields and parses the
// binary data section. Encoding detection rules:
//   - level table present → run-length encoding
//   - dataLength == Px*Py*2 → 16-bit little-endian
//   - dataLength == Px*Py → 8-bit single-byte
func (c *Composite) identifyAndParse(rd *bufio.Reader) error {
	values := c.Px * c.Py

	switch {
	case c.level != nil:
		return c.parseRunlength(rd)
	case c.dataLength == values*2:
		return c.parseLittleEndian(rd)
	case c.dataLength == values:
		return c.parseSingleByte(rd)
	default:
		return newError("identifyAndParse", "unknown encoding: data length does not match any known format")
	}
}

// parseData parses the composite data and writes the related fields.
// This method requires header data to be already written.
func (c *Composite) parseData(reader *bufio.Reader) error {
	if c.Px == 0 || c.Py == 0 {
		return newError("parseData", "parsed header data required")
	}

	// Ensure precision multiplier is set (default: 10^0 = 1.0 when constructed
	// without parseHeader, e.g. in tests).
	if c.precisionMult == 0 {
		c.precisionMult = 1.0
	}

	// Allocate PlainData as a single contiguous backing array sliced into rows.
	// This reduces heap allocations from c.Py+1 to 2 and improves cache locality
	// during row-by-row decoding (parseLittleEndian, parseSingleByte).
	flat := make([]float32, c.Px*c.Py)
	c.PlainData = make([][]float32, c.Py)
	for i := range c.PlainData {
		c.PlainData[i] = flat[i*c.Px : (i+1)*c.Px]
	}

	return c.identifyAndParse(reader)
}

// arrangeData slices plain data into its data layers or strips the preceding
// vertical projection.
func (c *Composite) arrangeData() {
	if c.Dy <= 0 {
		c.Dy = c.Py // fallback: treat as single-layer composite
	}
	if c.Py%c.Dy == 0 { // multiple layers are linked downwards
		c.DataZ = make([][][]float32, c.Py/c.Dy)
		for i := range c.DataZ {
			c.DataZ[i] = c.PlainData[c.Dy*i : c.Dy*(i+1)] // split layers
		}
	} else { // only use bottom most part of plain data
		c.DataZ = [][][]float32{c.PlainData[c.Py-c.Dy:]} // strip elevation
	}

	c.Dz = len(c.DataZ)
	c.Data = c.DataZ[0] // alias
}
