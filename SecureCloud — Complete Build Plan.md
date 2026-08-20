# SecureCloud — Complete Build Plan
### OpenChoreo Security Deployment Lab
**Start:** Sat 8 Aug 2026 · **Deadline:** Fri 28 Aug 2026 · **Working days:** 21

---

## 0. Read this first

### 0.1 The honest reality check

You have 21 days, coursework running in parallel, and you have never built a
microservices + Kubernetes + CI/CD project before. That is a real constraint,
not a pep-talk problem.

Three things follow from it:

1. **The business logic is not the project.** Your Auth/Scan/Notification
   services should be almost embarrassingly simple. The impressive part is the
   platform story: how they are built, deployed, connected, secured, and
   observed. Do not spend Day 12 improving your scanner.
2. **The riskiest step is Day 1, not Day 18.** If OpenChoreo will not run on
   your laptop, the entire plan changes. So Phase 0 is "prove the platform runs"
   — *before* you write a single line of application code. Most students do this
   backwards and discover the problem with four days left.
3. **A finished small project beats an unfinished big one.** Section 11 gives you
   explicit *cut lines* — what to drop, in what order, if you fall behind.

### 0.2 The one thing that decides whether this succeeds

For a WSO2 application, the artefact that gets read is your **README and
architecture doc**, and the thing that gets watched is your **5-minute demo
video**. Nobody clones your repo and runs it.

So: Phase 5 (docs + demo) is *not* optional polish. If you have to choose
between "add Postgres" and "record a good demo video," record the video.
Budget for it and protect those days.

### 0.3 How to use this document

- Work top to bottom. Each phase has a **Definition of Done**. Do not start the
  next phase until the current one is done — half-finished phases are how
  projects die.
- Every task maps to a **GitHub issue** (Section 9) and a **commit** (Section 10).
- Section 8 is a verified command/YAML reference. Trust it over blog posts.

---

## 1. Day 0 gate — check this in the next 30 minutes

Before planning anything, verify your machine can run this. Do it now.

### 1.1 Hardware requirement — and the 8 GB strategy

There are **two different requirements**, and conflating them is what makes this
look impossible on a modest machine:

| Install | Requirement | Covers |
|---|---|---|
| `./install.sh --version v1.2.2` | **4 GB RAM / 2 CPUs** | Control plane, data plane, gateway, Backstage portal — everything in Phases 0–3 |
| `+ --with-build --with-observability` | **8 GB RAM / 4 CPUs** | Adds the Workflow Plane (Argo CI) and Observability Plane — needed only in Phase 4 |

**This machine has 8 GB total.** That rules out running the heavy install
locally, but it does *not* rule out the project. Roughly 70% of the work — all
of Phases 0 through 3, which is the entire application and the whole
deploy/connect/promote story — runs fine on the 4 GB base install.

So the plan splits execution:

```text
Phases 0–3   (8–21 Aug)   →  LOCAL, base install, no flags
                             Build, deploy, connect, promote. 4 GB.

Phase 4      (22–24 Aug)  →  BORROWED BIG MACHINE, full install
                             CI/CD + observability. 16 GB.
                             Manifests are just YAML in your repo —
                             `kubectl apply -f platform/` and you're back.

Phase 5      (25–27 Aug)  →  LOCAL. Writing docs needs no cluster.
```

Your platform manifests and container images live in Git and GHCR, so moving
between machines costs about ten minutes, not a day. This is worth internalising:
the fact that the whole system is portable across clusters is *itself* a
demonstration of what the platform gives you. Say so in your architecture doc.

### 1.1a Where to run Phase 4

Ranked for a University of Moratuwa student:

| Option | Cost | Specs | Notes |
|---|---|---|---|
| **GitHub Codespaces** (Student Pack) | **Free** | 4-core / 16 GB | 180 core-hours/month free for verified students = ~45 h on a 4-core box. Phase 4 needs ~30 h. Sidesteps international card payment entirely. **Recommended.** |
| **University lab machine** | Free | Varies | Check whether CSE lab PCs have 16 GB. Zero setup friction if so. |
| **Contabo VPS (Singapore)** | ~€5–7/mo | 4 vCPU / 8 GB+ | Singapore datacentre → good latency from Sri Lanka. Monthly billing. |
| **Hetzner CX32** | €6.80/mo, €0.0113/hr | 4 vCPU / 8 GB | Hourly billing — 3 days ≈ €0.80. Cheapest by far, but EU-only, so ~150–200 ms SSH latency. |
| Oracle Cloud Always Free | Free | 4 OCPU / 24 GB ARM | ARM64. The docs warn about buildpack problems on ARM, and buildpacks are exactly what Phase 4 exercises. **Avoid for this.** |

**Set up the GitHub Student Developer Pack today** if you haven't
(education.github.com/pack) — verification can take a few days, and you need it
live by 22 Aug.

### 1.1b The port-forwarding gotcha (read this before you rent anything)

OpenChoreo's URLs are hostname-routed: `openchoreo.localhost:8080` and
`*.openchoreoapis.localhost:19080`. The Envoy gateway matches on the `Host`
header. This breaks in the obvious remote setups and works in the non-obvious
ones:

| Setup | Works? | Why |
|---|---|---|
| Codespaces **in a browser tab** | ✗ | Ports get rewritten to `https://xxx-19080.app.github.dev`; wrong `Host` header, gateway won't route |
| Codespaces via **VS Code Desktop** | ✓ | Forwards to real `localhost:19080`, `Host` header preserved |
| VPS via **SSH `-L` forwarding** | ✓ | Same reason |
| VPS accessed by public IP | ✗ | No `.localhost` resolution, no matching `Host` |

For a VPS:

```bash
ssh -L 8080:localhost:8080 -L 19080:localhost:19080 user@your-vm
```

Then browse `http://openchoreo.localhost:8080` on your own machine as normal.
Subdomains of `.localhost` resolve to 127.0.0.1, so the wildcard hostnames work
through the tunnel.

### 1.2a Verify what Docker actually has

```bash
docker run --rm alpine:latest sh -c "echo 'Memory:'; free -h; echo; echo 'CPU Cores:'; nproc"
```

Target for the base install: **≥ 4.5 GB and ≥ 2 cores** visible to Docker. If
you're short, see §1.4 — on an 8 GB machine, how you run Docker matters more
than how much RAM you have.

### 1.4 Squeezing the base install onto 8 GB

Do all of these before Phase 0:

**If on Linux** — you're in the best position. Native Docker, no VM tax. Just:
- Work in a TTY or a lightweight WM during cluster sessions
- Close Chrome/Chromium while the cluster runs (it can hold 2–3 GB alone)
- Add swap so pressure causes slowness rather than OOM kills:
  ```bash
  sudo fallocate -l 8G /swapfile && sudo chmod 600 /swapfile
  sudo mkswap /swapfile && sudo swapon /swapfile
  echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
  ```

**If on Windows** — this is your situation. Docker Desktop is the wrong tool at
8 GB; its VM plus Windows-side backend services cost 1–2 GB you cannot spare.
**Full setup in Appendix A** — do that before anything else.

**If on macOS** — use Colima rather than Docker Desktop, same reasoning:
```bash
colima start --cpu 4 --memory 5
```

**Universally:**
- Run `./uninstall.sh` when you finish a session; don't leave the cluster idling
- Never run Docker Compose (Phase 1) and the k3d cluster at the same time
- If a pod is stuck `Pending` with insufficient-memory events, that's the signal
  to stop and free RAM rather than to wait

### 1.5 Go/no-go

| Result of the base install | Action |
|---|---|
| Installs clean, sample app + Backstage load | **Green.** Proceed with the split plan. |
| Installs but pods restart or the machine is unusable | Apply §1.4 fully, retry once. Still bad → move Phases 2–4 to Codespaces entirely. |
| Won't install at all | Do everything on Codespaces. The plan is unchanged; only the location is. |

### 1.2 Prerequisites checklist

```bash
docker --version      # Engine 26.0+ recommended
git --version
go version            # 1.22+
node --version        # 20+
```

If Docker is missing or old, fix that first. Everything else can wait.

### 1.3 The 90-minute smoke test (do this today)

This is Issue #1. Run the official quick start end to end **before** committing
to the project. Note the install command has **no flags** — that is deliberate,
and it is what makes this fit in 8 GB:

```bash
# 1. Start the pre-configured dev container
docker run --rm -it --name openchoreo-quick-start \
  --pull always \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --network=host \
  ghcr.io/openchoreo/quick-start:v1.2.2

# 2. Inside the container — install OpenChoreo (creates a k3d cluster)
#    BASE INSTALL ONLY. Do not add --with-build or --with-observability here;
#    those double the memory footprint and belong to Phase 4 on a bigger box.
./install.sh --version v1.2.2

# 3. If anything is stuck in 'pending', wait, then:
./check-status.sh

# 4. Deploy the sample app
./deploy-react-starter.sh
```

Then open:
- Sample app: `http://react-starter-development-default.openchoreoapis.localhost:19080`
- Backstage portal: `http://openchoreo.localhost:8080/`
  (login `admin@openchoreo.dev` / `Admin@123`)

**Gate:** If both URLs load, you are green — proceed. If not, spend the rest of
today debugging *this only*. Do not start writing services. Ask in the CNCF
Slack `#openchoreo` channel — the maintainers are active and this alone is worth
doing (see Section 13).

---

## 2. Scope — locked

### 2.1 What you are building

A three-service security platform, deployed and operated entirely through
OpenChoreo.

```text
                        User
                          │
                          ▼
                 Web Dashboard (Next.js)
                  ClusterComponentType:
                  deployment/web-application
                          │
              ┌───────────┴────────────┐
              ▼                        ▼
       Auth Service (Go)        Scan Service (Go)
       deployment/service       deployment/service
       - POST /register         - POST  /scan
       - POST /login            - GET   /scan/{id}
       - GET  /verify           - GET   /scans
                                        │
                                        ▼
                            Notification Service (Go)
                            deployment/service
                            - POST /notifications
                            - GET  /notifications

            All built, deployed, connected, and observed
                          via OpenChoreo v1.2.2
```

### 2.2 Deliberately simple business logic

Keep these *tiny*. Total application code target: **under 1,200 lines** across
all three services.

**Auth Service**
- `POST /register` → store `{email, bcrypt(password)}`
- `POST /login` → verify, return a signed JWT (HS256, secret from a Kubernetes secret)
- `GET /verify` → validate the `Authorization: Bearer` header, return the claims

**Scan Service**
- `POST /scan` with `{"target": "example.com"}` → create a scan record, run checks async, return `{"id": "...", "status": "queued"}`
- The "security scan" is three cheap, real checks:
  1. Does the host serve HTTPS, and is the TLS cert valid / not expired?
  2. Are `Strict-Transport-Security`, `X-Content-Type-Options`, and `Content-Security-Policy` headers present?
  3. Does HTTP redirect to HTTPS?
- Produce a score out of 100 and a list of findings.
- On completion → `POST` to the Notification Service.

**Notification Service**
- `POST /notifications` → store `{scan_id, message, created_at}`
- `GET /notifications` → list them

That is the whole product. Resist the urge to add more.

### 2.3 Storage decision

Use **SQLite, one database per service** (`modernc.org/sqlite`, pure Go — no
CGO, so your Docker images stay simple and small).

Why not Postgres: it adds a stateful dependency, provisioning complexity, and a
whole new failure mode into a 21-day window. Database-per-service is also the
*architecturally correct* microservices pattern, so this is a defensible design
choice, not a shortcut — say exactly that in your architecture doc.

Add persistence via an OpenChoreo **persistent-volume Trait** (Phase 3, Issue
#22). If the trait proves fiddly, run ephemeral for the demo and note it as a
known limitation. Postgres via the OpenChoreo `Resource` abstraction is a
**stretch goal only** — see cut line C1.

### 2.4 Explicitly out of scope

Write these in your README under "Non-goals". Naming what you deliberately
excluded reads as engineering maturity, not as gaps.

- Multi-cluster / multi-region data planes
- Production-grade auth (refresh tokens, OAuth2 providers, RBAC in-app)
- Real vulnerability scanning (CVE databases, port scanning, fuzzing)
- Horizontal autoscaling and load testing
- Email/SMS delivery in the Notification Service (it records, it doesn't send)

---

## 3. Tech stack and why

| Layer | Choice | Reason |
|---|---|---|
| Services | **Go 1.22**, stdlib `net/http` + `chi` router | You already know Go from SecureScan. Tiny static binaries → fast Docker builds → fast CI. Go is also WSO2-adjacent (OpenChoreo itself is Go). |
| Storage | **SQLite** (`modernc.org/sqlite`) | Zero-dependency, no CGO, database-per-service |
| Auth | **JWT HS256** (`golang-jwt/jwt/v5`) | Simple, stateless, works across services |
| Dashboard | **Next.js 14+ (App Router)** | You already know it from OnTime. Reuses existing skill instead of spending learning budget. |
| Container build | **Dockerfile** (multi-stage) | More predictable than Buildpacks. You control the outcome. Buildpacks become an optional extra (Issue #31). |
| Platform | **OpenChoreo v1.2.2** on k3d | Pin the version. Do not float. |
| CI (repo) | **GitHub Actions** | Lint/test/build on every PR |
| CD (platform) | **OpenChoreo Workflow Plane** | Argo Workflows under the hood; this is the "built-in CI/CD" story |
| Observability | **OpenChoreo Observability Plane** | Logs, metrics, alerts through the platform |

> **Pin everything to `v1.2.2`.** OpenChoreo moves fast and the CRD schema
> changed significantly at v1.0. When you search for help, use the version
> selector on `openchoreo.dev/docs` — a v0.x tutorial will give you YAML that
> no longer validates. This is the single most common way to lose a day.

---

## 4. Repository structure

One repository, monorepo layout. For a solo 21-day project, multi-repo costs you
time and buys you nothing.

```text
securecloud/
├── .github/
│   ├── ISSUE_TEMPLATE/
│   │   ├── task.md
│   │   └── bug.md
│   ├── workflows/
│   │   ├── ci.yaml                 # lint + test + build on every PR
│   │   └── images.yaml             # build & push images to GHCR on main
│   └── pull_request_template.md
├── docs/
│   ├── architecture.md             # ← the doc that gets read
│   ├── openchoreo-notes.md         # your learning log (see §7 Phase 0)
│   ├── runbook.md                  # how to run/operate it
│   ├── adr/
│   │   ├── 0001-monorepo.md
│   │   ├── 0002-sqlite-per-service.md
│   │   └── 0003-docker-over-buildpacks.md
│   └── images/
│       ├── architecture.png
│       ├── backstage-catalog.png
│       └── observability.png
├── services/
│   ├── auth/
│   │   ├── cmd/server/main.go
│   │   ├── internal/{handler,store,token}/
│   │   ├── Dockerfile
│   │   └── go.mod
│   ├── scan/
│   │   ├── cmd/server/main.go
│   │   ├── internal/{handler,store,checks,notify}/
│   │   ├── Dockerfile
│   │   └── go.mod
│   └── notification/
│       ├── cmd/server/main.go
│       ├── internal/{handler,store}/
│       ├── Dockerfile
│       └── go.mod
├── web/                            # Next.js dashboard
│   ├── app/
│   ├── Dockerfile
│   └── package.json
├── platform/                       # ← all OpenChoreo manifests
│   ├── project.yaml
│   ├── auth/{component.yaml,workload.yaml}
│   ├── scan/{component.yaml,workload.yaml}
│   ├── notification/{component.yaml,workload.yaml}
│   ├── web/{component.yaml,workload.yaml}
│   └── secrets/secret-reference.yaml
├── deploy/
│   └── compose.yaml                # local dev only
├── scripts/
│   ├── bootstrap.sh                # install OpenChoreo
│   ├── deploy-all.sh
│   └── smoke-test.sh
├── Makefile
├── .gitignore
├── LICENSE                         # Apache-2.0
└── README.md
```

---

## 5. GitHub setup (do this in Phase 0, ~1 hour)

### 5.1 Create the repo

```bash
gh repo create securecloud --public \
  --description "Cloud-native security platform built on OpenChoreo — microservices, CI/CD, and observability on Kubernetes"
```

Add topics: `openchoreo`, `kubernetes`, `microservices`, `platform-engineering`,
`golang`, `devops`, `cncf`, `internal-developer-platform`.

### 5.2 Labels

```bash
gh label create "phase-0" --color "0E8A16" --description "Foundations"
gh label create "phase-1" --color "1D76DB" --description "Services"
gh label create "phase-2" --color "5319E7" --description "First deploy"
gh label create "phase-3" --color "B60205" --description "Full system"
gh label create "phase-4" --color "D93F0B" --description "CI/CD + observability"
gh label create "phase-5" --color "FBCA04" --description "Docs + demo"

gh label create "service:auth" --color "C2E0C6"
gh label create "service:scan" --color "C2E0C6"
gh label create "service:notification" --color "C2E0C6"
gh label create "service:web" --color "C2E0C6"
gh label create "platform" --color "BFD4F2" --description "OpenChoreo manifests"
gh label create "infra" --color "BFD4F2"
gh label create "docs" --color "D4C5F9"
gh label create "blocked" --color "000000"
gh label create "stretch" --color "EEEEEE" --description "Cut first if behind"
```

### 5.3 Milestones

Create these with due dates — GitHub will then show you burn-down progress,
which is genuinely motivating and screenshots well.

```bash
gh api repos/:owner/securecloud/milestones -f title="M0 · Foundations"       -f due_on="2026-08-10T23:59:59Z" -f description="OpenChoreo running locally, repo scaffolded, K8s fundamentals understood"
gh api repos/:owner/securecloud/milestones -f title="M1 · Services"          -f due_on="2026-08-14T23:59:59Z" -f description="Three Go services + dashboard running locally via Docker Compose"
gh api repos/:owner/securecloud/milestones -f title="M2 · First Deploy"      -f due_on="2026-08-17T23:59:59Z" -f description="Auth Service live on OpenChoreo from a pre-built image"
gh api repos/:owner/securecloud/milestones -f title="M3 · Full System"       -f due_on="2026-08-21T23:59:59Z" -f description="All services deployed, connected, and reachable through the gateway"
gh api repos/:owner/securecloud/milestones -f title="M4 · CI/CD + Observability" -f due_on="2026-08-24T23:59:59Z" -f description="Source-to-deploy pipeline and observability wired up"
gh api repos/:owner/securecloud/milestones -f title="M5 · Ship"              -f due_on="2026-08-27T23:59:59Z" -f description="README, architecture doc, diagrams, demo video"
```

### 5.4 Project board

```bash
gh project create --owner @me --title "SecureCloud"
```

Columns: `Backlog` → `This Phase` → `In Progress` → `In Review` → `Done`.
Rule: **max 2 items in "In Progress"**. This is the discipline that stops you
from having six half-finished things on Aug 27.

### 5.5 Issue template

`.github/ISSUE_TEMPLATE/task.md`:

```markdown
---
name: Task
about: A unit of work
title: ''
labels: ''
---

## Goal
<!-- One sentence: what is true after this is done? -->

## Acceptance criteria
- [ ]
- [ ]

## Notes
<!-- Links to docs, gotchas, decisions -->
```

### 5.6 PR template

`.github/pull_request_template.md`:

```markdown
## What
<!-- One paragraph -->

## Why
<!-- Link the issue: Closes #N -->

## How I verified
<!-- Commands run, screenshots, curl output -->

## Checklist
- [ ] Tests pass locally (`make test`)
- [ ] Manifests validated (`kubectl apply --dry-run=server`)
- [ ] Docs updated if behaviour changed
```

### 5.7 Branch protection (optional but looks good)

Require a PR to merge to `main`, require CI to pass. Yes, even solo. It forces
the discipline and it makes your repo look like a team repo.

---

## 6. Git workflow

### 6.1 Branch naming

```text
feat/auth-jwt-issuing
feat/scan-tls-checks
fix/scan-nil-pointer-on-timeout
chore/ci-go-matrix
docs/architecture-diagram
platform/notification-component
```

### 6.2 Conventional Commits

```text
<type>(<scope>): <subject>

[optional body — explain WHY, not what]

[optional footer]
```

**Types:** `feat` · `fix` · `docs` · `chore` · `refactor` · `test` · `ci` · `build` · `perf`

**Scopes:** `auth` · `scan` · `notification` · `web` · `platform` · `ci` · `docs` · `repo`

Good:
```text
feat(scan): add TLS certificate expiry check

Scans now flag certificates expiring within 30 days as a medium
finding. Uses crypto/tls with a 5s handshake timeout so a dead host
cannot stall the scan worker.

Closes #14
```

Bad: `update stuff`, `fix`, `wip`, `asdf`

### 6.3 The daily loop

```bash
git checkout main && git pull
git checkout -b feat/scan-tls-checks

# ... work, committing in small logical chunks ...
git add -p
git commit -m "feat(scan): add TLS certificate expiry check"

git push -u origin feat/scan-tls-checks
gh pr create --fill --milestone "M1 · Services"
# CI runs → merge
gh pr merge --squash --delete-branch
```

**Commit at least once every working day.** A green contribution graph across
21 consecutive days is itself a signal to a reviewer.

---

## 7. The phases

---

### PHASE 0 — Foundations
**Sat 8 – Mon 10 Aug · Milestone M0**

> **Goal:** Prove the platform runs, understand what you're looking at, and have
> a repo ready. Zero application code this phase.

#### Sat 8 Aug (heavy day — use the weekend)

| Time | Task | Issue |
|---|---|---|
| 0:00–0:30 | Hardware gate (§1.1) — decide go/no-go | #1 |
| 0:30–2:00 | Run the quick start end to end; both URLs load | #1 |
| 2:00–3:30 | Explore Backstage: catalog, components, environments | #2 |
| 3:30–5:00 | Docker fundamentals refresher: images vs containers, layers, multi-stage builds, `docker build/run/logs/exec` | #3 |

#### Sun 9 Aug

| Time | Task | Issue |
|---|---|---|
| 0:00–2:30 | Kubernetes fundamentals — **only** these: Pod, Deployment, Service, Namespace, ConfigMap, Secret, and `kubectl get/describe/logs/exec`. Ignore everything else for now. | #3 |
| 2:30–4:00 | Walk the OpenChoreo resource chain by hand (commands in §8.3). Understand: Component → ComponentRelease → ReleaseBinding → Deployment+Service+HTTPRoute | #4 |
| 4:00–5:00 | Write `docs/openchoreo-notes.md` — explain the chain in your own words | #4 |

> **Why the notes file matters:** in an interview you will be asked "so what
> does OpenChoreo actually *do*?" The person who can answer that clearly beats
> the person who just ran the scripts. Write it down while it's fresh.

#### Mon 10 Aug

| Time | Task | Issue |
|---|---|---|
| 0:00–1:00 | Create repo, labels, milestones, project board, templates (§5) | #5 |
| 1:00–2:00 | Scaffold directory structure, `.gitignore`, `LICENSE`, `Makefile`, stub `README.md` | #6 |
| 2:00–3:00 | Deploy the GCP microservices demo (`./deploy-gcp-demo.sh`) and read its manifests — this is your reference for multi-service wiring | #7 |

#### Definition of Done — M0
- [ ] Appendix A complete: Docker Engine in WSL2, `.wslconfig` applied, repo inside the WSL2 filesystem
- [ ] `./install.sh --version v1.2.2` (base install) completes clean
- [ ] Sample app + Backstage both reachable **from the Windows browser**
- [ ] Codespaces dry run passed (Appendix B.1) — DinD and k3d work
- [ ] `.devcontainer/devcontainer.json` committed
- [ ] GitHub Student Developer Pack application submitted (needed by 22 Aug)
- [ ] You can explain Component → ComponentRelease → ReleaseBinding → K8s resources without looking it up
- [ ] Repo exists with structure, labels, milestones, board, templates
- [ ] `docs/openchoreo-notes.md` has ≥ 400 words in your own words

---

### PHASE 1 — Services running locally
**Tue 11 – Fri 14 Aug · Milestone M1**

> **Goal:** All three services + dashboard working on your laptop under Docker
> Compose. No OpenChoreo yet. Get the code boring and correct first.

#### Tue 11 Aug — Auth Service
- Scaffold `services/auth`, chi router, SQLite store
- `POST /register`, `POST /login` (bcrypt), `GET /verify` (JWT HS256)
- `GET /healthz` and `GET /readyz` — **every service gets these**, the platform
  uses them for probes
- Structured JSON logging to stdout (`log/slog`) — this is what makes the
  observability plane useful later
- Unit tests for the token package
- Multi-stage `Dockerfile`
- Issues: #8, #9, #10

#### Wed 12 Aug — Scan Service
- Scaffold `services/scan`, SQLite store
- `POST /scan`, `GET /scan/{id}`, `GET /scans`
- Three checks: TLS validity/expiry, security headers, HTTP→HTTPS redirect
- Async worker (a goroutine + buffered channel is plenty)
- Scoring logic + findings list
- Every outbound call gets a **timeout and a context** — a hung scan is the most
  likely way your demo breaks live
- Issues: #11, #12, #13, #14

#### Thu 13 Aug — Notification Service + wiring
- Scaffold `services/notification`
- `POST /notifications`, `GET /notifications`
- Scan → Notification call on completion, **URL read from an env var**
  (`NOTIFICATION_SERVICE_URL`), never hardcoded — this is what lets OpenChoreo
  inject the address later
- All service URLs and secrets from env vars, with sane local defaults
- Issues: #15, #16, #17

#### Fri 14 Aug — Dashboard + Compose
- Next.js dashboard: login page, "new scan" form, scan list, scan detail,
  notifications panel
- Keep it plain. Tailwind, no component library, no animations.
- `deploy/compose.yaml` running all four + a smoke test script
- Issues: #18, #19, #20

#### Definition of Done — M1
- [ ] `docker compose up` starts all four services
- [ ] `scripts/smoke-test.sh` passes: register → login → scan → poll → notification appears
- [ ] Every service has `/healthz`, `/readyz`, structured JSON logs
- [ ] No hardcoded URLs or secrets anywhere
- [ ] `make test` green; CI passing on `main`

---

### PHASE 2 — First deploy to OpenChoreo
**Sat 15 – Mon 17 Aug · Milestone M2**

> **Goal:** Auth Service live on OpenChoreo, deployed from a pre-built image.
> **One service only.** Deploy-from-image is the lower-risk path; get it working
> before you touch source builds.

#### Sat 15 Aug (heavy day)
- Push images to GHCR (`ghcr.io/<you>/securecloud-auth:v0.1.0`), make them public
- Write `platform/project.yaml`
- Write `platform/auth/component.yaml` + `workload.yaml` (§8.4 for the shape)
- `kubectl apply` → debug until it deploys
- Issues: #21, #22, #23

**Expect this day to be frustrating.** Budget the whole day for one service. The
debugging loop:

```bash
kubectl get component auth-service -n default
kubectl describe component auth-service -n default
kubectl get componentrelease,releasebinding -n default
kubectl get deployment,svc,httproute -A -l openchoreo.dev/component=auth-service
kubectl logs -n <dp-namespace> deployment/<name> --tail=100
```

Most first-deploy failures are one of: image not public, wrong `componentType`
name, port mismatch between the container and the workload endpoint, or a probe
failing because `/healthz` isn't wired up.

#### Sun 16 Aug
- Reach Auth through the gateway and get a real response
- Add the JWT signing secret via a `SecretReference` (not a plain env var)
- View the component in Backstage; take screenshots for the README
- Issues: #24, #25

#### Mon 17 Aug
- Add a persistent-volume trait so the SQLite file survives a restart
- Verify: create a user → delete the pod → user still there
- Document the whole flow in `docs/runbook.md`
- Issue: #26

#### Definition of Done — M2
- [ ] `curl http://development-default.openchoreoapis.localhost:19080/auth-http/healthz` returns 200
- [ ] Register + login work through the gateway
- [ ] JWT secret comes from a `SecretReference`
- [ ] Component visible and healthy in Backstage
- [ ] Data survives a pod restart
- [ ] Runbook documents deploy + rollback

---

### PHASE 3 — Full system
**Tue 18 – Fri 21 Aug · Milestone M3**

> **Goal:** All four components deployed, connected, and reachable. This is the
> centrepiece.

#### Tue 18 Aug — Scan Service
- `platform/scan/*` manifests, deploy, verify through gateway
- Issue: #27

#### Wed 19 Aug — Notification Service + service-to-service connection
- Deploy Notification with **project-internal visibility** (not external)
- Declare the Scan → Notification dependency in Scan's `Workload.spec.dependencies`
- Verify OpenChoreo injects the address as an env var and that the call works
- Issue: #28, #29

> **This is your most valuable single demo moment.** You did not write a URL,
> configure DNS, or write a NetworkPolicy — you *declared a dependency* and the
> platform wired up the address injection and the network policy. Make sure your
> demo video shows this explicitly. Copy the exact dependency syntax from the
> GCP microservices demo sample rather than guessing (see §8.5).

#### Thu 20 Aug — Dashboard
- Deploy as `deployment/web-application`
- Wire it to Auth and Scan endpoints
- Full end-to-end flow working in a browser
- Issue: #30

#### Fri 21 Aug — Promotion + hardening
- Promote Auth from `development` → `staging` through the deployment pipeline
- Show environment-specific config (different replica counts / resource limits)
- End-to-end smoke test against the deployed system
- Issues: #31, #32

#### Definition of Done — M3
- [ ] All four components healthy in Backstage
- [ ] Full flow works in the browser: login → scan → result → notification
- [ ] Scan→Notification uses a declared dependency, not a hardcoded URL
- [ ] Notification is *not* externally reachable (verify this — it's a security claim you'll make)
- [ ] At least one component promoted to a second environment
- [ ] Architecture diagram drafted

---

### PHASE 4 — CI/CD and observability
**Sat 22 – Mon 24 Aug · Milestone M4**

> **Goal:** The "I understand how software is operated" half of your story.
> **This phase runs on a borrowed 16 GB machine, not your PC** — see §1.1a.

#### Relocation checklist (30 minutes, do it Fri 21 Aug evening)

Do this the night *before*, not on Saturday morning. Discovering that
docker-in-docker doesn't work in your Codespace at 9am on the heavy day costs
you the day.

```bash
# On the big machine (Codespace via VS Code Desktop, or VPS over SSH):
git clone https://github.com/<you>/securecloud && cd securecloud

# Full install this time — the flags are the whole point of relocating
docker run --rm -it --name openchoreo-quick-start --pull always \
  -v /var/run/docker.sock:/var/run/docker.sock --network=host \
  ghcr.io/openchoreo/quick-start:v1.2.2

./install.sh --version v1.2.2 --with-build --with-observability
./check-status.sh

# Redeploy your whole system — images are already on GHCR
kubectl apply -f platform/
```

- [ ] Full install completes, `check-status.sh` all green
- [ ] Port forwarding working (§1.1b) — Backstage loads in your local browser
- [ ] All four components come back healthy
- [ ] Smoke test passes against the relocated cluster

If your system comes back up on a completely different cluster from nothing but
`git clone` + `kubectl apply`, **record that** — screenshot it, mention it in
the video. It's the cleanest possible evidence that you understood the point of
declarative platform configuration.

#### Sat 22 Aug — Source-to-deploy pipeline (heavy day)
- Convert **one** service (start with Notification — smallest blast radius) to
  build from source via the OpenChoreo Workflow Plane
- Trigger a build, watch the WorkflowRun, verify the new image deploys
- If it works: convert Auth and Scan too
- If it fights you after ~4 hours: **stop**, keep those two on the image path,
  and document the difference. One working source-build demo is enough.
- Issues: #33, #34

#### Sun 23 Aug — Observability
- Confirm logs from all services appear in the observability plane
- Add an alert rule trait (e.g. error rate, or pod restart count)
- Trigger it deliberately — break something and show the alert firing
- Capture screenshots: logs, metrics, the firing alert
- Issues: #35, #36

> A screenshot of an alert you *caused on purpose* is worth more than three
> dashboards nobody triggered.

#### Mon 24 Aug — Repo CI polish
- `ci.yaml`: `go vet`, `golangci-lint`, `go test ./...`, `docker build` for all services
- `images.yaml`: build and push tagged images to GHCR on push to `main`
- Add CI status badges to the README
- Issue: #37

#### Definition of Done — M4
- [ ] At least one service builds from source through the Workflow Plane
- [ ] Logs from all services visible in the observability plane
- [ ] One alert rule configured and demonstrably firing
- [ ] GitHub Actions green, badges in README
- [ ] Screenshots captured for docs

---

### PHASE 5 — Ship
**Tue 25 – Thu 27 Aug · Milestone M5**

> **Goal:** Make it legible to someone who will spend four minutes on it.
> **Do not add features this phase.**

#### Tue 25 Aug — Documentation
- `README.md` (structure in §12)
- `docs/architecture.md`: the plane model, the resource chain, why each decision
- Three ADRs (monorepo, SQLite-per-service, Docker over Buildpacks)
- Architecture diagram — draw it properly (Excalidraw or Mermaid), not ASCII
- Issues: #38, #39, #40

#### Wed 26 Aug — Demo video
Script it. 5 minutes, tight:

| Time | Content |
|---|---|
| 0:00–0:30 | What the system does (one sentence) + architecture diagram |
| 0:30–1:30 | Live browser demo: login → scan → result → notification |
| 1:30–2:30 | Backstage catalog: components, environments, health |
| 2:30–3:30 | Push a commit → build runs → new version deploys |
| 3:30–4:30 | Observability: logs, then break something and show the alert fire |
| 4:30–5:00 | What you learned; what you'd do next |

Record with OBS. Do 2–3 takes. Upload unlisted to YouTube, link in the README.
- Issue: #41

#### Thu 27 Aug — Final pass
- Fresh-clone test: can someone follow your README from zero?
- Fix every broken link and stale command
- Tag `v1.0.0`, write release notes
- Issues: #42, #43

#### Fri 28 Aug — Buffer
Deliberately empty. Something will have slipped. If nothing has, do a stretch
goal or rest.

#### Definition of Done — M5
- [ ] README complete with diagram, screenshots, quick start, video link
- [ ] Architecture doc explains the *why*
- [ ] Demo video under 6 minutes, uploaded, linked
- [ ] Clean clone → README steps → working system
- [ ] `v1.0.0` tagged

---

## 8. OpenChoreo reference

> **Verified against the official docs on 8 Aug 2026, OpenChoreo v1.2.2.**
> Items marked ⚠️ you must confirm yourself — I have not verified the exact
> field shape and I would rather tell you than have you debug my guess.

### 8.1 Install

```bash
docker run --rm -it --name openchoreo-quick-start \
  --pull always \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --network=host \
  ghcr.io/openchoreo/quick-start:v1.2.2

./install.sh --version v1.2.2 --with-build --with-observability
./check-status.sh          # if anything is pending
./uninstall.sh             # full teardown
```

`install.sh` is idempotent — safe to rerun if interrupted.

**Apple Silicon:** use Colima to avoid buildpack problems:
```bash
colima start --vm-type=vz --vz-rosetta --cpu 4 --memory 8
```

### 8.2 What the install creates for you

A control-plane namespace, a ClusterDataPlane, a ClusterWorkflowPlane (with
`--with-build`), three Environments (development/staging/production), a
DeploymentPipeline, a default Project, and ClusterComponentTypes.

Inspect them:

```bash
kubectl get namespaces -l openchoreo.dev/control-plane=true
kubectl get clusterdataplanes
kubectl get environments
kubectl get projects
kubectl get clustercomponenttypes
kubectl get project default -oyaml
```

**Discovery commands — use these instead of trusting any tutorial, including this one:**

```bash
kubectl get clustercomponenttypes        # what component types actually exist
kubectl get clustertraits                # what traits are available (PVC, alerts, ...)
kubectl explain workload.spec.endpoints  # exact endpoint field shape
kubectl explain workload.spec.dependencies
kubectl api-resources | grep openchoreo  # every OpenChoreo CRD on your cluster
```

### 8.3 The resource chain

```text
Component  →  ComponentRelease  →  ReleaseBinding  →  Deployment + Service + HTTPRoute
(intent)      (immutable snapshot)  (binds to env)     (running on the data plane)
```

```bash
kubectl get component auth-service -n default
kubectl get component auth-service -n default -o jsonpath='{.status.latestRelease.name}'
kubectl get componentrelease -n default
kubectl get releasebinding auth-service-development -n default
kubectl get deployment,svc,httproute -A -l openchoreo.dev/component=auth-service
```

The workloads land in a data-plane namespace like
`dp-default-default-development-*`, **not** in `default`. Control plane holds
intent; data plane runs the pods. Worth saying out loud in your video.

### 8.4 Component + Workload (✅ verified shape, v1.2)

Adapted from the official `go-greeter-service` sample:

```yaml
apiVersion: openchoreo.dev/v1alpha1
kind: Component
metadata:
  name: auth-service
  namespace: default
spec:
  owner:
    projectName: securecloud
  autoDeploy: true
  componentType:
    kind: ClusterComponentType
    name: deployment/service
---
apiVersion: openchoreo.dev/v1alpha1
kind: Workload
metadata:
  name: auth-service-workload
  namespace: default
spec:
  owner:
    componentName: auth-service
    projectName: securecloud
  endpoints:
    http:
      type: HTTP
      port: 8080
      visibility: [external]
  container:
    image: "ghcr.io/<your-username>/securecloud-auth:v0.1.0"
    env:
      - key: LOG_LEVEL
        value: "info"
      - key: JWT_SECRET
        valueFrom:
          secretKeyRef:
            key: jwt-secret
            name: securecloud-secrets
```

Default ClusterComponentTypes available out of the box:
- `deployment/service` — backend services and APIs
- `deployment/web-application` — frontend apps (gets its own subdomain)
- `cronjob/scheduled-task` — scheduled batch jobs

⚠️ **Check the allowed `visibility` values** with `kubectl explain` before
writing the Notification Service manifest — you want it project-internal, not
external, and I have only verified that `external` is valid.

### 8.5 Service-to-service dependencies (⚠️ verify syntax)

Conceptually: a Workload declares a dependency on another component's endpoint;
OpenChoreo resolves the address, injects it as an env var, and configures the
network policy for that traffic. That is exactly the Scan → Notification link.

I have **not** verified the exact field shape for v1.2. Get it from the working
reference sample rather than guessing:

```bash
./deploy-gcp-demo.sh      # 11 interconnected services
# then read the manifests in /samples/gcp-microservices-demo/ inside the container
```

Copy the dependency block from there. This is what the `deploy-gcp-demo.sh` step
in Phase 0 (Issue #7) is for — you're reading it as reference material, not just
running it.

### 8.6 URLs

```text
Service:   http://{environment}-{namespace}.openchoreoapis.localhost:19080/{endpoint-name}/{path}
Web app:   http://{component}-{environment}-{namespace}.openchoreoapis.localhost:19080
Backstage: http://openchoreo.localhost:8080/    (admin@openchoreo.dev / Admin@123)
```

Gateway is on host ports **19080** (HTTP) and **19443** (HTTPS) — already
exposed by the dev container, no port-forward needed.

Example:
```bash
curl http://development-default.openchoreoapis.localhost:19080/auth-http/healthz
```

### 8.7 Secrets

Use `SecretReference`, which creates an ExternalSecret that produces a Kubernetes
Secret you then reference from the workload:

```yaml
apiVersion: openchoreo.dev/v1alpha1
kind: SecretReference
metadata:
  name: securecloud-secrets
  namespace: default
spec:
  template:
    type: Opaque
  data:
    - secretKey: jwt-secret
      remoteRef:
        key: securecloud-jwt-secret
  refreshInterval: 1h
```

⚠️ Confirm how the secret store is backed in the local quick-start install
before relying on this — check the sample and `kubectl describe` the resulting
ExternalSecret.

### 8.8 Useful references

- Docs (set version to v1.2.x): https://openchoreo.dev/docs/
- Samples: https://github.com/openchoreo/openchoreo/tree/release-v1.2/samples
- Concepts: https://openchoreo.dev/docs/concepts/developer-abstractions/
- CNCF Slack `#openchoreo`: https://slack.cncf.io/
- GitHub Discussions: https://github.com/openchoreo/openchoreo/discussions

---

## 9. GitHub issues (copy-paste ready)

Create all of these on Day 1. Seeing 43 issues and closing them one by one is
what keeps momentum on a 21-day project.

### M0 · Foundations

**#1 — Verify local environment can run OpenChoreo** `phase-0` `infra`
- [ ] Docker reports ≥ 8 GB RAM and ≥ 4 CPUs available
- [ ] Quick-start container runs, `install.sh --version v1.2.2 --with-build --with-observability` completes
- [ ] Sample app URL loads
- [ ] Backstage loads and login works
- [ ] Go/no-go decision recorded in `docs/openchoreo-notes.md`

**#2 — Explore the Backstage developer portal** `phase-0` `docs`
- [ ] Locate the component catalog, environments, and deployment views
- [ ] Screenshot each for later use in the README

**#3 — Docker and Kubernetes fundamentals** `phase-0` `docs`
- [ ] Can explain: image vs container, layers, multi-stage build
- [ ] Can explain: Pod, Deployment, Service, Namespace, ConfigMap, Secret
- [ ] Comfortable with `kubectl get/describe/logs/exec`
- [ ] Notes written up

**#4 — Trace the OpenChoreo resource chain by hand** `phase-0` `platform` `docs`
- [ ] Run every command in §8.3 against the sample app
- [ ] Write the chain in your own words in `docs/openchoreo-notes.md` (≥ 400 words)

**#5 — Set up GitHub repo, labels, milestones, board, templates** `phase-0` `infra`

**#6 — Scaffold repository structure** `phase-0` `infra`
- [ ] Directory tree per §4, `.gitignore`, `LICENSE` (Apache-2.0), `Makefile`, stub `README.md`

**#7 — Deploy and study the GCP microservices demo** `phase-0` `platform`
- [ ] `./deploy-gcp-demo.sh` runs successfully
- [ ] Dependency/connection syntax between services extracted into `docs/openchoreo-notes.md`
- [ ] *This is your reference for Issue #29 — do not skip it*

### M1 · Services

**#8 — Auth Service: scaffold + health endpoints** `phase-1` `service:auth`
- [ ] chi router, `/healthz`, `/readyz`, structured JSON logging via `log/slog`
- [ ] Config from env vars with sane defaults

**#9 — Auth Service: register and login** `phase-1` `service:auth`
- [ ] SQLite store, bcrypt hashing, `POST /register`, `POST /login`
- [ ] Duplicate email and bad credentials return sensible status codes

**#10 — Auth Service: JWT issuing and verification** `phase-1` `service:auth`
- [ ] HS256 signing, secret from `JWT_SECRET`, `GET /verify`
- [ ] Unit tests for the token package
- [ ] Multi-stage Dockerfile, image under 30 MB

**#11 — Scan Service: scaffold + storage** `phase-1` `service:scan`

**#12 — Scan Service: TLS certificate check** `phase-1` `service:scan`
- [ ] Validity, expiry, flags certs expiring within 30 days
- [ ] 5s handshake timeout

**#13 — Scan Service: security headers + HTTPS redirect checks** `phase-1` `service:scan`
- [ ] HSTS, X-Content-Type-Options, CSP present/absent
- [ ] HTTP→HTTPS redirect detected

**#14 — Scan Service: async worker, scoring, API** `phase-1` `service:scan`
- [ ] `POST /scan`, `GET /scan/{id}`, `GET /scans`
- [ ] Background worker, statuses: queued → running → complete/failed
- [ ] Score out of 100 with a findings list
- [ ] Every outbound call has a context timeout

**#15 — Notification Service: scaffold + API** `phase-1` `service:notification`
- [ ] `POST /notifications`, `GET /notifications`, SQLite store

**#16 — Wire Scan → Notification** `phase-1` `service:scan`
- [ ] Scan POSTs on completion; URL from `NOTIFICATION_SERVICE_URL`
- [ ] Failure to notify does not fail the scan (log and continue)

**#17 — Externalise all config** `phase-1` `infra`
- [ ] Zero hardcoded URLs, ports, or secrets in any service
- [ ] `docs/runbook.md` lists every env var

**#18 — Dashboard: auth pages** `phase-1` `service:web`
- [ ] Login/register, token stored, protected routes

**#19 — Dashboard: scan flow** `phase-1` `service:web`
- [ ] New-scan form, scan list with status, scan detail with findings, notifications panel

**#20 — Docker Compose + smoke test** `phase-1` `infra`
- [ ] `docker compose up` starts everything
- [ ] `scripts/smoke-test.sh` runs the full flow and exits non-zero on failure

### M2 · First Deploy

**#21 — Publish images to GHCR** `phase-2` `infra`
- [ ] All three service images pushed and **public**
- [ ] Semver tags

**#22 — Write the OpenChoreo Project manifest** `phase-2` `platform`

**#23 — Deploy Auth Service from image** `phase-2` `platform` `service:auth`
- [ ] Component + Workload manifests applied
- [ ] `kubectl wait --for=condition=Ready component/auth-service` succeeds

**#24 — Reach Auth Service through the gateway** `phase-2` `platform`
- [ ] `/healthz` returns 200 via the gateway URL
- [ ] Register and login work end to end
- [ ] Component healthy in Backstage, screenshot taken

**#25 — Move the JWT secret to a SecretReference** `phase-2` `platform`
- [ ] No secret value in any committed manifest

**#26 — Add persistent storage trait** `phase-2` `platform`
- [ ] Check available traits with `kubectl get clustertraits`
- [ ] Data survives `kubectl delete pod`
- [ ] Deploy + rollback documented in `docs/runbook.md`

### M3 · Full System

**#27 — Deploy Scan Service** `phase-3` `platform` `service:scan`

**#28 — Deploy Notification Service (internal only)** `phase-3` `platform`
- [ ] Deployed with project-internal visibility
- [ ] **Verify it is NOT reachable from outside the cluster** — this is a security claim you'll make in the demo, so prove it

**#29 — Declare the Scan → Notification dependency** `phase-3` `platform`
- [ ] Dependency declared in Scan's Workload (syntax copied from the GCP demo, Issue #7)
- [ ] Address injected by the platform, not hardcoded
- [ ] End-to-end scan produces a notification

**#30 — Deploy the dashboard** `phase-3` `platform` `service:web`
- [ ] `deployment/web-application` type, own subdomain
- [ ] Full flow works in a browser

**#31 — Promote a component to staging** `phase-3` `platform`
- [ ] Auth promoted development → staging via the deployment pipeline
- [ ] Environment-specific config differs (replicas or resource limits)
- [ ] Screenshot the pipeline view

**#32 — End-to-end smoke test against the cluster** `phase-3` `infra`

### M4 · CI/CD + Observability

**#33 — Build Notification Service from source via Workflow Plane** `phase-4` `platform`
- [ ] Workflow configured, WorkflowRun triggered and succeeds
- [ ] Built image deploys automatically
- [ ] **Time-box: 4 hours. If it fails, stop and document — see cut line C3**

**#34 — Extend source builds to Auth and Scan** `phase-4` `platform` `stretch`

**#35 — Verify logs in the observability plane** `phase-4` `platform`
- [ ] Logs from all services queryable
- [ ] Screenshot

**#36 — Add and trigger an alert rule** `phase-4` `platform`
- [ ] Alert rule trait configured (error rate or restart count)
- [ ] Deliberately break something; capture the firing alert
- [ ] Screenshot

**#37 — GitHub Actions CI** `phase-4` `ci`
- [ ] `go vet`, `golangci-lint`, `go test ./...`, `docker build` all services
- [ ] Image push to GHCR on `main`
- [ ] Badges in README

### M5 · Ship

**#38 — Write README** `phase-5` `docs`
**#39 — Write architecture doc + 3 ADRs** `phase-5` `docs`
**#40 — Produce architecture diagram** `phase-5` `docs`
- [ ] Proper diagram (Excalidraw/Mermaid), exported to `docs/images/`

**#41 — Record demo video** `phase-5` `docs`
- [ ] Scripted, under 6 minutes, uploaded unlisted, linked in README

**#42 — Fresh-clone verification** `phase-5` `docs`
- [ ] Clone to a new directory, follow README exactly, note every gap, fix them

**#43 — Tag v1.0.0** `phase-5` `infra`

---

## 10. Commit history

Target: **~90 commits over 21 days**. Small and frequent beats large and rare.

Sample progression:

```text
# Phase 0
chore(repo): initialise repository with Apache-2.0 licence
chore(repo): add directory structure and Makefile
docs(notes): document OpenChoreo resource chain and plane model
docs(notes): capture service dependency syntax from GCP demo

# Phase 1
feat(auth): scaffold service with chi router and health endpoints
feat(auth): add SQLite user store with bcrypt password hashing
feat(auth): issue HS256 JWTs on successful login
test(auth): cover token issuing and verification
build(auth): add multi-stage Dockerfile
feat(scan): add TLS certificate expiry check
feat(scan): detect missing security headers
feat(scan): add async scan worker with context timeouts
feat(scan): notify notification service on scan completion
feat(notification): add notification store and REST API
feat(web): add login and registration pages
feat(web): add scan submission and results views
build(compose): add local development stack
test(e2e): add smoke test covering the full user flow

# Phase 2
ci(images): publish service images to GHCR
feat(platform): add SecureCloud project manifest
feat(platform): deploy auth service component from image
fix(platform): correct workload endpoint port to match container
feat(platform): move JWT secret to SecretReference
feat(platform): add persistent volume trait to auth service
docs(runbook): document deploy and rollback procedure

# Phase 3
feat(platform): deploy scan service component
feat(platform): deploy notification service with internal visibility
feat(platform): declare scan to notification endpoint dependency
fix(scan): read notification address from injected env var
feat(platform): deploy dashboard as web-application component
feat(platform): promote auth service to staging environment

# Phase 4
feat(platform): build notification service from source via workflow plane
feat(platform): add error rate alert rule trait
ci: add lint, test, and build workflow
ci: publish tagged images on push to main
docs: add CI status badges

# Phase 5
docs: write project README with architecture overview
docs(adr): record monorepo, sqlite, and docker build decisions
docs: add architecture diagram
docs: link demo video
chore(release): tag v1.0.0
```

---

## 11. Risks and cut lines

### 11.1 Risk register

| # | Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| R1 | 8 GB PC can't run even the base install | Medium | High | §1.4 tuning. Fallback: run Phases 2–4 entirely on Codespaces. Test on Day 1. |
| R1b | Student Pack not verified by 22 Aug | Medium | High | **Apply today.** Verification takes days. Backup: Hetzner hourly (~€1 for the phase). |
| R1c | Docker-in-Docker fails in Codespaces | Medium | Medium | Test it during Phase 0, not on 22 Aug. Fallback: VPS. |
| R2 | Version drift — following a v0.x tutorial | **High** | High | Pin v1.2.2. Always set the docs version selector. Prefer `kubectl explain` over blogs. |
| R3 | Cluster becomes unstable / OOM | Medium | High | `./install.sh` is idempotent. Practise a full teardown+rebuild early so it isn't scary at 11pm on Aug 26. |
| R4 | Source builds (Workflow Plane) don't work | Medium | Medium | Deploy-from-image path already works from Phase 2. Time-box to 4 hours (C3). |
| R5 | Coursework spike | **High** | Medium | Weekends are the heavy days by design. Cut lines below. |
| R6 | Scope creep on scan logic | **High** | Medium | Scope locked in §2.2. Anything else is a `stretch` label. |
| R7 | Demo breaks live during recording | Medium | Medium | Record Aug 26, not Aug 28. Multiple takes. Keep a working recording as soon as you have one. |

### 11.2 Cut lines — drop in this order

If you are behind, cut from the top. Do not improvise.

- **C1** — Postgres via the `Resource` abstraction. *Already excluded; never add it.*
- **C2** — Observability plane, if the relocation falls through. You already have structured JSON logs going to stdout, so `kubectl logs` demonstrates the same data without the plane. Write honestly in the README: "the observability plane requires 8 GB; I developed on 8 GB total, so logs are shown via kubectl rather than the aggregation layer." That reads as resourcefulness, not as a gap.
- **C3** — Source builds for Auth and Scan (Issue #34). One working source-build demo on Notification is sufficient.
- **C4** — Environment promotion to staging (Issue #31). Nice-to-have.
- **C5** — Persistent volume trait (Issue #26). Run ephemeral, document the limitation.
- **C6** — Dashboard polish. An ugly but working dashboard is fine.
- **C7** — Alert rules (Issue #36). Keep log aggregation; drop alerting.

**Never cut:** the README, the architecture doc, the diagram, or the demo video.
Those are the deliverable. A three-service system with excellent documentation
beats a five-service system nobody can understand.

---

## 12. README structure

```markdown
# SecureCloud
> A cloud-native security scanning platform built on OpenChoreo —
> microservices, GitOps CI/CD, and observability on Kubernetes.

[badges: CI, licence, Go version, OpenChoreo v1.2.2]

## Demo
[5-minute video](link)
![dashboard screenshot](docs/images/dashboard.png)

## What this is
Two paragraphs. What it does, and — more importantly — what it demonstrates:
building, deploying, connecting, securing, and observing a multi-service system
through an internal developer platform.

## Architecture
![diagram](docs/images/architecture.png)
The three services, how they connect, and how OpenChoreo's planes map onto them.

## Why OpenChoreo
Three paragraphs on the problem an IDP solves and what the platform gave you
that raw Kubernetes would not: declared dependencies instead of hardcoded URLs,
environment promotion, built-in CI, network policy by default.

## Tech stack
Table.

## Running it locally
### Quick start (Docker Compose)
### Full platform (OpenChoreo on k3d)
Exact, tested commands. Pinned versions.

## Project structure
Annotated tree.

## What I learned
The most-read section for a student project. Be specific and honest —
including what went wrong.

## Design decisions
Link the ADRs.

## Non-goals
The §2.4 list. Shows you scoped deliberately.

## Roadmap
What you'd build next.
```

---

## 13. Positioning for WSO2

Three things worth doing beyond the code:

**1. Contribute to OpenChoreo itself.** This is the highest-signal action
available to you, and it costs a few hours. While building this you *will* hit
a docs gap, a confusing error message, or a broken sample link. Open an issue.
Better, open a PR fixing it. A merged commit in `openchoreo/openchoreo` says
something no portfolio project can. The maintainers are active in CNCF Slack
`#openchoreo` and in GitHub Discussions, and the repo has a contributor guide.

**2. Write it up publicly.** A blog post — "Deploying a microservices security
platform on OpenChoreo as a student" — is genuinely useful content because very
little exists at that level. You already create technical content for
Instagram/Facebook, so this is an existing habit pointed at a higher-value
target. Link it from the README.

**3. Frame the pair correctly.** SecureScan shows you can build security
software. SecureCloud shows you understand how software is operated after it's
written. When you describe them together, that second sentence is the one that
differentiates you — most student portfolios only have the first.

One note on honesty: whatever you cut, say so plainly in the README. "I ran out
of time to enable the observability plane locally due to RAM constraints"
reads as self-aware engineering judgement. A README that overclaims and falls
apart under a single question does real damage. Reviewers can tell the
difference, and they respect the first far more.

---

## 14. Quick reference card

```text
DEADLINE                              Fri 28 Aug 2026
OPENCHOREO VERSION                    v1.2.2  (pin it)

PHASE 0  Foundations       8–10 Aug   M0   local, base install
PHASE 1  Services         11–14 Aug   M1   local, cluster off
PHASE 2  First deploy     15–17 Aug   M2   local, base install
PHASE 3  Full system      18–21 Aug   M3   local, base install
PHASE 4  CI/CD + obs      22–24 Aug   M4   ← 16 GB machine
PHASE 5  Ship             25–27 Aug   M5   local, no cluster
BUFFER                        28 Aug

INSTALL COMMANDS
  local  (8 GB):  ./install.sh --version v1.2.2
  big   (16 GB):  ./install.sh --version v1.2.2 --with-build --with-observability

MEMORY DISCIPLINE
  never run Docker Compose and the k3d cluster together
  close the browser during installs
  ./uninstall.sh when you finish a session

DAILY   commit something, update the board, max 2 items In Progress
WEEKLY  re-read the cut lines and be honest about where you are

DEBUG LOOP
  kubectl describe component <name> -n default
  kubectl get componentrelease,releasebinding -n default
  kubectl get deployment,svc,httproute -A -l openchoreo.dev/component=<name>
  kubectl logs -n <dp-namespace> deployment/<name> --tail=100

DISCOVERY (trust these over any tutorial)
  kubectl get clustercomponenttypes
  kubectl get clustertraits
  kubectl explain workload.spec.endpoints
  kubectl api-resources | grep openchoreo

HELP
  CNCF Slack #openchoreo · github.com/openchoreo/openchoreo/discussions
```

**Today's job:** Issue #1. Run the quick start. Everything else waits until you
know the platform runs on your machine.

---

## Appendix A — Windows 8 GB setup (do this first)

Budget **2 hours** on Fri 8 Aug, before Issue #1. Getting this wrong costs you
days of mysterious slowness later.

### A.1 Drop Docker Desktop, use Docker Engine inside WSL2

Docker Desktop on Windows runs its own WSL distro, a Windows-side backend
process, and a GUI. On a 16 GB machine that's invisible; on 8 GB it's 1–2 GB
you need for the cluster. Running Docker Engine natively inside your Ubuntu
WSL2 distro removes all of it.

It also fixes a second problem: the OpenChoreo quick start uses
`docker run --network=host`. Host networking for Linux containers has always
been awkward under Docker Desktop on Windows. Inside WSL2 with native Docker
Engine it behaves exactly as the docs assume.

```powershell
# In PowerShell (Admin)
wsl --install -d Ubuntu-22.04
wsl --update
wsl --version          # want WSL 2.x
```

If Docker Desktop is already installed, uninstall it, or at minimum disable
"Start Docker Desktop when you sign in" and never launch it.

```bash
# Inside Ubuntu (WSL2)
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
sudo systemctl enable --now docker      # systemd works in modern WSL2
# close and reopen the terminal, then:
docker run --rm hello-world
```

If systemd isn't enabled, add to `/etc/wsl.conf` and `wsl --shutdown`:

```ini
[boot]
systemd=true
```

### A.2 `.wslconfig`

Create `C:\Users\<you>\.wslconfig`:

```ini
[wsl2]
memory=5GB
processors=4
swap=8GB
localhostForwarding=true
```

Apply with `wsl --shutdown` in PowerShell, then reopen your terminal.

**Why 5 GB:** WSL2 defaults to 50% of physical RAM, which is 4 GB here — right
at OpenChoreo's floor, with no headroom for image pulls. 5 GB leaves the cluster
room to breathe. Windows keeps ~3 GB, which is enough *if* you keep the browser
closed during cluster sessions. The 8 GB swap file means memory pressure makes
things slow rather than killing pods outright.

> **Do not add `autoMemoryReclaim=gradual`.** You will find it recommended
> everywhere for low-RAM WSL setups, and it looks perfect for this situation.
> Microsoft has documented that it breaks the Docker daemon when dockerd runs as
> a service inside WSL — which is exactly the setup in §A.1. The failure is
> confusing and intermittent. Skip it.

If you're on Windows 11 22H2+ and hit networking oddities, `networkingMode=mirrored`
under `[wsl2]` is worth trying, but don't enable it pre-emptively — one variable
at a time.

### A.3 Trim Windows itself

Every 200 MB matters at this size.

- Settings → Apps → Startup: disable everything non-essential
- Close Chrome/Edge entirely during cluster work (it can hold 2–3 GB alone)
- Use `wsl --shutdown` between sessions to hand memory back to Windows
- Check the baseline: Task Manager → Performance → Memory. If Windows idles
  above 4 GB, you have startup bloat to clear before OpenChoreo will fit.

> **Worth checking:** many 8 GB laptops have a free SODIMM slot or one
> replaceable stick. A 16 GB upgrade is often inexpensive and would remove this
> entire constraint — including letting you run Phase 4 locally. Check your
> model before you commit to the split plan. If the RAM is soldered, carry on
> as planned.

### A.4 The mistake that costs the most time

**Keep the repository inside the WSL2 filesystem, never on `/mnt/c/`.**

```bash
# Correct
cd ~ && git clone https://github.com/<you>/securecloud
# → /home/<you>/securecloud

# Wrong — cross-filesystem I/O is 10–20x slower
cd /mnt/c/Users/<you>/Projects && git clone ...
```

Go builds, `npm install`, and Docker build contexts all hammer the filesystem.
On `/mnt/c` a Next.js install that takes 40 seconds takes 8 minutes. Open the
project with VS Code's **WSL** extension (`code .` from inside Ubuntu) — it
looks identical but runs in Linux.

Also set line endings, or shell scripts will fail with cryptic `\r` errors:

```bash
git config --global core.autocrlf input
```

### A.5 Verify

```bash
# Inside WSL2 Ubuntu
free -h        # ~5 GB total
nproc          # 4
docker run --rm alpine sh -c "free -h; nproc"
```

Then run Issue #1 (§1.3) from inside WSL2 Ubuntu — **not** from PowerShell.

### A.6 Accessing the URLs from your Windows browser

WSL2 forwards `localhost` from Windows into the distro, and `.localhost`
subdomains resolve to 127.0.0.1, so `http://openchoreo.localhost:8080` and
`http://react-starter-development-default.openchoreoapis.localhost:19080`
should just work in Edge or Chrome.

If forwarding misbehaves, get the distro IP and add explicit hosts entries as a
fallback (Windows' hosts file has no wildcard support, so you list the specific
hostnames you use):

```bash
wsl hostname -I     # e.g. 172.28.114.3
```

Then in `C:\Windows\System32\drivers\etc\hosts` (as Administrator):

```text
172.28.114.3  openchoreo.localhost
172.28.114.3  development-default.openchoreoapis.localhost
```

Note the WSL2 IP changes on reboot unless you're using mirrored networking.

---

## Appendix B — Codespaces for Phase 4

### B.1 Do a dry run during Phase 0

**Add this to Issue #1.** Spend one hour in Phase 0 confirming that
docker-in-docker and k3d work in a Codespace. It costs ~4 core-hours out of your
180. Finding out on the morning of 22 Aug that it doesn't work costs you the
phase.

You don't need a full OpenChoreo install for the dry run — just confirm
`docker run --rm hello-world` and `k3d cluster create test` succeed inside the
Codespace.

### B.2 `.devcontainer/devcontainer.json`

Commit this in Phase 0 so Phase 4 is one click:

```json
{
  "name": "securecloud-openchoreo",
  "image": "mcr.microsoft.com/devcontainers/base:ubuntu-22.04",
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {},
    "ghcr.io/devcontainers/features/kubectl-helm-minikube:1": {
      "minikube": "none"
    },
    "ghcr.io/devcontainers/features/go:1": {}
  },
  "hostRequirements": {
    "cpus": 4,
    "memory": "16gb",
    "storage": "32gb"
  },
  "forwardPorts": [8080, 19080, 19443],
  "portsAttributes": {
    "8080":  { "label": "Backstage Portal" },
    "19080": { "label": "OpenChoreo Gateway (HTTP)" },
    "19443": { "label": "OpenChoreo Gateway (HTTPS)" }
  },
  "remoteUser": "vscode"
}
```

### B.3 Connect with VS Code Desktop, not the browser

This is not a preference — it is the difference between the gateway routing and
not. See §1.1b. In the browser, Codespaces rewrites forwarded ports to
`https://xxx-19080.app.github.dev`, the `Host` header no longer matches, and
Envoy returns 404 for every request.

```bash
gh codespace code    # opens the codespace in VS Code Desktop
```

Ports then forward to real `localhost` on your Windows machine, and
`http://openchoreo.localhost:8080` works as it does locally.

### B.4 Core-hour discipline

You get **180 core-hours/month**. A 4-core machine burns 4 core-hours per wall
hour, so your budget is **45 hours**. Phase 4 needs roughly 30. That fits — but
only with discipline:

| Habit | Cost of getting it wrong |
|---|---|
| Set idle timeout to 30 min (Settings → Codespaces) | Default is fine; longer is not |
| `gh codespace stop` when you finish for the day | Leaving it overnight burns ~32 core-hours — most of your Phase 4 budget in one night |
| Delete the codespace after Phase 4 | Storage counts against your 20 GB/month |

Check usage at any time: GitHub → Settings → Billing → Codespaces.

### B.5 If a Codespace runs out of disk

The full install with `--with-build --with-observability` pulls a lot of images
into a 32 GB disk. If you hit `no space left on device`:

```bash
docker system prune -af --volumes
```

Re-pull only what you need. If it recurs, drop `--with-observability` and lean
on cut line C2.
