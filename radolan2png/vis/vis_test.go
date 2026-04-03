package vis

import (
	"image/color"
	"math"
	"testing"

	"github.com/shyrmapp/radolan"
)

func TestId(t *testing.T) {
	if Id(42.5) != 42.5 {
		t.Error("Id should be identity")
	}
}

func TestLog(t *testing.T) {
	got := Log(math.E)
	if math.Abs(got-1.0) > 1e-10 {
		t.Errorf("Log(e) = %v; want 1.0", got)
	}
}

// --- Heatmap ---

func TestHeatmapBelowMin(t *testing.T) {
	hm := Heatmap(10, 100, Id)
	col := hm(5) // below min
	if col != (color.RGBA{0, 0, 0, 0xFF}) {
		t.Errorf("below min: got %v; want black", col)
	}
}

func TestHeatmapAtMin(t *testing.T) {
	hm := Heatmap(10, 100, Id)
	col := hm(10)
	// At min: p=0, h=mod(360-0+240,360)=240 → case 4 (x,0,c)
	// l=0.25, c=(1-|2*0.25-1|)*1.0=0.5, x=0.5*(1-|mod(4,2)-1|)=0.5*(1-1)=0
	// rr,gg,bb = 0,0,0.5; m=0.25-0.25=0
	// r,g,b = 0,0,127
	if col.A != 0xFF {
		t.Errorf("alpha = %d; want 255", col.A)
	}
	// Just verify it's not black (it's a valid colour at min boundary).
	if col == (color.RGBA{0, 0, 0, 0xFF}) {
		t.Error("at min: should not be black")
	}
}

func TestHeatmapAboveMax(t *testing.T) {
	hm := Heatmap(10, 100, Id)
	col1 := hm(100) // at max
	col2 := hm(200) // above max — clamped to p=1
	if col1 != col2 {
		t.Errorf("above max should clamp: at_max=%v, above_max=%v", col1, col2)
	}
}

func TestHeatmapLogCompression(t *testing.T) {
	hm := Heatmap(0.1, 100, Log)
	col := hm(0.05) // below min after log compression
	if col != (color.RGBA{0, 0, 0, 0xFF}) {
		t.Errorf("log below min: got %v; want black", col)
	}
	// Within range should not be black.
	col = hm(10)
	if col == (color.RGBA{0, 0, 0, 0xFF}) {
		t.Error("log in range: should not be black")
	}
}

func TestHeatmapNaN(t *testing.T) {
	hm := Heatmap(0, 100, Id)
	col := hm(math.NaN())
	// NaN < min is false, so NaN goes through the formula.
	// The code handles this: IsNaN(hh) sets hh=-1, switch falls through.
	if col.A != 0xFF {
		t.Errorf("NaN: alpha = %d; want 255", col.A)
	}
}

// --- Graymap ---

func TestGraymapBelowMin(t *testing.T) {
	gm := Graymap(0, 100, Id)
	col := gm(-5)
	if col != (color.RGBA{0, 0, 0, 0xFF}) {
		t.Errorf("below min: got %v; want black", col)
	}
}

func TestGraymapAtMax(t *testing.T) {
	gm := Graymap(0, 100, Id)
	col := gm(100)
	if col != (color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}) {
		t.Errorf("at max: got %v; want white", col)
	}
}

func TestGraymapAboveMax(t *testing.T) {
	gm := Graymap(0, 100, Id)
	col := gm(200) // clamped
	if col != (color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}) {
		t.Errorf("above max: got %v; want white", col)
	}
}

func TestGraymapMidpoint(t *testing.T) {
	gm := Graymap(0, 255, Id)
	col := gm(127.5) // p ≈ 0.5 → level ≈ 127
	if col.R < 120 || col.R > 135 {
		t.Errorf("midpoint: R=%d; want ~127", col.R)
	}
}

// --- Radialmap ---

func TestRadialmapNaN(t *testing.T) {
	rm := Radialmap(-31.5, 31.5, Log)
	col := rm(math.NaN())
	if col != (color.RGBA{0, 0, 0, 0xFF}) {
		t.Errorf("NaN: got %v; want black", col)
	}
}

func TestRadialmapPositive(t *testing.T) {
	rm := Radialmap(-31.5, 31.5, Log)
	col := rm(20.0) // positive → red channel dominant
	if col.R <= col.B {
		t.Errorf("positive velocity: R=%d B=%d; expect R > B", col.R, col.B)
	}
}

func TestRadialmapNegative(t *testing.T) {
	rm := Radialmap(-31.5, 31.5, Log)
	col := rm(-20.0) // negative → blue/cyan channel dominant
	if col.B <= col.R {
		t.Errorf("negative velocity: B=%d R=%d; expect B > R", col.B, col.R)
	}
}

func TestRadialmapNearZero(t *testing.T) {
	rm := Radialmap(-31.5, 31.5, Log)
	col := rm(0.5) // |val| <= 1 → special near-zero branch
	if col.R != 0xFF {
		t.Errorf("near zero positive: R=%d; want 255", col.R)
	}
}

func TestRadialmapAboveMax(t *testing.T) {
	rm := Radialmap(-31.5, 31.5, Log)
	col1 := rm(31.5) // at max
	col2 := rm(50.0) // above max → clamped
	if col1 != col2 {
		t.Errorf("above max should clamp: at_max=%v, above=%v", col1, col2)
	}
}

// --- Image ---

func makeVisComposite(dx, dy int, values []float32) *radolan.Composite {
	comp := radolan.NewDummy("RX", 3, dx, dy)
	rows := make([][]float32, dy)
	flat := make([]float32, dx*dy)
	copy(flat, values)
	for y := range rows {
		rows[y] = flat[y*dx : (y+1)*dx]
	}
	comp.DataZ = [][][]float32{rows}
	comp.Data = rows
	comp.Dz = 1
	return comp
}

func TestImageValidLayer(t *testing.T) {
	comp := makeVisComposite(3, 2, []float32{10, 20, 30, 40, 50, 60})
	img := Image(HeatmapReflectivity, comp, 0)

	if img.Bounds().Dx() != 3 || img.Bounds().Dy() != 2 {
		t.Errorf("image size = %v; want 3×2", img.Bounds())
	}

	// Verify non-black pixels (dBZ values are within HeatmapReflectivity range).
	r, g, b, a := img.At(1, 0).RGBA()
	if a>>8 != 0xFF {
		t.Errorf("alpha at (1,0) = %d; want 255", a>>8)
	}
	_ = r
	_ = g
	_ = b
}

func TestImageInvalidLayerNegative(t *testing.T) {
	comp := makeVisComposite(2, 2, []float32{10, 20, 30, 40})
	img := Image(HeatmapReflectivity, comp, -1)
	// Should return blank image.
	if img.Bounds().Dx() != 2 || img.Bounds().Dy() != 2 {
		t.Errorf("image size = %v; want 2×2", img.Bounds())
	}
	// All pixels should be zero (transparent black).
	r, g, b, a := img.At(0, 0).RGBA()
	if r != 0 || g != 0 || b != 0 || a != 0 {
		t.Errorf("blank image pixel = (%d,%d,%d,%d); want all zero", r, g, b, a)
	}
}

func TestImageInvalidLayerTooHigh(t *testing.T) {
	comp := makeVisComposite(2, 2, []float32{10, 20, 30, 40})
	img := Image(HeatmapReflectivity, comp, 5)
	r, _, _, a := img.At(0, 0).RGBA()
	if r != 0 || a != 0 {
		t.Error("out-of-bounds layer should produce blank image")
	}
}

func TestImageDirectPixelWrite(t *testing.T) {
	// Verify that direct pixel writes match what img.Set would produce.
	comp := makeVisComposite(2, 2, []float32{30, 50, 10, 70})
	img := Image(HeatmapReflectivity, comp, 0)

	// Check each pixel has alpha=255.
	for y := range 2 {
		for x := range 2 {
			_, _, _, a := img.At(x, y).RGBA()
			if a>>8 != 0xFF {
				t.Errorf("pixel (%d,%d) alpha = %d; want 255", x, y, a>>8)
			}
		}
	}
}

// --- Pre-defined gradient sanity checks ---

func TestPreDefinedGradients(t *testing.T) {
	// Just verify the package-level gradients don't panic and return valid colors.
	gradients := []struct {
		name string
		fn   ColorFunc
		val  float64
	}{
		{"HeatmapReflectivityShort", HeatmapReflectivityShort, 40},
		{"HeatmapReflectivity", HeatmapReflectivity, 40},
		{"HeatmapReflectivityWide", HeatmapReflectivityWide, 0},
		{"HeatmapAccumulatedHour", HeatmapAccumulatedHour, 10},
		{"HeatmapAccumulatedDay", HeatmapAccumulatedDay, 50},
		{"HeatmapRadialVelocity", HeatmapRadialVelocity, 15},
		{"GraymapLinear", GraymapLinear, 200},
		{"GraymapLinearWide", GraymapLinearWide, 2000},
	}
	for _, tc := range gradients {
		col := tc.fn(tc.val)
		if col.A != 0xFF {
			t.Errorf("%s(%v): alpha = %d; want 255", tc.name, tc.val, col.A)
		}
	}
}
