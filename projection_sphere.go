package radolan

import (
	"math"
)

// values described in [1]
const (
	earthRadius = 6370.04 // km

	junctionNorth = 60.0 // N
	junctionEast  = 10.0 // E
)

func (c *Composite) projectSphere(north, east float64) (x, y float64) {
	lambda0 := junctionEast * degToRad
	phi0 := junctionNorth * degToRad
	lambda := east * degToRad
	phi := north * degToRad

	m := (1.0 + math.Sin(phi0)) / (1.0 + math.Sin(phi))
	x = earthRadius * m * math.Cos(phi) * math.Sin(lambda-lambda0)
	y = earthRadius * m * math.Cos(phi) * math.Cos(lambda-lambda0)

	// offset correction
	x -= c.offx
	y -= c.offy

	// scaling
	x /= c.Rx
	y /= c.Ry

	return
}
