# radolan

Go package for parsing DWD RADOLAN and RADVOR-RE weather radar composites and working with the DWD polar stereographic grid.

[![CI](https://github.com/shyrmapp/radolan/actions/workflows/ci.yml/badge.svg)](https://github.com/shyrmapp/radolan/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/shyrmapp/radolan.svg)](https://pkg.go.dev/github.com/shyrmapp/radolan)

## Background

[DWD Open Data](https://opendata.dwd.de/climate_environment/CDC/grids_germany/5_minutes/) publishes radar composites in the RADOLAN binary format covering Germany and surrounding areas. This package parses those composites and provides coordinate projection, Z-R conversion, and spatial sampling utilities.

This is a production-hardened fork of [jonnyschaefer/radolan](https://github.com/jonnyschaefer/radolan) (MIT). It has diverged significantly from the original and is not intended to be merged back upstream. Key changes:

- **`ErrUnknownUnit` tolerance in `NewComposites`**: RV composites (5-min precipitation rate) trigger this error on valid data. The upstream library discards such composites; this fork continues and returns them — the data is usable.
- **Format ≥ 5 projection fix**: RADVOR-RE nowcast composites use `Format=5` on the DE1200 national grid. The upstream projection returns NaN for this combination. This fork applies the correct WGS84 polar stereographic math.
- **`PrecipitationRateAdaptive`**: Adaptive Z-R conversion that selects between JossWaldvogel70, Aniol80, and MarshallPalmer55 by intensity — closer to DWD operational practice than a single fixed relation.
- **`NeighbourhoodSample`**: Spatial averaging over a grid neighbourhood, returning mean mm/h, max mm/h, and rain coverage fraction — useful for point precipitation estimation without single-pixel noise.
- **`ProjectionFunc`**: Returns a pre-calibrated closure for repeated projection without per-call composite overhead — important for tight rendering loops.
- **Go 1.22+ modernisation**: Range-over-int, `slices.SortFunc`, other idiomatic updates.

## Installation

```sh
go get github.com/shyrmapp/radolan
```

Requires Go 1.22 or later.

## Usage

### Parse a single composite (RADVOR-RE nowcast)

```go
f, err := os.Open("RV2025032615050.bin.gz")
if err != nil { log.Fatal(err) }
defer f.Close()

comp, err := radolan.NewComposite(f)
if err != nil && err != radolan.ErrUnknownUnit {
    log.Fatal(err)
}

// Project a lat/lng to grid coordinates.
x, y := comp.Project(52.52, 13.41) // Berlin
mmh := radolan.PrecipitationRateAdaptive(comp.At(int(x), int(y)))
fmt.Printf("Berlin: %.2f mm/h\n", mmh)
```

### Parse a tar.bz2 archive (RADOLAN RV, 5-min composites)

RV archives contain multiple composites. `NewComposites` tolerates `ErrUnknownUnit`
and returns all frames sorted by forecast time.

```go
f, err := os.Open("RV_latest.tar.bz2")
if err != nil { log.Fatal(err) }
defer f.Close()

composites, err := radolan.NewComposites(f)
if err != nil { log.Fatal(err) }
fmt.Printf("Loaded %d frames, latest at %s\n",
    len(composites), composites[len(composites)-1].ForecastTime)
```

### Spatial sampling around a point

```go
proj := comp.ProjectionFunc()
x, y := proj(lat, lng)
avgMMH, maxMMH, coverage := comp.NeighbourhoodSample(int(x), int(y), 2)
fmt.Printf("avg %.2f mm/h, max %.2f mm/h, coverage %.0f%%\n",
    avgMMH, maxMMH, coverage*100)
```

### Z-R conversion

```go
// Adaptive (recommended): selects relation by intensity
mmh := radolan.PrecipitationRateAdaptive(dBZ)

// Fixed relation (DWD operational Germany)
mmh = radolan.PrecipitationRate(radolan.Aniol80, dBZ)

// Inverse: mm/h → dBZ
dbz := radolan.Reflectivity(radolan.Aniol80, mmh)
```

## Supported products

| Product | Grid              | Description                      | Notes                     |
|---------|-------------------|----------------------------------|---------------------------|
| EX      | Middle-European   | Reflectivity                     |                           |
| FX      | National          | Nowcast reflectivity             |                           |
| FZ      | National          | Nowcast reflectivity             |                           |
| PE      | Local             | Echo top                         |                           |
| PF      | Local             | Reflectivity                     |                           |
| PG      | National picture  | Reflectivity                     |                           |
| PR      | Local             | Doppler radial velocity          |                           |
| PX      | Local             | Reflectivity                     |                           |
| PZ      | Local             | 3D reflectivity CAPPI            |                           |
| RV      | DE1200            | 5-min precipitation rate (mm/h)  | `ErrUnknownUnit` expected |
| RW      | National          | Hourly accumulated precipitation |                           |
| RX      | National          | Reflectivity                     |                           |
| SF      | National          | Daily accumulated precipitation  |                           |
| WN      | DE1200 Sphere     | Nowcast reflectivity             |                           |
| WN      | DE1200 WGS84      | Nowcast reflectivity             |                           |
| WX      | Extended national | Reflectivity                     |                           |
| YW      | DE1200            | 5-min precipitation rate (mm/h)  |                           |

RADVOR-RE (`RV`/`YW`, Format=5, DE1200 national grid) is fully supported including the WGS84 polar stereographic projection.

## Coordinate projection

DWD uses a polar stereographic projection with a spherical Earth model for most grids, and a WGS84 ellipsoid model for DE1200 Format≥5 (RADOLAN RV, RADVOR-RE). This package handles both automatically based on the composite header.

```go
// Grid coordinate from lat/lng
x, y := comp.Project(north, east)

// Pre-calibrated closure for tight loops (avoids repeated header lookups)
project := comp.ProjectionFunc()
x, y = project(north, east)
```

Coordinates are zero-indexed from the top-left corner of the grid. Non-integer values indicate sub-pixel position; use `int(x)` for direct array access or bilinear interpolation for smoother results.

## Data sources

- **RADOLAN RV** (5-min observed): `https://opendata.dwd.de/weather/radar/composit/rv/`
- **RADVOR-RE** (5-min nowcast, 2h ahead): `https://opendata.dwd.de/weather/radar/radvor/rv/`
- **DWD RADOLAN product documentation**: [DWD RADOLAN/RADVOR System Description](https://www.dwd.de/DE/leistungen/radolan/radolan.html)

## License

MIT. Derived from [jonnyschaefer/radolan](https://github.com/jonnyschaefer/radolan) (also MIT).
