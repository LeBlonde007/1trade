package catalog

import (
	"fmt"
	"sort"
	"time"
)

// Model lifecycle (F10). A model is active, then deprecated (still served, with Deprecation / Sunset
// headers and a console banner and CLI warning), then retired at its sunset (410 Gone, pointing at
// its replacement). Customers get at least MinNotice between the announcement and the sunset.

// Status values for Pricing.Status.
const (
	StatusActive     = "active"
	StatusDeprecated = "deprecated"
	StatusRetired    = "retired"
)

// MinNotice is the minimum time between announcing a deprecation and the sunset (F10: 30 days).
const MinNotice = 30 * 24 * time.Hour

// Deprecation schedules a model's retirement.
type Deprecation struct {
	AnnouncedAt time.Time `json:"announced_at"`
	SunsetAt    time.Time `json:"sunset_at"`
	Replacement string    `json:"replacement,omitempty"`
}

// meta is each model's category and context window. Every catalog entry must have a row
// (TestEveryModelHasMetadata); context length is 0 where it does not apply (images, audio, video).
var meta = map[string]struct {
	category string
	context  int
}{
	"llama-3.1-70b":        {"text-general", 131072},
	"llama-3.1-8b":         {"text-small", 131072},
	"llama-3.2-1b":         {"text-small", 131072},
	"qwen2.5-1.5b":         {"text-small", 32768},
	"claude-opus-4.8":      {"text-general", 200000},
	"claude-sonnet-4.5":    {"text-general", 200000},
	"gpt-5":                {"text-general", 400000},
	"gpt-4o":               {"text-general", 128000},
	"deepseek-v3.2":        {"text-general", 128000},
	"qwen3-32b":            {"text-general", 32768},
	"nemotron-vision":      {"image-understanding", 32768},
	"doc-writer":           {"text-docs", 128000},
	"code-writer":          {"text-code", 128000},
	"agent-operator":       {"agent", 0},
	"stable-diffusion-3.5": {"image-generation", 0},
	"flux-schnell":         {"image-generation", 0},
	"gpt-image-1.5":        {"image-generation", 0},
	"elevenlabs-tts":       {"speech-tts", 0},
	"qwen3-tts":            {"speech-tts", 0},
	"whisper-large-v3":     {"speech-stt", 0},
	"wan-t2v":              {"video-generation", 0},
	"bge-m3":               {"embeddings", 8192},
	"e5-large":             {"embeddings", 512},
}

// deprecations is the reviewed lifecycle schedule: the quarterly catalog review adds rows here (see
// docs/catalog-refresh.md). Empty means every model is active. A var so tests can schedule one.
var deprecations = map[string]Deprecation{}

// init fills each entry's category and context window from meta.
func init() {
	for i := range models {
		if m, ok := meta[models[i].ID]; ok {
			models[i].Trade1.Category, models[i].Trade1.ContextLength = m.category, m.context
		}
	}
}

// StatusAt is the model's lifecycle status at now, and its deprecation schedule (nil if none).
func StatusAt(id string, now time.Time) (string, *Deprecation) {
	d, ok := deprecations[id]
	switch {
	case !ok:
		return StatusActive, nil
	case !now.Before(d.SunsetAt):
		return StatusRetired, &d
	case !now.Before(d.AnnouncedAt):
		return StatusDeprecated, &d
	}
	return StatusActive, nil // scheduled but not yet announced
}

// withStatus stamps m's lifecycle fields for now.
func withStatus(m Model, now time.Time) Model {
	m.Trade1.Status, m.Trade1.Deprecation = StatusAt(m.ID, now)
	return m
}

// Listing is ListServable with lifecycle status applied, retired models removed, in catalog order.
func Listing(modelMap map[string]string, now time.Time) []Model {
	var out []Model
	for _, m := range ListServable(modelMap) {
		if m = withStatus(m, now); m.Trade1.Status != StatusRetired {
			out = append(out, m)
		}
	}
	return out
}

// LookupAt is Lookup with lifecycle status applied for now.
func LookupAt(id string, now time.Time) (Model, bool) {
	m, ok := Lookup(id)
	if !ok {
		return Model{}, false
	}
	return withStatus(m, now), true
}

// validateDeprecations checks a schedule: known models, at least MinNotice before the sunset, and a
// replacement that exists and is not itself being retired.
func validateDeprecations(ds map[string]Deprecation) error {
	ids := make([]string, 0, len(ds))
	for id := range ds {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		d := ds[id]
		if _, ok := Lookup(id); !ok {
			return fmt.Errorf("catalog: deprecation for unknown model %s", id)
		}
		if d.SunsetAt.Sub(d.AnnouncedAt) < MinNotice {
			return fmt.Errorf("catalog: %s gets less than 30 days' notice", id)
		}
		if d.Replacement != "" {
			if _, ok := Lookup(d.Replacement); !ok {
				return fmt.Errorf("catalog: %s's replacement %s is not in the catalog", id, d.Replacement)
			}
			if _, gone := ds[d.Replacement]; gone {
				return fmt.Errorf("catalog: %s's replacement %s is itself deprecated", id, d.Replacement)
			}
		}
	}
	return nil
}

// SetDeprecations replaces the lifecycle schedule after validating it (30 days' notice, known
// models, live replacements), and returns a func that restores the previous one. It is how a
// reviewed schedule is loaded, and how tests schedule one. Not safe to call while serving.
func SetDeprecations(ds map[string]Deprecation) (restore func(), err error) {
	if err := validateDeprecations(ds); err != nil {
		return nil, err
	}
	prev := deprecations
	deprecations = ds
	return func() { deprecations = prev }, nil
}
