package radolan

import "testing"

// BenchmarkNeighbourhoodSampleInterior measures the interior fast path (no bounds checks).
func BenchmarkNeighbourhoodSampleInterior(b *testing.B) {
	comp := makeBenchComposite(900, 900)
	b.ResetTimer()
	for b.Loop() {
		comp.NeighbourhoodSample(450, 450, 2)
	}
}

// BenchmarkNeighbourhoodSampleEdge measures the edge slow path (bounds checks active).
func BenchmarkNeighbourhoodSampleEdge(b *testing.B) {
	comp := makeBenchComposite(900, 900)
	b.ResetTimer()
	for b.Loop() {
		comp.NeighbourhoodSample(0, 0, 2)
	}
}

// BenchmarkNeighbourhoodSampleFullGrid measures cost of sampling all 810k pixels —
// the pre-computation workload in the server's grid processing step.
func BenchmarkNeighbourhoodSampleFullGrid(b *testing.B) {
	comp := makeBenchComposite(900, 900)
	b.ResetTimer()
	for b.Loop() {
		for y := 0; y < comp.Dy; y++ {
			for x := 0; x < comp.Dx; x++ {
				comp.NeighbourhoodSample(x, y, 2)
			}
		}
	}
}

// BenchmarkPrecipitationRateAdaptive measures Z-R conversion throughput.
func BenchmarkPrecipitationRateAdaptive(b *testing.B) {
	for b.Loop() {
		PrecipitationRateAdaptive(35.0)
	}
}

func makeBenchComposite(dx, dy int) *Composite {
	comp := &Composite{Dx: dx, Dy: dy, Dz: 1}
	flat := make([]float32, dx*dy)
	// Fill with a realistic dBZ pattern: mix of dry, stratiform, convective.
	for i := range flat {
		switch i % 5 {
		case 0:
			flat[i] = 0 // dry
		case 1:
			flat[i] = 15 // drizzle
		case 2:
			flat[i] = 28 // stratiform
		case 3:
			flat[i] = 40 // convective
		case 4:
			flat[i] = NaN // no-data
		}
	}
	rows := make([][]float32, dy)
	for y := range rows {
		rows[y] = flat[y*dx : (y+1)*dx]
	}
	comp.DataZ = [][][]float32{rows}
	comp.Data = rows
	return comp
}
