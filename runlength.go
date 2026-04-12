package radolan

import (
	"bufio"
	"io"
)

// parseRunlength parses the runlength encoded composite and writes into the
// previously created PlainData field of the composite. Data is vertically
// flipped to match the coordinate system used by parseLittleEndian and
// parseSingleByte (row 0 = southernmost row in geographic terms).
func (c *Composite) parseRunlength(reader *bufio.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return newError("parseRunlength", err.Error())
	}

	last := len(c.PlainData) - 1
	rows := c.splitRunlengthRows(data)

	if len(rows) != len(c.PlainData) {
		return newError("parseRunlength", "row count mismatch")
	}

	for i, line := range rows {
		err := c.decodeRunlength(c.PlainData[last-i], line) // vertically flipped
		if err != nil {
			return err
		}
	}

	return nil
}

// splitRunlengthRows splits run-length encoded binary data into per-row byte slices.
// Each row starts with a line-number byte (0–255) and is terminated by 0x0A.
// When the line number equals 0x0A (rows >= 10), the delimiter byte is the same value,
// so simple newline splitting would corrupt the data. This function handles that
// by scanning the data and identifying real line terminators.
func (c *Composite) splitRunlengthRows(data []byte) [][]byte {
	var rows [][]byte
	pos := 0

	for pos < len(data) && len(rows) < c.Py {
		start := pos

		// Skip line number byte.
		pos++

		// Scan data bytes until we find the line terminator (0x0A).
		// After the line number, any 0x0A byte is the terminator since
		// valid offset bytes are >= 16 and 0x0A as a value byte would
		// encode 0 repetitions (a no-op). A degenerate encoder emitting
		// 0x0A as a zero-repetition value byte would be misread as a
		// terminator, but no real DWD encoder produces such data.
		found := false
		for pos < len(data) {
			if data[pos] == 0x0A {
				rows = append(rows, data[start:pos])
				pos++ // skip the newline delimiter
				found = true
				break
			}
			pos++
		}

		// If no terminator was found for this row, include remaining data
		// as the last row (handles files without trailing newline).
		if !found && start < len(data) {
			rows = append(rows, data[start:])
			break
		}
	}

	return rows
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
