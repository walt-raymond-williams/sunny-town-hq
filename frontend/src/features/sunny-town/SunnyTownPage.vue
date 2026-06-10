<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { normalizeContainerInventorySlots, normalizeStudentInventorySlots } from '../../api/inventoryApi'
import {
  hotbarIndexForEvent,
  useSunnyTownInventoryActions,
} from '../../composables/useSunnyTownInventoryActions'
import {
  nearestSunnyTownChest,
  useSunnyTownChestInteractions,
} from '../../composables/useSunnyTownChestInteractions'
import { useSunnyTownMessageEffects } from '../../composables/useSunnyTownMessageEffects'
import {
  nearestSunnyTownNpc,
  useSunnyTownNpcInteractions,
} from '../../composables/useSunnyTownNpcInteractions'
import {
  movementDirectionForEvent,
  movementInputEventOptions,
  moveSendIntervalMs,
  useSunnyTownMovement,
} from '../../composables/useSunnyTownMovement'
import { useSunnyTownLocalPlayer } from '../../composables/useSunnyTownLocalPlayer'
import { useSunnyTownPlacement } from '../../composables/useSunnyTownPlacement'
import { useSunnyTownRemotePlayers } from '../../composables/useSunnyTownRemotePlayers'
import { useSunnyTownRenderer } from '../../composables/useSunnyTownRenderer'
import { useSunnyTownSocket } from '../../composables/useSunnyTownSocket'
import { useSunnyTownToolUseAnimation } from '../../composables/useSunnyTownToolUseAnimation'
import { useSunnyTownWorldState } from '../../composables/useSunnyTownWorldState'
import { useRouteAccess } from '../../composables/useRouteAccess'
import { useStudentInventoryStore } from '../../stores/studentInventory'
import type { InventoryStorageRef } from '../../types/inventory'
import type {
  SunnyTownContainerOpenMessage,
  SunnyTownContainerTransferMessage,
  SunnyTownEquipmentChangedMessage,
  SunnyTownMap,
  SunnyTownMoveMessage,
  SunnyTownNpc,
  SunnyTownPlaceObjectMessage,
  SunnyTownPlayer,
  SunnyTownServerMessage,
  SunnyTownToolUseMessage,
  SunnyTownWorldObject,
} from '../../types/sunnyTown'
import SunnyTownCanvas from './SunnyTownCanvas.vue'
import SunnyTownChestPanel from './SunnyTownChestPanel.vue'
import SunnyTownDialogue from './SunnyTownDialogue.vue'
import SunnyTownHud from './SunnyTownHud.vue'
import SunnyTownInventoryPanel from './SunnyTownInventoryPanel.vue'
import SunnyTownSchoolworkPanel from './SunnyTownSchoolworkPanel.vue'
import SunnyTownShop from './SunnyTownShop.vue'
import {
  drawCollectible,
  drawNpc,
  drawPlayer,
} from './rendering/characterDrawing'
import { drawMap } from './rendering/mapDrawing'
import {
  drawPlacementPreview,
  drawWorldObjects,
} from './rendering/objectDrawing'

const npcInteractionRadius = 54
const toolUseDurationMs = 360

const router = useRouter()
const { ensureStudentAccess } = useRouteAccess()
const inventoryStore = useStudentInventoryStore()
const movement = useSunnyTownMovement()
const localPlayerState = useSunnyTownLocalPlayer()
const placement = useSunnyTownPlacement()
const remotePlayerState = useSunnyTownRemotePlayers()
const toolUseAnimation = useSunnyTownToolUseAnimation()
const worldState = useSunnyTownWorldState()
const chestInteractions = useSunnyTownChestInteractions({
  sendOpen: sendChestOpen,
})
const selfId = ref('')
const starBalance = ref(0)
const gameToast = ref('')
const inventoryActions = useSunnyTownInventoryActions(inventoryStore, {
  onEquipmentChanged() {
    applyLocalEquipmentVisuals()
    notifyEquipmentChanged()
  },
  onHotbarSelectionChanged() {
    placement.clearHover()
    draw()
  },
  onHotbarUpdated() {
    draw()
  },
})
const {
  activeDialogueLine,
  activeDialogueNpc,
  activeSchoolworkNpc,
  activeShopNpc,
  closeDialogue,
  closeOverlays: closeNpcOverlays,
  dialogueProgress,
  hasActiveOverlay,
  interactWith: interactWithNpc,
  isLoadingSchoolwork,
  isPurchasing,
  isSubmittingSchoolwork,
  nearbyNpc,
  buyShopItem,
  openTrade,
  refreshNearby: refreshNearbyNpc,
  schoolworkAnswer,
  schoolworkAssignment,
  schoolworkError,
  schoolworkNotice,
  schoolworkOpen,
  shopError,
  shopNotice,
  shopOpen,
  shopStock,
  shopStockCapacity,
  startSchoolwork,
  submitSchoolworkAnswer,
} = useSunnyTownNpcInteractions({
  loadInventory: () => inventoryStore.loadInventory(),
  setInventoryItems: (items) => inventoryStore.setItems(items),
  starBalance,
})
const {
  activeChest,
  applyContainerError,
  applyContainerOpened,
  applyContainerTransferCommitted,
  canDepositIntoActiveChest,
  canWithdrawFromActiveChest,
  chestError,
  closeChest,
  containerSlots,
  hasActiveOverlay: hasActiveChestOverlay,
  inspectChest,
  isLoadingChest,
  isTransferringChestSlot,
  nearbyChest,
  refreshNearby: refreshNearbyChest,
  startContainerTransfer,
} = chestInteractions
const canvas = ref<HTMLCanvasElement | null>(null)
const {
  activeMap,
  applyPlacedObject,
  applyRemovedObject,
  applySnapshot,
  collectibles,
  npcs,
  players,
  worldObjects,
} = worldState
const placementHoverGrid = placement.hoverGrid
const {
  clearSelectedHotbarSlot,
  craftInventoryRecipe,
  craftingPanelOpen,
  equipInventorySlotDrop,
  inventoryOpen,
  placingStoneBlock,
  selectedHotbarIndex,
  selectedHotbarItemKey,
  selectHotbarSlot,
  showAllCraftingRecipes,
  stoneBlockQuantity,
  toggleCraftingPanel,
  toggleInventory,
  unequipInventorySlot,
} = inventoryActions
let moveSeq = 0
let lastMoveSendAtMs = 0
let lastSentMoveJson = ''

const {
  connected,
  error,
  isOpen: isSunnyTownSocketOpen,
  send: sendSunnyTownMessage,
  start: startSunnyTownSocket,
  status,
  stop: stopSunnyTownSocket,
} = useSunnyTownSocket({
  async onSession(nextSession) {
    starBalance.value = nextSession.wallet.starBalance
    inventoryStore.setSessionInventory(nextSession.inventory, nextSession.hotbar)
    await nextTick()
  },
  onMessage: handleServerMessage,
})
const messageEffects = useSunnyTownMessageEffects({
  craftingPanelOpen,
  error,
  gameToast,
  inventoryStore,
  starBalance,
})

const renderer = useSunnyTownRenderer(canvas, {
  drawScene,
  onFrame(deltaSeconds) {
    predictSelf(deltaSeconds)
    refreshNpcInteractionState()
  },
})
const draw = renderer.draw

const playerCount = computed(() => players.value.length)

onMounted(async () => {
  const canAccess = await ensureStudentAccess()
  if (!canAccess) {
    return
  }

  window.addEventListener('keydown', handleKeyDown, movementInputEventOptions)
  window.addEventListener('keyup', handleKeyUp, movementInputEventOptions)
  window.addEventListener('blur', handleInputCancel)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  window.addEventListener('resize', handleResize)
  renderer.start()
  await startSunnyTownSocket()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeyDown, movementInputEventOptions)
  window.removeEventListener('keyup', handleKeyUp, movementInputEventOptions)
  window.removeEventListener('blur', handleInputCancel)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  window.removeEventListener('resize', handleResize)
  renderer.stop()
  stopSunnyTownSocket()
  localPlayerState.clear()
  remotePlayerState.clear()
})

function handleServerMessage(message: SunnyTownServerMessage) {
  if (message.type === 'hello') {
    selfId.value = message.selfId || ''
    applyMapState(message)
    return
  }
  if (message.type === 'snapshot') {
    if (!applySnapshot(message)) {
      return
    }
    remotePlayerState.recordSnapshots(players.value, selfId.value, message.serverTimeMs || Date.now())
    syncLocalSelfFromSnapshot()
    return
  }
  if (message.type === 'map_changed') {
    applyMapState(message)
    return
  }
  if (message.type === 'reward_committed') {
    messageEffects.applyRewardCommitted(message)
    return
  }
  if (message.type === 'reward_failed') {
    messageEffects.applyRewardFailed()
    return
  }
  if (message.type === 'resource_committed') {
    messageEffects.applyResourceCommitted(message)
    return
  }
  if (message.type === 'map_object_placed') {
    applyPlacedObject(message)
    messageEffects.applyPlacedObjectCommitted(message)
    draw()
    return
  }
  if (message.type === 'map_object_removed') {
    applyRemovedObject(message)
    messageEffects.applyRemovedObjectCommitted(message)
    draw()
    return
  }
  if (message.type === 'container_opened') {
    applyContainerOpened(message.container ? normalizeContainerInventorySlots(message.container) : null)
    return
  }
  if (message.type === 'container_transfer_committed') {
    if (message.inventory) {
      inventoryStore.setInventorySlots(normalizeStudentInventorySlots(message.inventory))
    }
    applyContainerTransferCommitted(message.container ? normalizeContainerInventorySlots(message.container) : null)
    return
  }
  if (message.type === 'resource_failed') {
    messageEffects.applyResourceFailed()
    return
  }
  if (message.type === 'error') {
    if (activeChest.value && message.code?.startsWith('container_')) {
      applyContainerError(chestErrorMessage(message.code))
      return
    }
    error.value = message.code || 'Sunny Town received an invalid message'
  }
}

function handleKeyDown(event: KeyboardEvent) {
  if (event.code === 'Escape') {
    if (hasActiveOverlay() || hasActiveChestOverlay() || inventoryOpen.value) {
      event.preventDefault()
      closeNpcOverlays()
      closeChest()
      inventoryOpen.value = false
    }
    return
  }

  if (isEditableKeyboardTarget(event.target)) {
    return
  }

  const hotbarIndex = hotbarIndexForEvent(event)
  if (hotbarIndex !== null) {
    event.preventDefault()
    selectHotbarSlot(hotbarIndex)
    return
  }

  if (event.code === 'KeyE' || event.key.toLowerCase() === 'e') {
    event.preventDefault()
    toggleInventory()
    return
  }
  if (event.code === 'KeyF' || event.key.toLowerCase() === 'f') {
    event.preventDefault()
    handlePrimaryInteraction()
    return
  }

  const direction = movementDirectionForEvent(event)
  if (!direction) {
    return
  }
  event.preventDefault()
  movement.press(direction)
  refreshLocalMovementState()
  sendMove(true)
}

function handleKeyUp(event: KeyboardEvent) {
  const direction = movementDirectionForEvent(event)
  if (!direction) {
    return
  }
  event.preventDefault()
  movement.release(direction)
  refreshLocalMovementState()
  sendMove(true)
}

function handleVisibilityChange() {
  if (document.visibilityState === 'hidden') {
    handleInputCancel()
  }
}

function handleInputCancel() {
  if (!movement.hasInput()) {
    return
  }
  movement.clear()
  refreshLocalMovementState()
  sendMove(true)
}

function handleResize() {
  draw()
}

function setCanvas(element: HTMLCanvasElement) {
  canvas.value = element
  draw()
}

function handleCanvasPointerDown(event: PointerEvent) {
  if (event.button !== 0) {
    return
  }
  if (placingStoneBlock.value) {
    event.preventDefault()
    placeStoneBlockAtPointer(event)
    return
  }
  if (hasActiveOverlay() || hasActiveChestOverlay() || inventoryOpen.value) {
    return
  }
  event.preventDefault()
  useEquippedTool()
}

function handleCanvasPointerMove(event: PointerEvent) {
  if (!placingStoneBlock.value || inventoryOpen.value) {
    return
  }
  placement.setHoverFromPointer(event, activeMap.value, canvas.value, cameraForCanvas())
  draw()
}

function handleCanvasPointerLeave() {
  placement.clearHover()
  draw()
}

function placeStoneBlockAtPointer(event: PointerEvent) {
  const grid = placement.gridFromPointer(event, activeMap.value, canvas.value, cameraForCanvas())
  if (!grid || stoneBlockQuantity.value < 1 || !isSunnyTownSocketOpen()) {
    return
  }
  const message: SunnyTownPlaceObjectMessage = {
    type: 'place_object',
    itemKey: 'stone_block',
    gridX: grid.gridX,
    gridY: grid.gridY,
    clientTimeMs: Date.now(),
  }
  sendSunnyTownMessage(JSON.stringify(message))
}

function notifyEquipmentChanged() {
  if (!isSunnyTownSocketOpen()) {
    return
  }
  const message: SunnyTownEquipmentChangedMessage = { type: 'equipment_changed' }
  sendSunnyTownMessage(JSON.stringify(message))
}

function applyLocalEquipmentVisuals() {
  const equipment = { ...inventoryStore.equippedVisuals }
  localPlayerState.applyEquipment(equipment)
  players.value = players.value.map((player) => (
    player.id === selfId.value ? { ...player, equipment } : player
  ))
  draw()
}

function cameraForCanvas(): { x: number; y: number } {
  const target = canvas.value
  if (!target) {
    return { x: 0, y: 0 }
  }
  const rect = target.getBoundingClientRect()
  return currentCamera(rect.width, rect.height)
}

function currentCamera(viewWidth: number, viewHeight: number): { x: number; y: number } {
  const map = activeMap.value
  if (!map) {
    return { x: 0, y: 0 }
  }
  const renderedPlayers = renderedSunnyTownPlayers()
  const self = renderedPlayers.find((player) => player.id === selfId.value) || renderedPlayers[0]
  const worldWidth = map.width * map.tileSize
  const worldHeight = map.height * map.tileSize
  return {
    x: clamp((self?.x || worldWidth / 2) - viewWidth / 2, 0, Math.max(0, worldWidth - viewWidth)),
    y: clamp((self?.y || worldHeight / 2) - viewHeight / 2, 0, Math.max(0, worldHeight - viewHeight)),
  }
}

function applyMapState(message: SunnyTownServerMessage) {
  worldState.applyMapState(message)
  remotePlayerState.clear()
  localPlayerState.clear()
  lastSentMoveJson = ''
  toolUseAnimation.clear()
  placement.clearHover()
  movement.clear()
  closeNpcOverlays()
  closeChest()
  nearbyNpc.value = null
  nearbyChest.value = null
  syncLocalSelfFromSnapshot()
  draw()
}

async function handlePrimaryInteraction() {
  if (interactWithNearbyNpc()) {
    return
  }
  if (inspectChest(nearestChestToSelf())) {
    handleInputCancel()
    return
  }
  useEquippedTool()
}

function sendChestOpen(chest: SunnyTownWorldObject): boolean {
  if (!isSunnyTownSocketOpen()) {
    return false
  }
  const message: SunnyTownContainerOpenMessage = {
    type: 'container_open',
    objectSource: chest.source,
    objectId: chest.id,
    clientTimeMs: Date.now(),
  }
  return sendSunnyTownMessage(JSON.stringify(message))
}

function transferChestStack(source: InventoryStorageRef, destination: InventoryStorageRef) {
  if (!activeChest.value || !isSunnyTownSocketOpen()) {
    applyContainerError('Sunny Town connection is not ready.')
    return
  }
  startContainerTransfer()
  const message: SunnyTownContainerTransferMessage = {
    type: 'container_transfer',
    objectSource: activeChest.value.source,
    objectId: activeChest.value.id,
    source,
    destination,
    clientTimeMs: Date.now(),
  }
  if (!sendSunnyTownMessage(JSON.stringify(message))) {
    applyContainerError('Sunny Town connection is not ready.')
  }
}

function interactWithNearbyNpc(): boolean {
  if (!activeMap.value) {
    return false
  }
  const result = interactWithNpc(nearestNpcToSelf())
  if (result.openedSchoolwork) {
    handleInputCancel()
  }
  return result.handled
}

function useEquippedTool() {
  const player = localPlayerState.current(players.value, selfId.value)
  const toolKey = selectedHotbarItemKey.value
  if (!player || !toolKey) {
    return
  }
  if (toolKey !== 'pickaxe') {
    return
  }

  toolUseAnimation.start(toolKey, player.facing, performance.now(), toolUseDurationMs)

  if (isSunnyTownSocketOpen()) {
    const message: SunnyTownToolUseMessage = {
      type: 'tool_use',
      toolKey,
      x: Math.round(player.x * 10) / 10,
      y: Math.round(player.y * 10) / 10,
      facing: player.facing,
      clientTimeMs: Date.now(),
    }
    sendSunnyTownMessage(JSON.stringify(message))
  }
  draw()
}

function refreshLocalMovementState() {
  localPlayerState.refreshMovement(movement.currentInput(), activeMap.value, worldObjects.value)
}

function sendMove(force = false) {
  const localSelf = localPlayerState.local()
  if (!isSunnyTownSocketOpen() || !localSelf) {
    return
  }

  const now = performance.now()
  const moveState = {
    x: Math.round(localSelf.x * 10) / 10,
    y: Math.round(localSelf.y * 10) / 10,
    facing: localSelf.facing,
    moving: localSelf.moving,
  }
  const moveStateJson = JSON.stringify(moveState)
  if (!force && now - lastMoveSendAtMs < moveSendIntervalMs) {
    return
  }
  if (!force && !localSelf.moving && moveStateJson === lastSentMoveJson) {
    return
  }
  const message: SunnyTownMoveMessage = {
    type: 'move',
    seq: ++moveSeq,
    x: moveState.x,
    y: moveState.y,
    facing: moveState.facing,
    moving: moveState.moving,
    clientTimeMs: Date.now(),
  }
  lastMoveSendAtMs = now
  lastSentMoveJson = moveStateJson
  sendSunnyTownMessage(JSON.stringify(message))
}

function drawScene(context: CanvasRenderingContext2D, width: number, height: number) {
  const map = activeMap.value
  if (!map) {
    context.fillStyle = '#8fcf85'
    context.fillRect(0, 0, width, height)
    return
  }

  const renderedPlayers = renderedSunnyTownPlayers()
  const camera = currentCamera(width, height)
  const cameraX = camera.x
  const cameraY = camera.y

  drawMap(context, map, cameraX, cameraY, width, height)
  drawWorldObjects(context, worldObjects.value, cameraX, cameraY)
  drawPlacementPreview(
    context,
    map,
    cameraX,
    cameraY,
    placingStoneBlock.value ? placementHoverGrid.value : null,
    placementHoverGrid.value ? canPlaceStoneBlock(map, placementHoverGrid.value.gridX, placementHoverGrid.value.gridY) : false,
  )
  for (const collectible of collectibles.value) {
    if (collectible.active) {
      drawCollectible(context, collectible, cameraX, cameraY)
    }
  }
  for (const npc of renderedSunnyTownNpcs()) {
    drawNpc(context, npc, cameraX, cameraY, nearbyNpc.value?.id || '')
  }
  for (const player of renderedPlayers) {
    drawPlayer(context, player, cameraX, cameraY, {
      selfId: selfId.value,
      selectedToolKey: selectedHotbarItemKey.value,
      toolUseProgress: currentToolUseProgress,
    })
  }
}

function refreshNpcInteractionState() {
  refreshNearbyNpc(nearestNpcToSelf())
  refreshNearbyChest(nearestChestToSelf())
}

function nearestNpcToSelf(): SunnyTownNpc | null {
  const map = activeMap.value
  const self = localPlayerState.current(players.value, selfId.value)
  if (!map) {
    return null
  }
  return nearestSunnyTownNpc(renderedSunnyTownNpcs(), self, npcInteractionRadius)
}

function nearestChestToSelf(): SunnyTownWorldObject | null {
  const self = localPlayerState.current(players.value, selfId.value)
  return nearestSunnyTownChest(worldObjects.value, self)
}

function renderedSunnyTownNpcs(): SunnyTownNpc[] {
  const map = activeMap.value
  if (!map) {
    return []
  }
  return npcs.value.length > 0 ? npcs.value : map.npcs
}

function renderedSunnyTownPlayers(): SunnyTownPlayer[] {
  const selfSnapshot = players.value.find((player) => player.id === selfId.value)
  const remotePlayers = players.value.filter((player) => player.id !== selfId.value)
  const renderedRemotePlayers = remotePlayerState.smoothPlayers(remotePlayers)
  return localPlayerState.renderedPlayers(selfSnapshot, renderedRemotePlayers)
}

function predictSelf(deltaSeconds: number) {
  if (!selfId.value || !connected.value || !activeMap.value) {
    return
  }

  const selfSnapshot = players.value.find((player) => player.id === selfId.value)
  if (!selfSnapshot) {
    return
  }
  const input = movement.currentInput()
  localPlayerState.predict(selfSnapshot, input, activeMap.value, worldObjects.value, deltaSeconds)
  sendMove(false)
}

function syncLocalSelfFromSnapshot() {
  if (!selfId.value) {
    return
  }

  const selfSnapshot = players.value.find((player) => player.id === selfId.value)
  if (!selfSnapshot) {
    localPlayerState.clear()
    return
  }
  localPlayerState.syncFromSnapshot(selfSnapshot)
}

function canPlaceStoneBlock(map: SunnyTownMap, gridX: number, gridY: number): boolean {
  const self = localPlayerState.current(players.value, selfId.value)
  return placement.canPlaceStoneBlock(map, gridX, gridY, stoneBlockQuantity.value, worldObjects.value, self)
}

function currentToolUseProgress(toolKey: string): number | null {
  return toolUseAnimation.progress(toolKey, performance.now())
}

function isEditableKeyboardTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false
  }

  const tagName = target.tagName.toLowerCase()
  return tagName === 'input' || tagName === 'textarea' || tagName === 'select' || target.isContentEditable
}

function chestErrorMessage(code: string): string {
  const messages: Record<string, string> = {
    container_access_denied: 'Move closer to the chest to use this storage.',
    container_open_failed: 'Storage could not be opened.',
    container_transfer_failed: 'That item could not be moved.',
    invalid_container_transfer: 'That storage move is not allowed.',
  }
  return messages[code] || 'Storage action failed.'
}

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value))
}

function backToPet() {
  router.push('/student')
}
</script>

<template>
  <SunnyTownHud
    :connected="connected"
    :error="error"
    :game-toast="gameToast"
    :hotbar-slots="inventoryStore.hotbarSlots"
    :player-count="playerCount"
    :selected-hotbar-index="selectedHotbarIndex"
    :star-balance="starBalance"
    :status="status"
    @back="backToPet"
    @select-hotbar-slot="selectHotbarSlot"
  >
    <template #canvas>
      <SunnyTownCanvas
        @pointer-down="handleCanvasPointerDown"
        @pointer-leave="handleCanvasPointerLeave"
        @pointer-move="handleCanvasPointerMove"
        @ready="setCanvas"
      />
    </template>
      <div v-if="nearbyNpc && !activeDialogueNpc && !activeSchoolworkNpc && !activeChest && !inventoryOpen" class="sunny-town-talk-hint">
        <v-icon icon="mdi-chat" size="small" />
        <span>F {{ nearbyNpc.name }}</span>
      </div>
      <div v-else-if="nearbyChest && !activeChest && !activeDialogueNpc && !activeSchoolworkNpc && !activeShopNpc && !inventoryOpen" class="sunny-town-talk-hint">
        <v-icon icon="mdi-treasure-chest" size="small" />
        <span>F {{ nearbyChest.name || 'Storage Chest' }}</span>
      </div>
      <SunnyTownDialogue
        v-if="activeDialogueNpc"
        :line="activeDialogueLine"
        :npc="activeDialogueNpc"
        :progress="dialogueProgress"
        @close="closeDialogue"
      />
      <SunnyTownShop
        v-if="activeShopNpc"
        :is-purchasing="isPurchasing"
        :npc="activeShopNpc"
        :open="shopOpen"
        :shop-error="shopError"
        :shop-notice="shopNotice"
        :shop-stock="shopStock"
        :shop-stock-capacity="shopStockCapacity"
        :star-balance="starBalance"
        @buy="buyShopItem"
        @close="closeNpcOverlays"
        @open-trade="openTrade"
      />
      <SunnyTownSchoolworkPanel
        v-if="activeSchoolworkNpc"
        :answer="schoolworkAnswer"
        :assignment="schoolworkAssignment"
        :is-loading="isLoadingSchoolwork"
        :is-submitting="isSubmittingSchoolwork"
        :notice="schoolworkNotice"
        :npc="activeSchoolworkNpc"
        :open="schoolworkOpen"
        :schoolwork-error="schoolworkError"
        @close="closeNpcOverlays"
        @start="startSchoolwork"
        @submit="submitSchoolworkAnswer"
        @update-answer="schoolworkAnswer = $event"
      />
      <SunnyTownChestPanel
        v-if="activeChest"
        :can-deposit="canDepositIntoActiveChest"
        :can-withdraw="canWithdrawFromActiveChest"
        :chest="activeChest"
        :container="containerSlots"
        :error="chestError"
        :is-loading="isLoadingChest"
        :is-transferring="isTransferringChestSlot"
        @close="closeChest"
        @transfer="transferChestStack"
      />
      <SunnyTownInventoryPanel
        v-if="inventoryOpen"
        :crafting-panel-open="craftingPanelOpen"
        :selected-hotbar-index="selectedHotbarIndex"
        :show-all-crafting-recipes="showAllCraftingRecipes"
        @clear-hotbar="clearSelectedHotbarSlot"
        @close="inventoryOpen = false"
        @craft-recipe="craftInventoryRecipe"
        @equip-inventory-slot-drop="equipInventorySlotDrop"
        @select-hotbar-slot="selectHotbarSlot"
        @toggle-crafting-panel="toggleCraftingPanel"
        @unequip-slot="unequipInventorySlot"
        @update-show-all-crafting-recipes="showAllCraftingRecipes = $event"
      />
  </SunnyTownHud>
</template>
