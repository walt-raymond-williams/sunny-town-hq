<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getNextStudentAssignment, submitStudentAnswer } from '../../api/studentAssignmentsApi'
import { purchaseShopItem } from '../../api/shopApi'
import {
  nearestSunnyTownNpc,
  useSunnyTownNpcInteractions,
} from '../../composables/useSunnyTownNpcInteractions'
import {
  canPlaceStoneBlock as canPlaceStoneBlockOnMap,
  movementDirectionForEvent,
  movementInputEventOptions,
  moveSendIntervalMs,
  useSunnyTownMovement,
} from '../../composables/useSunnyTownMovement'
import { useSunnyTownLocalPlayer } from '../../composables/useSunnyTownLocalPlayer'
import { useSunnyTownRemotePlayers } from '../../composables/useSunnyTownRemotePlayers'
import { useSunnyTownRenderer } from '../../composables/useSunnyTownRenderer'
import { useSunnyTownSocket } from '../../composables/useSunnyTownSocket'
import { useSunnyTownToolUseAnimation } from '../../composables/useSunnyTownToolUseAnimation'
import { useStudentInventoryStore } from '../../stores/studentInventory'
import type { EquipmentSlot } from '../../types/inventory'
import type {
  SunnyTownCollectible,
  SunnyTownEquipmentChangedMessage,
  SunnyTownMap,
  SunnyTownMoveMessage,
  SunnyTownNpc,
  SunnyTownPlaceObjectMessage,
  SunnyTownPlacedObject,
  SunnyTownPlayer,
  SunnyTownResourceNode,
  SunnyTownServerMessage,
  SunnyTownToolUseMessage,
  SunnyTownWorldObject,
} from '../../types/sunnyTown'
import SunnyTownCanvas from './SunnyTownCanvas.vue'
import SunnyTownDialogue from './SunnyTownDialogue.vue'
import SunnyTownHud from './SunnyTownHud.vue'
import SunnyTownInventoryPanel from './SunnyTownInventoryPanel.vue'
import SunnyTownSchoolworkPanel from './SunnyTownSchoolworkPanel.vue'
import SunnyTownShop from './SunnyTownShop.vue'
import {
  legacyWorldObjects,
  placedObjectToWorldObject,
  sameWorldObject,
  worldObjectToPlacedObject,
  worldObjectToResourceNode,
} from './worldObjects'

const npcInteractionRadius = 54
const toolUseDurationMs = 360

const router = useRouter()
const inventoryStore = useStudentInventoryStore()
const movement = useSunnyTownMovement()
const localPlayerState = useSunnyTownLocalPlayer()
const remotePlayerState = useSunnyTownRemotePlayers()
const toolUseAnimation = useSunnyTownToolUseAnimation()
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
const activeMap = ref<SunnyTownMap | null>(null)
const players = ref<SunnyTownPlayer[]>([])
const collectibles = ref<SunnyTownCollectible[]>([])
const resourceNodes = ref<SunnyTownResourceNode[]>([])
const placedObjects = ref<SunnyTownPlacedObject[]>([])
const worldObjects = ref<SunnyTownWorldObject[]>([])
const selfId = ref('')
const starBalance = ref(0)
const gameToast = ref('')
const inventoryOpen = ref(false)
const craftingPanelOpen = ref(false)
const showAllCraftingRecipes = ref(false)
const selectedHotbarIndex = ref(0)
const placementHoverGrid = ref<{ gridX: number; gridY: number } | null>(null)
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

const renderer = useSunnyTownRenderer(canvas, {
  drawScene,
  onFrame(deltaSeconds) {
    predictSelf(deltaSeconds)
    refreshNpcInteractionState()
  },
})
const draw = renderer.draw

const playerCount = computed(() => players.value.length)
const stoneBlockQuantity = computed(() => inventoryStore.items.find((item) => item.key === 'stone_block')?.quantity || 0)
const selectedHotbarSlot = computed(() => inventoryStore.hotbarSlots[selectedHotbarIndex.value] || null)
const selectedHotbarItem = computed(() => selectedHotbarSlot.value?.item || null)
const selectedHotbarItemKey = computed(() => selectedHotbarItem.value?.key || '')
const placingStoneBlock = computed(() => selectedHotbarItemKey.value === 'stone_block' && stoneBlockQuantity.value > 0)

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
    if (message.mapId && activeMap.value && message.mapId !== activeMap.value.id) {
      return
    }
    players.value = message.players || []
    collectibles.value = message.collectibles || []
    resourceNodes.value = message.resourceNodes || []
    placedObjects.value = message.placedObjects || placedObjects.value
    worldObjects.value = message.worldObjects || legacyWorldObjects(resourceNodes.value, placedObjects.value)
    remotePlayerState.recordSnapshots(players.value, selfId.value, message.serverTimeMs || Date.now())
    syncLocalSelfFromSnapshot()
    return
  }
  if (message.type === 'map_changed') {
    applyMapState(message)
    return
  }
  if (message.type === 'reward_committed') {
    starBalance.value = message.newStarBalance ?? starBalance.value + (message.amount || 0)
    gameToast.value = `+${message.amount || 1} star`
    window.setTimeout(() => {
      gameToast.value = ''
    }, 1200)
    return
  }
  if (message.type === 'reward_failed') {
    error.value = 'That star could not be saved. Try again in a moment.'
    return
  }
  if (message.type === 'resource_committed') {
    const amount = message.amount || 1
    const resourceKey = message.resourceKey || 'rock'
    gameToast.value = `+${amount} ${resourceKey}`
    if (message.quantity !== undefined) {
      inventoryStore.setItemQuantity(resourceKey, message.quantity)
    }
    if (craftingPanelOpen.value) {
      void inventoryStore.loadCraftingRecipes()
    }
    window.setTimeout(() => {
      gameToast.value = ''
    }, 1200)
    return
  }
  if (message.type === 'map_object_placed') {
    if (message.placedObject) {
      placedObjects.value = [
        ...placedObjects.value.filter((object) => object.id !== message.placedObject?.id),
        message.placedObject,
      ]
    }
    const placedWorldObject = message.worldObject
    if (placedWorldObject) {
      worldObjects.value = [
        ...worldObjects.value.filter((object) => !sameWorldObject(object, placedWorldObject)),
        placedWorldObject,
      ]
    } else if (message.placedObject) {
      worldObjects.value = [
        ...worldObjects.value.filter((object) => object.source !== 'placed' || object.id !== message.placedObject?.id),
        placedObjectToWorldObject(message.placedObject),
      ]
    }
    if (message.resourceKey && message.quantity !== undefined) {
      inventoryStore.setItemQuantity(message.resourceKey, message.quantity)
      gameToast.value = 'Stone block placed'
      window.setTimeout(() => {
        gameToast.value = ''
      }, 1200)
    }
    draw()
    return
  }
  if (message.type === 'map_object_removed') {
    if (message.placedObject) {
      placedObjects.value = placedObjects.value.filter((object) => object.id !== message.placedObject?.id)
    }
    const removedWorldObject = message.worldObject
    if (removedWorldObject) {
      worldObjects.value = worldObjects.value.filter((object) => !sameWorldObject(object, removedWorldObject))
    } else if (message.placedObject) {
      worldObjects.value = worldObjects.value.filter((object) => object.source !== 'placed' || object.id !== message.placedObject?.id)
    }
    if (message.resourceKey && message.quantity !== undefined) {
      inventoryStore.setItemQuantity(message.resourceKey, message.quantity)
      gameToast.value = '+1 stone_block'
      window.setTimeout(() => {
        gameToast.value = ''
      }, 1200)
    }
    if (craftingPanelOpen.value) {
      void inventoryStore.loadCraftingRecipes()
    }
    draw()
    return
  }
  if (message.type === 'resource_failed') {
    error.value = 'That resource could not be saved. Try again in a moment.'
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
  placementHoverGrid.value = gridFromPointer(event)
  draw()
}

function handleCanvasPointerLeave() {
  placementHoverGrid.value = null
  draw()
}

async function toggleInventory() {
  inventoryOpen.value = !inventoryOpen.value
  if (inventoryOpen.value) {
    craftingPanelOpen.value = true
    await inventoryStore.loadInventory()
    await inventoryStore.loadHotbar()
    await inventoryStore.loadCraftingRecipes()
  }
}

async function toggleCraftingPanel() {
  craftingPanelOpen.value = !craftingPanelOpen.value
  if (craftingPanelOpen.value) {
    await inventoryStore.loadCraftingRecipes()
  }
}

async function craftInventoryRecipe(recipeKey: string) {
  await inventoryStore.craftRecipe(recipeKey)
  await inventoryStore.loadHotbar()
}

function selectHotbarSlot(index: number) {
  selectedHotbarIndex.value = index
  placementHoverGrid.value = null
  draw()
}

function hotbarIndexForEvent(event: KeyboardEvent): number | null {
  if (!/^Digit[1-5]$/.test(event.code)) {
    return null
  }
  return Number(event.code.slice(5)) - 1
}

function placeStoneBlockAtPointer(event: PointerEvent) {
  const grid = gridFromPointer(event)
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

async function equipInventoryItem(itemKey: string, slot: EquipmentSlot | '') {
  if (!slot) {
    return
  }
  await inventoryStore.equipItem(slot, itemKey)
  applyLocalEquipmentVisuals()
  notifyEquipmentChanged()
}

async function unequipInventorySlot(slot: EquipmentSlot) {
  await inventoryStore.unequipItem(slot)
  applyLocalEquipmentVisuals()
  notifyEquipmentChanged()
}

async function assignInventoryItemToSelectedHotbarSlot(itemKey: string) {
  await inventoryStore.setHotbarSlot(selectedHotbarIndex.value + 1, itemKey)
  draw()
}

async function clearSelectedHotbarSlot() {
  await inventoryStore.setHotbarSlot(selectedHotbarIndex.value + 1, '')
  draw()
}

function applyLocalEquipmentVisuals() {
  const equipment = { ...inventoryStore.equippedVisuals }
  localPlayerState.applyEquipment(equipment)
  players.value = players.value.map((player) => (
    player.id === selfId.value ? { ...player, equipment } : player
  ))
  draw()
}

function gridFromPointer(event: PointerEvent): { gridX: number; gridY: number } | null {
  const map = activeMap.value
  const target = canvas.value
  if (!map || !target) {
    return null
  }
  const rect = target.getBoundingClientRect()
  const camera = currentCamera(rect.width, rect.height)
  const worldX = event.clientX - rect.left + camera.x
  const worldY = event.clientY - rect.top + camera.y
  const gridX = Math.floor(worldX / map.tileSize)
  const gridY = Math.floor(worldY / map.tileSize)
  if (gridX < 0 || gridY < 0 || gridX >= map.width || gridY >= map.height) {
    return null
  }
  return { gridX, gridY }
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
  if (message.map) {
    activeMap.value = {
      ...message.map,
      portals: message.map.portals || [],
      npcs: message.map.npcs || [],
      resourceNodes: message.map.resourceNodes || [],
      blockedRects: message.map.blockedRects || [],
      starSpawns: message.map.starSpawns || [],
      spawns: message.map.spawns || [],
    }
  }
  players.value = message.players || []
  collectibles.value = message.collectibles || []
  resourceNodes.value = message.resourceNodes || []
  placedObjects.value = message.placedObjects || []
  worldObjects.value = message.worldObjects || legacyWorldObjects(resourceNodes.value, placedObjects.value)
  remotePlayerState.clear()
  localPlayerState.clear()
  lastSentMoveJson = ''
  toolUseAnimation.clear()
  placementHoverGrid.value = null
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
  drawWorldObjects(context, cameraX, cameraY)
  drawPlacementPreview(context, map, cameraX, cameraY)
  for (const collectible of collectibles.value) {
    if (collectible.active) {
      drawCollectible(context, collectible, cameraX, cameraY)
    }
  }
  for (const npc of map.npcs) {
    drawNpc(context, npc, cameraX, cameraY)
  }
  for (const player of renderedPlayers) {
    drawPlayer(context, player, cameraX, cameraY)
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
  return canPlaceStoneBlockOnMap(map, gridX, gridY, stoneBlockQuantity.value, worldObjects.value, self)
}

function drawMap(
  context: CanvasRenderingContext2D,
  map: SunnyTownMap,
  cameraX: number,
  cameraY: number,
  width: number,
  height: number,
) {
  context.fillStyle = map.id === 'sunny-town-v1'
    ? '#8fcf85'
    : map.id === 'sunny-town-classroom' ? '#b7c6da' : map.id === 'forest-crossing-v1' ? '#6fb27a' : '#cda66f'
  context.fillRect(0, 0, width, height)

  if (map.id === 'sunny-town-v1') {
    context.fillStyle = '#d6bd79'
    context.fillRect(0 - cameraX, 420 - cameraY, map.width * map.tileSize, 124)
    context.fillRect(570 - cameraX, 0 - cameraY, 140, map.height * map.tileSize)
  } else if (map.id === 'sunny-town-classroom') {
    context.fillStyle = '#e4d4b5'
    context.fillRect(64 - cameraX, 64 - cameraY, map.width * map.tileSize - 128, map.height * map.tileSize - 128)
    context.fillStyle = '#3f596f'
    context.fillRect(224 - cameraX, 82 - cameraY, 192, 44)
  } else if (map.id === 'forest-crossing-v1') {
    context.fillStyle = '#7ac27d'
    context.fillRect(0, 0, width, height)
    context.fillStyle = '#d1b06a'
    context.fillRect(0 - cameraX, 420 - cameraY, map.width * map.tileSize, 124)
    context.fillStyle = '#3b88a3'
    context.fillRect(560 - cameraX, 0 - cameraY, 96, 384)
    context.fillRect(560 - cameraX, 576 - cameraY, 96, 384)
    context.fillStyle = '#a87c42'
    context.fillRect(548 - cameraX, 384 - cameraY, 120, 192)
    context.strokeStyle = '#765631'
    context.lineWidth = 4
    context.strokeRect(548 - cameraX, 384 - cameraY, 120, 192)
  } else {
    context.fillStyle = '#d9bd8d'
    context.fillRect(64 - cameraX, 64 - cameraY, map.width * map.tileSize - 128, map.height * map.tileSize - 128)
  }

  context.strokeStyle = 'rgba(255, 255, 255, 0.18)'
  context.lineWidth = 1
  for (let x = -cameraX % map.tileSize; x < width; x += map.tileSize) {
    context.beginPath()
    context.moveTo(x, 0)
    context.lineTo(x, height)
    context.stroke()
  }
  for (let y = -cameraY % map.tileSize; y < height; y += map.tileSize) {
    context.beginPath()
    context.moveTo(0, y)
    context.lineTo(width, y)
    context.stroke()
  }

  for (const blocked of map.blockedRects) {
    context.fillStyle = map.id === 'sunny-town-v1'
      ? blocked.width > 400 || blocked.height > 400 ? '#4f8a5b' : '#7e6b52'
      : map.id === 'sunny-town-classroom'
        ? blocked.width > 260 || blocked.height > 260 ? '#516070' : '#8a6f4d'
        : map.id === 'forest-crossing-v1'
          ? blocked.width > 90 || blocked.height > 90 ? '#3e7a45' : '#6b6f57'
          : blocked.width > 260 || blocked.height > 260 ? '#6d4f38' : '#8b6748'
    context.fillRect(blocked.x - cameraX, blocked.y - cameraY, blocked.width, blocked.height)
    if (map.id === 'forest-crossing-v1' && blocked.width <= 160 && blocked.height <= 160) {
      context.fillStyle = '#2f6b3b'
      context.beginPath()
      context.arc(blocked.x + blocked.width / 2 - cameraX, blocked.y + blocked.height / 2 - cameraY, Math.min(blocked.width, blocked.height) / 2, 0, Math.PI * 2)
      context.fill()
    }
  }

  for (const portal of map.portals) {
    context.fillStyle = '#3d2c22'
    context.fillRect(portal.x - cameraX, portal.y - cameraY, portal.width, portal.height)
    context.strokeStyle = '#f1d28f'
    context.lineWidth = 2
    context.strokeRect(portal.x - cameraX + 2, portal.y - cameraY + 2, portal.width - 4, portal.height - 4)
  }
}

function drawPlacedObject(
  context: CanvasRenderingContext2D,
  object: SunnyTownPlacedObject,
  cameraX: number,
  cameraY: number,
) {
  const x = object.x - cameraX
  const y = object.y - cameraY
  context.save()
  context.fillStyle = '#69727c'
  context.fillRect(x + 3, y + 3, object.width - 6, object.height - 6)
  context.strokeStyle = '#353b42'
  context.lineWidth = 2
  context.strokeRect(x + 3, y + 3, object.width - 6, object.height - 6)
  context.fillStyle = '#8d98a3'
  context.fillRect(x + 7, y + 7, object.width - 14, 5)
  context.fillStyle = '#4d555e'
  context.fillRect(x + 7, y + object.height - 12, object.width - 14, 4)
  context.restore()
}

function drawWorldObjects(context: CanvasRenderingContext2D, cameraX: number, cameraY: number) {
  for (const object of worldObjects.value) {
    if (!object.active) {
      continue
    }
    if (object.kind === 'stone_block' && object.itemKey === 'stone_block') {
      drawPlacedObject(context, worldObjectToPlacedObject(object), cameraX, cameraY)
      continue
    }
    if (object.kind === 'rock_node' && object.resourceKind === 'rock') {
      drawResourceNode(context, worldObjectToResourceNode(object), cameraX, cameraY)
    }
  }
}

function drawPlacementPreview(
  context: CanvasRenderingContext2D,
  map: SunnyTownMap,
  cameraX: number,
  cameraY: number,
) {
  if (!placingStoneBlock.value || !placementHoverGrid.value) {
    return
  }
  const { gridX, gridY } = placementHoverGrid.value
  const valid = canPlaceStoneBlock(map, gridX, gridY)
  const x = gridX * map.tileSize - cameraX
  const y = gridY * map.tileSize - cameraY
  context.save()
  context.globalAlpha = 0.72
  context.fillStyle = valid ? '#8d98a3' : '#b94a48'
  context.fillRect(x + 3, y + 3, map.tileSize - 6, map.tileSize - 6)
  context.globalAlpha = 1
  context.strokeStyle = valid ? '#f7e08a' : '#ffcbc7'
  context.lineWidth = 2
  context.strokeRect(x + 2, y + 2, map.tileSize - 4, map.tileSize - 4)
  context.restore()
}

function drawResourceNode(
  context: CanvasRenderingContext2D,
  node: SunnyTownResourceNode,
  cameraX: number,
  cameraY: number,
) {
  if (!node.active) {
    return
  }
  const x = node.x - cameraX
  const y = node.y - cameraY
  context.save()
  context.translate(x, y)
  context.fillStyle = '#6b737b'
  context.strokeStyle = '#343a40'
  context.lineWidth = 3
  context.beginPath()
  context.moveTo(-node.radius, 4)
  context.lineTo(-node.radius * 0.55, -node.radius * 0.7)
  context.lineTo(node.radius * 0.25, -node.radius)
  context.lineTo(node.radius, -node.radius * 0.1)
  context.lineTo(node.radius * 0.7, node.radius * 0.75)
  context.lineTo(-node.radius * 0.45, node.radius)
  context.closePath()
  context.fill()
  context.stroke()
  context.fillStyle = '#bcd5e8'
  context.beginPath()
  context.arc(node.radius * 0.25, -node.radius * 0.35, 4, 0, Math.PI * 2)
  context.fill()
  const hits = Math.max(0, node.hits || 0)
  if (hits > 0) {
    context.strokeStyle = '#23282e'
    context.lineWidth = 2
    context.beginPath()
    context.moveTo(-node.radius * 0.15, -node.radius * 0.75)
    context.lineTo(node.radius * 0.05, -node.radius * 0.25)
    context.lineTo(-node.radius * 0.2, node.radius * 0.15)
    context.stroke()
  }
  if (hits > 1) {
    context.beginPath()
    context.moveTo(node.radius * 0.2, -node.radius * 0.45)
    context.lineTo(node.radius * 0.45, -node.radius * 0.05)
    context.lineTo(node.radius * 0.25, node.radius * 0.45)
    context.stroke()
  }
  context.restore()
}

function drawPlayer(context: CanvasRenderingContext2D, player: SunnyTownPlayer, cameraX: number, cameraY: number) {
  const x = player.x - cameraX
  const y = player.y - cameraY
  const isSelf = player.id === selfId.value

  context.fillStyle = 'rgba(0, 0, 0, 0.18)'
  context.beginPath()
  context.ellipse(x, y + 16, 18, 7, 0, 0, Math.PI * 2)
  context.fill()

  const gearKey = player.equipment?.gear || ''
  const accessoryKey = player.equipment?.accessory || ''
  const toolKey = isSelf ? selectedHotbarItemKey.value : player.equipment?.tool || ''
  const toolProgress = isSelf ? currentToolUseProgress(toolKey) : null

  context.fillStyle = gearKey === 'sunny_hoodie' ? '#f06f38' : (isSelf ? '#27746f' : '#5c6bc0')
  context.beginPath()
  context.arc(x, y, 16, 0, Math.PI * 2)
  context.fill()

  if (toolKey === 'pickaxe') {
    drawPickaxe(context, x, y, player.facing, toolProgress)
  }

  if (gearKey === 'sunny_hoodie') {
    context.fillStyle = '#2f7d72'
    context.fillRect(x - 10, y + 2, 20, 8)
    context.strokeStyle = '#f4d48e'
    context.lineWidth = 2
    context.beginPath()
    context.moveTo(x, y + 2)
    context.lineTo(x, y + 10)
    context.stroke()
  }

  if (accessoryKey === 'star_cap') {
    context.fillStyle = '#f2c84b'
    context.beginPath()
    context.ellipse(x, y - 14, 14, 6, 0, 0, Math.PI * 2)
    context.fill()
    context.fillStyle = '#365d9f'
    context.fillRect(x - 9, y - 20, 18, 8)
    context.fillStyle = '#ffffff'
    context.beginPath()
    context.moveTo(x, y - 21)
    context.lineTo(x + 3, y - 16)
    context.lineTo(x + 8, y - 16)
    context.lineTo(x + 4, y - 13)
    context.lineTo(x + 6, y - 8)
    context.lineTo(x, y - 11)
    context.lineTo(x - 6, y - 8)
    context.lineTo(x - 4, y - 13)
    context.lineTo(x - 8, y - 16)
    context.lineTo(x - 3, y - 16)
    context.closePath()
    context.fill()
  }

  context.fillStyle = '#ffffff'
  context.beginPath()
  context.arc(x - 5, y - 4, 3, 0, Math.PI * 2)
  context.arc(x + 5, y - 4, 3, 0, Math.PI * 2)
  context.fill()

  context.fillStyle = '#17212b'
  context.font = '700 12px Inter, sans-serif'
  context.textAlign = 'center'
  context.fillText(player.displayName, x, y - 24)
}

function currentToolUseProgress(toolKey: string): number | null {
  return toolUseAnimation.progress(toolKey, performance.now())
}

function drawPickaxe(
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  facing: SunnyTownPlayer['facing'],
  swingProgress: number | null,
) {
  const swinging = swingProgress !== null
  const direction = directionVector(facing)
  const side = facing === 'left' || facing === 'right' ? -1 : 1
  const baseX = x + direction.x * 17 + (facing === 'up' || facing === 'down' ? 13 : 0)
  const baseY = y + direction.y * 14 + (facing === 'left' || facing === 'right' ? 2 : 6)
  const swingAngle = swinging ? (Math.sin(swingProgress * Math.PI) * 1.2 - 0.6) * side : 0
  const restingAngle = facing === 'left'
    ? -0.8
    : facing === 'right'
      ? 0.8
      : facing === 'up'
        ? -0.35
        : 0.35

  context.save()
  context.translate(baseX, baseY)
  context.rotate(restingAngle + swingAngle)
  context.lineCap = 'round'
  context.strokeStyle = swinging ? '#f6d56f' : '#7b4b24'
  context.lineWidth = swinging ? 5 : 4
  context.beginPath()
  context.moveTo(0, 12)
  context.lineTo(0, -13)
  context.stroke()
  context.strokeStyle = '#5f6b75'
  context.lineWidth = swinging ? 6 : 5
  context.beginPath()
  context.moveTo(-10, -14)
  context.quadraticCurveTo(0, -21, 12, -14)
  context.stroke()
  context.restore()
}

function directionVector(facing: SunnyTownPlayer['facing']) {
  switch (facing) {
    case 'up':
      return { x: 0, y: -1 }
    case 'down':
      return { x: 0, y: 1 }
    case 'left':
      return { x: -1, y: 0 }
    case 'right':
      return { x: 1, y: 0 }
  }
}

function drawNpc(context: CanvasRenderingContext2D, npc: SunnyTownNpc, cameraX: number, cameraY: number) {
  const x = npc.x - cameraX
  const y = npc.y - cameraY
  const isNearby = nearbyNpc.value?.id === npc.id

  context.fillStyle = 'rgba(0, 0, 0, 0.18)'
  context.beginPath()
  context.ellipse(x, y + 16, 18, 7, 0, 0, Math.PI * 2)
  context.fill()

  context.fillStyle = npc.spriteKey === 'keeper' ? '#8b4f9f' : npc.spriteKey === 'teacher' ? '#2f6b8f' : '#b96b4f'
  context.beginPath()
  context.arc(x, y, 16, 0, Math.PI * 2)
  context.fill()

  context.fillStyle = '#f7d7b5'
  context.beginPath()
  context.arc(x, y - 4, 9, 0, Math.PI * 2)
  context.fill()

  context.fillStyle = '#17212b'
  context.beginPath()
  context.arc(x - 3, y - 6, 1.5, 0, Math.PI * 2)
  context.arc(x + 3, y - 6, 1.5, 0, Math.PI * 2)
  context.fill()

  context.strokeStyle = isNearby ? '#f1d28f' : 'rgba(255, 255, 255, 0.38)'
  context.lineWidth = isNearby ? 3 : 2
  context.beginPath()
  context.arc(x, y, 19, 0, Math.PI * 2)
  context.stroke()

  context.fillStyle = '#17212b'
  context.font = '700 12px Inter, sans-serif'
  context.textAlign = 'center'
  context.fillText(npc.name, x, y - 28)
}

function drawCollectible(
  context: CanvasRenderingContext2D,
  collectible: SunnyTownCollectible,
  cameraX: number,
  cameraY: number,
) {
  const x = collectible.x - cameraX
  const y = collectible.y - cameraY
  context.save()
  context.translate(x, y)
  context.fillStyle = '#f6c945'
  context.strokeStyle = '#7a5a00'
  context.lineWidth = 2
  context.beginPath()
  for (let index = 0; index < 10; index++) {
    const radius = index % 2 === 0 ? 15 : 7
    const angle = -Math.PI / 2 + (index * Math.PI) / 5
    const px = Math.cos(angle) * radius
    const py = Math.sin(angle) * radius
    if (index === 0) {
      context.moveTo(px, py)
    } else {
      context.lineTo(px, py)
    }
  }
  context.closePath()
  context.fill()
  context.stroke()
  context.restore()
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
