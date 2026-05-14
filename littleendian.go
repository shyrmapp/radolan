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

	dBZ := c.DataUnit == Unit_dBZ

	for i := range dst {
		lo := line[2*i]
		hi := line[2*i+1]

		if hi&(1<<5) != 0 {
			dst[i] = NaN
			continue
		}

		value := int(lo) | (int(hi&0x0F) << 8)
		if hi&(1<<6) != 0 {
			value *= -1
		}

		conv := c.rvp6Raw(value)

		if dBZ {
			dst[i] = toDBZ(conv)
		} else {
			dst[i] = conv
		}
	}

	return nil
}
