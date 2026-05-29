# Exascale — Hardware & Datacenter Integration Primer

> Pairs with `exascale_beginner_primer.md`. That one taught you the
> trading / web stack. This one teaches you the *physical* and *protocol*
> side: what GPUs actually are, how a datacenter plugs their hardware
> into the venue, and the reasoning behind every architectural choice.
>
> Read top-to-bottom in ~45 minutes. No prior systems knowledge assumed.

---

## 1. GPU hardware — the terms

### What is a GPU vs a CPU?

A **CPU** (Intel / AMD / Apple Silicon) is a few very smart cores. Great
at sequential logic — running your OS, web servers, business code.

A **GPU** (NVIDIA / AMD / Intel) is *thousands* of dumber cores that all
do the same simple math operation in parallel. Originally built for
shading pixels in video games (millions of pixels = millions of identical
math ops). Turns out: training and running AI models is also "millions of
identical math ops on big matrices." So the gaming hardware accidentally
became the AI hardware.

The key number on a GPU is **FLOPS** — Floating-Point Operations Per
Second. An H100 does ~1,000 trillion (1 *peta*flop) FP16 operations per
second. A high-end CPU does ~0.5 trillion. The GPU is ~2,000× faster
for the right workload.

### The hardware vocabulary

| Term | What it means |
|---|---|
| **GPU** | The chip itself. |
| **VRAM** | The GPU's dedicated memory (separate from your computer's RAM). AI models live here while running. If a model doesn't fit in VRAM, you can't run it on that GPU. |
| **HBM** | "High-Bandwidth Memory" — the fancy stacked VRAM modern AI GPUs use. H100 has 80 GB of HBM3. |
| **Tensor cores** | Specialized circuits inside the GPU that do matrix multiplications very fast. Modern AI = matrix multiplications, so this is the headline feature. |
| **CUDA** | NVIDIA's programming layer. Software talks to GPUs via CUDA. If you've heard "NVIDIA's moat is CUDA," this is why — everyone wrote their AI code against CUDA, so switching to AMD means rewriting. |
| **NVLink** | A super-fast cable between GPUs in the same machine. Lets 8 GPUs act as if they were one bigger GPU. |
| **InfiniBand** | A super-fast cable between *machines*. Lets a whole rack of GPUs act as a single supercomputer. Bypasses normal Ethernet to cut latency from ~50μs to ~1μs. |
| **NCCL** | NVIDIA Collective Communications Library. How multiple GPUs synchronize their work. The thing you test in the validation suite. |

### The GPUs Exascale trades

| Symbol | Hardware | VRAM | Use case | Spot price (2026 ref) |
|---|---|---|---|---|
| `H100-SPOT` | NVIDIA H100 80GB | 80 GB HBM3 | Workhorse for training + heavy inference | ~$2.99 / GPU-hour |
| `H200-SPOT` | NVIDIA H200 141GB | 141 GB HBM3e | Bigger memory, fits bigger models in one card | ~$3.84 / GPU-hour |
| `B200-SPOT` *(future)* | NVIDIA Blackwell B200 | 192 GB HBM3e | Next-gen, 2.5× faster than H100 | TBD |
| `MI300X-SPOT` *(future)* | AMD MI300X | 192 GB HBM3 | NVIDIA alternative, ROCm not CUDA | TBD |

**Why H100 is "the unit":** it's what most production LLMs train on
today. Anthropic's Claude, OpenAI's GPT-4, Meta's Llama — all H100
(or A100 / H200 variants). It's the closest the industry has to a
standard SKU, which is exactly what you need to make a commodity market.

### Why GPUs are expensive

A single H100 is **$30,000-$40,000** at retail. They're hard to
manufacture (TSMC 4N process, advanced packaging), demand massively
outstrips supply, and NVIDIA has the monopoly. A datacenter with
1,000 H100s has $30M+ in just chips, before networking, cooling,
real estate, or power.

This scarcity is *the* reason a commodity market makes sense — when
something is scarce and standardized, you trade it.

### What's a "GPU-hour"?

The unit of capacity Exascale trades. **One GPU-hour = one H100 (or
H200, etc.) running at 100% utilization for one wall-clock hour.**

If a partner has 100 H100s and rents them out 24h/day at 80%
utilization, they have `100 × 24 × 0.80 = 1,920 H100-hours` of supply
per day. At $2.99/hr that's $5,740/day gross.

---

## 2. What a datacenter physically looks like

You don't need to know the plumbing, but you need to know the shape
of the thing so the screens you're building make sense.

```
A typical AI datacenter:
                                                                 
   ┌──────────────────── Building ────────────────────┐
   │                                                  │
   │   ┌── Hall A ──┐  ┌── Hall B ──┐  ┌── Hall C ──┐ │
   │   │ 16 racks   │  │ 16 racks   │  │ 16 racks   │ │
   │   └────────────┘  └────────────┘  └────────────┘ │
   │                                                  │
   │   Each rack ≈ 8 servers tall                     │
   │   Each server has 8 GPUs (NVLinked)              │
   │                                                  │
   │   So one rack ≈ 64 GPUs                          │
   │   One hall   ≈ 1,024 GPUs                        │
   │   One building ≈ ~3,000 GPUs                     │
   └──────────────────────────────────────────────────┘
```

**Inside one rack**, the servers (called "DGX H100" boxes from NVIDIA,
or HGX-based whitelabel boxes from others) are wired:
- 8 GPUs per server, all NVLinked (talk to each other at 900 GB/s).
- Servers in the same rack are InfiniBanded together (3.2 Tb/s per
  link). 8 servers → 64 GPUs that behave like a single supercomputer.
- Racks talk to each other over more InfiniBand or RoCE Ethernet at
  the cluster level.

**Why this matters for software:** when you "launch a job on 64 H100s,"
the right datacenter assignment puts those 64 GPUs on the same
InfiniBand fabric. Putting them across two buildings = 100× slower for
training jobs because the GPUs have to exchange gradients constantly.
The venue's matching engine has to be **fabric-aware**, not just
GPU-count-aware.

**Other things datacenters care about** that show up in our UI:

- **PUE** (Power Usage Effectiveness) — `total power / IT power`.
  1.0 = perfect, every watt powers compute. Real DCs run 1.1–1.5.
  Anything > 1.6 is dated.
- **Tier I-IV** — Uptime Institute's redundancy rating. Tier III = no
  scheduled downtime for maintenance. Tier IV = also no unscheduled
  downtime (dual everything). Exascale requires Tier III minimum.
- **Region** — physical location. Latency-sensitive workloads (real-time
  inference) want the partner in the same continent as the buyer.

---

## 3. The integration — how a datacenter plugs into Exascale

This is the meaty part. You're connecting a multimillion-dollar
GPU fleet to a market venue. The integration has to be:
- **Trustless** — the venue can't trust the DC to be honest about
  utilization, fills, or hardware claims.
- **Push-based** — DCs are often behind firewalls; we can't dial in.
- **Reversible** — the DC must be able to leave the venue immediately.
- **Auditable** — every byte of capacity, every fill, every settlement
  cryptographically attestable.

### Step-by-step partner onboarding

```
DC ops engineer                    Exascale venue
─────────────────────────────────────────────────────
1. Signs partnership agreement                                 (legal)
2. Logs into /datacenter/register                              (UI)
3. Declares: 200 H100s in Reno NV, Tier III, PUE 1.18          (form)
4. Receives install token: ix_8a91…0c                         (secret)

5. SSH into one of their hosts (their normal access)
6. Runs the one-liner:
       curl -sL https://exa.sc/agent | \
         sudo SITE=pdc_7c2a INSTALL_TOKEN=ix_8a91…0c sh

7. Agent process starts:
       ┌─────────────────────────────────────────┐
       │ exascale-agent (Go binary, signed)      │
       │  ├─ probes local hardware (nvidia-smi)  │
       │  ├─ runs validation (NCCL allreduce,    │
       │  │   CUDA sanity, disk IO, net egress)  │
       │  └─ opens outbound mTLS WebSocket to    │
       │     wss://api.exascale.com/dc/v1/link   │
       └─────────────────────────────────────────┘

8. Heartbeat shows up in /datacenter/connect (live)
9. DC ops marks the host as "production" → venue starts
   routing fills to it
```

The whole thing takes ~5 minutes. The screen for it is **D1** in
`docs/exascale_exchange_demo_gaps.md`.

### The agent — what it actually does

The agent is a small (~12 MB) Go binary that runs on every GPU host
the DC contributes. It does **four jobs**, nothing more:

1. **Heartbeat.** Every 5 seconds, send `{ site, host, gpu_utilization,
   memory_used, temperature, ts }` to the venue.
2. **Capability advertisement.** On startup and every 60 seconds, send
   the host's "menu": what GPUs are present, what VRAM, what's free
   right now, what's busy, what models are pre-cached on disk.
3. **Job receiver.** When the venue assigns a fill ("run this container
   for buyer X for 90 seconds"), the agent receives the signed job
   manifest, validates the signature, pulls the container if not
   cached, launches it in an isolated cgroup, streams stderr/stdout
   back through the WebSocket, and reports completion + execution
   hash.
4. **Self-update.** If the venue ships a new agent version, the agent
   verifies the signature, downloads, and restarts itself.

That's the entire surface area. **The agent does NOT** open inbound
ports, accept SSH, run arbitrary shell, or trust anything the venue
sends without a valid Ed25519 signature.

### Protocol shape (so the audit story makes sense)

```
┌───────────────┐                            ┌────────────────────┐
│  DC HOST      │                            │  EXASCALE VENUE    │
│  (agent.go)   │                            │  (matching engine) │
└───────┬───────┘                            └─────────┬──────────┘
        │                                              │
        │ ─── connect ─────────────────────────────────│
        │     wss://api.exascale.com/dc/v1/link        │
        │     headers: Authorization: Bearer <jwt>     │
        │     mTLS cert pinned                          │
        │                                              │
        │ ─── heartbeat (5s) ──────────────────────────│
        │     { site, host, gpu_util[0..7], temp, ts } │
        │                                              │
        │ <── job_assign ─────────────────────────────  │
        │     { job_id, model_uri, container_digest,   │
        │       buyer_id_masked, qty_credits,           │
        │       sig: ed25519 (signed by venue),        │
        │       deadline_ms }                           │
        │                                              │
        │ ─── job_started ─────────────────────────────│
        │     { job_id, started_at, container_hash }   │
        │                                              │
        │ ─── job_chunk (streaming) ───────────────────│
        │     { job_id, seq, payload, exec_hash }       │
        │                                              │
        │ ─── job_complete ────────────────────────────│
        │     { job_id, finished_at, total_tokens,     │
        │       wall_time_ms, exit_code, attest_hash } │
        │                                              │
        │ <── settle_ack ─────────────────────────────  │
        │     { job_id, settlement_id, audit_block }   │
        │                                              │
```

Every message is hash-chained to the previous one in that job. The
final settlement writes a block to the audit chain (`#14,287`-style
block numbers you see in `/enterprise/audit`).

---

## 4. Why these choices — and what we ruled out

This is where most of the "why" lives. Beginner readers should
understand each tradeoff.

### Why an agent, not SSH?

**SSH approach (rejected):** the venue holds the DC's SSH keys, dials
in when it needs to run a job. NVIDIA's NGC and a few academic
clusters work this way.

**Why we didn't:**
- DCs *hate* giving outsiders SSH credentials. It's a security
  non-starter for almost every enterprise.
- SSH is connection-oriented; the venue would need to keep N
  connections open to N hosts. Doesn't scale to thousands of nodes.
- SSH dispatch can't carry signed job manifests cleanly — you'd
  layer signing on top, which means two protocols.
- Inbound firewall holes are a procurement nightmare.

**Agent + outbound WebSocket (chosen):**
- DC owns the install. Can revoke at any second (just `systemctl stop
  exascale-agent`).
- Outbound only — works through any firewall that allows HTTPS.
- Long-lived WebSocket holds the connection so dispatch latency is
  ~5ms, not the 200ms a new HTTPS handshake would cost.
- Signed binary + signed messages = the DC can prove (and the venue
  can prove) what was sent.

### Why WebSocket, not gRPC or REST?

- **REST polling** would mean the DC asks every few seconds "any jobs
  for me?" — wastes bandwidth and adds 1-5 seconds of dispatch
  latency. Killed for trading-style workloads.
- **gRPC** is a great fit *if* both ends are easy to upgrade.
  Real-world DCs run mixed Linux distros with patchy http/2 stacks.
  gRPC also doesn't play nicely with most corporate firewall proxies.
- **WebSocket** is just HTTPS upgraded. Goes through every proxy,
  carries binary frames, low-latency. Battle-tested for trading
  (most crypto exchanges use it).

### Why containers (not VMs, not bare metal)?

When a job arrives, the agent needs to launch the buyer's workload
in a way that's:
- isolated (the buyer can't see other tenants' data),
- fast to start (every 100ms matters when you're billing by the second),
- deterministic (the same model on the same hardware should produce
  the same result for the same input — audit story).

| Option | Isolation | Startup | Verdict |
|---|---|---|---|
| Bare metal | None | Instant | Killed — multitenancy impossible |
| Full VM (KVM) | Strong | 10-30s | Killed — startup too slow for billing |
| Container (Docker, containerd) | Medium | 1-3s | **Chosen** |
| nspawn / Firecracker microVM | Strong | <1s | Future option for higher-trust tiers |

Containers run inside a cgroup that pins exactly the GPUs assigned and
nothing else (via `nvidia-container-runtime`). The buyer's job sees
only its own GPUs and a clean filesystem.

### Why pre-cache model artifacts on the DC?

**Cold-start problem:** the first request to "run Llama-3-70B" needs
to copy ~140 GB of weights from somewhere to the GPU. Over the public
internet that's 5-20 minutes. The buyer paid for compute, not
download time.

**Solution (D2 in the gaps doc):** the venue pre-fetches the top-N
popular models to every active partner site overnight. The agent
reports `models_cached: ['llama3-70b', 'claude-opus-4-7', …]`. The
matching engine only routes jobs to sites that already have the
model. First-token latency drops from minutes to ~500ms.

This is also why the **datacenter library** screen (`/datacenter/library`,
not yet built) needs storage-pressure gauges — pre-cached models cost
disk.

### Why "mint" capacity instead of just having it appear?

A datacenter has, say, 10,000 GPU-hours of free time next month. They
don't have to sell *all* of it on the venue — they might want to keep
some for internal jobs or private customers.

**The mint flow (D3 in the gaps doc)** is the partner saying "I
hereby commit 4,000 of those 10,000 hours as `H100-FWD-30D`
contracts, floor price $2.85/hr." Until they mint, the venue doesn't
know about that capacity. After they mint, those credits exist as
tradeable supply.

This is the same primitive as a company issuing bonds: the asset
existed (future cashflows), but the *tradeable security* didn't until
issuance.

### Why hash-chain everything?

Trading venues have to convince regulators and auditors that the
record is tamper-proof. Two cheap protections:

1. **Per-event SHA-256.** Every audit event hashes the previous event
   + its own payload. Mutating event #14,277 invalidates every event
   after it, instantly detectable.
2. **Periodic Merkle root publishing.** Every 50 blocks, the merkle
   root of the chain is published to a public-write-once medium (an
   S3 bucket with object lock, or an actual blockchain for the
   paranoid). External auditors can verify the chain ex-post.

This is what you see in `/enterprise/audit` — every row has a block
# and hash, every detail panel shows the prev hash, signature, and
"verify against ledger" button.

### Why mTLS, not OAuth?

OAuth is great for human-driven web logins. mTLS (mutual TLS,
certificate-based) is better for machine-to-machine because:
- The DC's host *is* the identity, not a user-typeable secret that
  can leak. Cert lives on disk in `/etc/exascale/agent.crt`.
- Cert rotation is automatable; no human re-typing tokens.
- The venue can revoke a single host's cert without affecting the
  rest of the site (different from invalidating an OAuth client).

The install token (`ix_8a91…0c` in step 4 above) is a one-time
bootstrap secret. After the first successful link, the agent gets
its long-term mTLS cert and the install token becomes useless.

### Why Ed25519 signatures, not RSA?

- Smaller (64-byte signatures vs 256+ for RSA-2048).
- Faster to verify on GPU hosts that have plenty of CPU spare but
  hate doing crypto work that contends with the workload.
- Modern, no known weaknesses, and what every new crypto protocol
  picks. (The other valid pick is BLS, useful only if you need
  signature aggregation — we don't for v1.)

---

## 5. The full economic flow (one diagram)

How $1 of buyer money becomes $0.85 of partner revenue:

```
Buyer (AI Co) ─┐
$1 USD wire    │ deposit → /wallet
               ▼
        ┌──────────────────┐
        │ cash leg: $1.00  │
        └──────────────────┘
               │ buys 995 AI credits at $0.001005 (T2 in gaps doc)
               ▼
        ┌──────────────────┐
        │ credit leg: 995  │
        └──────────────────┘
               │ submits inference job (A2)
               ▼
        ┌──────────────────┐  matching engine
        │   venue queue    │  picks best fill
        └──────────────────┘  (DC with model cached, low latency)
               │
               ▼
   DC partner (Northstar Reno)
   runs the container, returns the result
   wall_time × $rate = $0.95 due
               │
               ▼
        ┌──────────────────┐  fee split:
        │  venue fee 10%   │   $0.10 → Exascale
        │  partner net 90% │   $0.85 → Northstar (queued to weekly wire)
        └──────────────────┘
               │
               ▼
        Audit chain block #14,287
        (every step above attested)
```

Every screen in the project sits somewhere on this diagram. Once you
can find your screen on the diagram, you understand its purpose.

---

## 6. The five protocols you'll see referenced

| Protocol | Where you see it | What it does |
|---|---|---|
| **WSS** (WebSocket over TLS) | Agent ↔ venue link | Long-lived push channel |
| **mTLS** | Agent ↔ venue auth | Cert-based identity |
| **Ed25519** | Job manifests, audit chain | Lightweight signatures |
| **SHA-256** | Per-block chain hashing | Detect tampering |
| **gRPC** | Internal venue services only | High-throughput RPC |

You won't write any of this in the frontend codebase. But when you're
building screens like the audit log or the agent install page, you're
visualizing artefacts these protocols produce — hashes, signatures,
heartbeats, settlement blocks. Knowing what they *mean* helps you
present them well.

---

## 7. Common questions a beginner asks

**Q: Can a datacenter cheat by reporting fake capacity?**
Yes, in theory. We mitigate by:
- Validation suite at onboarding (real NCCL allreduce — can't fake).
- Spot-check jobs that the venue knows the expected hash for.
- Buyer reputation feedback (job didn't return the model output we
  expected → that fill is disputed → partner's reputation drops).
- Hardware attestation roadmap: NVIDIA's "GPU attestation" feature on
  H100+ allows the GPU itself to cryptographically prove what code it
  ran. Slow rollout, but it's coming.

**Q: What stops a buyer's job from being malicious to the host?**
Containers + a strict runtime policy: no network (other than back to
the venue), no host filesystem mount, GPU access limited to the
allocated cards, CPU/RAM cgroup limits. Buyers also can't pick
arbitrary container images — only signed published artefacts from the
venue catalog (or their own images that pass a security scan).

**Q: How does the venue handle GPU failure mid-job?**
The agent reports failure (`job_complete` with `exit_code: -1` and a
diagnostic). The job re-enters the queue and is dispatched to another
host. The buyer's clock pauses (they're not billed for the failed
attempt). The original DC eats the loss for the failed window and
their reputation takes a small hit if it happens often.

**Q: Why not just put this on a blockchain?**
We use a hash-chain (block-numbered, signature-attested) — same audit
properties, 1000× cheaper than running on Ethereum, no token speculation,
no MEV. We publish the Merkle root to a public medium periodically
for ex-post verification. That gives us the trust without the
overhead.

**Q: What about open-source models? Who owns the rights?**
The buyer picks the model from a catalog (`/inference`). Each model
entry tracks its license (Apache 2.0, Llama Community License, etc.)
and the venue blocks commercially-restricted models for buyers without
the right enterprise tier. Partner DCs aren't on the hook for IP — the
catalog ensures only properly-licensed weights ever land on their disks.

---

## 8. If you remember nothing else

1. **H100 = the standard GPU SKU.** $30K each, 80 GB VRAM, what most
   AI runs on today. We trade GPU-*hours* of it.
2. **DCs install an agent**, not give us SSH. The agent dials out
   over WSS+mTLS so DC firewalls don't have to open.
3. **Every job is signed** end-to-end and **every settlement** writes
   an audit block. That's how the venue convinces auditors it's not
   making things up.
4. **Models are pre-cached** at partner sites so cold-start latency
   doesn't kill the buyer experience.
5. **Containers, not VMs, not bare metal.** Sub-second startup with
   strong-enough isolation.

When you build a screen that references hashes, agents, fabric, or
settlements — re-read the relevant section here to make sure the
visual matches the system.
