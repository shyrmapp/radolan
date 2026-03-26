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

// TestNeighbourhoodSample verifies sampling behaviour for a small synthetic composite.
func TestNeighbourhoodSample(t *testing.T) {
	// Build a 5×5 composite where the centre pixel (2,2) is a known dBZ value
	// and all surrounding pixels are dry (0 dBZ → below threshold).
	comp := NewDummy("RX", 3, 5, 5)
	comp.DataZ = [][][]float32{make([][]float32, 5)}
	for y := range comp.DataZ[0] {
		comp.DataZ[0][y] = make([]float32, 5)
	}
	comp.Data = comp.DataZ[0]
	comp.Dz = 1

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
