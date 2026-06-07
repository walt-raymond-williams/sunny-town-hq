# Test Case Coverage Matrix

This document maps `docs/USER_STORIES_AND_REQUIREMENTS.md` to concrete test cases and current coverage. It is based on the current docs and the existing code/tests as of this audit.

Coverage labels:

- `Covered`: meaningful automated tests already exist.
- `Partial`: some important behavior is tested, but notable acceptance criteria are not.
- `Gap`: no meaningful automated coverage was found.
- `Manual/build`: best covered today by build checks or manual browser verification unless the project adds UI/e2e tooling.

Verification run during this audit:

```powershell
go test -count=1 ./cmd/hq
go test -count=1 ./cmd/sunny-town ./internal/sunnytownauth
```

Both passed.

## Summary

Current automated coverage is strongest for:

- Sunny Town map loading, rooms, movement validation, portals, collectibles, mining, resource-node lifecycle, NPC map validation, and map scoping.
- HQ durable reward/resource idempotency.
- HQ inventory, equipment, crafting, shop purchase, map-object placement/removal, pet feeding, saved Sunny Town position, and hotbar helper behavior.
- Sunny Town join-token signing and verification.

Current automated coverage is weakest for:

- HTTP auth and role enforcement across most HQ endpoints.
- Teacher assignment create/list/grade/reset/delete workflows.
- Student assignment next/submit/graded workflows.
- Pet play/sleep/wake/decay behavior.
- Sunny Town WebSocket handler behavior such as unknown messages, message size limits, rate limiting, ping, and `equipment_changed`.
- Frontend behavior. There are no frontend unit/component/e2e test dependencies in `frontend/package.json`; current verification is `npm run build`.

Recommendation: add more tests, but target the highest-leverage backend gaps first. Full browser automation is useful later, but it is not the best immediate return unless frontend regressions become frequent.

## Recommended Test Backlog

### P0: Add HQ Schoolwork Workflow Tests

Worth it: yes. These are core app workflows, currently under-tested, and inexpensive to cover as handler or DB-level tests.

Proposed cases:

- Teacher can create assignment with valid category/prompt/expected answer.
- Invalid category is rejected.
- Student can fetch next unanswered assignment.
- Student can submit an answer once.
- Student cannot submit the same active assignment twice.
- Teacher can grade latest active attempt pass/fail with feedback.
- Passing grade awards one cookie once.
- Teacher can reset latest active attempt and preserve feedback/history.
- Reset assignment becomes answerable again.
- Student graded endpoint returns only graded attempts for that student.
- Teacher delete removes assignment from lists.

Likely location: `cmd/hq/assignment_test.go`

### P0: Add HQ Auth/Role Boundary Tests

Worth it: yes. Authorization bugs are high-impact and easy to regress as endpoints grow.

Proposed cases:

- Unauthenticated requests to `/api/...` are rejected.
- Student-only endpoints reject teacher-only users.
- Teacher-only endpoints reject student-only users.
- `/api/internal/sunny-town/*` endpoints reject missing/wrong `X-HQ-Service-Secret`.
- Sunny Town session endpoint rejects non-student users.

Likely location: `cmd/hq/auth_handler_test.go` or endpoint-specific handler tests.

### P1: Add Pet Lifecycle Tests

Worth it: yes. The pet lifecycle has time-based rules that are easy to break and hard to verify manually.

Proposed cases:

- `playWithStudentPet` increases happiness and spends energy.
- Play is rejected or bounded when energy is insufficient.
- `putStudentPetToSleep` sets sleep fields.
- Sleeping recovery reaches 100 energy and wakes automatically after the configured duration.
- Early wake stops recovery and subtracts happiness.
- Awake decay reduces hunger/happiness/energy over elapsed time.
- Awake energy reaching 0 puts pet to sleep.

Likely location: `cmd/hq/pet_service_test.go`

### P1: Add Sunny Town WebSocket Protocol Tests

Worth it: yes for server protocol edge cases, but keep these focused. The room tests already cover most game rules.

Proposed cases:

- Unknown message produces an `error` response.
- Invalid JSON or oversized message closes/rejects appropriately.
- `ping` produces expected keepalive response if supported.
- `equipment_changed` reloads equipment and updates future snapshots.
- `place_object` success broadcasts `map_object_placed` and sends inventory quantity only to acting player.
- Pickaxe removal of placed block broadcasts `map_object_removed`.
- Tool cooldown blocks repeated mining swings inside the cooldown window.

Likely location: `cmd/sunny-town/websocket_test.go` or additional room/client tests.

### P1: Add Schema/User Sync Tests

Worth it: yes if schema churn continues. Starter equipment and app-user role sync are foundational but currently inferred from migration code.

Proposed cases:

- Database migrations seed catalog item types.
- Existing students receive starter equipment without duplicating quantities.
- `internal/hq/users.Store.SyncAuthenticated` creates app user, roles, wallet/pet defaults, starter equipment, and default hotbar.
- Removing an owned equipped item causes equipment load to hide it.

Likely location: `internal/hq/users/users_test.go`

### P2: Add Frontend Type-Level or Unit Tests

Worth it: moderate. Add only after choosing a frontend test stack such as Vitest plus Vue Test Utils. Today `npm run build` already gives useful TypeScript and production-bundle coverage.

Proposed cases:

- Assignment status helpers choose latest graded/current attempts correctly.
- Student inventory store loads inventory/equipment/recipes and refreshes inventory after equip/unequip/craft.
- Sunny Town API client builds the expected session and hotbar requests.
- Auth helper handles token refresh failure by returning the user to login.

Likely location: `frontend/src/**/*.test.ts`

### P2: Add Minimal Browser Smoke Tests

Worth it: later. This becomes valuable after frontend behavior stabilizes and logged-in test users can be provisioned reliably.

Proposed cases:

- Student can log in, open assignments, submit an answer, open grades, open pet inventory.
- Teacher can log in, create an assignment, grade/reset/delete it.
- Student can enter Sunny Town and receive the first map.
- Sunny Town inventory overlay opens with `E`; pressing `E` does not break movement.

Likely tool: Playwright, once added to `frontend/` or an integration test folder.

## Coverage By Requirement Area

| Requirement area | Coverage | Existing tests | Recommended action |
|---|---:|---|---|
| Keycloak authentication and role sync | Partial | `internal/sunnytownauth/token_test.go` covers Sunny Town join token role requirement; no broad HQ bearer-token handler tests found. | Add P0 auth/role handler tests. |
| Local-network host consistency | Manual/build | No automated tests found. | Keep as setup docs/manual verification unless this becomes a recurring support issue. |
| HQ/Sunny Town service boundary | Partial | HQ internal resource endpoint secret test; resource/reward commits use HQ helpers. | Add service-secret tests for every internal endpoint. |
| Health checks/runtime | Manual/build | No direct health handler test found; verified through commands. | Optional low-value unit test; Docker health checks are enough. |
| Teacher create/list/review/grade/reset/delete | Gap | No assignment workflow tests found. | Add P0 schoolwork tests. |
| Student next assignment/submit/grades/history | Gap | No assignment workflow tests found. | Add P0 schoolwork tests. |
| Pet view/feed | Partial | `TestFeedStudentPetConsumesCookieInventory`, `TestFeedStudentPetRequiresCookieInventory`. | Add P1 pet lifecycle tests for play/sleep/wake/decay. |
| Stars as wallet currency | Covered | `TestCommitSunnyTownRewardIdempotent`, `TestApplyGameResultCreditsPetStarsOnce`, shop ledger tests. | Maintain. |
| Inventory ownership | Partial | Inventory mutations are covered through feed/shop/craft/equipment/map-object tests. | Add direct `loadStudentInventory` hides zero-quantity items if regressions occur. |
| Equipment | Covered | `TestEquipStudentItemRequiresOwnedEquippableItem`, reject cookie/unowned, unequip, tool slot. | Add wrong-slot test; otherwise good. |
| Starter equipment | Gap | No direct schema/sync test found. | Add P1 schema/user sync tests. |
| Shop purchase | Covered | Buy cookie, insufficient stars, invalid purchase. | Maintain. |
| Crafting | Covered | Craft success, insufficient ingredients, unknown recipe. | Maintain. |
| Sunny Town session/join token | Partial | Join-token sign/verify; room join saved position; join target validation. | Add handler test for `POST /api/student/sunny-town/session` response and role rejection. |
| Restore last Sunny Town position | Covered | `TestSaveSunnyTownPositionUpsertsLastLocation`, `TestRoomJoinUsesSavedPosition`, `TestWorldJoinUsesClaimMap`. | Maintain. |
| Local movement feel | Partial | Server accepted-state tests exist; no frontend local prediction tests. | Backend is covered; frontend remains manual/build unless adding UI tests. |
| Server-accepted movement | Covered | Movement, bounds, out-of-order, blocked geometry, placed object, no drift, last processed seq. | Maintain. |
| Multiplayer map scoping | Covered | Current map snapshots, interior sharing, NPC scoping. | Maintain. |
| Map loading and validation | Covered | Checked-in maps, duplicate maps, portal targets, resources, NPC validation. | Maintain. |
| Portals | Covered | Door transfer, forest transfer, target portal guard. | Maintain. |
| Stars/collectibles | Covered | Pickup overlap, pickup requires overlap, indoor map no stars, HQ idempotent reward. | Maintain. |
| NPC/shop/schoolwork definitions | Partial | NPC map validation and scoped map messages covered; frontend interaction flow not covered. | Add UI/e2e later if NPC regressions occur. |
| Mining nodes | Covered | Requires pickaxe/inventory, range, three swings, inactive/respawn, resource snapshots, HQ idempotent resource commit. | Add cooldown test. |
| Placed stone block placement/removal | Partial | HQ placement/removal persistence and room placement validation; movement blocked by placed object. | Add Sunny Town message/broadcast tests. |
| WebSocket client/server protocol | Partial | Room-level behavior covered; direct WebSocket edge cases mostly absent. | Add P1 protocol tests. |
| Frontend inventory store/equipment broadcast | Manual/build | No frontend tests found. | Add P2 unit tests after adopting Vitest. |
| Verification commands | Covered by process | `go test -count=1` passed in this audit. | Continue running scoped checks before finalizing changes. |

## Suggested Test Case IDs

Use these IDs when turning the backlog into test files. Keep IDs stable so future docs and failures can refer to them.

### Auth and Roles

- `TC-AUTH-001`: unauthenticated API request returns unauthorized.
- `TC-AUTH-002`: student endpoint rejects teacher-only user.
- `TC-AUTH-003`: teacher endpoint rejects student-only user.
- `TC-AUTH-004`: internal Sunny Town endpoint rejects missing service secret.
- `TC-AUTH-005`: internal Sunny Town endpoint rejects wrong service secret.
- `TC-AUTH-006`: Sunny Town session rejects non-student user.
- `TC-AUTH-007`: Sunny Town join token rejects expired token.
- `TC-AUTH-008`: Sunny Town join token rejects wrong secret.

### Schoolwork

- `TC-SCHOOL-001`: teacher creates valid assignment.
- `TC-SCHOOL-002`: teacher create rejects invalid category.
- `TC-SCHOOL-003`: student fetches next unanswered assignment.
- `TC-SCHOOL-004`: student category filter returns matching category only.
- `TC-SCHOOL-005`: student submits answer and current attempt is created.
- `TC-SCHOOL-006`: duplicate active submission is rejected.
- `TC-SCHOOL-007`: teacher lists answered assignments needing review.
- `TC-SCHOOL-008`: teacher grades pass with feedback.
- `TC-SCHOOL-009`: passing grade awards exactly one cookie.
- `TC-SCHOOL-010`: teacher grades fail with feedback.
- `TC-SCHOOL-011`: teacher reset preserves old attempt and makes assignment answerable.
- `TC-SCHOOL-012`: student sees reset attempt history.
- `TC-SCHOOL-013`: student graded endpoint excludes ungraded attempts.
- `TC-SCHOOL-014`: teacher deletes assignment.

### Pet

- `TC-PET-001`: feed consumes one cookie and increases hunger.
- `TC-PET-002`: feed without cookie is rejected.
- `TC-PET-003`: play increases happiness and spends energy.
- `TC-PET-004`: pet stats remain within 0-100.
- `TC-PET-005`: put to sleep sets sleeping state.
- `TC-PET-006`: sleep completes energy recovery and auto-wakes.
- `TC-PET-007`: early wake subtracts happiness and stops recovery.
- `TC-PET-008`: elapsed awake decay reduces stats.
- `TC-PET-009`: zero awake energy causes sleep.

### Inventory, Equipment, Shop, Crafting

- `TC-INV-001`: inventory API omits zero-quantity items.
- `TC-INV-002`: stars are not returned as inventory.
- `TC-INV-003`: starter items are granted to existing students.
- `TC-INV-004`: starter items are granted to new synced students.
- `TC-EQ-001`: equip owned gear succeeds.
- `TC-EQ-002`: equip unowned item fails.
- `TC-EQ-003`: equip non-equippable cookie fails.
- `TC-EQ-004`: equip wrong slot fails.
- `TC-EQ-005`: unequip clears slot without consuming inventory.
- `TC-SHOP-001`: buy cookie spends stars and increments inventory.
- `TC-SHOP-002`: buy cookie rejects insufficient stars.
- `TC-CRAFT-001`: craft stone block consumes 4 rock and creates 1 stone block.
- `TC-CRAFT-002`: craft rejects missing ingredients without mutation.

### Sunny Town

- `TC-ST-001`: session response includes room/map/avatar/ws/token/expiry/wallet.
- `TC-ST-002`: join uses saved position.
- `TC-ST-003`: no saved position joins default map.
- `TC-ST-MOVE-001`: accepted move updates position/facing/moving.
- `TC-ST-MOVE-002`: out-of-order move is ignored.
- `TC-ST-MOVE-003`: invalid or out-of-bounds move is rejected/clamped.
- `TC-ST-MOVE-004`: blocked geometry rejects movement.
- `TC-ST-MOVE-005`: placed object rejects movement.
- `TC-ST-MOVE-006`: snapshot includes last processed sequence.
- `TC-ST-MAP-001`: checked-in maps load successfully.
- `TC-ST-MAP-002`: duplicate map ID is rejected.
- `TC-ST-MAP-003`: missing portal target is rejected.
- `TC-ST-PORTAL-001`: portal sends player to target map.
- `TC-ST-PORTAL-002`: portal target guard prevents immediate bounce.
- `TC-ST-MULTI-001`: same-map players see each other.
- `TC-ST-MULTI-002`: different-map players do not see each other.
- `TC-ST-STAR-001`: star pickup enqueues idempotent reward from accepted overlap.
- `TC-ST-STAR-002`: non-overlap does not enqueue reward.
- `TC-ST-MINE-001`: mining requires pickaxe.
- `TC-ST-MINE-002`: mining requires range.
- `TC-ST-MINE-003`: mining requires three accepted swings.
- `TC-ST-MINE-004`: inactive node cannot be mined and later respawns.
- `TC-ST-MINE-005`: repeated mining inside cooldown is blocked.
- `TC-ST-BUILD-001`: place stone block consumes inventory and persists object.
- `TC-ST-BUILD-002`: occupied placement rolls back inventory.
- `TC-ST-BUILD-003`: placed object appears in hello/snapshot/map_changed.
- `TC-ST-BUILD-004`: pickaxe removes placed block and refunds inventory.
- `TC-ST-WS-001`: unknown WebSocket message returns error.
- `TC-ST-WS-002`: oversized WebSocket message is rejected.
- `TC-ST-WS-003`: equipment changed reloads HQ equipment.

## Practical Guidance

Do not try to close every gap at once. The best near-term payoff is:

1. Add schoolwork workflow tests.
2. Add auth/role tests around endpoint access.
3. Add pet lifecycle tests.
4. Add a small number of Sunny Town protocol tests for cases not already covered by room tests.

Frontend tests are worth adding once there is a real frontend test framework decision. Until then, `npm run build` remains a useful baseline because it runs `vue-tsc --noEmit` and the production Vite build.
