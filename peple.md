# SecureCloud — 3-Member Complete Team Workflow

**Project:** SecureCloud — OpenChoreo Security Deployment Lab
**Start:** 8 August 2026
**Final Deadline:** 28 August 2026
**Team Size:** 3 members
**Target Release:** `v1.0.0`

---

# 1. Team Structure

Do **not** divide the project into “backend person / frontend person / documentation person” and let everyone work independently until the end.

Instead, give everyone a technical ownership area while making integration and code review shared responsibilities.

| Member       | Primary Role                   | Main Ownership                                                                                                           |
| ------------ | ------------------------------ | ------------------------------------------------------------------------------------------------------------------------ |
| **Member 1** | Platform & Auth Lead           | Auth Service, OpenChoreo, Kubernetes manifests, secrets, deployment, promotion, Workflow Plane                           |
| **Member 2** | Scan & Security Lead           | Scan Service, security checks, backend testing, observability, alerting                                                  |
| **Member 3** | Web, Notification & QA/CI Lead | Notification Service, Next.js dashboard, Docker Compose, GitHub Actions, E2E testing, release/documentation coordination |

All three members must still understand the complete architecture.

---

# 2. Code Ownership

## Member 1 — Platform & Auth

Primary directories:

```text
services/auth/
├── cmd/server/main.go
└── internal/
    ├── handler/
    ├── store/
    └── token/

platform/
├── project.yaml
├── auth/
│   ├── component.yaml
│   └── workload.yaml
├── secrets/
└── workflow-related manifests
```

Responsible for:

* User registration
* Login
* bcrypt password hashing
* JWT generation
* JWT verification
* `/healthz`
* `/readyz`
* SQLite user storage
* Auth Dockerfile
* OpenChoreo project
* SecretReference
* Auth deployment
* persistent volume
* environment promotion
* Workflow Plane/source build

---

# 3. Member 2 — Scan & Security

Primary directories:

```text
services/scan/
├── cmd/server/main.go
└── internal/
    ├── handler/
    ├── store/
    ├── checks/
    └── notify/

platform/
└── scan/
    ├── component.yaml
    └── workload.yaml
```

Responsible for:

* Scan creation
* Scan history
* TLS validation
* Certificate expiry detection
* Security-header checking
* HTTP → HTTPS redirect checking
* Security score calculation
* Findings
* async scan worker
* request timeouts
* Scan → Notification client code
* scan unit tests
* observability
* alerting
* security verification

---

# 4. Member 3 — Web, Notification, CI & QA

Primary directories:

```text
services/notification/
├── cmd/server/main.go
└── internal/
    ├── handler/
    └── store/

web/
deploy/
scripts/
.github/
```

Responsible for:

* Notification API
* notification storage
* Next.js dashboard
* Login/Register UI
* Scan submission UI
* Scan history
* Scan results
* Notification panel
* Docker Compose
* smoke-test script
* GitHub Actions
* GHCR automation
* E2E testing
* README coordination
* demo recording coordination
* release

---

# 5. Shared Files

These are not permanently owned by one member:

```text
README.md
Makefile
docs/
.gitignore
LICENSE
```

However, only **one member edits a shared document at a time**.

This prevents unnecessary Git conflicts.

---

# 6. GitHub Setup

Use one monorepo:

```text
securecloud/
```

Protect `main`.

Nobody pushes directly to `main`.

Required workflow:

```text
GitHub Issue
     ↓
Branch
     ↓
Code
     ↓
Local Tests
     ↓
Commit
     ↓
Push
     ↓
Pull Request
     ↓
Another member reviews
     ↓
CI passes
     ↓
Squash Merge
     ↓
Delete branch
```

---

# 7. Branch Naming

Use the issue number in the branch.

Examples:

```text
feat/9-auth-register-login
feat/12-scan-tls-check
feat/15-notification-api
feat/18-web-auth-pages

test/14-scan-worker-tests
fix/29-notification-dependency
platform/23-auth-component
ci/37-github-actions
docs/39-architecture
```

Never use branches such as:

```text
nirmal-work
test
new
final
final2
member1
```

---

# 8. Commit Standard

Use Conventional Commits.

Format:

```text
<type>(<scope>): <description>
```

Types:

```text
feat
fix
test
docs
ci
build
chore
refactor
```

Scopes:

```text
auth
scan
notification
web
platform
ci
docs
repo
```

Examples for Member 1:

```text
feat(auth): add user registration endpoint
feat(auth): add bcrypt password hashing
feat(auth): issue JWT after successful login
test(auth): cover JWT verification
feat(platform): deploy auth service through OpenChoreo
```

Examples for Member 2:

```text
feat(scan): add TLS certificate validation
feat(scan): detect missing security headers
feat(scan): add HTTPS redirect check
feat(scan): add asynchronous worker
test(scan): cover scoring logic
```

Examples for Member 3:

```text
feat(notification): add notification REST API
feat(web): add login page
feat(web): add scan results view
test(e2e): add full system smoke test
ci: run tests on pull requests
```

Every member should commit frequently.

Do not create one giant commit at the end of a day.

---

# 9. Pull Request Rules

Every PR must satisfy:

```text
[ ] Linked GitHub issue
[ ] Small enough to review
[ ] Code builds
[ ] Unit tests pass
[ ] Existing tests still pass
[ ] No secrets committed
[ ] No hardcoded service URLs
[ ] Relevant documentation updated
[ ] CI green
[ ] Reviewed by another member
```

Backend/platform PR:

```bash
make test

go vet ./...

kubectl apply --dry-run=server -f platform/
```

Frontend PR:

```bash
npm run lint
npm run build
```

---

# 10. Review Matrix

Primary reviewer should be outside the code owner's area.

| Author   | Preferred Reviewer |
| -------- | ------------------ |
| Member 1 | Member 2           |
| Member 2 | Member 3           |
| Member 3 | Member 1           |

For security-sensitive changes such as JWT, secrets, internal service visibility, all three should review.

---

# 11. PHASE 0 — Foundations

## 8–10 August

**Goal:** No application development yet.

First prove that OpenChoreo and the development environment work.

---

## August 8

### Member 1

Own Issue #1.

Tasks:

```text
Verify Docker resources
Run OpenChoreo quick-start
Install OpenChoreo v1.2.2
Deploy sample application
Verify Backstage
Verify gateway
```

Record every failure and solution.

Commit:

```text
docs(platform): record OpenChoreo environment verification
```

### Member 2

Study only the Kubernetes concepts required by this project:

```text
Pod
Deployment
Service
Namespace
ConfigMap
Secret
```

Practice:

```bash
kubectl get
kubectl describe
kubectl logs
kubectl exec
```

Start:

```text
docs/openchoreo-notes.md
```

Commit:

```text
docs(notes): add Kubernetes fundamentals
```

### Member 3

Create/configure GitHub repository.

Set up:

```text
labels
milestones
project board
issue templates
PR template
branch protection
```

Also test GitHub Codespaces.

Commit:

```text
chore(repo): configure project collaboration workflow
```

---

# August 9

### Member 1

Trace:

```text
Component
    ↓
ComponentRelease
    ↓
ReleaseBinding
    ↓
Deployment
Service
HTTPRoute
```

Document what each does.

### Member 2

Deploy and examine the OpenChoreo GCP microservices example.

Most important task:

Find the working syntax for:

```text
service → service dependency
```

because Scan → Notification needs it later.

### Member 3

Scaffold repository:

```text
services/auth
services/scan
services/notification
web
platform
deploy
scripts
docs
.github
```

Create:

```text
Makefile
.gitignore
LICENSE
README.md
```

---

# August 10

## Entire Team — Foundation Integration Day

No one starts feature development until:

```text
[ ] OpenChoreo works
[ ] Backstage works
[ ] sample application works
[ ] Codespaces test completed
[ ] GitHub board configured
[ ] project structure committed
[ ] dependency syntax understood
[ ] everyone understands architecture
```

Create all project issues and assign owners.

Tag milestone:

```text
M0 · Foundations
```

---

# 12. PHASE 1 — Application Development

## 11–14 August

This is where all three members work in parallel.

---

# August 11

## Member 1 — Auth

Implement:

```text
GET /healthz
GET /readyz
POST /register
POST /login
```

Create SQLite user repository.

Recommended code structure:

```text
handler/
store/
token/
```

Tests:

```text
successful registration
duplicate registration
successful login
incorrect password
unknown account
```

---

## Member 2 — Scan Foundation

Implement:

```text
GET /healthz
GET /readyz

POST /scan
GET /scan/{id}
GET /scans
```

Create:

```text
Scan
Finding
ScanStatus
Store
```

Statuses:

```text
queued
running
complete
failed
```

---

## Member 3 — Notification

Implement:

```text
GET /healthz
GET /readyz
POST /notifications
GET /notifications
```

Create SQLite notification store.

Tests:

```text
create notification
invalid request
list notifications
empty database
health endpoint
```

---

# August 12

## Member 1

Implement JWT.

Environment variable:

```text
JWT_SECRET
```

Functions should approximately separate into:

```text
GenerateToken()
ValidateToken()
ParseBearerToken()
```

Test:

```text
valid token
expired token
wrong signing secret
malformed token
missing token
```

Create multi-stage Dockerfile.

---

## Member 2

Implement security checks.

### TLS

Check:

```text
TLS handshake succeeds
certificate valid
certificate not expired
certificate expiry date
```

### Security headers

Check:

```text
Strict-Transport-Security
X-Content-Type-Options
Content-Security-Policy
```

### Redirect

Determine whether:

```text
http://target
```

redirects to:

```text
https://target
```

Every network call gets a timeout.

---

## Member 3

Start Next.js.

Pages:

```text
/login
/register
/dashboard
/scans/[id]
```

Implement authentication forms first.

Keep UI basic.

Do not spend time on animations.

---

# August 13

## Member 1

Finish Auth.

Add:

```text
structured JSON logging
environment configuration
error handling
Docker image
```

Run complete Auth unit test suite.

Open Auth PR.

---

## Member 2

Implement:

```text
async worker
scan queue
score calculation
finding generation
notification client
```

Important rule:

If Notification Service fails:

```text
Scan must still complete.
```

Log the notification failure.

Do not fail the scan.

Tests:

```text
queued → running
running → complete
timeout → failed
score calculation
notification failure
```

---

## Member 3

Connect:

```text
Scan → Notification
```

using:

```text
NOTIFICATION_SERVICE_URL
```

Finish web:

```text
login
registration
new scan
scan list
scan detail
notifications
```

---

# August 14

## Member 1

Review configuration across all services.

Ensure:

```text
no hardcoded URL
no committed password
no committed JWT secret
ports configurable
database paths configurable
```

## Member 2

Perform backend regression tests.

Run:

```bash
go test ./...
go vet ./...
```

Fix edge cases.

## Member 3

Create:

```text
deploy/compose.yaml
scripts/smoke-test.sh
```

Smoke test must execute:

```text
register
   ↓
login
   ↓
receive JWT
   ↓
submit scan
   ↓
poll scan
   ↓
receive result
   ↓
notification exists
```

---

## M1 Gate

Do not proceed unless:

```text
docker compose up
```

starts the complete system.

And:

```bash
scripts/smoke-test.sh
```

passes.

---

# 13. PHASE 2 — First OpenChoreo Deployment

## 15–17 August

---

# August 15

## Member 1

Own OpenChoreo deployment.

Create:

```text
platform/project.yaml

platform/auth/component.yaml
platform/auth/workload.yaml
```

Deploy Auth.

Debug until healthy.

---

## Member 2

Build production images.

Verify:

```text
auth
scan
notification
```

containers independently.

Publish versioned GHCR images.

Example:

```text
ghcr.io/<team>/securecloud-auth:v0.1.0
ghcr.io/<team>/securecloud-scan:v0.1.0
ghcr.io/<team>/securecloud-notification:v0.1.0
```

---

## Member 3

Create deployment verification checklist.

Test:

```text
gateway health endpoint
register through gateway
login through gateway
invalid credentials
```

Capture useful screenshots.

---

# August 16

## Member 1

Move JWT secret into OpenChoreo SecretReference.

No actual secret values in Git.

Verify Auth through gateway.

## Member 2

Attack/test Auth deployment.

Verify:

```text
invalid token rejected
missing token rejected
bad login rejected
health probe available
```

## Member 3

Update:

```text
docs/runbook.md
```

Document deployment commands and gateway URLs.

---

# August 17

## Member 1

Add persistent-volume trait.

Test pod restart.

## Member 2

Verification:

```text
register user
delete Auth pod
wait for replacement
login again
```

User must still exist.

## Member 3

Run complete regression test.

Update runbook.

---

# 14. PHASE 3 — Full OpenChoreo System

## 18–21 August

---

# August 18

## Member 1

Create/deploy Scan OpenChoreo manifests.

## Member 2

Verify real Scan behavior inside OpenChoreo:

```text
TLS
headers
redirect
score
database
timeouts
```

## Member 3

Prepare dashboard runtime configuration for OpenChoreo URLs.

---

# August 19

## Member 1

Implement OpenChoreo dependency:

```text
Scan
 ↓
Notification
```

No hardcoded hostname.

## Member 2

Deploy Notification Service as **internal only**.

Perform security test:

External access should fail.

Internal Scan → Notification access should succeed.

## Member 3

Perform E2E test:

```text
submit scan
scan completes
notification generated
```

---

# August 20

## Member 3 — Main Owner

Deploy dashboard as:

```text
deployment/web-application
```

## Member 1

Assist with OpenChoreo configuration.

## Member 2

Perform regression/security testing after UI deployment.

Final browser flow:

```text
Register
   ↓
Login
   ↓
Dashboard
   ↓
Create scan
   ↓
Scan runs
   ↓
Results
   ↓
Notification
```

---

# August 21

## Member 1

Promote Auth:

```text
development
     ↓
staging
```

Create environment-specific configuration.

## Member 2

Run complete deployed-system smoke test.

Verify Notification cannot be accessed externally.

## Member 3

Create first architecture diagram draft and capture deployment screenshots.

---

# M3 Gate

Required:

```text
[ ] Auth deployed
[ ] Scan deployed
[ ] Notification deployed
[ ] Web deployed
[ ] browser flow works
[ ] Scan → Notification dependency works
[ ] Notification is internal
[ ] smoke test passes
[ ] staging promotion demonstrated
```

At this point, the project is already portfolio-ready.

Everything after this improves the platform-engineering story.

---

# 15. PHASE 4 — CI/CD + Observability

## 22–24 August

Use the larger machine/Codespace planned for the full OpenChoreo installation.

---

# August 22

## Member 1 — Workflow Plane

Configure source build for **Notification Service first**.

Flow:

```text
Git source
   ↓
OpenChoreo Workflow Plane
   ↓
build
   ↓
container image
   ↓
deployment
```

Do not spend unlimited time debugging it.

If Notification source build works, only then attempt Auth and Scan.

---

## Member 2

Prepare monitoring verification.

Check structured logs.

Create failure scenarios.

Examples:

```text
bad scan target
HTTP timeout
service restart
500 response
```

---

## Member 3

Begin GitHub Actions CI.

PR pipeline:

```text
checkout
   ↓
Go setup
   ↓
go vet
   ↓
go test
   ↓
lint
   ↓
Docker builds
   ↓
Next.js lint/build
```

---

# August 23

## Member 1

Extend Workflow Plane builds if stable.

Otherwise help integration.

## Member 2

Own Observability.

Verify logs from:

```text
Auth
Scan
Notification
```

Create alert.

Trigger it deliberately.

Capture:

```text
logs screenshot
metrics screenshot
alert screenshot
```

## Member 3

Run failure-injection tests and verify the observable events appear correctly.

---

# August 24

## Member 1

Infrastructure cleanup and deployment validation.

## Member 2

Run full backend test suite.

Check regression after CI/CD changes.

## Member 3

Finish GitHub Actions.

Create image publishing workflow.

`main` pipeline:

```text
merge
 ↓
test
 ↓
build
 ↓
container image
 ↓
GHCR
```

Add badges to README.

---

# 16. Testing Strategy

Testing is mandatory before merge.

---

## Auth Unit Tests

Test:

```text
registration succeeds
duplicate email fails
password stored hashed
valid login succeeds
wrong password fails
JWT generated
JWT validated
expired JWT rejected
tampered JWT rejected
missing Authorization rejected
```

Important packages:

```text
internal/token
internal/store
internal/handler
```

---

# Scan Unit Tests

This service deserves the largest test suite.

Test:

```text
TLS valid
TLS invalid
TLS expired
TLS close to expiry
HSTS present
HSTS missing
X-Content-Type-Options present
CSP present
HTTP → HTTPS redirects
no redirect
timeout
score calculation
status transitions
notification success
notification failure
```

Use Go's testing facilities and `httptest` where practical instead of relying entirely on public internet hosts.

---

# Notification Tests

Test:

```text
POST valid notification
POST malformed JSON
POST missing scan_id
GET empty notifications
GET existing notifications
SQLite persistence
health
ready
```

---

# Frontend Tests

Because of the deadline, prioritize:

```text
lint
production build
critical UI tests
E2E smoke test
```

Critical user flows:

```text
login
submit scan
view scan
view notification
```

Do not spend two days trying to reach 100% React test coverage.

---

# 17. Integration Testing

Create one script:

```text
scripts/smoke-test.sh
```

It should fail immediately if any step fails.

Pseudo-flow:

```text
health check
    ↓
register
    ↓
login
    ↓
extract token
    ↓
create scan
    ↓
extract scan ID
    ↓
poll status
    ↓
verify complete
    ↓
verify findings
    ↓
GET notifications
    ↓
verify notification exists
```

Run it against:

1. Docker Compose
2. OpenChoreo development
3. relocated/full cluster
4. final release

---

# 18. Platform Testing

These are not traditional unit tests, but they are essential.

### Manifest validation

```bash
kubectl apply --dry-run=server -f platform/
```

### Health

```text
Auth /healthz → 200
Scan /healthz → 200
Notification /healthz → 200
```

### Persistence

```text
create data
delete pod
wait
check data
```

### Security boundary

Notification Service:

```text
external request → must fail
Scan internal request → must succeed
```

### Environment promotion

```text
development → staging
```

Verify both environments independently.

---

# 19. PHASE 5 — Documentation & Release

## 25–27 August

**Feature freeze starts here.**

No new functionality unless necessary to fix a blocking defect.

---

# August 25

## Member 1

Write:

```text
docs/architecture.md
```

Explain:

```text
OpenChoreo architecture
Component
ComponentRelease
ReleaseBinding
Data Plane
Workflow Plane
service dependencies
secrets
promotion
```

Write ADRs.

---

## Member 2

Document:

```text
security scanning design
testing approach
security boundaries
observability
known limitations
non-goals
```

Help produce architecture diagram.

---

## Member 3

Lead README.

Required structure:

```text
Project overview
Architecture
Why OpenChoreo
Features
Technology stack
Screenshots
Local setup
OpenChoreo setup
Project structure
Testing
CI/CD
Observability
Design decisions
Known limitations
Demo
```

---

# August 26 — Demo Day

Freeze a known-good commit before recording.

Example tag/branch:

```text
demo-candidate
```

Member 3 coordinates recording.

Suggested 5-minute demo:

```text
0:00–0:30
Member 3
Project introduction + architecture

0:30–1:30
Member 3
Browser demo

1:30–2:30
Member 1
OpenChoreo / Backstage architecture

2:30–3:30
Member 1
CI/CD / deployment

3:30–4:30
Member 2
Logs, security boundary and alert

4:30–5:00
Team
Lessons learned / future work
```

Keep at least one working recording immediately after you have it.

---

# August 27 — Release Candidate

## Member 2 — Independent Fresh-Clone Test

Member 2 should behave as if they have never seen the project.

```bash
git clone ...
```

Then follow README exactly.

Every missing command or incorrect instruction becomes a bug.

This is extremely important.

## Member 1

Fix deployment/documentation issues discovered during fresh-clone testing.

## Member 3

Final README cleanup.

Create release notes.

After tests pass:

```bash
git tag -a v1.0.0 -m "SecureCloud v1.0.0"
git push origin v1.0.0
```

Create GitHub Release.

---

# August 28 — BUFFER / FINAL DEADLINE

Do not schedule new features.

Only:

```text
critical bug fixes
broken documentation
demo/link fixes
README corrections
release verification
```

Final checklist:

```text
[ ] main CI green
[ ] Docker Compose works
[ ] unit tests green
[ ] smoke test green
[ ] OpenChoreo deployment works
[ ] browser E2E works
[ ] Notification internal-only verified
[ ] README complete
[ ] architecture document complete
[ ] architecture diagram complete
[ ] screenshots available
[ ] demo video available
[ ] v1.0.0 release exists
```

---

# 20. Existing GitHub Issue Ownership

Use the existing 43-issue structure instead of throwing it away.

## Member 1 — Platform/Auth

Primary ownership:

```text
#1   OpenChoreo environment
#4   OpenChoreo resource chain
#7   GCP microservices reference
#8   Auth scaffold
#9   Register/login
#10  JWT
#22  OpenChoreo project
#23  Auth deployment
#25  SecretReference
#26  Persistent storage
#29  Scan → Notification dependency
#31  staging promotion
#33  Workflow Plane build
#34  additional source builds
#39  architecture/ADRs
```

---

## Member 2 — Scan/Security

Primary ownership:

```text
#3   Docker/Kubernetes fundamentals
#11  Scan scaffold
#12  TLS check
#13  headers + redirect
#14  worker/scoring/API
#17  configuration security review
#21  container image verification
#27  Scan deployment
#28  Notification security visibility test
#32  cluster smoke test
#35  logs
#36  alerting
#40  architecture diagram
```

---

## Member 3 — Web/Notification/QA

Primary ownership:

```text
#2   Backstage exploration/screenshots
#5   GitHub setup
#6   repository scaffold
#15  Notification API
#16  Scan → Notification local wiring
#18  Web authentication
#19  Web scan flow
#20  Docker Compose/smoke test
#24  gateway verification
#30  dashboard deployment
#37  GitHub Actions
#38  README
#41  demo
#42  fresh-clone verification
#43  v1.0.0 release
```

---

# 21. Daily Team Workflow

At the beginning of every work session:

```bash
git checkout main
git pull
```

Check GitHub board.

Choose an assigned issue.

Move:

```text
Backlog
   ↓
In Progress
```

Create branch.

Example:

```bash
git checkout -b feat/12-scan-tls-check
```

Work.

Run tests.

Commit small changes:

```bash
git add -p
git commit -m "feat(scan): add TLS certificate validation"
```

Push:

```bash
git push -u origin feat/12-scan-tls-check
```

Open PR.

Another member reviews.

CI passes.

Merge.

Move issue:

```text
In Review
   ↓
Done
```

Then pull the latest `main` before starting anything else.

---

# 22. Daily Team Sync

Have one short synchronization every day.

Each member answers only:

```text
1. What did I finish?
2. What am I doing next?
3. What is blocking me?
4. Do I need another member's code?
```

Do not turn it into a one-hour meeting.

The purpose is dependency management.

---

# 23. Important Dependency Order

Some tasks cannot truly be parallel.

```text
OpenChoreo verification
        ↓
Local services
        ↓
Docker Compose
        ↓
Auth deployed
        ↓
Scan deployed
        ↓
Notification internal deployment
        ↓
Scan → Notification dependency
        ↓
Web deployment
        ↓
Full E2E
        ↓
CI/CD + Observability
        ↓
Documentation
        ↓
Demo
        ↓
Release
```

Never bypass this chain because one member “has nothing to do.”

They should review/test/document instead.

---

# 24. What NOT to Add

Do not use the three-person team as an excuse to add:

```text
CVE scanner
port scanner
RBAC
OAuth login
refresh-token system
email notifications
SMS notifications
PostgreSQL migration
complex dashboard animations
multi-region deployment
load testing infrastructure
extra microservices
```

Finish the intended project first.

---

# 25. Scope-Cut Rule

If behind schedule, remove optional functionality before compromising delivery.

Suggested removal order:

```text
1. Additional database/platform experiments
2. Full observability if infrastructure unavailable
3. Source builds for Auth and Scan
4. Staging promotion
5. Persistent volume
6. Dashboard visual polish
7. Alerting
```

Never remove:

```text
README
architecture document
architecture diagram
working core system
tests
demo video
```

---

# 26. Final Responsibility Rule

Although each member owns an area:

**Member 1 must be able to explain Scan and Web.**

**Member 2 must be able to explain Auth and OpenChoreo.**

**Member 3 must be able to explain backend service communication and deployment.**

Before August 28, perform one architecture walkthrough where each person explains another person's component.

That proves this is genuinely a group project instead of three disconnected assignments.

---

# Final Target

By the end of August 27 the repository should already contain:

```text
SecureCloud v1.0.0

✓ 3 Go microservices
✓ Next.js dashboard
✓ SQLite database-per-service
✓ JWT authentication
✓ TLS/header/redirect security scanning
✓ Dockerized services
✓ Docker Compose local environment
✓ OpenChoreo deployment
✓ declared service dependency
✓ Kubernetes secret handling
✓ internal Notification Service
✓ environment promotion
✓ GitHub Actions CI
✓ container publishing
✓ Workflow Plane demonstration
✓ structured logging
✓ observability demonstration
✓ automated unit tests
✓ automated smoke test
✓ architecture documentation
✓ ADRs
✓ screenshots
✓ demo video
✓ clean GitHub issue/PR history
```

**August 28 remains a buffer, not a development day.**
