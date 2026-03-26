package radolan

import (
	"math"
)

var NaN = float32(math.NaN())

func IsNaN(f float32) (is bool) {
	return f != f
}

// Z-R relationship
type ZR struct {
	// intermediate caching
	c1 float64 // 10*b
	c2 float64 // a^(-1/b)
	c3 float64 // 10^(1/(10*b))
	c4 float64 // 10 * log10(a)
}

// Common Z-R relationships
var (
	Aniol80 = NewZR(256, 1.42) // DWD operational Germany, described in [5]

	// Doelling98 is the MeteoSwiss operational Z-R relationship.
	Doelling98 = NewZR(316, 1.50)

	// JossWaldvogel70 is a European stratiform/drizzle Z-R relationship (a=300, b=1.50)
	// commonly cited in German meteorological literature. Note: the original Joss &
	// Waldvogel (1970) paper proposed Z=25R^1.71 for drizzle specifically; this variant
	// (a=300, b=1.50) reflects a widely-used regional adaptation.
	JossWaldvogel70 = NewZR(300, 1.50)

	MarshallPalmer55 = NewZR(200, 1.60) // operational use in austria
)

// New Z-R returns a Z-R relationship mathematically expressed as Z = a * R^b
func NewZR(A, B float64) ZR {
	c1 := 10.0 * B
	c2 := math.Pow(A, -1.0/B)
	c3 := math.Pow(10.0, 1/c1)
	c4 := 10.0 * math.Log10(A)

	return ZR{c1, c2, c3, c4}
}

// PrecipitationRate returns the estimated precipitation rate in mm/h for the given
// reflectivity factor and Z-R relationship.
func PrecipitationRate(relation ZR, dBZ float32) (rate float64) {
	return relation.c2 * math.Pow(relation.c3, float64(dBZ))
}

// PrecipitationRateAdaptive returns the estimated precipitation rate (mm/h) using
// a regime-adaptive Z-R relationship selected by echo intensity:
//   - dBZ ≤ 0:  sub-noise floor → 0 mm/h
//   - dBZ < 20: drizzle/light stratiform → JossWaldvogel70 (a=300, b=1.50)
//   - dBZ < 35: operational stratiform → Aniol80 (a=256, b=1.42)
//   - dBZ ≥ 35: convective/hail-contaminated → MarshallPalmer55 (a=200, b=1.60)
//
// At convective thresholds (dBZ ≥ 35), Aniol80 is known to overestimate QPE
// relative to rain-gauge networks in German summer convection due to hail
// contamination and partial beam blockage (Krämer et al. 2008, HESS). The switch
// to MarshallPalmer55 applies a more conservative estimate in this regime.
// JossWaldvogel70 reduces the overestimate Aniol80 produces for drizzle DSDs.
func PrecipitationRateAdaptive(dBZ float32) float64 {
	if dBZ <= 0 {
		return 0
	}
	switch {
	case dBZ >= 35:
		return PrecipitationRate(MarshallPalmer55, dBZ)
	case dBZ < 20:
		return PrecipitationRate(JossWaldvogel70, dBZ)
	default:
		return PrecipitationRate(Aniol80, dBZ)
	}
}

// Reflectivity returns the estimated reflectivity factor for the given precipitation
// rate (mm/h) and Z-R relationship.
func Reflectivity(relation ZR, rate float64) (dBZ float32) {
	return float32(relation.c4 + relation.c1*math.Log10(rate))
}

// toDBZ converts the given radar video processor values (rvp-6) to radar reflectivity
// factors in decibel relative to Z (dBZ).
func toDBZ(rvp6 float32) (dBZ float32) {
	return rvp6/2.0 - 32.5
}

// toRVP6 converts the given radar reflectivity factors (dBZ) to radar video processor
// values (rvp-6).
func toRVP6(dBZ float32) float32 {
	return (dBZ + 32.5) * 2
}

// rvp6Raw converts the raw value to radar video processor values (rvp-6) by applying the
// products precision field.
func (c *Composite) rvp6Raw(value int) float32 {
	return float32(value) * float32(math.Pow10(c.precision))
}
