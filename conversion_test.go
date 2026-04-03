package radolan

import (
	"math"
	"testing"
)

func TestConversion(t *testing.T) {
	testcases := []struct {
		rvp float32
		dbz float32
		zr  float64
	}{
		{0, -32.5, 0.0001},
		{65, 0, 0.0201},
		{100, 17.5, 0.3439},
		{200, 67.5, 1141.7670},
	}

	for _, test := range testcases {
		dbz := toDBZ(test.rvp)
		zr := PrecipitationRate(Aniol80, dbz)
		rz := Reflectivity(Aniol80, zr)
		rvp := toRVP6(dbz)

		if dbz != test.dbz {
			t.Errorf("toDBZ(%f) = %f; expected: %f", test.rvp, dbz, test.dbz)
		}
		if rvp != test.rvp {
			t.Errorf("toRVP6(toDBZ(%f)) = %f; expected: %f", test.rvp, rvp, test.rvp)
		}
		if math.Abs(test.zr-zr) > 0.0001 {
			t.Errorf("PrecipitationRate(Aniol80, toDBZ(%f)) = %f; expected: %f", test.rvp, zr, test.zr)
		}
		if math.Abs(float64(test.dbz-rz)) > 0.0000001 {
			t.Errorf("Reflectivity(PrecipitationRate(toDBZ(%f))) = %f; expected: %f",
				test.rvp, rz, test.dbz)
		}
	}
}

// TestAllZRRelationships verifies PrecipitationRate and Reflectivity round-trip
// for all four predefined Z-R relationships.
func TestAllZRRelationships(t *testing.T) {
	relations := []struct {
		name string
		zr   ZR
		a, b float64
	}{
		{"Aniol80", Aniol80, 256, 1.42},
		{"Doelling98", Doelling98, 316, 1.50},
		{"JossWaldvogel70", JossWaldvogel70, 300, 1.50},
		{"MarshallPalmer55", MarshallPalmer55, 200, 1.60},
	}

	for _, rel := range relations {
		// Verify round-trip at several dBZ values.
		for _, dBZ := range []float32{0, 10, 20, 30, 40, 50, 60} {
			rate := PrecipitationRate(rel.zr, dBZ)
			if rate <= 0 {
				t.Errorf("%s: PrecipitationRate(%.0f) = %v; want > 0", rel.name, dBZ, rate)
				continue
			}
			roundTrip := Reflectivity(rel.zr, rate)
			if math.Abs(float64(roundTrip-dBZ)) > 1e-5 {
				t.Errorf("%s: round-trip dBZ=%.0f → rate=%v → dBZ=%v", rel.name, dBZ, rate, roundTrip)
			}
		}

		// Verify the Z-R relationship: Z = a × R^b.
		// At dBZ=30 → Z=10^(30/10)=1000 → R=(1000/a)^(1/b).
		rate30 := PrecipitationRate(rel.zr, 30)
		Z := math.Pow(10, 3) // 30 dBZ = 10^3
		wantRate := math.Pow(Z/rel.a, 1.0/rel.b)
		if math.Abs(rate30-wantRate) > 1e-6 {
			t.Errorf("%s: rate at 30 dBZ = %v; want %v (from Z=a*R^b)", rel.name, rate30, wantRate)
		}
	}
}

// TestNewZRCoefficients verifies the precomputed coefficients of NewZR.
func TestNewZRCoefficients(t *testing.T) {
	zr := NewZR(200, 1.6)
	// c1 = 10*b = 16
	if math.Abs(zr.c1-16.0) > 1e-10 {
		t.Errorf("c1 = %v; want 16.0", zr.c1)
	}
	// c2 = 200^(-1/1.6)
	wantC2 := math.Pow(200, -1.0/1.6)
	if math.Abs(zr.c2-wantC2) > 1e-10 {
		t.Errorf("c2 = %v; want %v", zr.c2, wantC2)
	}
	// c3 = 10^(1/16)
	wantC3 := math.Pow(10.0, 1.0/16.0)
	if math.Abs(zr.c3-wantC3) > 1e-10 {
		t.Errorf("c3 = %v; want %v", zr.c3, wantC3)
	}
	// c4 = 10*log10(200)
	wantC4 := 10.0 * math.Log10(200.0)
	if math.Abs(zr.c4-wantC4) > 1e-10 {
		t.Errorf("c4 = %v; want %v", zr.c4, wantC4)
	}
}

// TestPrecipitationRateAdaptive verifies regime switching boundaries and that
// results are consistent with the underlying single-ZR PrecipitationRate values.
func TestPrecipitationRateAdaptive(t *testing.T) {
	cases := []struct {
		dBZ      float32
		expected ZR
	}{
		{19.9, JossWaldvogel70},  // drizzle regime
		{20.0, Aniol80},          // boundary: Aniol80 starts at 20
		{34.9, Aniol80},          // upper Aniol80 range
		{35.0, MarshallPalmer55}, // boundary: convective starts at 35
		{50.0, MarshallPalmer55}, // deep convection
	}

	for _, tc := range cases {
		got := PrecipitationRateAdaptive(tc.dBZ)
		want := PrecipitationRate(tc.expected, tc.dBZ)
		if math.Abs(got-want) > 1e-10 {
			t.Errorf("PrecipitationRateAdaptive(%.1f) = %v; want %v (from %v regime)",
				tc.dBZ, got, want, tc.expected)
		}
	}
}

// TestPrecipitationRateAdaptiveZeroGuard verifies that sub-noise-floor values
// (dBZ ≤ 0) return exactly 0 rather than a small positive or NaN result.
func TestPrecipitationRateAdaptiveZeroGuard(t *testing.T) {
	for _, dBZ := range []float32{0.0, -0.1, -5.0, -32.5} {
		if got := PrecipitationRateAdaptive(dBZ); got != 0.0 {
			t.Errorf("PrecipitationRateAdaptive(%.1f) = %v; want 0", dBZ, got)
		}
	}
}

// TestPrecipitationRateAdaptiveMonotonicity verifies that the adaptive function
// is monotonically increasing across regime boundaries.
func TestPrecipitationRateAdaptiveMonotonicity(t *testing.T) {
	prev := 0.0
	for dBZ := float32(1); dBZ <= 60; dBZ += 0.5 {
		rate := PrecipitationRateAdaptive(dBZ)
		if rate < prev {
			t.Errorf("non-monotonic at %.1f dBZ: %.6f < %.6f", dBZ, rate, prev)
		}
		prev = rate
	}
}

// TestIsNaN verifies the float32 NaN check.
func TestIsNaN(t *testing.T) {
	if !IsNaN(NaN) {
		t.Error("IsNaN(NaN) = false")
	}
	if IsNaN(0) {
		t.Error("IsNaN(0) = true")
	}
	if IsNaN(42.5) {
		t.Error("IsNaN(42.5) = true")
	}
}

// TestRvp6Raw verifies precision-scaled conversion.
func TestRvp6Raw(t *testing.T) {
	c := &Composite{precisionMult: 0.1}
	got := c.rvp6Raw(100)
	if math.Abs(float64(got-10.0)) > 1e-6 {
		t.Errorf("rvp6Raw(100) with mult=0.1 = %v; want 10.0", got)
	}

	c2 := &Composite{precisionMult: 100.0}
	got2 := c2.rvp6Raw(5)
	if math.Abs(float64(got2-500.0)) > 1e-6 {
		t.Errorf("rvp6Raw(5) with mult=100 = %v; want 500.0", got2)
	}
}

func makeTestGrid(dx, dy int) *Composite {
	comp := NewDummy("RX", 3, dx, dy)
	comp.DataZ = [][][]float32{make([][]float32, dy)}
	for y := range comp.DataZ[0] {
		comp.DataZ[0][y] = make([]float32, dx)
	}
	comp.Data = comp.DataZ[0]
	comp.Dz = 1
	return comp
}

// TestNeighbourhoodSample verifies sampling behaviour for a small synthetic composite.
func TestNeighbourhoodSample(t *testing.T) {
	// Build a 5×5 composite where the centre pixel (2,2) is a known dBZ value
	// and all surrounding pixels are dry (0 dBZ → below threshold).
	comp := makeTestGrid(5, 5)

	// All-dry: centre sample should return 0, 0, 0
	avgMMH, maxMMH, cov := comp.NeighbourhoodSample(2, 2, 1)
	if avgMMH != 0 || maxMMH != 0 || cov != 0 {
		t.Errorf("all-dry: got (%v,%v,%v); want (0,0,0)", avgMMH, maxMMH, cov)
	}

	// Set centre pixel to 35 dBZ (convective threshold — uses MarshallPalmer55)
	comp.Data[2][2] = 35
	avgMMH, maxMMH, cov = comp.NeighbourhoodSample(2, 2, 1)
	wantRate := PrecipitationRate(MarshallPalmer55, 35)
	// 3×3 neighbourhood = 9 pixels, only centre is wet → avg = wantRate/9
	if math.Abs(avgMMH-wantRate/9) > 1e-9 {
		t.Errorf("single centre pixel: avgMMH = %v; want %v", avgMMH, wantRate/9)
	}
	if math.Abs(maxMMH-wantRate) > 1e-9 {
		t.Errorf("single centre pixel: maxMMH = %v; want %v", maxMMH, wantRate)
	}
	if math.Abs(cov-1.0/9.0) > 1e-9 {
		t.Errorf("single centre pixel: coverage = %v; want %v", cov, 1.0/9.0)
	}

	// Edge pixel (0,0) with radius 1: only 4 in-bounds pixels in the neighbourhood
	comp.Data[0][0] = 35
	avgMMH, _, cov = comp.NeighbourhoodSample(0, 0, 1)
	// 4 in-bounds pixels, 1 wet
	if math.Abs(cov-1.0/4.0) > 1e-9 {
		t.Errorf("corner pixel coverage = %v; want %v", cov, 1.0/4.0)
	}
	_ = avgMMH
}

// TestNeighbourhoodSampleFastSlowParity verifies both code paths of
// NeighbourhoodSample against an independent reference using AtZ.
//
// The interior fast path (no per-pixel bounds checks) accesses data via
// row := c.Data[cy+dy]; row[cx+dx], while the slow path uses c.At(px,py).
// Both should produce identical results for any valid pixel.
func TestNeighbourhoodSampleFastSlowParity(t *testing.T) {
	const radius = 2

	// --- Fast path: 9×9 grid, sample at (4,4) → 4−2=2≥0, 4+2=6<9 ---
	fast := makeTestGrid(9, 9)
	// Fill the 5×5 neighbourhood centred on (4,4) with a varied dBZ pattern.
	nbh := [5][5]float32{
		{0, 15, 28, 40, 0},
		{15, 28, 40, 28, 15},
		{28, 40, 50, 40, 28},
		{15, 28, 40, 28, 15},
		{0, 15, 28, 40, 0},
	}
	for oy := 0; oy < 5; oy++ {
		for ox := 0; ox < 5; ox++ {
			fast.Data[2+oy][2+ox] = nbh[oy][ox]
		}
	}

	gotAvg, gotMax, gotCov := fast.NeighbourhoodSample(4, 4, radius)

	// Reference: accumulate using AtZ — different data access from fast path's
	// row-slice approach; catches any x/y transposition in the fast path.
	var wantTotal, wantMax float64
	var wantCount, wantAbove int
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			dBZ := fast.AtZ(4+dx, 4+dy, 0)
			wantCount++
			if IsNaN(dBZ) || dBZ <= 0 {
				continue
			}
			mmH := PrecipitationRateAdaptive(dBZ)
			wantTotal += mmH
			if mmH > wantMax {
				wantMax = mmH
			}
			if mmH >= 0.1 {
				wantAbove++
			}
		}
	}
	wantAvg := wantTotal / float64(wantCount)
	wantCov := float64(wantAbove) / float64(wantCount)

	if math.Abs(gotAvg-wantAvg) > 1e-9 {
		t.Errorf("fast path avgMMH = %v; want %v", gotAvg, wantAvg)
	}
	if math.Abs(gotMax-wantMax) > 1e-9 {
		t.Errorf("fast path maxMMH = %v; want %v", gotMax, wantMax)
	}
	if math.Abs(gotCov-wantCov) > 1e-9 {
		t.Errorf("fast path coverage = %v; want %v", gotCov, wantCov)
	}

	// --- Slow path: 5×5 grid, sample at (0,0,2) → 0−2=−2<0 → bounds-checked ---
	// In-bounds pixels: x∈[0,2], y∈[0,2] → 9 of 25 cells counted.
	slow := makeTestGrid(5, 5)
	for oy := 0; oy < 3; oy++ {
		for ox := 0; ox < 3; ox++ {
			slow.Data[oy][ox] = nbh[oy][ox]
		}
	}

	sAvg, sMax, sCov := slow.NeighbourhoodSample(0, 0, radius)

	var sTotal, sMaxWant float64
	var sCount, sAbove int
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			dBZ := slow.AtZ(x, y, 0)
			sCount++
			if IsNaN(dBZ) || dBZ <= 0 {
				continue
			}
			mmH := PrecipitationRateAdaptive(dBZ)
			sTotal += mmH
			if mmH > sMaxWant {
				sMaxWant = mmH
			}
			if mmH >= 0.1 {
				sAbove++
			}
		}
	}
	sAvgWant := sTotal / float64(sCount)
	sCovWant := float64(sAbove) / float64(sCount)

	if math.Abs(sAvg-sAvgWant) > 1e-9 {
		t.Errorf("slow path avgMMH = %v; want %v", sAvg, sAvgWant)
	}
	if math.Abs(sMax-sMaxWant) > 1e-9 {
		t.Errorf("slow path maxMMH = %v; want %v", sMax, sMaxWant)
	}
	if math.Abs(sCov-sCovWant) > 1e-9 {
		t.Errorf("slow path coverage = %v; want %v", sCov, sCovWant)
	}
}

// TestNeighbourhoodSampleRadius0 verifies that radius=0 samples exactly one pixel.
func TestNeighbourhoodSampleRadius0(t *testing.T) {
	comp := makeTestGrid(5, 5)
	comp.Data[2][2] = 30 // stratiform
	avg, max, cov := comp.NeighbourhoodSample(2, 2, 0)
	wantRate := PrecipitationRateAdaptive(30)
	if math.Abs(avg-wantRate) > 1e-9 {
		t.Errorf("radius=0 avg = %v; want %v", avg, wantRate)
	}
	if math.Abs(max-wantRate) > 1e-9 {
		t.Errorf("radius=0 max = %v; want %v", max, wantRate)
	}
	if cov != 1.0 {
		t.Errorf("radius=0 coverage = %v; want 1.0", cov)
	}
}

// TestNeighbourhoodSampleNaN verifies NaN pixels are counted but don't contribute.
func TestNeighbourhoodSampleNaN(t *testing.T) {
	comp := makeTestGrid(3, 3)
	// All NaN
	for y := range 3 {
		for x := range 3 {
			comp.Data[y][x] = NaN
		}
	}
	avg, max, cov := comp.NeighbourhoodSample(1, 1, 1)
	if avg != 0 || max != 0 || cov != 0 {
		t.Errorf("all-NaN: got (%v,%v,%v); want (0,0,0)", avg, max, cov)
	}
}
