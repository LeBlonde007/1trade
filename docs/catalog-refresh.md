# Model catalog refresh (F10)

The catalog is curated: the top 3–5 models per category, kept current. This is how it changes.

## Cadence

- **Quarterly review.** For each category, check what is state of the art, what customers use (the
  gateway's per-model usage and `latency_p50_ms`), and what costs us too much to serve.
- **Promote** a new state-of-the-art model within 2–4 weeks of its release.
- **Deprecate** with at least 30 days' notice.

## Adding a model

1. Add a row to `services/inference-gateway/internal/catalog/catalog.go`: id, name, modality,
   credit type, unit and price.
2. Add its category and context window to `meta` in `catalog/lifecycle.go`.
   `TestEveryModelHasMetadata` fails until you do.
3. Map it to an upstream in the deployment's `INFERENCE_MODEL_MAP`. It is listed only where it can
   be served.

## Deprecating a model

1. Add it to `deprecations` in `catalog/lifecycle.go`, with `AnnouncedAt`, `SunsetAt` and a
   `Replacement`. The schedule is validated: the sunset is at least 30 days after the announcement,
   and the replacement exists and is not itself being retired. `TestEveryModelHasMetadata` checks
   the shipped schedule.
2. From `AnnouncedAt`, customers see it everywhere:
   - **API:** the model still serves, with `Deprecation`, `Sunset` and successor `Link` headers.
     `/v1/models` shows `status: deprecated` and the schedule.
   - **Console:** the playground shows a banner with a one-click switch to the replacement.
   - **CLI:** every call to the model prints a warning on stderr. `1trade catalog` flags it.
   - **Email:** notify affected tenants at announcement and 7 days before the sunset (this is manual
     today).
3. At `SunsetAt` the model retires. Every inference endpoint answers `410 model_retired` naming the
   replacement, and `/v1/models` stops listing it. Remove the row at the next review.
