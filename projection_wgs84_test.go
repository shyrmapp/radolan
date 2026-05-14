package radolan

import (
	"testing"
)

func Test_DE1200_WGS84(t *testing.T) {
	DE1200Grid := [][]float64{
		[]float64{55.86208711, 1.463301510, 0, 0},       // NW
		[]float64{55.84543856, 18.73161645, 1100, 0},    // NE
		[]float64{45.68460578, 16.58086935, 1100, 1200}, // SE
		[]float64{45.69642538, 3.566994635, 0, 1200},    // SW
	}

	comp := NewDummy("WN", 5, 1100, 1200)

	for _, v := range DE1200Grid {
		rx, ry := comp.Project(v[0], v[1])
		ex, ey := v[2], v[3]

		if dist(rx, ry, ex, ey) > 0.000001 {
			t.Errorf("comp.Project(%#v, %#v) = (%#v, %#v); expected: (%#v, %#v)", v[0], v[1], rx, ry, ex, ey)
		}
	}
}

// TestProjectionFunc verifies that ProjectionFunc returns a closure that produces
// identical results to c.Project for both DE1200 (WGS84) and national (sphere) grids.
func TestProjectionFunc(t *testing.T) {
	cases := []struct {
		name   string
		format int
		dx, dy int
		points [][]float64 // {north, east}
	}{
		{
			name:   "DE1200 WGS84 (Format 5)",
			format: 5, dx: 1100, dy: 1200,
			points: [][]float64{
				{55.86208711, 1.463301510}, // NW corner
				{55.84543856, 18.73161645}, // NE corner
				{45.68460578, 16.58086935}, // SE corner
				{45.69642538, 3.566994635}, // SW corner
				{48.8975, 9.1919},          // Ludwigsburg
				{52.5200, 13.4050},         // Berlin
				{48.1351, 11.5820},         // Munich
			},
		},
		{
			name:   "National grid sphere (Format 5)",
			format: 5, dx: 900, dy: 900,
			points: [][]float64{
				{54.5877, 2.0715},  // NW corner
				{47.0705, 14.6209}, // SE corner
				{48.8975, 9.1919},  // Ludwigsburg
				{52.5200, 13.4050}, // Berlin
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			comp := NewDummy("WN", tc.format, tc.dx, tc.dy)
			fn := comp.ProjectionFunc()
			if fn == nil {
				t.Fatal("ProjectionFunc returned nil")
			}
			for _, pt := range tc.points {
				north, east := pt[0], pt[1]
				cx, cy := comp.Project(north, east)
				fx, fy := fn(north, east)
				if dist(cx, cy, fx, fy) > 1e-9 {
					t.Errorf("point (%.6f, %.6f): Project=(%v,%v) ProjectionFunc=(%v,%v)",
						north, east, cx, cy, fx, fy)
				}
			}
		})
	}
}
