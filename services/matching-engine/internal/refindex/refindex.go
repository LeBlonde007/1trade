// Package refindex serves the AI credit index reads (openapi/index.yaml) while index-service does not
// yet exist.
//
// **This is a keep-warm mock, and it labels itself as one.** KW01 gives `index-service` ownership of
// the real computation from M5; until then the methodology page needs a real interface to build
// against, so the reads live here. Every print returned carries `provisional: true` and
// `source: "mock"`, and every collection carries `is_mock: true` — a caller can never mistake a
// simulated print for a published index value. When index-service ships, this package is deleted and
// the BFF's upstream base changes; the contract and the client do not.
//
// What *is* real here is the integrity machinery, because that is the part worth exercising early
// (immutable commitment #2): prints are hash-chained with SHA-256 over a canonical serialization,
// anchored at a fixed genesis, so the chain for a given day is identical no matter how much history the
// caller asks for. Tampering with any print breaks every hash after it.
package refindex

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/exascale/matching-engine/internal/domain"
	"github.com/exascale/matching-engine/internal/marketdata"
)

// MethodologyVersion is the version prints are computed under. Bumping it is a published,
// committee-reviewed act — see docs/index-methodology.md §8.
const MethodologyVersion = "1.2"

// MethodologyEffective is the date the current methodology version took effect.
const MethodologyEffective = "2026-04-01"

// WindowMinutes is the observation window each print is computed over.
const WindowMinutes = 120

// TrimPct is the share of observations retained after outlier filtering.
const TrimPct = "95.0"

// VolumeFloor is the minimum observed volume for a constituent to enter a print.
const VolumeFloor = "25000.000000"

// publicationHourUTC is the daily print time (16:00 UTC), per phase6_v2.md §11.5.
const publicationHourUTC = 16

// genesis is the first publication date — methodology v1.0's effective date. The chain is anchored
// here, so a print's chain_hash never depends on how many days the caller requested.
var genesis = time.Date(2025, 10, 15, publicationHourUTC, 0, 0, 0, time.UTC)

// indexRef and indexVol drive the simulated index level. The index tracks the ai_index credit, so it
// reuses that product's reference level from the catalog.
const indexPeriodDays = 45.0

// Print is one index print.
type Print struct {
	PrintID            string
	Value              float64
	PublishedAt        time.Time
	MethodologyVersion string
	ObservationCount   int
	ExcludedCount      int
	ChainHash          string
	PrevChainHash      string
	Provisional        bool
	Source             string
}

// Constituent is one disclosed input to the index, with its weight.
type Constituent struct {
	Name       string
	CreditType string
	Weight     string // fixed-point 4dp; weights sum to 1
	Source     string
	ObservedAt time.Time
}

// constituents is the disclosed weight schedule for MethodologyVersion. Weights sum to exactly 1.0000.
// Every credit_type is drawn from the canonical enum (credit-types.md §1) — the index never invents an
// input. Sources are the Phase 1 observation set from KW01: real platform transactions, not trades,
// because no trades exist while the exchange is paused.
var constituents = []Constituent{
	{Name: "Text credit consumption", CreditType: domain.CreditText, Weight: "0.4200", Source: "Realized consumption rates"},
	{Name: "Image credit consumption", CreditType: domain.CreditImage, Weight: "0.1500", Source: "Realized consumption rates"},
	{Name: "Speech credit consumption", CreditType: domain.CreditSpeech, Weight: "0.0800", Source: "Realized consumption rates"},
	{Name: "Video credit consumption", CreditType: domain.CreditVideo, Weight: "0.0500", Source: "Realized consumption rates"},
	{Name: "Embeddings credit consumption", CreditType: domain.CreditEmbeddings, Weight: "0.1000", Source: "Prepaid purchase prices"},
	{Name: "H100 capacity cost", CreditType: domain.CreditH100, Weight: "0.2000", Source: "Reserved-capacity transaction prices"},
}

// Constituents returns the disclosed weight schedule, stamped with the observation time for the print
// covering t. The slice is copied so callers cannot mutate the schedule.
func Constituents(t time.Time) []Constituent {
	pub := LastPublication(t)
	out := make([]Constituent, len(constituents))
	copy(out, constituents)
	for i := range out {
		// Each constituent is observed at some point inside the window before publication.
		offset := time.Duration(marketdata.Hash01(out[i].CreditType+"obs", pub.Unix()) * float64(WindowMinutes) * float64(time.Minute))
		out[i].ObservedAt = pub.Add(-offset)
	}
	return out
}

// LastPublication returns the most recent 16:00 UTC publication instant at or before t.
func LastPublication(t time.Time) time.Time {
	u := t.UTC()
	pub := time.Date(u.Year(), u.Month(), u.Day(), publicationHourUTC, 0, 0, 0, time.UTC)
	if pub.After(u) {
		pub = pub.AddDate(0, 0, -1)
	}
	return pub
}

// NextPublication returns the next 16:00 UTC publication instant strictly after t.
func NextPublication(t time.Time) time.Time {
	return LastPublication(t).AddDate(0, 0, 1)
}

// History returns prints for the last `days` publications, oldest first. The chain is always walked
// from genesis so each print's chain_hash is absolute; only the tail is returned.
func History(t time.Time, days int) []Print {
	if days < 1 {
		return nil
	}
	last := LastPublication(t)
	if last.Before(genesis) {
		return nil
	}
	total := int(last.Sub(genesis).Hours()/24) + 1
	all := make([]Print, 0, total)
	prev := ""
	for i := 0; i < total; i++ {
		pub := genesis.AddDate(0, 0, i)
		p := compute(pub, prev)
		all = append(all, p)
		prev = p.ChainHash
	}
	if days >= len(all) {
		return all
	}
	return all[len(all)-days:]
}

// Latest returns the most recent print, or false when the current time predates genesis.
func Latest(t time.Time) (Print, bool) {
	h := History(t, 1)
	if len(h) == 0 {
		return Print{}, false
	}
	return h[len(h)-1], true
}

// compute builds the print published at pub, chained to prev. The value comes from the simulated index
// curve; the observation counts and the chain hash are computed the way the real service will.
func compute(pub time.Time, prev string) Print {
	ref, ok := domain.Find("EAI-IDX")
	if !ok {
		// The catalog always contains the index product; fall back to a sane level rather than panicking
		// in a request path.
		ref = domain.Product{Reference: 0.001005, Vol: 0.012}
	}
	value := marketdata.PriceAt("index", ref.Reference, ref.Vol, pub.Unix(), indexPeriodDays)

	observations := 4000 + int(marketdata.Hash01("obs", pub.Unix())*4000)
	excluded := int(float64(observations) * (0.005 + marketdata.Hash01("exc", pub.Unix())*0.02))

	p := Print{
		PrintID:            marketdata.DeterministicID("index-print", pub.Unix()),
		Value:              value,
		PublishedAt:        pub,
		MethodologyVersion: MethodologyVersion,
		ObservationCount:   observations,
		ExcludedCount:      excluded,
		PrevChainHash:      prev,
		// Mock prints are always provisional: nothing has been reconciled, because nothing was computed
		// from real settled observations.
		Provisional: true,
		Source:      "mock",
	}
	p.ChainHash = chainHash(prev, p)
	return p
}

// chainHash computes hash(prev_chain_hash || canonical_json(row)), the same construction the credit
// ledger uses. The canonical form is built with an explicit, fixed key order and fixed-point values —
// never Go's map iteration or float formatting, either of which would make the hash unstable.
func chainHash(prev string, p Print) string {
	canonical := fmt.Sprintf(
		`{"print_id":"%s","value":"%.6f","published_at":"%s","methodology_version":"%s","observation_count":%d,"excluded_count":%d,"provisional":%t,"source":"%s"}`,
		p.PrintID, p.Value, p.PublishedAt.UTC().Format(time.RFC3339), p.MethodologyVersion,
		p.ObservationCount, p.ExcludedCount, p.Provisional, p.Source,
	)
	sum := sha256.Sum256([]byte(prev + canonical))
	return hex.EncodeToString(sum[:])
}

// VerifyChain re-derives every hash in a print series and reports whether the chain is intact. The API
// does not need this, but it is the property the whole design exists to provide, so it is exercised by
// the package's tests and available to any audit tooling.
func VerifyChain(prints []Print) bool {
	prev := ""
	for i, p := range prints {
		if i > 0 && p.PrevChainHash != prev {
			return false
		}
		if chainHash(p.PrevChainHash, p) != p.ChainHash {
			return false
		}
		prev = p.ChainHash
	}
	return true
}
