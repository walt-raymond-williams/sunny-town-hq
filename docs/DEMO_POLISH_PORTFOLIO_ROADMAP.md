# Demo Polish And Portfolio Roadmap

Issue: #59
Epic: #39
Discovery: `docs/DEMO_POLISH_PORTFOLIO_DISCOVERY.md`

## Outcome

The repository should be easy to understand, run, review, and demo from the README without requiring a reviewer to mine the planning docs first.

The first demo story should be:

```text
HQ learning app context
  -> student login
  -> Sunny Town
  -> full-viewport realtime world
  -> hotbar-selected pickaxe/mining/inventory/progression
  -> Cookie Keeper home/work routine with visual cues and debug inspectability
```

## Execution Order

1. Reuse #7: update the README and add first demo assets.
2. Add a small runtime/demo smoke issue if README verification reveals that demo users or local-network setup are still too easy to miss.
3. Reuse #5: clean frontend production bundle warnings.
4. Add optional Playwright/demo-readiness coverage after the README path is stable.

## Child Issue Drafts

### Reuse #7: Add README Architecture Highlights And Demo Assets

Existing issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/7

Recommended revised scope:

As a reviewer, I want the README to show the product, the strongest demo path, and the architecture choices worth noticing so I can evaluate the project quickly.

Acceptance criteria:

- Add a short "Demo Path" or "What To Demo" section to `README.md`.
- Surface the imported local demo users:
  - `playwright-student` / `playwright`
  - `playwright-teacher` / `playwright`
- Add or link committed demo media under `docs/assets/demo/`.
- Use the Cookie Keeper life/work loop as the primary Sunny Town demo story.
- Include a compact architecture highlights section covering:
  - HQ durable state ownership
  - Sunny Town realtime state ownership
  - server-authoritative gameplay validation
  - service-authenticated internal APIs
  - idempotent ledgers for retries
  - Docker Compose reproducible runtime
  - disciplined AI-assisted engineering workflow
- Keep setup instructions accurate and link to `docs/current/RUNTIME.md` and `docs/PROJECT_SETUP.md` for details.
- Mention local-network host consistency prominently near Quick Start.
- Do not commit generated `web/` assets or local logs.

Implementation notes:

- Start from `docs/DEMO_POLISH_PORTFOLIO_DISCOVERY.md`.
- Store media in `docs/assets/demo/`.
- Prefer one screenshot plus one short routine GIF/video poster in the first pass.
- Keep README concise. Link to `docs/current/` for deep technical detail.
- If a video/GIF is too large, commit a poster screenshot and document the external clip/link decision instead.

Verification:

```powershell
git diff --check
docker compose -f deploy/docker-compose.yml config
```

If runtime instructions are changed:

```powershell
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Expected health response for both services: `ok`.

### New Draft: Verify Fresh Demo Runtime And Demo Users

Suggested title: `Verify fresh demo runtime and README demo login path`

As a maintainer, I want the README demo path to be verified against a fresh or known-clean local runtime so reviewers do not hit login/setup confusion before seeing the product.

Acceptance criteria:

- Verify `docker compose -f deploy/docker-compose.yml up -d --build` starts the full local stack.
- Verify HQ and Sunny Town health endpoints return `ok`.
- Verify the README-described demo host matches Keycloak issuer expectations.
- Verify `playwright-student` can log in and reach Sunny Town in a fresh Keycloak realm.
- Verify `playwright-teacher` can log in and reach the teacher workspace in a fresh Keycloak realm.
- Document the expected behavior when an existing Keycloak volume predates the imported demo users.
- Add a short troubleshooting note if needed, without turning README into a long setup manual.

Implementation notes:

- This may be handled inside #7 if the README change is small and the runtime is already clean.
- Prefer documenting a volume-reset caveat over adding risky automatic reset commands.
- Do not add production reset endpoints for demo convenience.
- If manual browser verification is required, record exact host, account, and result in the issue comment.

Verification:

```powershell
docker compose -f deploy/docker-compose.yml config
docker compose -f deploy/docker-compose.yml up -d --build
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18080/healthz | Select-Object -ExpandProperty Content
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/healthz | Select-Object -ExpandProperty Content
```

Manual browser verification:

- Log in at `http://127.0.0.1:18080`.
- Use `playwright-student` / `playwright`.
- Open Sunny Town and confirm the game surface renders.
- Log out or use a separate browser context.
- Use `playwright-teacher` / `playwright`.
- Confirm the teacher workspace loads.

### Reuse #5: Reduce Frontend Production Bundle Warnings

Existing issue: https://github.com/walt-raymond-williams/sunny-town-hq/issues/5

Recommended revised scope:

As a maintainer, I want `npm run build` output to be clean enough for demo/readiness work, while keeping bundle changes low-risk.

Acceptance criteria:

- Normalize `studentAssignmentsApi.ts` imports so Vite no longer warns that the module is both dynamically and statically imported, or document why the warning remains acceptable.
- Evaluate large chunk warnings after the import cleanup.
- Add route-level splitting or manual chunks only if it clearly reduces first-load/demo friction without making the frontend harder to maintain.
- Keep behavior unchanged.

Implementation notes:

- Current warning sources include:
  - `frontend/src/composables/useStudentAssignments.ts`
  - `frontend/src/composables/useSunnyTownNpcInteractions.ts`
- Prefer the smallest import-shape fix before introducing chunk configuration.
- Do not chase perfect bundle size unless measured demo load time is bad.

Verification:

```powershell
cd frontend
npm run build
```

### New Draft: Add First Demo Smoke Checklist Or Playwright Slice

Suggested title: `Add first demo readiness smoke checklist or Playwright slice`

As a maintainer, I want a repeatable demo-readiness check so the README path does not silently regress after future feature work.

Acceptance criteria:

- Decide whether the first version is a manual checklist, a Playwright smoke test, or both.
- Cover at least:
  - app loads from Docker-served HQ
  - student demo login works
  - Sunny Town route loads
  - Sunny Town canvas renders nonblank
  - inventory/hotbar panel can open
  - Cookie Keeper is visible or routine state can be inspected
- Link any browser tests to existing `docs/PLAYWRIGHT_E2E_TEST_PLAN.md` IDs where practical.
- Keep deterministic backend checks separate from browser checks.

Implementation notes:

- This should follow #7 so the README demo path is stable before automation encodes it.
- If Playwright login is brittle, start with a committed manual checklist and create a narrower follow-up for automation.
- Prefer the Docker-served app at `http://127.0.0.1:18080`.

Verification:

```powershell
task test
task frontend:build
task compose:rebuild-runtime
task health
```

If Playwright is added:

```powershell
cd frontend
npm run test:e2e
```

## Asset Storage Rule

Use:

```text
docs/assets/demo/
```

Commit:

- small PNG screenshots
- small optimized GIFs only if size is reasonable
- poster screenshots for external videos

Do not commit:

- generated `web/` assets
- local logs
- raw screen recordings
- large binary exports that make the repo unpleasant to clone

## Recommended #39 Status After #59

Move #39 from `status:needs-design` to a normal execution posture once #59 is closed. The recommended next executable issue is #7, followed by the fresh demo runtime verification issue if #7 does not absorb it.

Keep #39 open until:

- README front door is updated.
- First demo assets exist or an explicit no-asset decision is documented.
- Demo runtime/login path is verified.
- Bundle warning posture is either cleaned up or documented as acceptable.
- A repeatable demo-readiness checklist or smoke test exists.
