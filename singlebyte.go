package radolan

import (
	"bufio"
	"io"
)

// parseSingleByte parses the single byte encoded composite as described in [1] and writes
// into the previously created PlainData field of the composite.
// A single line buffer is reused across all rows to avoid per-row heap allocation.
func (c *Composite) parseSingleByte(reader *bufio.Reader) error {
	line := make([]byte, c.Dx) // one allocation for the whole composite
	last := len(c.PlainData) - 1
	for i := range c.PlainData {
		if _, err := io.ReadFull(reader, line); err != nil {
			return newError("parseSingleByte", err.Error())
		}
		if err := c.decodeSingleByte(c.PlainData[last-i], line); err != nil { // vertically flipped
			return err
		}
	}
	return nil
}

// decodeSingleByte decodes the source line and writes to the given destination.
func (c *Composite) decodeSingleByte(dst []float32, line []byte) error {
	if len(dst) != len(line) {
		return newError("decodeSingleByte", "wrong destination or source size")
	}

	dBZ := c.DataUnit == Unit_dBZ

	for i, v := range line {
		if v == 250 {
			dst[i] = NaN
			continue
		}

		conv := c.rvp6Raw(int(v))

		if dBZ {
			dst[i] = toDBZ(conv)
		} else {
			dst[i] = conv
		}
	}

	return nil
}

// rvp6SingleByte converts the raw byte of single byte encoded
// composite products to radar video processor values (rvp-6). NaN may be returned
// when the no-data flag is set.
func (c *Composite) rvp6SingleByte(value byte) float32 {
	if value == 250 { // error code: no-data
		return NaN
	}

	conv := c.rvp6Raw(int(value)) // set decimal point

	// not sure if single byte formats are even used for other things than dBZ (RX, dBZ)
	if c.DataUnit != Unit_dBZ {
		return conv
	}

	return toDBZ(conv)
}
