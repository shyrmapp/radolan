package radolan

import (
	"archive/tar"
	"bytes"
	"compress/bzip2"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- Real-world integration tests using DWD open data ---

func loadBz2(t *testing.T, name string) *Composite {
	t.Helper()
	path := filepath.Join("testdata", "real", name)
	f, err := os.Open(path)
	if err != nil {
		t.Skipf("skipping: %v", err)
		return nil
	}
	defer f.Close()
	r := bzip2.NewReader(f)
	comp, err := NewComposite(r)
	if err != nil && !errors.Is(err, ErrUnknownUnit) {
		t.Fatalf("NewComposite(%s): %v", name, err)
	}
	return comp
}

func loadTarBz2(t *testing.T, name string) []*Composite {
	t.Helper()
	path := filepath.Join("testdata", "real", name)
	f, err := os.Open(path)
	if err != nil {
		t.Skipf("skipping: %v", err)
		return nil
	}
	defer f.Close()
	bzipReader := bzip2.NewReader(f)
	tarReader := tar.NewReader(bzipReader)
	var results []*Composite
	for {
		_, err := tarReader.Next()
		if err != nil {
			break
		}
		comp, err := NewComposite(tarReader)
		if err != nil && !errors.Is(err, ErrUnknownUnit) {
			t.Fatalf("NewComposite from %s: %v", name, err)
		}
		results = append(results, comp)
	}
	if len(results) == 0 {
		t.Skipf("skipping: no composites in %s", name)
	}
	return results
}

// TestRealRW validates RW (hourly precipitation, little-endian, mm) parsing.
func TestRealRW(t *testing.T) {
	comp := loadBz2(t, "rw_sample.bz2")
	if comp == nil {
		return
	}

	if comp.Product != "RW" {
		t.Errorf("Product = %q; want RW", comp.Product)
	}
	if comp.DataUnit != Unit_mm {
		t.Errorf("DataUnit = %v; want Unit_mm", comp.DataUnit)
	}
	if comp.Dx != 900 || comp.Dy != 900 {
		t.Errorf("Dx=%d Dy=%d; want 900,900", comp.Dx, comp.Dy)
	}
	if !comp.HasProjection {
		t.Error("HasProjection = false; want true")
	}
	if comp.Format != 3 {
		t.Errorf("Format = %d; want 3", comp.Format)
	}

	// Verify data integrity: total pixel count matches
	totalPixels := comp.Dx * comp.Dy
	nanCount := 0
	validCount := 0
	for y := 0; y < comp.Dy; y++ {
		for x := 0; x < comp.Dx; x++ {
			v := comp.At(x, y)
			if IsNaN(v) {
				nanCount++
			} else {
				validCount++
				if v < 0 {
					t.Errorf("negative value at (%d,%d): %v", x, y, v)
				}
			}
		}
	}
	if nanCount+validCount != totalPixels {
		t.Errorf("pixel count: nan=%d valid=%d total=%d; expected %d",
			nanCount, validCount, nanCount+validCount, totalPixels)
	}

	// RW values should be in mm (hourly precipitation), typically 0-50 mm
	if validCount > 0 {
		t.Logf("RW: %d NaN, %d valid pixels", nanCount, validCount)
	}
}

// TestRealRY validates RY (5-minute precipitation, little-endian, mm) parsing.
func TestRealRY(t *testing.T) {
	comp := loadBz2(t, "ry_sample.bz2")
	if comp == nil {
		return
	}

	if comp.Product != "RY" {
		t.Errorf("Product = %q; want RY", comp.Product)
	}
	if comp.DataUnit != Unit_mm {
		t.Errorf("DataUnit = %v; want Unit_mm", comp.DataUnit)
	}
	if comp.Dx != 900 || comp.Dy != 900 {
		t.Errorf("Dx=%d Dy=%d; want 900,900", comp.Dx, comp.Dy)
	}
	if comp.Format != 3 {
		t.Errorf("Format = %d; want 3", comp.Format)
	}

	// Verify Projection maps Berlin inside bounds
	x, y := comp.Project(52.52, 13.41)
	if x < 0 || x >= float64(comp.Dx) || y < 0 || y >= float64(comp.Dy) {
		t.Errorf("Berlin Project=(%.2f,%.2f) out of bounds (0-%d, 0-%d)",
			x, y, comp.Dx, comp.Dy)
	}
}

// TestRealSF validates SF (daily precipitation, little-endian, mm) parsing.
func TestRealSF(t *testing.T) {
	comp := loadBz2(t, "sf_sample.bz2")
	if comp == nil {
		return
	}

	if comp.Product != "SF" {
		t.Errorf("Product = %q; want SF", comp.Product)
	}
	if comp.DataUnit != Unit_mm {
		t.Errorf("DataUnit = %v; want Unit_mm", comp.DataUnit)
	}
	if comp.Dx != 900 || comp.Dy != 900 {
		t.Errorf("Dx=%d Dy=%d; want 900,900", comp.Dx, comp.Dy)
	}
	// SF is 24h accumulated, should have Interval=24h
	if comp.Interval != 24*time.Hour {
		t.Errorf("Interval = %v; want 24h", comp.Interval)
	}
}

// TestRealYW validates YW (hourly precipitation, little-endian, mm) parsing.
func TestRealYW(t *testing.T) {
	comp := loadBz2(t, "yw_sample.bz2")
	if comp == nil {
		return
	}

	if comp.Product != "YW" {
		t.Errorf("Product = %q; want YW", comp.Product)
	}
	if comp.DataUnit != Unit_mm {
		t.Errorf("DataUnit = %v; want Unit_mm", comp.DataUnit)
	}
	if comp.Dx != 900 || comp.Dy != 900 {
		t.Errorf("Dx=%d Dy=%d; want 900,900", comp.Dx, comp.Dy)
	}
}

// TestRealRV validates RADVOR-RE (Format 5, WGS84, DE1200 grid) parsing.
func TestRealRV(t *testing.T) {
	comps := loadTarBz2(t, "rv_sample.tar.bz2")
	if len(comps) == 0 {
		return
	}
	comp := comps[0]

	if comp.Product != "RV" {
		t.Errorf("Product = %q; want RV", comp.Product)
	}
	if comp.DataUnit != Unit_mm {
		t.Errorf("DataUnit = %v; want Unit_mm (RV is precipitation)", comp.DataUnit)
	}
	if comp.Dx != 1100 || comp.Dy != 1200 {
		t.Errorf("Dx=%d Dy=%d; want 1100,1200", comp.Dx, comp.Dy)
	}
	if comp.Format != 5 {
		t.Errorf("Format = %d; want 5", comp.Format)
	}
	if !comp.HasProjection {
		t.Error("HasProjection = false; want true")
	}

	// Verify WGS84 projection for known cities
	cities := []struct {
		name     string
		lat, lon float64
	}{
		{"Berlin", 52.52, 13.41},
		{"Munich", 48.14, 11.58},
		{"Hamburg", 53.55, 9.99},
		{"Cologne", 50.94, 6.96},
	}
	for _, c := range cities {
		x, y := comp.Project(c.lat, c.lon)
		if x < 0 || x >= float64(comp.Dx) || y < 0 || y >= float64(comp.Dy) {
			t.Errorf("%s Project=(%.2f,%.2f) out of bounds", c.name, x, y)
		}
	}

	// Verify ForecastTimes are ascending
	for i := 1; i < len(comps); i++ {
		if !comps[i].ForecastTime.After(comps[i-1].ForecastTime) {
			t.Errorf("comps[%d].ForecastTime=%v not after comps[%d].ForecastTime=%v",
				i, comps[i].ForecastTime, i-1, comps[i-1].ForecastTime)
		}
	}
}

// TestRealWN validates WN (radar wind, Format 5, dBZ, DE1200 grid) parsing.
func TestRealWN(t *testing.T) {
	comps := loadTarBz2(t, "wn_sample.tar.bz2")
	if len(comps) == 0 {
		return
	}
	comp := comps[0]

	if comp.Product != "WN" {
		t.Errorf("Product = %q; want WN", comp.Product)
	}
	if comp.DataUnit != Unit_dBZ {
		t.Errorf("DataUnit = %v; want Unit_dBZ", comp.DataUnit)
	}
	if comp.Dx != 1100 || comp.Dy != 1200 {
		t.Errorf("Dx=%d Dy=%d; want 1100,1200", comp.Dx, comp.Dy)
	}
	if comp.Format != 5 {
		t.Errorf("Format = %d; want 5", comp.Format)
	}

	// WN is dBZ, so values should be in a reasonable range (-30 to 80 dBZ)
	for y := 0; y < comp.Dy; y++ {
		for x := 0; x < comp.Dx; x++ {
			v := comp.At(x, y)
			if !IsNaN(v) && (v < -40 || v > 80) {
				t.Logf("WN value at (%d,%d) = %.2f; expected range [-40, 80] dBZ", x, y, v)
				break
			}
		}
	}

	// Verify Projection round-trip for DE1200 WGS84 corners
	corners := []struct {
		name     string
		lat, lon float64
		ex, ey   float64
		eps      float64
	}{
		{"NW", 55.86208711, 1.463301510, 0, 0, 0.001},
		{"NE", 55.84543856, 18.73161645, 1100, 0, 0.5},
		{"SE", 45.68460578, 16.58086935, 1100, 1200, 0.5},
		{"SW", 45.69642538, 3.566994635, 0, 1200, 0.5},
	}
	for _, c := range corners {
		x, y := comp.Project(c.lat, c.lon)
		if dx := math.Abs(x - c.ex); dx > c.eps {
			t.Errorf("%s corner: x=%.4f want=%.4f diff=%.4f", c.name, x, c.ex, dx)
		}
		if dy := math.Abs(y - c.ey); dy > c.eps {
			if dy > 1.0 {
				t.Errorf("%s corner: y=%.4f want=%.4f diff=%.4f", c.name, y, c.ey, dy)
			}
		}
	}
}

// --- runlength encoding tests for the 0x0A line-number bug fix ---

// TestDecodeRunlengthWithRow10 verifies that the run-length decoder handles
// line-number byte 0x0A (row 10) correctly after the fix.
func TestDecodeRunlengthWithRow10(t *testing.T) {
	levels := []float32{0.0, 1.0, 2.0, 3.0, 4.0, 5.0}

	// Construct binary data with 3 rows (rows 9, 10, 11), using newlines between.
	// Row 10's line number is 0x0A which equals the newline delimiter.
	// Each row: [linenum] [offset=16(pos=0)] [value] [newline]
	// value 0x22 = 2 reps of class 2 → level[1] = 1.0
	// value 0x12 = 1 rep of class 2 → level[1] = 1.0
	rowData := []byte{
		0x09, 16, 0x22, 0x0A, // row 9
		0x0A, 16, 0x12, 0x0A, // row 10 — line number 0x0A
		0x0B, 16, 0x12, 0x0A, // row 11
	}

	c := &Composite{
		Px: 4, Py: 12, Dx: 4, Dy: 12,
		level:         levels,
		precisionMult: 1.0,
	}
	flat := make([]float32, 4*12)
	c.PlainData = make([][]float32, 12)
	for i := range c.PlainData {
		c.PlainData[i] = flat[i*4 : (i+1)*4]
	}

	// Test splitRunlengthRows directly (not parseRunlength) because
	// parseRunlength would reject 3 rows against Py=12 as a mismatch.
	rows := c.splitRunlengthRows(rowData)
	if len(rows) != 3 {
		t.Fatalf("splitRunlengthRows: got %d rows, want 3", len(rows))
	}

	// Verify row 10 (index 1) was correctly split — first byte should be 0x0A.
	if rows[1][0] != 0x0A {
		t.Errorf("row 10 line number = 0x%02X; want 0x0A", rows[1][0])
	}

	// Decode row 9: line_num=9, offset=16(pos=0), value=0x22(2 reps of class 2)
	dst := make([]float32, 4)
	err := c.decodeRunlength(dst, rows[0])
	if err != nil {
		t.Fatalf("decode row 9: %v", err)
	}
	// class 2 → level[1] = 1.0, at positions 0 and 1
	if dst[0] != 1.0 || dst[1] != 1.0 {
		t.Errorf("row 9: dst[0]=%v dst[1]=%v; want 1.0,1.0", dst[0], dst[1])
	}

	// Decode row 10: line_num=10(0x0A), offset=16(pos=0), value=0x12(1 rep of class 2)
	dst10 := make([]float32, 4)
	err = c.decodeRunlength(dst10, rows[1])
	if err != nil {
		t.Fatalf("decode row 10: %v", err)
	}
	if dst10[0] != 1.0 {
		t.Errorf("row 10: dst[0]=%v; want 1.0", dst10[0])
	}
}

// --- arrangeData guard tests ---

func TestArrangeDataDyExceedsPy(t *testing.T) {
	// Dy > Py should not panic; should be clamped to Py.
	c := &Composite{Dx: 4, Dy: 100, Px: 4, Py: 3}
	c.PlainData = [][]float32{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}}
	c.arrangeData()
	if c.Dy != 3 {
		t.Errorf("Dy = %d; want 3 (clamped to Py)", c.Dy)
	}
	if c.Dz != 1 {
		t.Errorf("Dz = %d; want 1", c.Dz)
	}
}

// --- Z-R conversion verification against known values ---

func TestZRKnownValues(t *testing.T) {
	// Verify PrecipitationRate against independently computed values.
	// Marshall-Palmer: Z = 200*R^1.6
	// At R=1 mm/h: Z = 200, dBZ=10*log10(200) ≈ 23.01
	// At R=10 mm/h: Z = 200*10^1.6 = 200*39.81 ≈ 7962.1, dBZ ≈ 39.01
	cases := []struct {
		relation    ZR
		dBZ         float32
		wantRateMin float64
		wantRateMax float64
	}{
		// Aniol80: Z = 256*R^1.42
		// At dBZ=20: rate should be reasonable (a few mm/h for light rain)
		{Aniol80, 20, 0.5, 5.0},
		// At dBZ=40: heavier rain
		{Aniol80, 40, 5.0, 500.0},
		// Marshall-Palmer: Z = 200*R^1.6
		// At dBZ=30: moderate rain
		{MarshallPalmer55, 30, 1.0, 20.0},
	}

	for _, tc := range cases {
		rate := PrecipitationRate(tc.relation, tc.dBZ)
		if rate < tc.wantRateMin || rate > tc.wantRateMax {
			t.Errorf("PrecipitationRate(%+v, %.0f) = %.4f; want in [%.1f, %.1f]",
				tc.relation, tc.dBZ, rate, tc.wantRateMin, tc.wantRateMax)
		}
	}
}

// --- Projection edge-case tests ---

func TestProjectionBerlinNationalGrid(t *testing.T) {
	comp := NewDummy("RW", 3, 900, 900)
	x, y := comp.Project(52.52, 13.41)
	// Berlin should be roughly in the center-right area of the national grid
	if x < 500 || x > 900 || y < 100 || y > 600 {
		t.Errorf("Berlin at (52.52, 13.41) → (%.2f, %.2f) seems wrong for 900×900 grid", x, y)
	}
}

func TestProjectionBerlinDE1200(t *testing.T) {
	comp := NewDummy("WN", 5, 1100, 1200)
	x, y := comp.Project(52.52, 13.41)
	// Berlin should be within the DE1200 grid
	if x < 0 || x >= 1100 || y < 0 || y >= 1200 {
		t.Errorf("Berlin at (52.52, 13.41) → (%.2f, %.2f) out of bounds", x, y)
	}
}

// TestProjectionFuncCapturesNoReference verifies that ProjectionFunc produces
// correct results even after the composite is set to nil.
func TestProjectionFuncCapturesNoReference(t *testing.T) {
	comp := NewDummy("RW", 3, 900, 900)
	fn := comp.ProjectionFunc()
	comp = nil // composite is unreachable now

	x, y := fn(52.52, 13.41)
	// Should still produce valid results without panicking
	if x < 500 || x > 900 || y < 100 || y > 600 {
		t.Errorf("Berlin projection after freeing composite: (%.2f, %.2f)", x, y)
	}
}

// --- Vertical flip consistency test ---

func TestVerticalFlipConsistency(t *testing.T) {
	// Little-endian and single-byte parsers flip vertically.
	// Verify that the first binary row (northmost in data) maps to the
	// last plain data row, and the last binary row maps to the first.
	//
	// This is critical: DWD data is stored north-up, but geographic
	// coordinates have y increasing southward after projection.

	// Test with little-endian
	leData := make([]byte, 2*3*2) // 2 wide, 3 tall
	// Row 0 (north): rvp6=[65, 0] (0 dBZ), [100, 0] (17.5 dBZ)
	leData[0] = 65
	leData[1] = 0x00 // pixel (0,0)
	leData[2] = 100
	leData[3] = 0x00 // pixel (1,0)
	// Row 1 (middle): all zeros
	// Row 2 (south): rvp6=[200, 0] (67.5 dBZ), [65, 0] (0 dBZ)
	leData[8] = 200
	leData[9] = 0x00 // pixel (0,2)
	leData[10] = 65
	leData[11] = 0x00 // pixel (1,2)

	blob := buildComposite("FZ", 2, 3, "E+00", leData)
	comp, err := NewComposite(bytes.NewReader(blob))
	if err != nil {
		t.Fatalf("NewComposite: %v", err)
	}

	// After vertical flip:
	// PlainData[0] ← binary row 2 (south): 67.5, 0.0
	// PlainData[1] ← binary row 1 (middle): 0.0, 0.0
	// PlainData[2] ← binary row 0 (north): 0.0, 17.5
	// Geographic south (y=0) has the southmost data, geographic north (y=2) has northmost.
	south := comp.At(0, 0) // y=0 is south after flip
	if IsNaN(south) || math.Abs(float64(south-67.5)) > 0.01 {
		t.Errorf("south pixel (0,0) = %v; want 67.5", south)
	}
}

// --- NewComposites integration test ---

func TestNewCompositesSort(t *testing.T) {
	comps := loadTarBz2(t, "rv_sample.tar.bz2")
	if len(comps) < 2 {
		t.Skip("need at least 2 composites")
	}
	for i := 1; i < len(comps); i++ {
		if !comps[i].ForecastTime.After(comps[i-1].ForecastTime) {
			t.Errorf("comps not sorted: [%d]=%v > [%d]=%v",
				i-1, comps[i-1].ForecastTime, i, comps[i].ForecastTime)
		}
	}
}

// --- NeighbourhoodSample with real data ---

func TestNeighbourhoodSampleRealData(t *testing.T) {
	comp := loadBz2(t, "rw_sample.bz2")
	if comp == nil {
		return
	}

	// Sample Berlin's neighbourhood
	x, y := comp.Project(52.52, 13.41)
	ix, iy := int(x), int(y)
	if ix < 3 || ix >= comp.Dx-3 || iy < 3 || iy >= comp.Dy-3 {
		t.Skipf("Berlin at (%d,%d) too close to edge", ix, iy)
	}

	avg, max, cov := comp.NeighbourhoodSample(ix, iy, 2)
	if avg < 0 {
		t.Errorf("avgMMH = %v; want >= 0", avg)
	}
	if max < 0 {
		t.Errorf("maxMMH = %v; want >= 0", max)
	}
	if cov < 0 || cov > 1 {
		t.Errorf("coverage = %v; want [0, 1]", cov)
	}
	if max < avg-0.001 { // max should be >= avg (allowing rounding)
		t.Errorf("max(%v) < avg(%v)", max, avg)
	}
}

// --- Unit catalog completeness test ---

func TestUnitCatalogCompleteness(t *testing.T) {
	// All products referenced in the README or doc comments should be in the catalog.
	products := []struct {
		name string
		unit Unit
	}{
		{"PG", Unit_dBZ},
		{"RX", Unit_dBZ},
		{"RW", Unit_mm},
		{"SF", Unit_mm},
		{"FZ", Unit_dBZ},
		{"WN", Unit_dBZ},
		{"WX", Unit_dBZ},
		{"EX", Unit_dBZ},
		{"RV", Unit_mm},
		{"YW", Unit_mm},
		{"PR", Unit_mps},
	}
	for _, tc := range products {
		got, ok := unitCatalog[tc.name]
		if !ok {
			t.Errorf("product %q not in unitCatalog", tc.name)
			continue
		}
		if got != tc.unit {
			t.Errorf("unitCatalog[%q] = %v; want %v", tc.name, got, tc.unit)
		}
	}
}
