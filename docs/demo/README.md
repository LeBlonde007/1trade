# Investor demo pack — how to turn this into a video with NotebookLM

This folder is a **NotebookLM source pack** for an investor video grounded in the *current* product
(platform-first) and the *real* UI. NotebookLM reads sources you upload and generates a **Video
Overview** (narrated slides) or **Audio Overview** (podcast-style); you steer it with a focus
prompt. You upload these docs — I can't run NotebookLM for you.

## The sources (upload all three to one NotebookLM notebook)
1. **`INVESTOR_NARRATIVE.md`** — the story + traction (the spine).
2. **`UI_WALKTHROUGH.md`** — the 31 real screens, grouped platform vs kept-warm.
3. **`VIDEO_SCRIPT.md`** — the 90s / 3-min arc the overview should follow.

Optional extra sources for richer grounding (already in the repo):
- `docs/re/exascale_gtm_focus_update.md` (the platform-first pivot — **authoritative**)
- `docs/plans/STATUS.md` (what's built/verified — the traction proof)
- `docs/plans/DESIGN_SYSTEM.md` (the institutional aesthetic)
- Do **NOT** upload `docs/other/exascale_pitch_and_video.md` as-is — it's the older trading-first
  pitch and will pull the narration back to "the exchange is the product." It's superseded.

## Steering prompt (paste into NotebookLM's "customize" / focus box)
> Audience: seed/Series-A investors. Tone: confident, concrete, institutional — no AI buzzwords as
> adjectives. Tell the **platform-first** story: Exascale is building the commodity market for AI
> compute, and is **already earning revenue** by selling inference + GPU compute to AI startups,
> paid in prepaid credits (no license needed). Emphasize that the **credit ledger is built,
> deployed, and verified** (append-only, hash-chained, atomic) — real software, not a mockup. Frame
> the **exchange (order book, market maker, price index) as the funded "act two," paused pending a
> license** — never as live today. Walk the real screens in this order: inference → compute →
> wallet → enterprise billing/audit → datacenter → (act-two) trade/portfolio. Treat all dollar
> figures as illustrative. Close on: "the commodity market for AI compute — already earning, before
> the exchange even opens."

## Recommended NotebookLM settings
- **Video Overview**, ~3 minutes, investor audience. (Audio Overview is a great second asset for
  warm-intro emails.)
- After generation, review for: any claim that the exchange is *live* (must say paused), and any
  number that should be flagged as illustrative.

## Best results: pair NotebookLM narration with a real screen recording
NotebookLM narrates from the docs (+ any screenshots you upload). For the most convincing investor
asset, **also screen-record the live UI** and cut it under the narration:
1. `make web` → open `http://localhost:3000` (mock-data mode; the candles tick and burn meters move).
2. Record at 1440px following `VIDEO_SCRIPT.md` (drive the `/onboarding/tour` persona tour).
3. Upload key screenshots to NotebookLM so the Video Overview shows real screens, or edit the
   recording under the generated audio.

## Before sending to real investors
- Replace every mock number (overpay %, TAM/SAM/SOM, the ask, milestones).
- Keep the "exchange paused / demo" ribbon visible on all trading shots.
- Have counsel review the credits-as-prepaid-service-units framing (tracked as feature F22).
