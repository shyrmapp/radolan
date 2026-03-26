package radolan

import (
	"bufio"
	"io"
)

// parseLittleEndian parses the little endian encoded composite as described in [1] and [3].
// Result are written into the previously created PlainData field of the composite.
// A single line buffer is reused across all rows to avoid per-row heap allocation.
func (c *Composite) parseLittleEndian(reader *bufio.Reader) error {
	line := make([]byte, c.Dx*2) // one allocation for the whole composite
	last := len(c.PlainData) - 1
	for i := range c.PlainData {
		if _, err := io.ReadFull(reader, line); err != nil {
			return newError("parseLittleEndian", err.Error())
		}
		if err := c.decodeLittleEndian(c.PlainData[last-i], line); err != nil { // vertically flipped
			return err
		}
	}
	return nil
}

// decodeLittleEndian decodes the source line and writes to the given destination.
func (c *Composite) decodeLittleEndian(dst []float32, line []byte) error {
	if len(line)%2 != 0 || len(dst)*2 != len(line) {
		return newError("decodeLittleEndian", "wrong destination or source size")
	}

	for i := range dst {
		tuple := [2]byte{line[2*i], line[2*i+1]}
		dst[i] = c.rvp6LittleEndian(tuple)
	}

	return nil
}

// rvp6LittleEndian converts the raw two byte tuple of little endian encoded composite products
// to radar video processor values (rvp-6). NaN may be returned when the no-data flag is set.
func (c *Composite) rvp6LittleEndian(tuple [2]byte) float32 {
	var value int = 0x0F & int(tuple[1])
	value = (value << 8) + int(tuple[0])

	if tuple[1]&(1<<5) != 0 { // error code: no-data
		return NaN
	}

	if tuple[1]&(1<<6) != 0 { // flag: negative value
		value *= -1
	}

	conv := c.rvp6Raw(value) // set decimal point

	// little endian encoded formats are also used for mm/h
	if c.DataUnit != Unit_dBZ {
		return conv
	}

	// Even though this format supports negative values and custom
	// precision they do not make use of this and we still have to subtract
	// the bias and scale it (RADVOR FX, dBZ)
	return toDBZ(conv)
}
