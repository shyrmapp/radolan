package radolan

import (
	"bufio"
	"strings"
	"testing"
	"time"
)

type headerTestcase struct {
	// head of file
	test string

	// expected
	expBinary       string
	expProduct      string
	expCaptureTime  time.Time
	expForecastTime time.Time
	expInterval     time.Duration
	expDx           int
	expDy           int
	expDataLength   int
	expPrecision    int
	expLevel        []float32
}

func TestParseHeaderPG(t *testing.T) {
	ht := &headerTestcase{}

	// head of file
	ht.test = "PG262115100000616BY22205LV 6  1.0 19.0 28.0 37.0 46.0 55.0CS0MX 0MS " +
		"88<boo,ros,emd,hnr,umd,pro,ess,fld,drs,neu,nhb,oft,eis,tur,isn,fbg,mem> " +
		"are used, BG460460\x03binarycontent"

	// expected
	ht.expBinary = "binarycontent"
	ht.expProduct = "PG"
	// Header stores "2115" → 21:15 UTC. "23:15 CEST" (UTC+2) is the same instant.
	ht.expCaptureTime = time.Date(2016, 6, 26, 21, 15, 0, 0, time.UTC)
	ht.expForecastTime = time.Date(2016, 6, 26, 21, 15, 0, 0, time.UTC)
	ht.expDx = 460
	ht.expDy = 460
	ht.expDataLength = 22205 - 159 // BY - header_etx_length
	ht.expPrecision = 0
	ht.expLevel = []float32{1.0, 19.0, 28.0, 37.0, 46.0, 55.0}

	testParseHeader(t, ht)
}

func TestParseHeaderFZ(t *testing.T) {
	ht := &headerTestcase{}

	// head of file
	ht.test = "FZ282105100000716BY 405160VS 3SW   2.13.1PR E-01INT   5GP 450x 450VV 100MF " +
		"00000002MS 66<boo,ros,emd,hnr,umd,pro,ess,drs,neu,nhb,oft,eis,tur,isn,fbg,mem>" +
		"\x03binarycontent"

	// ht.expected values
	ht.expBinary = "binarycontent"

	ht.expProduct = "FZ"
	// Header stores "2305"/"0045" → 21:05 and 22:45 UTC. CEST (UTC+2) equivalents
	// are 23:05 and 00:45+1day, which are the same instants.
	ht.expCaptureTime = time.Date(2016, 7, 28, 21, 5, 0, 0, time.UTC)
	ht.expForecastTime = time.Date(2016, 7, 28, 22, 45, 0, 0, time.UTC)
	ht.expInterval = 5 * time.Minute
	ht.expDx = 450
	ht.expDy = 450
	ht.expDataLength = 405160 - 154 // BY - header_etx_length
	ht.expPrecision = -1
	ht.expLevel = []float32(nil)

	testParseHeader(t, ht)
}

func testParseHeader(t *testing.T, ht *headerTestcase) {
	dummy := &Composite{}
	reader := bufio.NewReader(strings.NewReader(ht.test))

	// run
	if err := dummy.parseHeader(reader); err != nil {
		t.Errorf("%s.parseHeader(): returned error: %#v", ht.expProduct, err.Error())
	}

	// test results
	// Product
	if dummy.Product != ht.expProduct {
		t.Errorf("%s.parseHeader(): Product: %#v; expected: %#v", ht.expProduct,
			dummy.Product, ht.expProduct)
	}

	// CaptureTime
	if !dummy.CaptureTime.Equal(ht.expCaptureTime) {
		t.Errorf("%s.parseHeader(): CaptureTime: %#v; expected: %#v", ht.expProduct,
			dummy.CaptureTime.String(), ht.expCaptureTime.String())
	}

	// ForecastTime
	if !dummy.ForecastTime.Equal(ht.expForecastTime) {
		t.Errorf("%s.parseHeader(): ForecastTime: %#v; expected: %#v", ht.expProduct,
			dummy.ForecastTime.String(), ht.expForecastTime.String())
	}

	// Interval
	if dummy.Interval != ht.expInterval {
		t.Errorf("%s.parseHeader(): Interval: %#v; expected: %#v", ht.expProduct,
			dummy.Interval.String(), ht.expInterval.String())
	}

	// Dx Dy
	if dummy.Dx != ht.expDx || dummy.Dy != ht.expDy {
		t.Errorf("%s.parseHeader(): Dx: %d Dy: %d; expected Dx: %d Dy: %d", ht.expProduct,
			dummy.Dx, dummy.Dy, ht.expDx, ht.expDy)
	}

	// dataLength
	if dummy.dataLength != ht.expDataLength {
		t.Errorf("%s.parseHeader(): dataLength: %#v; expected: %#v", ht.expProduct,
			dummy.dataLength, ht.expDataLength)
	}

	// precision
	if dummy.precision != ht.expPrecision {
		t.Errorf("%s.parseHeader(): precision: %#v; expected: %#v", ht.expProduct,
			dummy.precision, ht.expPrecision)
	}

	// level
	for i := range ht.expLevel {
		if len(dummy.level) != len(ht.expLevel) || dummy.level[i] != ht.expLevel[i] {
			t.Errorf("%s.parseHeader(): level: %#v; expected: %#v", ht.expProduct,
				dummy.level, ht.expLevel)
		}
	}

	// check consistency
	if line, _ := reader.ReadString('\n'); line != ht.expBinary {
		t.Errorf("%s.parseHeader(): binary data corrupted", ht.expProduct)
	}
}

// --- Header edge case tests ---

func TestParseHeaderTooShort(t *testing.T) {
	// Header shorter than 22 bytes should fail.
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader("AB\x03")))
	if err == nil {
		t.Fatal("expected error for too-short header")
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderNoETX(t *testing.T) {
	// No ETX delimiter → ReadString fails with EOF.
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader("RX010000100000124BY1234567890")))
	if err == nil {
		t.Fatal("expected error when ETX is missing")
	}
}

func TestParseHeaderMissingBY(t *testing.T) {
	// Valid length but no BY field → data length parse fails.
	header := "RX010000100000124" + strings.Repeat(" ", 10) + "GP 900x 900\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error when BY field is missing")
	}
	if !strings.Contains(err.Error(), "data length") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderBadTime(t *testing.T) {
	// Corrupt timestamp bytes → time.Parse fails.
	header := "RXxxxxxx10000xxxx" + "BY 100GP 900x 900\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error for bad timestamp")
	}
	if !strings.Contains(err.Error(), "capture time") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderBadVV(t *testing.T) {
	// VV field with non-numeric content.
	header := "RX010000100000124BY 100GP 900x 900VV abc\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error for bad VV")
	}
	if !strings.Contains(err.Error(), "forecast time") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderBadINT(t *testing.T) {
	// INT field with non-numeric content.
	header := "RX010000100000124BY 100GP 900x 900INT abc\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error for bad INT")
	}
	if !strings.Contains(err.Error(), "interval") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderBadGP(t *testing.T) {
	// GP field with bad format.
	header := "RX010000100000124BY 100GP notvalid\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error for bad GP")
	}
	if !strings.Contains(err.Error(), "dimensions (GP)") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderBadBG(t *testing.T) {
	// BG field with bad format — no GP, so falls through to BG.
	header := "RX010000100000124BY 100BG abcdef\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error for bad BG")
	}
	if !strings.Contains(err.Error(), "dimensions (BG)") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderNoDimensionInfo(t *testing.T) {
	// Product not in dimensionCatalog and no GP/BG field.
	header := "ZZ010000100000124BY 100\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error for missing dimension info")
	}
	if !strings.Contains(err.Error(), "no dimension information") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderDimensionCatalogFallback(t *testing.T) {
	// PX is in dimensionCatalog (200x224, 200x200, 1km).
	// No GP/BG in header → falls back to catalog lookup.
	header := "PX010000100000124BY 100PR E+00LV 6  1.0 19.0 28.0 37.0 46.0 55.0\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Px != 200 || c.Py != 224 {
		t.Errorf("Px=%d Py=%d; want 200, 224", c.Px, c.Py)
	}
	if c.Dx != 200 || c.Dy != 200 {
		t.Errorf("Dx=%d Dy=%d; want 200, 200", c.Dx, c.Dy)
	}
	if c.Rx != 1.0 || c.Ry != 1.0 {
		t.Errorf("Rx=%f Ry=%f; want 1.0, 1.0", c.Rx, c.Ry)
	}
}

func TestParseHeaderBadPrecision(t *testing.T) {
	// Precision field "E" with non-numeric value.
	header := "RX010000100000124BY 100GP 900x 900PR E abc\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error for bad precision")
	}
	if !strings.Contains(err.Error(), "precision") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderPositivePrecision(t *testing.T) {
	// PR E+02 → precision = 2, precisionMult = 100.
	header := "RX010000100000124BY 100GP 900x 900PR E+02\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.precision != 2 {
		t.Errorf("precision = %d; want 2", c.precision)
	}
	if c.precisionMult != 100.0 {
		t.Errorf("precisionMult = %v; want 100", c.precisionMult)
	}
}

func TestParseHeaderNoPrecision(t *testing.T) {
	// No PR/E field → precision stays 0, precisionMult = 1.0.
	header := "RX010000100000124BY 100GP 900x 900\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.precision != 0 {
		t.Errorf("precision = %d; want 0", c.precision)
	}
	if c.precisionMult != 1.0 {
		t.Errorf("precisionMult = %v; want 1.0", c.precisionMult)
	}
}

func TestParseHeaderLevelTooShort(t *testing.T) {
	// LV field with only 1-char value → len(lv) < 2 triggers "too short".
	// "LVx" where x is a single non-uppercase char before the next uppercase key.
	header := "PG010000100000124BY 100BG460460LVxCS0\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error for too-short LV")
	}
	if !strings.Contains(err.Error(), "level field too short") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderLevelBadCount(t *testing.T) {
	// LV count is not numeric.
	header := "PG010000100000124BY 100BG460460LV xx  1.0\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error for bad level count")
	}
	if !strings.Contains(err.Error(), "level count") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderLevelBadFormat(t *testing.T) {
	// LV count says 6 but string is too short.
	header := "PG010000100000124BY 100BG460460LV 6  1.0\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error for invalid level format")
	}
	if !strings.Contains(err.Error(), "invalid level format") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderLevelBadValue(t *testing.T) {
	// LV with correct count/length but non-numeric value.
	header := "PG010000100000124BY 100BG460460LV 1 xxxx\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error for invalid level value")
	}
	if !strings.Contains(err.Error(), "invalid level value") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderBadVS(t *testing.T) {
	// VS field with non-numeric value.
	header := "RX010000100000124BY 100GP 900x 900VS abc\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err == nil {
		t.Fatal("expected error for bad VS")
	}
	if !strings.Contains(err.Error(), "format value") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHeaderWeeklyInterval(t *testing.T) {
	// W1-W4 products multiply the interval by 10.
	for _, product := range []string{"W1", "W2", "W3", "W4"} {
		header := product + "010000100000124BY 100GP 900x 900INT  60\x03"
		c := &Composite{}
		err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", product, err)
		}
		want := 600 * time.Minute // 60 * 10
		if c.Interval != want {
			t.Errorf("%s: Interval = %v; want %v", product, c.Interval, want)
		}
	}
}

func TestParseHeader12LevelNegativeValues(t *testing.T) {
	// Test a 12-level header with negative values (e.g. Doppler velocity classes).
	header := "PR010000100000124BY 100BG460460" +
		"LV12-31.5-24.5-17.5-10.5 -5.5 -1.0  1.0  5.5 10.5 17.5 24.5 31.5\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []float32{-31.5, -24.5, -17.5, -10.5, -5.5, -1.0, 1.0, 5.5, 10.5, 17.5, 24.5, 31.5}
	if len(c.level) != len(want) {
		t.Fatalf("level length = %d; want %d", len(c.level), len(want))
	}
	for i := range want {
		if c.level[i] != want[i] {
			t.Errorf("level[%d] = %v; want %v", i, c.level[i], want[i])
		}
	}
}

func TestParseHeaderUnknownUnit(t *testing.T) {
	// RV is not in unitCatalog → DataUnit should be Unit_unknown.
	header := "RV010000100000124BY 100GP 900x 900\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.DataUnit != Unit_unknown {
		t.Errorf("DataUnit = %v; want Unit_unknown", c.DataUnit)
	}
}

func TestParseHeaderKnownUnit(t *testing.T) {
	// RW is in unitCatalog → DataUnit should be Unit_mm.
	header := "RW010000100000124BY 100GP 900x 900\x03"
	c := &Composite{}
	err := c.parseHeader(bufio.NewReader(strings.NewReader(header)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.DataUnit != Unit_mm {
		t.Errorf("DataUnit = %v; want Unit_mm", c.DataUnit)
	}
}

// --- splitHeader tests ---

func TestSplitHeaderEmpty(t *testing.T) {
	// Empty string: the final assignment runs with zero-length slices,
	// producing map["":""] — this is fine since parseHeader never calls
	// splitHeader with empty input.
	m := splitHeader("")
	if len(m) != 1 {
		t.Errorf("expected 1 entry for empty header, got %v", m)
	}
}

func TestSplitHeaderNoKey(t *testing.T) {
	// Input starts with non-uppercase → returns empty map.
	m := splitHeader("123ABC")
	if len(m) != 0 {
		t.Errorf("expected empty map when no key prefixes value, got %v", m)
	}
}

func TestSplitHeaderSimple(t *testing.T) {
	m := splitHeader("BY 100GP 900x 900")
	if m["BY"] != " 100" {
		t.Errorf("BY = %q; want %q", m["BY"], " 100")
	}
	if m["GP"] != " 900x 900" {
		t.Errorf("GP = %q; want %q", m["GP"], " 900x 900")
	}
}
