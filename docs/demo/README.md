# Investor demo pack — how to turn this into a video with NotebookLM

This folder is a **NotebookLM source pack** for an investor video grounded in the *current* product
(platform-first) and the *real* UI. NotebookLM reads sources you upload and generates a **Video
Overview** (narrated slides) or **Audio Overview** (podcast-style); you steer it with a focus
prompt. You upload these docs — I can't run NotebookLM for you.

## The sources (upload all three to one NotebookLM notebook)
1. **`INVESTOR_NARRATIVE.md`** — the story + traction (the spine).
2. **`UI_WALKTHROUGH.md`** — the 31 real screens, grouped platform + exchange.
3. **`VIDEO_SCRIPT.md`** — the 90s / 3-min arc the overview should follow.

**Upload ONLY these three.** Do **not** add the internal build docs as NotebookLM sources — several
of them (`docs/re/exascale_gtm_focus_update.md`, `docs/plans/STATUS.md`, `docs/plans/MANAGEMENT_PLAN.md`,
`docs/plans/DESIGN_SYSTEM.md`) describe the exchange as **paused / kept-warm / license-gated** for
internal build sequencing, and NotebookLM would surface that in the video — the opposite of the
intended framing. Likewise do **not** upload `docs/other/exascale_pitch_and_video.md` (the older
trading-first pitch). The three demo-pack docs are written to be the complete, self-contained source.

## Steering prompt (paste into NotebookLM's "customize" / focus box)
> Audience: seed/Series-A investors. Tone: confident, concrete, institutional — no AI buzzwords as
> adjectives. Lead with the **platform**: Exascale is building the commodity market for AI compute,
> and is **already earning revenue** by selling inference + GPU compute to AI startups, paid in
> prepaid credits. Emphasize that the **credit ledger is built, deployed, and verified**
> (append-only, hash-chained, atomic) — real software, not a mockup. Present the **exchange (order
> book, market maker, price index) as a real, working part of the product** — the market the
> platform powers. **Do NOT describe the exchange as paused, coming-soon, mock, or pending a
> license, and do NOT mention licensing or trading being on hold.** (Also do not claim it is
> processing live real-money trades or that the venue is licensed — describe what it is and does.)
> Walk the real screens in this order: inference → compute → wallet → enterprise billing/audit →
> datacenter → trade/portfolio. Treat all dollar figures as illustrative. Close on: "Exascale — the
> commodity market for AI compute."

## Recommended NotebookLM settings
- **Video Overview**, ~3 minutes, investor audience. (Audio Overview is a great second asset for
  warm-intro emails.)
- After generation, review for: no "paused / coming-soon / pending-license" language about the
  exchange (the video presents it as part of the product); also no claim that it's processing live
  real-money trades or is licensed; and every dollar figure flagged as illustrative.

## Best results: pair NotebookLM narration with a real screen recording
NotebookLM narrates from the docs (+ any screenshots you upload). For the most convincing investor
asset, **also screen-record the live UI** and cut it under the narration:
1. `make web` → open `http://localhost:3000` (mock-data mode; the candles tick and burn meters move).
2. Record at 1440px following `VIDEO_SCRIPT.md` (drive the `/onboarding/tour` persona tour).
3. Upload key screenshots to NotebookLM so the Video Overview shows real screens, or edit the
   recording under the generated audio.

## Before sending to real investors
- Replace every mock number (overpay %, TAM/SAM/SOM, the ask, milestones).
- The video presents the exchange as part of the product (no paused/coming-soon caveats). Keep the
  voice-over to *what it is and does* — avoid asserting it's processing live real-money trades or is
  licensed, so a diligence call doesn't catch a claim that isn't yet true.
- Have counsel review the credits-as-prepaid-service-units framing (tracked as feature F22).
