# radolan

[![Go Reference](https://pkg.go.dev/badge/github.com/shyrmapp/radolan.svg)](https://pkg.go.dev/github.com/shyrmapp/radolan)
[![Go Report Card](https://goreportcard.com/badge/github.com/shyrmapp/radolan)](https://goreportcard.com/report/github.com/shyrmapp/radolan)
[![CI](https://github.com/shyrmapp/radolan/actions/workflows/ci.yml/badge.svg)](https://github.com/shyrmapp/radolan/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-97%25-brightgreen)](https://github.com/shyrmapp/radolan)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Go package for parsing DWD RADOLAN / RADVOR-RE weather radar composites. Supports coordinate projection, Z-R conversion, and spatial sampling. Zero external dependencies.

## Installation

```sh
go get github.com/shyrmapp/radolan
```

Requires **Go 1.26+**.

## Quick start

```go
package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/shyrmapp/radolan"
)

func main() {
	f, err := os.Open("raa01-rv_10000-latest-dwd---bin.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	comp, err := radolan.NewComposite(f)
	if err != nil && !errors.Is(err, radolan.ErrUnknownUnit) {
		log.Fatal(err)
	}

	x, y := comp.Project(52.52, 13.41) // Berlin
	mmh := radolan.PrecipitationRateAdaptive(comp.At(int(x), int(y)))
	fmt.Printf("Berlin: %.2f mm/h\n", mmh)
}
```

## API overview

### Parsing

| Function | Description |
|---|---|
| `NewComposite(io.Reader)` | Parse a single binary composite |
| `NewComposites(io.Reader)` | Parse a `.tar.bz2` archive of composites (sorted by forecast time) |
| `NewDummy(product, format, dx, dy)` | Create a blank composite for projection-only use |

### Data access

| Method | Description |
|---|---|
| `comp.At(x, y)` | Raw value at pixel (x, y) — layer 0 |
| `comp.AtZ(x, y, z)` | Raw value at voxel (x, y, z) — for 3D products |
| `comp.NeighbourhoodSample(cx, cy, radius)` | Spatial average: mean mm/h, max mm/h, rain coverage fraction |

### Coordinate projection

| Method | Description |
|---|---|
| `comp.Project(lat, lon)` | Convert WGS84 coordinates to grid (x, y) |
| `comp.ProjectionFunc()` | Pre-calibrated closure for tight loops |

### Z-R conversion

| Function | Description |
|---|---|
| `PrecipitationRateAdaptive(dBZ)` | Regime-adaptive dBZ → mm/h (recommended) |
| `PrecipitationRate(relation, dBZ)` | Fixed Z-R: dBZ → mm/h |
| `Reflectivity(relation, mmh)` | Inverse: mm/h → dBZ |

Built-in Z-R relationships: `Aniol80` (DWD operational), `Doelling98` (MeteoSwiss), `JossWaldvogel70` (stratiform), `MarshallPalmer55` (convective). Custom relations via `NewZR(a, b)`.

### Types and sentinels

- **`Composite`** — parsed radar composite with header metadata, grid data, and projection
- **`Unit`** — data unit enum: `Unit_dBZ`, `Unit_mm`, `Unit_km`, `Unit_mps`, `Unit_unknown`
- **`ErrUnknownUnit`** — returned when the product's unit is not in the catalog; data is still usable (check with `errors.Is`)
- **`NaN`** / `IsNaN(f)` — float32 NaN for no-data pixels

## Usage examples

### Parse a tar.bz2 archive

```go
f, err := os.Open("RV_latest.tar.bz2")
if err != nil {
	log.Fatal(err)
}
defer f.Close()

composites, err := radolan.NewComposites(f)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("%d frames, latest: %s\n",
	len(composites), composites[len(composites)-1].ForecastTime)
```

### Spatial sampling

```go
project := comp.ProjectionFunc()
x, y := project(lat, lng)
avgMMH, maxMMH, coverage := comp.NeighbourhoodSample(int(x), int(y), 2)
fmt.Printf("avg %.2f mm/h, max %.2f mm/h, coverage %.0f%%\n",
	avgMMH, maxMMH, coverage*100)
```

### Z-R conversion

```go
// Adaptive (recommended): selects relation by intensity
mmh := radolan.PrecipitationRateAdaptive(dBZ)

// Fixed relation
mmh = radolan.PrecipitationRate(radolan.Aniol80, dBZ)

// Inverse: mm/h → dBZ
dbz := radolan.Reflectivity(radolan.Aniol80, mmh)
```

## Supported products

| Product | Grid | Unit | Description |
|---------|------|------|-------------|
| EX | Middle-European (1400×1500) | dBZ | Reflectivity |
| FX, FZ | National (900×900) | dBZ | Nowcast reflectivity |
| PE | Local | km | Echo top |
| PF, PX | Local | dBZ | Reflectivity |
| PG | National picture (920×920) | dBZ | Reflectivity |
| PR | Local | m/s | Doppler radial velocity |
| PZ | Local | dBZ | 3D reflectivity (CAPPI) |
| RV, YW | DE1200 (1100×1200) | mm | 5-min precipitation |
| RW | National (900×900) | mm | Hourly precipitation |
| RX | National (900×900) | dBZ | Reflectivity |
| SF | National (900×900) | mm | Daily precipitation |
| WN | DE1200 (1100×1200) | dBZ | Nowcast reflectivity |
| WX | Extended national (900×1100) | dBZ | Reflectivity |

RV composites return `ErrUnknownUnit` — this is expected. The data is valid.

## Coordinate projection

DWD uses two projection models depending on the composite format:

- **Format < 5**: Polar stereographic with spherical Earth (R = 6370.04 km, φ₀ = 60°N, λ₀ = 10°E)
- **Format ≥ 5** (DE1200, RADVOR-RE): WGS84 ellipsoid polar stereographic

The library detects the format from the header and applies the correct projection automatically. Grid coordinates are zero-indexed from the top-left corner.

## Background

This is a production fork of [jonnyschaefer/radolan](https://github.com/jonnyschaefer/radolan) (MIT). Key changes from upstream:

- **`ErrUnknownUnit` tolerance**: RV composites trigger this on valid data. Upstream discards them; this fork returns them.
- **Format ≥ 5 projection**: Upstream returns NaN for DE1200/RADVOR-RE. This fork applies WGS84 polar stereographic.
- **`PrecipitationRateAdaptive`**: Regime-adaptive Z-R conversion (JossWaldvogel70 / Aniol80 / MarshallPalmer55 by intensity).
- **`NeighbourhoodSample`**: Spatial averaging with coverage statistics.
- **`ProjectionFunc`**: Pre-calibrated projection closure for rendering loops.
- **Go 1.26 modernisation**: Range-over-int, `errors.Is` sentinels, precomputed precision multipliers, no `init()`.

## Data sources

- [RADOLAN RV (5-min observed)](https://opendata.dwd.de/weather/radar/composit/rv/)
- [RADVOR-RE (5-min nowcast)](https://opendata.dwd.de/weather/radar/radvor/rv/)
- [DWD RADOLAN format specification (PDF)](https://www.dwd.de/DE/leistungen/radolan/radolan_info/radolan_radvor_op_komposit_format_pdf.pdf)

## Contributing

Contributions welcome. Please ensure `go test -race ./...` and `go vet ./...` pass before submitting.

## License

MIT — see [LICENSE](LICENSE). Derived from [jonnyschaefer/radolan](https://github.com/jonnyschaefer/radolan).
