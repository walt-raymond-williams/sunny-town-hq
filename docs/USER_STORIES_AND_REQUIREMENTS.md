# User Stories and Requirements

This document captures current product requirements inferred from the existing architecture, setup, feature, and testing docs. It is intended as a bridge from project documentation to future automated and manual test cases.

Status labels:

- `Implemented`: described as current behavior in the docs.
- `Current expectation`: required by current architecture or testing guidance, even when not directly described as a user-facing feature.
- `Future`: mentioned as a future direction and not part of the current test target.

## Actors

- `Student`: authenticated Keycloak user with the `student` role.
- `Teacher`: authenticated Keycloak user with the `teacher` role.
- `Sunny Town service`: realtime WebSocket service that validates live world actions.
- `HQ service`: durable system of record for users, schoolwork, pet, wallet, inventory, equipment, placed map objects, and saved Sunny Town position.

## Cross-Cutting Requirements

### REQ-AUTH-001: Keycloak Authentication

Status: `Implemented`

The app must use Keycloak login for teacher and student users. The frontend obtains bearer tokens, and HQ validates those tokens before serving authenticated APIs.

Acceptance criteria:

- A user can log in through the configured Keycloak `hq` realm.
- HQ validates the bearer token issuer, audience, signature, and roles.
- HQ creates or updates local `app_user` and `app_user_role` rows on first authenticated request.
- Student-only APIs reject users without the `student` role.
- Teacher-only APIs reject users without the `teacher` role.

Source docs: `README.md`, `docs/PROJECT_SETUP.md`, `docs/KEYCLOAK_SETUP.md`

### REQ-AUTH-002: Local-Network Host Consistency

Status: `Current expectation`

The browser-facing HQ host and Keycloak issuer host must match the hostname or IP address used by the client.

Acceptance criteria:

- A browser session opened through `localhost` uses a Keycloak issuer based on `localhost`.
- A browser session opened through a laptop LAN IP uses a Keycloak issuer based on that LAN IP.
- Phone/tablet users on the same Wi-Fi do not rely on `localhost` to reach the laptop-hosted app.
- Invalid Keycloak redirect URI issues are resolved by adding the exact HQ URL to the Keycloak client.

Source docs: `README.md`, `docs/PROJECT_SETUP.md`, `docs/KEYCLOAK_SETUP.md`

### REQ-SVC-001: Service Boundary

Status: `Implemented`

HQ must own durable state. Sunny Town must own live realtime world state. Sunny Town must not write the HQ database directly.

Acceptance criteria:

- HQ owns account, role, assignment, attempt, pet, wallet, inventory, equipment, map-object, and saved Sunny Town position records.
- Sunny Town owns connected players, room membership, accepted positions, live collectibles, resource node state, portals, and gameplay validation.
- Sunny Town commits durable changes only through HQ internal HTTP endpoints.
- HQ internal Sunny Town endpoints require `X-HQ-Service-Secret`.

Source docs: `README.md`, `docs/SUNNY_TOWN_ARCHITECTURE.md`, `docs/archive/INVENTORY_AND_EQUIPMENT.md`

### REQ-RUNTIME-001: Health Checks

Status: `Current expectation`

HQ and Sunny Town must expose health checks for local integration verification.

Acceptance criteria:

- `GET http://127.0.0.1:18080/healthz` returns `ok` for HQ.
- `GET http://127.0.0.1:18082/healthz` returns `ok` for Sunny Town.
- Docker Compose can rebuild and run HQ and Sunny Town together.

Source docs: `README.md`, `docs/PROJECT_SETUP.md`, `docs/TESTING_GUIDELINES.md`

## Teacher Schoolwork Stories

### STORY-TCHR-001: Create Assignment

Status: `Implemented`

As a teacher, I want to create categorized text-prompt assignments so that students have schoolwork to complete.

Acceptance criteria:

- A teacher can create an assignment with category, prompt, and expected answer.
- Supported categories are `MATH`, `SCIENCE`, and `READING`.
- A non-teacher cannot create assignments.
- Created assignments are persisted in PostgreSQL.

Source docs: `ARCHITECTURE.md`, `README.md`

### STORY-TCHR-002: Review Submitted Work

Status: `Implemented`

As a teacher, I want to view submitted assignments so that I can grade student work.

Acceptance criteria:

- A teacher can view all assignments.
- A teacher can filter or identify assignments that need review.
- Submitted answers include the latest active student attempt.
- A non-teacher cannot access teacher review APIs.

Source docs: `ARCHITECTURE.md`, `README.md`

### STORY-TCHR-003: Grade Work

Status: `Implemented`

As a teacher, I want to mark a submission pass or fail and optionally leave feedback so that the student can see results.

Acceptance criteria:

- A teacher can mark the latest active attempt as passed.
- A teacher can mark the latest active attempt as failed.
- A teacher can save optional feedback.
- A passing grade awards the student one cookie only once for that attempt.
- Re-saving a passing result does not duplicate the cookie reward.

Source docs: `ARCHITECTURE.md`

### STORY-TCHR-004: Reset Assignment Attempt

Status: `Implemented`

As a teacher, I want to reset a submitted assignment while preserving history so that the student can try again.

Acceptance criteria:

- A teacher can mark the latest active attempt reset.
- Resetting can preserve feedback for the historical attempt.
- A reset assignment becomes answerable again by the student.
- Historical attempts remain visible for review.

Source docs: `ARCHITECTURE.md`

### STORY-TCHR-005: Delete Assignment

Status: `Implemented`

As a teacher, I want to delete assignments so that obsolete work can be removed.

Acceptance criteria:

- A teacher can delete an assignment.
- A non-teacher cannot delete assignments.
- Deleted assignments no longer appear in student or teacher assignment lists.

Source docs: `ARCHITECTURE.md`

## Student Schoolwork Stories

### STORY-STUD-001: Answer Next Assignment

Status: `Implemented`

As a student, I want to see one unanswered assignment at a time so that I can focus on the next piece of work.

Acceptance criteria:

- A student can request the next unanswered assignment.
- A student can filter next assignment by `MATH`, `SCIENCE`, or `READING`.
- A student can submit a text answer.
- After submission, the student is advanced to the next unanswered assignment.
- When no unanswered assignments remain, the student sees a completion state.

Source docs: `ARCHITECTURE.md`, `README.md`

### STORY-STUD-002: See Reset Attempt History

Status: `Implemented`

As a student, I want to see previous attempts when work is reset so that I can learn from prior feedback.

Acceptance criteria:

- A reset assignment can be shown again as unanswered.
- Previous attempts for that assignment are available below the answer form.
- Previous feedback is preserved when the teacher provided it.

Source docs: `ARCHITECTURE.md`

### STORY-STUD-003: Review Grades

Status: `Implemented`

As a student, I want to review graded work and subject progress so that I can understand how I am doing.

Acceptance criteria:

- A student can view graded attempts grouped by question.
- Grades include pass/fail status.
- The student can see category pass percentages for Math, Science, and Reading.
- Ungraded active attempts do not count as passed or failed grades.

Source docs: `ARCHITECTURE.md`, `README.md`

## Pet Stories

### STORY-PET-001: View Pet State

Status: `Implemented`

As a student, I want to view my pet's hunger, happiness, energy, sleep state, and cookie count so that I can care for it.

Acceptance criteria:

- The student pet UI shows cookie count and pet stats.
- Pet stats are loaded from the Connect RPC pet service.
- The Student page can receive live pet state updates through `WatchPetState`.
- Pet stats are persisted per student.

Source docs: `ARCHITECTURE.md`, `README.md`

### STORY-PET-002: Feed Pet

Status: `Implemented`

As a student, I want to feed my pet with cookies so that its hunger improves.

Acceptance criteria:

- Feeding consumes one cookie from inventory-backed cookie ownership.
- Feeding increases hunger and caps hunger at 100.
- Feeding cannot create negative cookie quantities.
- The legacy JSON feed endpoint may exist, but the main frontend pet path uses Connect RPC.

Source docs: `ARCHITECTURE.md`, `docs/archive/INVENTORY_AND_EQUIPMENT.md`

### STORY-PET-003: Play With Pet

Status: `Implemented`

As a student, I want to play with my pet so that its happiness improves.

Acceptance criteria:

- Playing increases happiness.
- Playing costs 10 energy.
- Pet state remains within valid stat bounds.

Source docs: `ARCHITECTURE.md`

### STORY-PET-004: Pet Sleep And Recovery

Status: `Implemented`

As a student, I want my pet to sleep and recover energy so that pet care has a simple resource loop.

Acceptance criteria:

- A student can put the pet to sleep.
- Sleeping restores energy linearly toward 100 over about ten minutes.
- The pet wakes automatically when sleep recovery completes.
- Automatic wake grants 10 happiness.
- Waking early stops recovery and subtracts 30 happiness.
- If awake energy reaches 0, the pet falls asleep.

Source docs: `ARCHITECTURE.md`

## Inventory, Wallet, Equipment, and Crafting Stories

### REQ-INV-001: Stars Are Wallet Currency

Status: `Implemented`

Stars must be stored as wallet currency, not inventory items.

Acceptance criteria:

- `student_wallet` stores current star balance.
- `student_star_ledger` records star gains and spends.
- Inventory APIs do not return a `star` item.
- Sunny Town star pickup rewards update wallet state through HQ.

Source docs: `docs/archive/INVENTORY_AND_EQUIPMENT.md`, `docs/SUNNY_TOWN_ARCHITECTURE.md`

### REQ-INV-002: Inventory Item Ownership

Status: `Implemented`

Inventory must track positive student-owned item quantities.

Acceptance criteria:

- `inventory_item_type` defines valid item keys and metadata.
- `student_inventory_item` stores per-student item quantities.
- Quantity cannot be negative.
- Student inventory API returns only items with quantity greater than zero.
- Cookies, rock, crystal, and stone blocks are inventory items.

Source docs: `docs/archive/INVENTORY_AND_EQUIPMENT.md`

### STORY-INV-001: View Inventory

Status: `Implemented`

As a student, I want to view my inventory from the Student Pet page and Sunny Town so that I know what I own.

Acceptance criteria:

- The Student Pet page has an inventory dialog.
- Pressing `E` in Sunny Town toggles the inventory overlay.
- Inventory shows item quantities for non-equippable items.
- Inventory shows equip status for equippable items.
- Student Pet and Sunny Town show consistent cookie quantities.

Source docs: `docs/archive/INVENTORY_AND_EQUIPMENT.md`, `docs/TESTING_GUIDELINES.md`

### STORY-INV-002: Equip Items

Status: `Implemented`

As a student, I want to equip owned gear, accessories, and tools so that my avatar and available actions change.

Acceptance criteria:

- Equipment slots include `gear`, `accessory`, and `tool`.
- A student can equip an owned item into its matching slot.
- A student cannot equip an unowned item.
- A student cannot equip a non-equippable item such as `cookie`.
- A student cannot equip an item into the wrong slot.
- Equipping does not decrement item quantity.
- Unequipping clears the slot and does not remove the item from inventory.

Source docs: `docs/archive/INVENTORY_AND_EQUIPMENT.md`

### STORY-INV-003: Starter Equipment

Status: `Implemented`

As a student, I should receive starter equipment so that I can use Sunny Town features immediately.

Acceptance criteria:

- Existing student accounts receive `sunny_hoodie`, `star_cap`, and `pickaxe` during schema migration or backfill.
- New student accounts receive `sunny_hoodie`, `star_cap`, and `pickaxe` during authenticated user sync.
- Starter items appear in the inventory API with positive quantity.

Source docs: `docs/archive/INVENTORY_AND_EQUIPMENT.md`

### STORY-INV-004: Buy Shop Item

Status: `Implemented`

As a student, I want to buy cookies with wallet stars so that Sunny Town shops can affect durable inventory.

Acceptance criteria:

- The Cookie Keeper shop supports purchasing `cookie`.
- Purchase verifies sufficient wallet stars.
- Purchase subtracts stars from `student_wallet`.
- Purchase records a negative `student_star_ledger` row with source `shop_purchase`.
- Purchase increments cookie inventory.
- Purchase returns updated star balance and inventory.

Source docs: `docs/archive/INVENTORY_AND_EQUIPMENT.md`, `docs/SUNNY_TOWN_ARCHITECTURE.md`

### STORY-CRAFT-001: Craft Stone Block

Status: `Implemented`

As a student, I want to craft stone blocks from rocks so that mined resources can become placeable map objects.

Acceptance criteria:

- Crafting recipes API returns known recipes and whether the student can craft them.
- The `stone_block` recipe consumes 4 `rock` and creates 1 `stone_block`.
- Crafting runs in an HQ database transaction.
- If the student lacks ingredients, no inventory changes.
- Crafting returns updated inventory and recipe availability.

Source docs: `docs/archive/INVENTORY_AND_EQUIPMENT.md`

## Sunny Town Entry and Session Stories

### STORY-ST-001: Start Sunny Town Session

Status: `Implemented`

As a student, I want to enter Sunny Town from the app so that I can play in the realtime world.

Acceptance criteria:

- The frontend calls `POST /api/student/sunny-town/session` using a student bearer token.
- HQ rejects non-student users.
- HQ returns room id, map id, avatar id, WebSocket URL, join token, expiry, and wallet state.
- The join token is short-lived and HMAC-signed by HQ.
- Sunny Town validates the join token before accepting the WebSocket.
- Sunny Town sends `hello` followed by periodic `snapshot` messages.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`

### STORY-ST-002: Restore Last Position

Status: `Implemented`

As a student, I want Sunny Town to reopen where I last left off so that my session feels continuous.

Acceptance criteria:

- Sunny Town saves the player's last accepted map, coordinates, and facing through HQ when the WebSocket leaves a room.
- HQ persists the position in `student_sunny_town_position`.
- A later Sunny Town session loads the saved position when possible.
- If no saved position exists, the player joins `sunny-town-v1`.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`, `docs/TESTING_GUIDELINES.md`

## Sunny Town Movement and Multiplayer Stories

### STORY-ST-MOVE-001: Local Movement Feel

Status: `Implemented`

As a student, I want movement to start and stop immediately so that Sunny Town feels responsive.

Acceptance criteria:

- The frontend moves and renders the local player every animation frame.
- The frontend sends movement samples on input changes and while moving.
- Releasing movement sends a final forced `moving: false` sample.
- Normal network lag does not pull the local player backward during normal play.
- Arrow keys and WASD both move the player.

Source docs: `docs/SUNNY_TOWN_MOVEMENT_MODEL.md`, `docs/TESTING_GUIDELINES.md`

### REQ-ST-MOVE-001: Server-Accepted Movement

Status: `Implemented`

Sunny Town must use server-accepted positions for shared state and gameplay effects.

Acceptance criteria:

- The server ignores out-of-order movement samples by sequence number.
- The server rejects invalid numeric coordinates.
- The server clamps accepted positions to map bounds.
- The server rejects movement into static blocked rectangles.
- The server rejects movement into placed map objects.
- Stars, mining, portals, and future gameplay effects use accepted server position, not raw client coordinates.
- Snapshots include `lastProcessedSeq`.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`, `docs/SUNNY_TOWN_MOVEMENT_MODEL.md`

### STORY-ST-MULTI-001: See Nearby Players

Status: `Implemented`

As a student, I want to see other students in the same Sunny Town map so that the world feels shared.

Acceptance criteria:

- Players in the same map appear in each other's snapshots.
- Remote players render from interpolated snapshots.
- Players in different maps do not appear in each other's snapshots.
- Player snapshots include display name, position, facing, movement state, avatar id, equipment visuals, and last processed movement sequence.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`, `docs/SUNNY_TOWN_MOVEMENT_MODEL.md`, `docs/TESTING_GUIDELINES.md`

## Sunny Town Map and Portal Stories

### REQ-ST-MAP-001: Load Map JSON

Status: `Implemented`

Sunny Town must load checked-in map JSON files and validate map relationships at startup.

Acceptance criteria:

- Maps are loaded from `SUNNY_TOWN_MAPS_DIR`.
- Current maps include `sunny-town-v1`, `sunny-town-house-1`, `sunny-town-classroom`, and `forest-crossing-v1`.
- Duplicate map IDs are rejected.
- Portals targeting missing maps are rejected.
- Resource node definitions are validated, including duplicate id rejection.
- The frontend receives map data from Sunny Town instead of relying on hardcoded map geometry.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`, `docs/TESTING_GUIDELINES.md`

### STORY-ST-PORTAL-001: Transition Between Maps

Status: `Implemented`

As a student, I want to walk through map portals so that I can move between Sunny Town areas.

Acceptance criteria:

- Walking into the outdoor southern building door transitions to `sunny-town-house-1`.
- Walking into the indoor south door transitions back to `sunny-town-v1`.
- Walking through the eastern path transitions to `forest-crossing-v1`.
- Portal overlap is detected from the server-accepted player rectangle.
- Portal transfer sends a `map_changed` message with the target map, players, collectibles, resource nodes, and placed objects.
- Portal targets do not intentionally place the player inside the destination portal trigger.
- The portal re-entry guard requires the player to leave a portal before another transfer.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`, `docs/TESTING_GUIDELINES.md`

## Sunny Town Collectible and Reward Stories

### STORY-ST-STAR-001: Collect Stars

Status: `Implemented`

As a student, I want to collect stars in Sunny Town so that I can earn wallet currency.

Acceptance criteria:

- Maps with `starSpawns` create in-memory star collectibles.
- A star is collected only when the accepted player position is within pickup radius.
- Collected stars are marked inactive and respawn later.
- Sunny Town sends an idempotent reward event to HQ.
- HQ records the event in `student_star_ledger`.
- HQ updates `student_wallet`.
- Sunny Town sends `reward_committed` or `reward_failed`.
- Maps without `starSpawns` do not enqueue star reward events.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`, `docs/TESTING_GUIDELINES.md`

## Sunny Town NPC, Shop, and Schoolwork Stories

### STORY-ST-NPC-001: Interact With NPCs

Status: `Implemented`

As a student, I want to interact with NPCs so that Sunny Town can connect to dialogue, shops, and schoolwork.

Acceptance criteria:

- Maps can define NPCs.
- NPCs can be dialogue-only, shop-backed, or activity-backed.
- Pressing `F` prioritizes nearby NPC or activity interaction before tool use.
- Schoolwork NPCs open the existing student assignment flow inside Sunny Town.
- Shop NPCs use the HQ shop API.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`, `docs/archive/INVENTORY_AND_EQUIPMENT.md`

## Sunny Town Mining and Resource Stories

### STORY-ST-MINE-001: Mine Resource Nodes

Status: `Implemented`

As a student, I want to mine resource nodes with a pickaxe so that I can earn resources.

Acceptance criteria:

- Resource nodes are loaded from map JSON.
- Mining requires the player to have the requested tool equipped.
- Mining requires the requested tool to be `pickaxe`.
- Mining requires the player to be close enough to an active resource node.
- Mining respects tool cooldown.
- Mining rejects missing tools, wrong tools, inactive nodes, and out-of-range players.
- A resource node depletes after three accepted pickaxe hits.
- Depleted nodes schedule respawn.
- Resource node depletion is in Sunny Town memory and resets if Sunny Town restarts.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`, `docs/archive/INVENTORY_AND_EQUIPMENT.md`, `docs/TESTING_GUIDELINES.md`

### REQ-ST-MINE-001: Idempotent Resource Commit

Status: `Implemented`

Mining rewards must be committed through HQ idempotently.

Acceptance criteria:

- Sunny Town rolls the current resource drop table after a node is depleted.
- Current drop table is 85 percent 1 `rock`, 10 percent 2 `rock`, and 5 percent 1 `crystal`.
- Sunny Town sends resource events to `POST /api/internal/sunny-town/resource-events`.
- HQ records events in `student_inventory_ledger` with unique `event_id` protection.
- HQ increments `student_inventory_item` only when the event id is new.
- Retrying the same event id does not double-award resources.
- Sunny Town sends `resource_committed` or `resource_failed`.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`, `docs/archive/INVENTORY_AND_EQUIPMENT.md`

## Sunny Town Placed Object Stories

### STORY-ST-BUILD-001: Place Stone Block

Status: `Implemented`

As a student, I want to place crafted stone blocks on the map so that I can edit the Sunny Town environment.

Acceptance criteria:

- The Sunny Town inventory overlay exposes a placement action when the player owns `stone_block`.
- Placement snaps to a grid tile.
- Sunny Town validates map bounds, static blocked rectangles, portals, NPCs, resource nodes, players, and existing placed objects.
- Sunny Town asks HQ to persist placement and consume one `stone_block`.
- HQ stores placed objects in `sunny_town_map_object` with uniqueness by room, map, grid x, and grid y.
- HQ rolls back inventory consumption if placement fails.
- Placed objects appear to other players in the same map.
- Placed objects block movement.
- Placed objects remain after reconnect and Sunny Town restart.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`, `docs/archive/INVENTORY_AND_EQUIPMENT.md`, `docs/TESTING_GUIDELINES.md`

### STORY-ST-BUILD-002: Remove Stone Block

Status: `Implemented`

As a student, I want to mine a placed stone block so that I can recover it.

Acceptance criteria:

- Pickaxe use checks nearby placed `stone_block` objects before resource nodes.
- Sunny Town validates the removal request.
- Sunny Town asks HQ to remove the map object and refund one `stone_block`.
- HQ deletes the persisted object and refunds inventory in one transaction.
- Sunny Town broadcasts `map_object_removed`.
- Other players in the same map see the block removed without waiting for a later reconnect.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`, `docs/archive/INVENTORY_AND_EQUIPMENT.md`

## WebSocket Protocol Requirements

### REQ-WS-001: Supported Client Messages

Status: `Implemented`

Sunny Town must support the current client-to-server WebSocket message set.

Acceptance criteria:

- Supported client messages are `move`, `equipment_changed`, `tool_use`, `place_object`, and `ping`.
- Unknown messages receive a small `error` response.
- WebSocket read size is capped.
- Move messages are rate-limited by a simple per-connection input rate window while moving.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`

### REQ-WS-002: Supported Server Messages

Status: `Implemented`

Sunny Town must emit the current server-to-client WebSocket message set.

Acceptance criteria:

- Supported server messages include `hello`, `snapshot`, `map_changed`, `reward_committed`, `reward_failed`, `resource_committed`, `resource_failed`, `map_object_placed`, `map_object_removed`, and `error`.
- `hello`, `snapshot`, and `map_changed` include map-relevant placed objects.
- Inventory quantities from map-object mutations are sent only to the acting player.

Source docs: `docs/SUNNY_TOWN_ARCHITECTURE.md`

## Frontend Requirements

### REQ-FE-001: App Shape

Status: `Current expectation`

The frontend must remain a usable app interface rather than a marketing or landing page.

Acceptance criteria:

- Student, teacher, pet, inventory, and Sunny Town workflows are directly usable from the app shell.
- Controls are compact and operational.
- Frontend changes build successfully with `npm run build` from `frontend/`.

Source docs: `AGENTS.md`, `README.md`, `docs/TESTING_GUIDELINES.md`

### REQ-FE-002: Student Inventory Store

Status: `Implemented`

The frontend inventory store must coordinate inventory, equipment, and crafting state.

Acceptance criteria:

- The store loads `/api/student/inventory`.
- The store loads `/api/student/equipment`.
- The store loads `/api/student/crafting/recipes`.
- The store exposes equipment visuals for Sunny Town rendering.
- After equipment changes, the store reloads inventory so `equipped` flags stay correct.
- Crafting updates inventory and recipe availability.

Source docs: `docs/archive/INVENTORY_AND_EQUIPMENT.md`

### REQ-FE-003: Equipment Change Broadcast

Status: `Implemented`

Sunny Town equipment changes must be committed through HQ before being broadcast to other players.

Acceptance criteria:

- The frontend calls the student equipment API to equip or unequip.
- After the HQ update succeeds, the frontend sends `{ "type": "equipment_changed" }` over the Sunny Town WebSocket.
- Sunny Town reloads equipment from HQ.
- Future snapshots include updated equipment visuals.
- Other players can see the updated avatar equipment.

Source docs: `docs/archive/INVENTORY_AND_EQUIPMENT.md`, `docs/SUNNY_TOWN_ARCHITECTURE.md`

## Verification Requirements

### REQ-TEST-001: Backend Verification

Status: `Current expectation`

Backend changes should be verified with Go tests appropriate to the changed scope.

Acceptance criteria:

- General backend changes pass `go test ./...`.
- Sunny Town-only backend changes pass `go test ./cmd/sunny-town`.
- HQ-only backend changes pass `go test ./cmd/hq`.

Source docs: `AGENTS.md`, `docs/TESTING_GUIDELINES.md`

### REQ-TEST-002: Frontend Verification

Status: `Current expectation`

Frontend changes should be verified with deterministic build checks.

Acceptance criteria:

- Frontend changes pass `npm run build` from `frontend/`.
- Type checks should be run with `npm run typecheck` when practical or when the change affects TypeScript contracts.
- Generated `web/` assets are kept out of commits.

Source docs: `AGENTS.md`, `docs/TESTING_GUIDELINES.md`

### REQ-TEST-003: Integration Verification

Status: `Current expectation`

Integration/runtime changes should be verified through Docker Compose when practical.

Acceptance criteria:

- `docker compose -f deploy\docker-compose.yml up -d --build --force-recreate hq sunny-town` rebuilds and runs HQ and Sunny Town.
- HQ health returns `ok`.
- Sunny Town health returns `ok`.
- Expected containers include healthy `hq-server` and `hq-sunny-town`.

Source docs: `AGENTS.md`, `docs/TESTING_GUIDELINES.md`

## Future Requirements Not Yet In Current Scope

These items are documented as future ideas or scaling paths and should not drive current acceptance tests unless the project intentionally promotes them.

- Multiple students as a richer product feature beyond current local setup.
- Assignment due dates and late status.
- Draft answers.
- File attachments.
- Multiple-choice questions.
- Parent dashboard.
- Printable assignments.
- HTTPS with a local reverse proxy.
- Nginx or other production-style frontend proxy.
- PostgREST as a limited learning/admin sidecar.
- Multiple Sunny Town processes, sticky routing, Redis/NATS event fan-out, or durable live node depletion.

Source docs: `ARCHITECTURE.md`, `docs/SUNNY_TOWN_ARCHITECTURE.md`
