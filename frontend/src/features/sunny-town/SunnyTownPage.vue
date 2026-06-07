<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getNextStudentAssignment, submitStudentAnswer } from '../../api/studentAssignmentsApi'
import { purchaseShopItem } from '../../api/shopApi'
import {
  hotbarIndexForEvent,
  useSunnyTownInventoryActions,
} from '../../composables/useSunnyTownInventoryActions'
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
import { useStudentInventoryStore } from '../../stores/studentInventory'
import type {
  SunnyTownEquipmentChangedMessage,
  SunnyTownMap,
  SunnyTownMoveMessage,
  SunnyTownNpc,
  SunnyTownPlaceObjectMessage,
  SunnyTownPlayer,
  SunnyTownServerMessage,
  SunnyTownToolUseMessage,
} from '../../types/sunnyTown'
import SunnyTownCanvas from './SunnyTownCanvas.vue'
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
const inventoryStore = useStudentInventoryStore()
const movement = useSunnyTownMovement()
const localPlayerState = useSunnyTownLocalPlayer()
const placement = useSunnyTownPlacement()
const remotePlayerState = useSunnyTownRemotePlayers()
const toolUseAnimation = useSunnyTownToolUseAnimation()
const worldState = useSunnyTownWorldState()
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
  refreshNearby: refreshNearbyNpc,
  schoolworkAnswer,
  schoolworkAssignment,
  schoolworkError,
  schoolworkNotice,
  schoolworkOpen,
  shopError,
  shopNotice,
  shopOpen,
} = useSunnyTownNpcInteractions()
const canvas = ref<HTMLCanvasElement | null>(null)
const {
  activeMap,
  applyPlacedObject,
  applyRemovedObject,
  applySnapshot,
  collectibles,
  players,
  worldObjects,
} = worldState
const selfId = ref('')
const starBalance = ref(0)
const gameToast = ref('')
const placementHoverGrid = placement.hoverGrid
const {
  assignInventoryItemToSelectedHotbarSlot,
  clearSelectedHotbarSlot,
  craftInventoryRecipe,
  craftingPanelOpen,
  equipInventoryItem,
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
  if (message.type === 'resource_failed') {
    messageEffects.applyResourceFailed()
    return
  }
  if (message.type === 'error') {
    error.value = message.code || 'Sunny Town received an invalid message'
  }
}

function handleKeyDown(event: KeyboardEvent) {
  if (event.code === 'Escape') {
    if (hasActiveOverlay() || inventoryOpen.value) {
      event.preventDefault()
      closeNpcOverlays()
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
  if (hasActiveOverlay() || inventoryOpen.value) {
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
  nearbyNpc.value = null
  syncLocalSelfFromSnapshot()
  draw()
}

function handlePrimaryInteraction() {
  if (interactWithNearbyNpc()) {
    return
  }
  useEquippedTool()
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

async function startSchoolwork() {
  if (!activeSchoolworkNpc.value || isLoadingSchoolwork.value) {
    return
  }
  schoolworkOpen.value = true
  schoolworkAssignment.value = null
  schoolworkAnswer.value = ''
  schoolworkError.value = ''
  schoolworkNotice.value = ''
  isLoadingSchoolwork.value = true

  try {
    schoolworkAssignment.value = await getNextStudentAssignment()
  } catch (caught) {
    schoolworkError.value = errorMessage(caught)
  } finally {
    isLoadingSchoolwork.value = false
  }
}

async function openTrade() {
  if (!activeShopNpc.value?.shop) {
    return
  }
  shopOpen.value = true
  shopError.value = ''
  shopNotice.value = ''
  await inventoryStore.loadInventory()
}

async function buyShopItem(itemKey: string) {
  const shop = activeShopNpc.value?.shop
  if (!shop || isPurchasing.value) {
    return
  }
  isPurchasing.value = true
  shopError.value = ''
  shopNotice.value = ''

  try {
    const purchase = await purchaseShopItem({
      shopId: shop.id,
      itemKey,
      quantity: 1,
    })
    starBalance.value = purchase.starBalance
    inventoryStore.setItems(purchase.inventory.items)
    shopNotice.value = 'Purchased.'
  } catch (caught) {
    shopError.value = caught instanceof Error ? caught.message : String(caught)
  } finally {
    isPurchasing.value = false
  }
}

async function submitSchoolworkAnswer() {
  if (!schoolworkAssignment.value || isSubmittingSchoolwork.value) {
    return
  }

  isSubmittingSchoolwork.value = true
  schoolworkError.value = ''
  schoolworkNotice.value = ''

  try {
    await submitStudentAnswer(schoolworkAssignment.value.id, schoolworkAnswer.value.trim())
    schoolworkNotice.value = 'Answer submitted.'
    schoolworkAnswer.value = ''
    schoolworkAssignment.value = await getNextStudentAssignment()
  } catch (caught) {
    schoolworkError.value = errorMessage(caught)
  } finally {
    isSubmittingSchoolwork.value = false
  }
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
  for (const npc of map.npcs) {
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
}

function nearestNpcToSelf(): SunnyTownNpc | null {
  const map = activeMap.value
  const self = localPlayerState.current(players.value, selfId.value)
  if (!map) {
    return null
  }
  return nearestSunnyTownNpc(map.npcs, self, npcInteractionRadius)
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

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
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
      <div v-if="nearbyNpc && !activeDialogueNpc && !activeSchoolworkNpc && !inventoryOpen" class="sunny-town-talk-hint">
        <v-icon icon="mdi-chat" size="small" />
        <span>F {{ nearbyNpc.name }}</span>
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
      <SunnyTownInventoryPanel
        v-if="inventoryOpen"
        :crafting-panel-open="craftingPanelOpen"
        :selected-hotbar-index="selectedHotbarIndex"
        :show-all-crafting-recipes="showAllCraftingRecipes"
        @assign-hotbar="assignInventoryItemToSelectedHotbarSlot"
        @clear-hotbar="clearSelectedHotbarSlot"
        @close="inventoryOpen = false"
        @craft-recipe="craftInventoryRecipe"
        @equip-item="equipInventoryItem"
        @toggle-crafting-panel="toggleCraftingPanel"
        @unequip-slot="unequipInventorySlot"
        @update-show-all-crafting-recipes="showAllCraftingRecipes = $event"
      />
  </SunnyTownHud>
</template>
