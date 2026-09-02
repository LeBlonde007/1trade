package marketdata

import (
	"encoding/binary"
	"encoding/hex"
	"hash/fnv"
	"math"
	"sort"
	"time"

	"github.com/trade1/matching-engine/internal/domain"
)

// Quote is a two-sided quote at an instant.
type Quote struct {
	Bid, Ask, Mid  float64
	BidSize        float64
	AskSize        float64
	Spread         float64
	SpreadBps      float64
	AsOf           time.Time
}

// Level is one aggregated order-book level.
type Level struct {
	Price      float64
	Size       float64
	Cumulative float64
}

// Book is an aggregated two-sided order book. Bids descend, asks ascend, and the two never cross.
type Book struct {
	Bids []Level
	Asks []Level
	AsOf time.Time
}

// Print is one tape print.
type Print struct {
	TradeID       string
	Price         float64
	Quantity      float64
	AggressorSide string
	Block         bool
	ExecutedAt    time.Time
}

// Candle is one OHLCV bar. Time is the bar-open in Unix seconds.
type Candle struct {
	Time                    int64
	Open, High, Low, Close  float64
	Volume                  float64
}

// Summary is a product's 24h rollup.
type Summary struct {
	Last         float64
	Open24h      float64
	High24h      float64
	Low24h       float64
	ChangePct24h float64
	Volume24h    float64
	SpreadBps    float64
}

// IntervalSeconds maps a contract interval string to its length. The bool reports whether the interval
// is one the contract allows — callers reject unknown intervals rather than guessing.
func IntervalSeconds(interval string) (int64, bool) {
	switch interval {
	case "1m":
		return 60, true
	case "5m":
		return 300, true
	case "15m":
		return 900, true
	case "1h":
		return 3600, true
	case "4h":
		return 14400, true
	case "1d":
		return 86400, true
	default:
		return 0, false
	}
}

// spreadFrac is the simulated half-spread as a fraction of mid, scaled by the product's volatility:
// more volatile products quote wider, which is how real market makers behave.
func spreadFrac(p domain.Product) float64 { return 0.00018 + p.Vol*0.010 }

// mid returns the product's simulated mid at t.
func mid(p domain.Product, t time.Time) float64 {
	return Price(p.ID, p.Reference, p.Vol, t.Unix())
}

// QuoteAt builds the two-sided quote for a product at t. Sizes at the touch are deterministic per
// two-second bucket so the quote visibly refreshes without flickering on every request.
func QuoteAt(p domain.Product, t time.Time) Quote {
	m := mid(p, t)
	tick := tickFor(p.QuotePrecision)
	half := m * spreadFrac(p)
	if half < tick/2 {
		half = tick / 2 // never quote a crossed or zero-width market
	}
	bucket := t.Unix() / 2
	bid := snap(m-half, tick)
	ask := snap(m+half, tick)
	if ask <= bid {
		ask = bid + tick
	}
	spread := ask - bid
	q := Quote{
		Bid:     bid,
		Ask:     ask,
		Mid:     (bid + ask) / 2,
		BidSize: logNormal(p.ID+"bidsz", bucket, baseLevelSize(p), 0.6),
		AskSize: logNormal(p.ID+"asksz", bucket, baseLevelSize(p), 0.6),
		Spread:  spread,
		AsOf:    t,
	}
	if q.Mid > 0 {
		q.SpreadBps = spread / q.Mid * 10000
	}
	return q
}

// baseLevelSize is the typical resting size at the touch, scaled so cheap credits quote in large
// unit counts and GPU-hours quote in small ones.
func baseLevelSize(p domain.Product) float64 {
	if p.Reference <= 0 {
		return 1000
	}
	// Aim for a roughly constant notional per level across products.
	return math.Max(4, 12000/math.Sqrt(p.Reference*1000))
}

// BookAt builds an aggregated order book to the requested depth. Sizes follow a power law away from
// the touch (deeper levels are larger), jittered per two-second bucket. Level prices step by at least
// one tick, so levels never collide after snapping.
func BookAt(p domain.Product, t time.Time, depth int) Book {
	if depth < 1 {
		depth = 1
	}
	q := QuoteAt(p, t)
	tick := tickFor(p.QuotePrecision)
	step := math.Max(tick, q.Mid*0.0008)
	bucket := t.Unix() / 2
	base := baseLevelSize(p)

	bids := make([]Level, 0, depth)
	asks := make([]Level, 0, depth)
	var cumBid, cumAsk float64
	for i := 0; i < depth; i++ {
		shape := math.Pow(float64(i)+1, 1.6) // power-law depth profile
		bidSize := logNormal(p.ID+"b", bucket*int64(depth+1)+int64(i), base*shape, 0.35)
		askSize := logNormal(p.ID+"a", bucket*int64(depth+1)+int64(i), base*shape, 0.35)
		cumBid += bidSize
		cumAsk += askSize
		bids = append(bids, Level{Price: snap(q.Bid-float64(i)*step, tick), Size: bidSize, Cumulative: cumBid})
		asks = append(asks, Level{Price: snap(q.Ask+float64(i)*step, tick), Size: askSize, Cumulative: cumAsk})
	}
	// Guarantee the contract's ordering even if snapping collapsed two adjacent levels.
	sort.Slice(bids, func(i, j int) bool { return bids[i].Price > bids[j].Price })
	sort.Slice(asks, func(i, j int) bool { return asks[i].Price < asks[j].Price })
	return Book{Bids: bids, Asks: asks, AsOf: t}
}

// printInterval is the mean seconds between prints for a product: busier (higher base volume) products
// print more often. Bounded so the tape neither stalls nor becomes a firehose.
func printInterval(p domain.Product) int64 {
	switch {
	case p.BaseVolume24h >= 9_000_000:
		return 3
	case p.BaseVolume24h >= 4_000_000:
		return 5
	case p.BaseVolume24h >= 1_000_000:
		return 9
	default:
		return 20
	}
}

// PrintsBefore returns up to limit recent prints for a product, newest first. Each print is keyed to a
// fixed time bucket, so as time advances new prints prepend and previously returned prints keep their
// exact id, price, size, and timestamp.
func PrintsBefore(p domain.Product, t time.Time, limit int) []Print {
	if limit < 1 {
		return nil
	}
	iv := printInterval(p)
	nowBucket := t.Unix() / iv
	base := baseLevelSize(p)
	out := make([]Print, 0, limit)
	for j := 0; j < limit; j++ {
		bucket := nowBucket - int64(j)
		// Jitter inside the bucket so prints aren't evenly spaced on the clock.
		offset := int64(hash01(p.ID+"toff", bucket) * float64(iv))
		ts := time.Unix(bucket*iv+offset, 0).UTC()
		if ts.After(t) {
			ts = t
		}
		m := Price(p.ID, p.Reference, p.Vol, ts.Unix())
		tick := tickFor(p.QuotePrecision)
		side := "buy"
		drift := 1.0
		if hash01(p.ID+"tside", bucket) < 0.5 {
			side = "sell"
			drift = -1.0
		}
		// Prints land at or just through the touch, on the aggressor's side.
		px := snap(m+drift*m*spreadFrac(p)*hash01(p.ID+"tpx", bucket), tick)
		if px <= 0 {
			px = tick
		}
		block := hash01(p.ID+"tblk", bucket) < 0.05
		qty := logNormal(p.ID+"tqty", bucket, base, 0.9)
		if block {
			qty *= 25 // rare outsized print the tape highlights
		}
		out = append(out, Print{
			TradeID:       DeterministicID(p.ID, "print", bucket),
			Price:         px,
			Quantity:      qty,
			AggressorSide: side,
			Block:         block,
			ExecutedAt:    ts,
		})
	}
	return out
}

// subSamples is how many points inside a bar are sampled to find its high and low. Four keeps candles
// honest (the wick reflects real intrabar movement) at negligible cost.
const subSamples = 4

// CandlesBefore returns up to limit OHLCV bars for an interval, oldest first, ending with the bar in
// progress. Closed bars are stable forever; only the last bar moves as time advances.
func CandlesBefore(p domain.Product, t time.Time, intervalSec int64, limit int) []Candle {
	if limit < 1 || intervalSec <= 0 {
		return nil
	}
	curOpen := (t.Unix() / intervalSec) * intervalSec
	out := make([]Candle, 0, limit)
	for k := limit - 1; k >= 0; k-- {
		barOpen := curOpen - int64(k)*intervalSec
		barEnd := barOpen + intervalSec
		isCurrent := barOpen == curOpen
		closeAt := barEnd
		if isCurrent {
			closeAt = t.Unix() // the live bar closes at "now"
		}
		o := Price(p.ID, p.Reference, p.Vol, barOpen)
		c := Price(p.ID, p.Reference, p.Vol, closeAt)
		hi := math.Max(o, c)
		lo := math.Min(o, c)
		for s := 1; s <= subSamples; s++ {
			ts := barOpen + int64(float64(s)/float64(subSamples+1)*float64(closeAt-barOpen))
			v := Price(p.ID, p.Reference, p.Vol, ts)
			hi = math.Max(hi, v)
			lo = math.Min(lo, v)
		}
		// A small deterministic wick beyond the sampled extremes, as real bars have.
		wick := (hi - lo) * 0.25 * hash01(p.ID+"wick", barOpen)
		hi += wick
		lo -= wick
		if lo <= 0 {
			lo = math.Min(o, c) * 0.999
		}
		// Volume scales with the bar length and is log-normal per bar; the live bar is pro-rated by how
		// much of it has elapsed, so it grows instead of appearing complete on arrival.
		barVol := p.BaseVolume24h / (86400 / float64(intervalSec))
		vol := logNormal(p.ID+"vol", barOpen, barVol/math.Max(p.Reference, 1e-9)/1000, 0.5)
		if isCurrent {
			elapsed := float64(t.Unix()-barOpen) / float64(intervalSec)
			if elapsed < 0.02 {
				elapsed = 0.02
			}
			vol *= elapsed
		}
		out = append(out, Candle{Time: barOpen, Open: o, High: hi, Low: lo, Close: c, Volume: vol})
	}
	return out
}

// SummaryAt builds a product's 24h rollup, sampling the price curve across the window for a true high
// and low rather than guessing from the endpoints.
func SummaryAt(p domain.Product, t time.Time) Summary {
	last := mid(p, t)
	open := Price(p.ID, p.Reference, p.Vol, t.Unix()-86400)
	hi, lo := math.Max(last, open), math.Min(last, open)
	const samples = 96 // every 15 minutes across the day
	for i := 0; i <= samples; i++ {
		ts := t.Unix() - 86400 + int64(float64(i)/samples*86400)
		v := Price(p.ID, p.Reference, p.Vol, ts)
		hi = math.Max(hi, v)
		lo = math.Min(lo, v)
	}
	var changePct float64
	if open != 0 {
		changePct = (last - open) / open * 100
	}
	dayBucket := t.Unix() / 86400
	q := QuoteAt(p, t)
	return Summary{
		Last:         last,
		Open24h:      open,
		High24h:      hi,
		Low24h:       lo,
		ChangePct24h: changePct,
		Volume24h:    logNormal(p.ID+"dayvol", dayBucket, p.BaseVolume24h, 0.25),
		SpreadBps:    q.SpreadBps,
	}
}

// DeterministicID derives a stable RFC-4122-shaped identifier from its inputs, so the same print or
// fill always carries the same id across requests. It is a content hash, not a random UUID: these
// objects are regenerated on every call and must keep their identity.
func DeterministicID(parts ...any) string {
	h := fnv.New64a()
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			_, _ = h.Write([]byte(v))
		case int64:
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], uint64(v))
			_, _ = h.Write(b[:])
		}
	}
	// Expand the 64-bit digest to 16 bytes by hashing twice with different suffixes.
	first := h.Sum64()
	h2 := fnv.New64a()
	var fb [8]byte
	binary.LittleEndian.PutUint64(fb[:], first)
	_, _ = h2.Write(fb[:])
	_, _ = h2.Write([]byte("hi"))
	second := h2.Sum64()

	var raw [16]byte
	binary.BigEndian.PutUint64(raw[0:8], first)
	binary.BigEndian.PutUint64(raw[8:16], second)
	raw[6] = (raw[6] & 0x0f) | 0x40 // version 4 shape
	raw[8] = (raw[8] & 0x3f) | 0x80 // RFC-4122 variant
	s := hex.EncodeToString(raw[:])
	return s[0:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:32]
}
