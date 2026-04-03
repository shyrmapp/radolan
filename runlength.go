package radolan

import (
	"bufio"
)

// parseRunlength parses the runlength encoded composite and writes into the
// previously created PlainData field of the composite.
func (c *Composite) parseRunlength(reader *bufio.Reader) error {
	for i := range c.PlainData {
		line, err := c.readLineRunlength(reader)
		if err != nil {
			return err
		}

		err = c.decodeRunlength(c.PlainData[i], line)
		if err != nil {
			return err
		}
	}

	return nil
}

// readLineRunlength reads a line until newline (non inclusive) from the given reader.
// This method is used to get a line of runlenth encoded data.
func (c *Composite) readLineRunlength(rd *bufio.Reader) (line []byte, err error) {
	line, err = rd.ReadBytes('\x0A')
	if err != nil {
		err = newError("readLineRunlength", err.Error())
	}
	length := len(line)
	if length > 0 {
		line = line[:length-1]
	}
	return
}

// decodeRunlength decodes the source line and writes to the given destination.
func (c *Composite) decodeRunlength(dst []float32, line []byte) error {
	// fill destination as runlength encoding will induce gaps
	for i := range dst {
		dst[i] = NaN
	}

	dstpos := 0
	offset := true
	for i, value := range line {
		if i == 0 { // skip useless line number
			continue
		}
		if offset { // calculate offset position
			if value < 16 {
				return newError("decodeRunlength", "invalid offset value")
			}
			dstpos += int(value) - 16 // update offset position
			offset = value == 255     // chained offset: next byte is also offset
		} else {
			// value [XXXX|YYYY] decodes to YYYY repeated XXXX times.
			runlength := int(value >> 4)
			value &= 0x0F

			for range runlength {
				if dstpos >= len(dst) {
					return newError("decodeRunlength", "destination size exceeded")
				}
				dst[dstpos] = c.rvp6Runlength(value)
				dstpos++
			}
		}
	}

	return nil
}

// rvp6Runlength sets the value of level based composite products to radar
// video processor values (rvp-6).
func (c *Composite) rvp6Runlength(value byte) float32 {
	if value == 0 {
		return NaN
	}
	value--

	if int(value) >= len(c.level) { // border markings
		return NaN
	}
	return c.level[value]
}
