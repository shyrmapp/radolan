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
