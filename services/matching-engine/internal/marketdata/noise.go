// Package marketdata simulates believable market data for the paused exchange (KW03 mock adapter).
//
// Two properties drive the whole design:
//
//  1. **Deterministic in time.** Every value is a pure function of (product, timestamp) — there is no
//     stored state and no RNG. Refetching a chart returns byte-identical history, a page refresh does
//     not rewrite the past, and two clients see the same market. A stateful random walk would fail all
//     three.
//  2. **Smooth, not jittery.** Prices come from fractal value noise (summed octaves of interpolated
//     lattice noise), which looks like a market at every zoom level, unlike white noise.
//
// This package performs no money math and touches no balance: it produces display data for a paused,
// paper-only surface. Values are float64 here and are serialized to fixed-point 6dp decimal strings at
// the API boundary (see internal/api). Nothing here settles, mints, burns, or debits.
package marketdata

import (
	"encoding/binary"
	"hash/fnv"
	"math"
)

// hash01 maps (seed, i) to a uniform value in [0,1). FNV-1a is not cryptographic — it does not need to
// be; it needs to be fast, stable across runs and platforms, and well-distributed, so the same
// timestamp always yields the same market.
func hash01(seed string, i int64) float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed))
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(i))
	_, _ = h.Write(b[:])
	// Drop the top bit so the conversion to float64 stays exact and non-negative.
	return float64(h.Sum64()&math.MaxInt64) / float64(math.MaxInt64+1.0)
}

// Hash01 exposes hash01 so sibling packages (portfolio) derive their own deterministic values from the
// same generator, keeping one source of simulated randomness across the service.
func Hash01(seed string, i int64) float64 { return hash01(seed, i) }

// LogNormalSize exposes logNormal for sibling packages deriving position and fill sizes.
func LogNormalSize(seed string, i int64, median, sigma float64) float64 {
	return logNormal(seed, i, median, sigma)
}

// hashSigned maps (seed, i) to a uniform value in [-1,1).
func hashSigned(seed string, i int64) float64 { return hash01(seed, i)*2 - 1 }

// smoothstep eases t in [0,1] with a cubic so interpolated noise has no visible corners at lattice
// points (a linear blend would produce a sawtooth chart).
func smoothstep(t float64) float64 { return t * t * (3 - 2*t) }

// valueNoise interpolates hashed lattice values around x, returning a smooth curve in [-1,1].
func valueNoise(seed string, x float64) float64 {
	i := int64(math.Floor(x))
	f := x - float64(i)
	a := hashSigned(seed, i)
	b := hashSigned(seed, i+1)
	return a + (b-a)*smoothstep(f)
}

// octaves is how many frequencies fbm sums. Five gives structure from the base period down to ~1/16th
// of it — enough detail for a 1-minute bar without the curve turning to noise.
const octaves = 5

// fbm sums octaves of value noise at doubling frequency and halving amplitude, normalized to about
// [-1,1]. This is what gives the price curve trends, swings, and micro-detail at once.
func fbm(seed string, x float64) float64 {
	var sum, amp, norm float64
	amp = 1
	freq := 1.0
	for o := 0; o < octaves; o++ {
		// Perturb the seed per octave so the octaves are independent rather than scaled copies.
		sum += valueNoise(seed+string(rune('a'+o)), x*freq) * amp
		norm += amp
		amp *= 0.5
		freq *= 2
	}
	if norm == 0 {
		return 0
	}
	return sum / norm
}

// basePeriod is the coarsest price cycle, in seconds. Half an hour: long enough that an intraday chart
// shows real trend, short enough that a demo viewer sees the market move while watching.
const basePeriod = 1800.0

// Price returns the simulated mid for a product at a Unix timestamp. It mean-reverts around ref with
// relative amplitude vol, and is strictly positive — a price of zero or below would be nonsense for
// every downstream consumer (spreads, percentage changes, chart scales).
func Price(seed string, ref, vol float64, unixSec int64) float64 {
	p := ref * (1 + vol*fbm(seed, float64(unixSec)/basePeriod))
	if floor := ref * 0.5; p < floor {
		return floor
	}
	if ceil := ref * 1.5; p > ceil {
		return ceil
	}
	return p
}

// PriceAt is Price over a longer cycle, for series sampled in days rather than seconds (the index
// print history). periodDays sets the coarsest cycle.
func PriceAt(seed string, ref, vol float64, unixSec int64, periodDays float64) float64 {
	p := ref * (1 + vol*fbm(seed, float64(unixSec)/(periodDays*86400)))
	if floor := ref * 0.5; p < floor {
		return floor
	}
	if ceil := ref * 1.5; p > ceil {
		return ceil
	}
	return p
}

// logNormal returns a deterministic log-normally distributed size in [1, ∞): the shape real trade
// sizes follow, with many small prints and rare large ones. sigma widens the tail.
func logNormal(seed string, i int64, median, sigma float64) float64 {
	// Box–Muller from two independent uniforms, guarding u1 away from 0 (log(0) is -Inf).
	u1 := hash01(seed+"u1", i)
	if u1 < 1e-12 {
		u1 = 1e-12
	}
	u2 := hash01(seed+"u2", i)
	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	v := median * math.Exp(sigma*z)
	if v < 1 {
		return 1
	}
	return v
}

// snap rounds a price to the product's tick so quoted levels sit on a real grid. A tick of zero or
// less is ignored rather than dividing by zero.
func snap(price, tick float64) float64 {
	if tick <= 0 {
		return price
	}
	return math.Round(price/tick) * tick
}

// tickFor returns the tick size implied by a product's quote precision (10^-precision), which is how
// the catalog's tick_size values are defined.
func tickFor(quotePrecision int) float64 {
	return math.Pow(10, -float64(quotePrecision))
}
