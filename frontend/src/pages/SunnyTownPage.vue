<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getNextStudentAssignment, submitStudentAnswer } from '../api/studentAssignmentsApi'
import { createSunnyTownSession } from '../api/sunnyTownApi'
import { purchaseShopItem } from '../api/shopApi'
import { useStudentInventoryStore } from '../stores/studentInventory'
import type { Assignment } from '../types/assignment'
import type { EquipmentSlot } from '../types/inventory'
import type {
  SunnyTownCollectible,
  SunnyTownEquipmentChangedMessage,
  SunnyTownMap,
  SunnyTownMoveMessage,
  SunnyTownNpc,
  SunnyTownPlayer,
  SunnyTownResourceNode,
  SunnyTownServerMessage,
  SunnyTownSession,
  SunnyTownToolUseMessage,
} from '../types/sunnyTown'

interface MovementInput {
  up: boolean
  down: boolean
  left: boolean
  right: boolean
}

type MovementDirection = keyof MovementInput

interface ToolUseAnimation {
  toolKey: string
  startedAt: number
  durationMs: number
  facing: SunnyTownPlayer['facing']
}

const playerSpeed = 150
const playerSize = 28
const moveSendIntervalMs = 50
const remoteInterpolationDelayMs = 150
const maxRemoteHistoryFrames = 12
const npcInteractionRadius = 54
const toolUseDurationMs = 360
const reconnectInitialDelayMs = 500
const reconnectMaxDelayMs = 8_000
const movementInputEventOptions = { capture: true }

const router = useRouter()
const inventoryStore = useStudentInventoryStore()
const canvas = ref<HTMLCanvasElement | null>(null)
const session = ref<SunnyTownSession | null>(null)
const activeMap = ref<SunnyTownMap | null>(null)
const players = ref<SunnyTownPlayer[]>([])
const collectibles = ref<SunnyTownCollectible[]>([])
const resourceNodes = ref<SunnyTownResourceNode[]>([])
const selfId = ref('')
const status = ref('Entering Sunny Town...')
const error = ref('')
const connected = ref(false)
const starBalance = ref(0)
const gameToast = ref('')
const inventoryOpen = ref(false)
const craftingPanelOpen = ref(false)
const showAllCraftingRecipes = ref(false)
const nearbyNpc = ref<SunnyTownNpc | null>(null)
const activeDialogueNpc = ref<SunnyTownNpc | null>(null)
const activeDialogueLineIndex = ref(0)
const activeShopNpc = ref<SunnyTownNpc | null>(null)
const shopOpen = ref(false)
const shopError = ref('')
const shopNotice = ref('')
const isPurchasing = ref(false)
const activeSchoolworkNpc = ref<SunnyTownNpc | null>(null)
const schoolworkOpen = ref(false)
const schoolworkAssignment = ref<Assignment | null>(null)
const schoolworkAnswer = ref('')
const schoolworkError = ref('')
const schoolworkNotice = ref('')
const isLoadingSchoolwork = ref(false)
const isSubmittingSchoolwork = ref(false)

const pressedDirections = new Set<MovementDirection>()
const remotePlayerHistories = new Map<string, Array<{ at: number; player: SunnyTownPlayer }>>()
let localSelf: SunnyTownPlayer | null = null
let renderedSelf: SunnyTownPlayer | null = null
let socket: WebSocket | null = null
let animationFrame = 0
let moveSeq = 0
let lastMoveSendAtMs = 0
let lastSentMoveJson = ''
let lastRenderTime = 0
let activeToolUse: ToolUseAnimation | null = null
let reconnectTimer = 0
let reconnectAttempts = 0
let shuttingDown = false

const playerCount = computed(() => players.value.length)
const activeDialogueLine = computed(() => activeDialogueNpc.value?.dialogue[activeDialogueLineIndex.value] || '')
const dialogueProgress = computed(() => {
  if (!activeDialogueNpc.value) {
    return ''
  }
  return `${activeDialogueLineIndex.value + 1}/${activeDialogueNpc.value.dialogue.length}`
})
const shopItems = computed(() => activeShopNpc.value?.shop?.items || [])
const visibleCraftingRecipes = computed(() => (
  showAllCraftingRecipes.value ? inventoryStore.craftingRecipes : inventoryStore.craftableRecipes
))

onMounted(async () => {
  window.addEventListener('keydown', handleKeyDown, movementInputEventOptions)
  window.addEventListener('keyup', handleKeyUp, movementInputEventOptions)
  window.addEventListener('blur', handleInputCancel)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  window.addEventListener('resize', handleResize)
  animationFrame = window.requestAnimationFrame(renderLoop)

  try {
    session.value = await createSunnyTownSession()
    starBalance.value = session.value.wallet.starBalance
    status.value = 'Connecting...'
    await nextTick()
    connect(session.value)
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
    status.value = 'Could not enter Sunny Town'
  }
})

onBeforeUnmount(() => {
  shuttingDown = true
  window.removeEventListener('keydown', handleKeyDown, movementInputEventOptions)
  window.removeEventListener('keyup', handleKeyUp, movementInputEventOptions)
  window.removeEventListener('blur', handleInputCancel)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  window.removeEventListener('resize', handleResize)
  window.cancelAnimationFrame(animationFrame)
  clearReconnectTimer()
  socket?.close()
  socket = null
  localSelf = null
  renderedSelf = null
  remotePlayerHistories.clear()
})

function connect(activeSession: SunnyTownSession) {
  const url = new URL(activeSession.websocketUrl)
  url.searchParams.set('token', activeSession.joinToken)
  const nextSocket = new WebSocket(url.toString())
  socket = nextSocket

  nextSocket.addEventListener('open', () => {
    if (socket !== nextSocket) {
      return
    }
    reconnectAttempts = 0
    connected.value = true
    status.value = 'Connected'
    error.value = ''
  })

  nextSocket.addEventListener('message', (event) => {
    if (socket !== nextSocket) {
      return
    }
    const message = parseServerMessage(event.data)
    if (!message) {
      return
    }
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
      recordRemoteSnapshots(players.value, message.serverTimeMs || Date.now())
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
    if (message.type === 'resource_failed') {
      error.value = 'That resource could not be saved. Try again in a moment.'
      return
    }
    if (message.type === 'error') {
      error.value = message.code || 'Sunny Town received an invalid message'
    }
  })

  nextSocket.addEventListener('close', () => {
    if (socket !== nextSocket) {
      return
    }
    connected.value = false
    socket = null
    if (shuttingDown) {
      status.value = 'Disconnected'
      return
    }
    scheduleReconnect()
  })

  nextSocket.addEventListener('error', () => {
    if (socket !== nextSocket) {
      return
    }
    error.value = 'Sunny Town connection failed'
  })
}

function scheduleReconnect() {
  if (shuttingDown || reconnectTimer) {
    return
  }

  const delay = Math.min(reconnectMaxDelayMs, reconnectInitialDelayMs * 2 ** reconnectAttempts)
  reconnectAttempts += 1
  status.value = 'Reconnecting...'
  reconnectTimer = window.setTimeout(() => {
    reconnectTimer = 0
    void reconnectSunnyTown()
  }, delay)
}

async function reconnectSunnyTown() {
  if (shuttingDown) {
    return
  }

  try {
    const nextSession = await createSunnyTownSession()
    session.value = nextSession
    starBalance.value = nextSession.wallet.starBalance
    connect(nextSession)
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
    scheduleReconnect()
  }
}

function clearReconnectTimer() {
  if (reconnectTimer) {
    window.clearTimeout(reconnectTimer)
    reconnectTimer = 0
  }
}

function parseServerMessage(data: unknown): SunnyTownServerMessage | null {
  if (typeof data !== 'string') {
    error.value = 'Sunny Town sent an unsupported message'
    return null
  }

  try {
    return JSON.parse(data) as SunnyTownServerMessage
  } catch {
    error.value = 'Sunny Town sent an invalid message'
    return null
  }
}

function handleKeyDown(event: KeyboardEvent) {
  if (event.code === 'Escape') {
    if (activeDialogueNpc.value || activeShopNpc.value || activeSchoolworkNpc.value || inventoryOpen.value) {
      event.preventDefault()
      closeNpcOverlays()
      inventoryOpen.value = false
    }
    return
  }

  if (isEditableKeyboardTarget(event.target)) {
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
  pressedDirections.add(direction)
  refreshLocalMovementState()
  sendMove(true)
}

function handleKeyUp(event: KeyboardEvent) {
  const direction = movementDirectionForEvent(event)
  if (!direction) {
    return
  }
  event.preventDefault()
  pressedDirections.delete(direction)
  refreshLocalMovementState()
  sendMove(true)
}

function handleVisibilityChange() {
  if (document.visibilityState === 'hidden') {
    handleInputCancel()
  }
}

function handleInputCancel() {
  if (pressedDirections.size === 0) {
    return
  }
  pressedDirections.clear()
  refreshLocalMovementState()
  sendMove(true)
}

function handleResize() {
  draw()
}

function handleCanvasPointerDown(event: PointerEvent) {
  if (event.button !== 0) {
    return
  }
  if (activeDialogueNpc.value || activeShopNpc.value || activeSchoolworkNpc.value || inventoryOpen.value) {
    return
  }
  event.preventDefault()
  useEquippedTool()
}

async function toggleInventory() {
  inventoryOpen.value = !inventoryOpen.value
  if (inventoryOpen.value) {
    await inventoryStore.loadInventory()
    if (craftingPanelOpen.value) {
      await inventoryStore.loadCraftingRecipes()
    }
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
}

function notifyEquipmentChanged() {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    return
  }
  const message: SunnyTownEquipmentChangedMessage = { type: 'equipment_changed' }
  socket.send(JSON.stringify(message))
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

function applyLocalEquipmentVisuals() {
  const equipment = { ...inventoryStore.equippedVisuals }
  if (localSelf) {
    localSelf.equipment = equipment
  }
  if (renderedSelf) {
    renderedSelf.equipment = equipment
  }
  players.value = players.value.map((player) => (
    player.id === selfId.value ? { ...player, equipment } : player
  ))
  draw()
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
  remotePlayerHistories.clear()
  localSelf = null
  renderedSelf = null
  lastSentMoveJson = ''
  activeToolUse = null
  pressedDirections.clear()
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
  if (activeSchoolworkNpc.value) {
    return true
  }
  if (activeShopNpc.value) {
    return true
  }
  if (activeDialogueNpc.value) {
    if (activeDialogueLineIndex.value < activeDialogueNpc.value.dialogue.length - 1) {
      activeDialogueLineIndex.value++
    } else {
      closeDialogue()
    }
    return true
  }

  const npc = nearestNpcToSelf()
  if (!npc) {
    return false
  }
  if (npc.shop) {
    openShopMenu(npc)
    return true
  }
  if (npc.activity?.type === 'schoolwork') {
    openSchoolworkMenu(npc)
    return true
  }
  activeDialogueNpc.value = npc
  activeDialogueLineIndex.value = 0
  return true
}

function useEquippedTool() {
  const player = localSelf || players.value.find((candidate) => candidate.id === selfId.value)
  const toolKey = player?.equipment?.tool || ''
  if (!player || !toolKey) {
    return
  }

  activeToolUse = {
    toolKey,
    startedAt: performance.now(),
    durationMs: toolUseDurationMs,
    facing: player.facing,
  }

  if (socket?.readyState === WebSocket.OPEN) {
    const message: SunnyTownToolUseMessage = {
      type: 'tool_use',
      toolKey,
      x: Math.round(player.x * 10) / 10,
      y: Math.round(player.y * 10) / 10,
      facing: player.facing,
      clientTimeMs: Date.now(),
    }
    socket.send(JSON.stringify(message))
  }
  draw()
}

function closeDialogue() {
  activeDialogueNpc.value = null
  activeDialogueLineIndex.value = 0
}

function openShopMenu(npc: SunnyTownNpc) {
  activeDialogueNpc.value = null
  activeDialogueLineIndex.value = 0
  activeSchoolworkNpc.value = null
  schoolworkOpen.value = false
  activeShopNpc.value = npc
  shopOpen.value = false
  shopError.value = ''
  shopNotice.value = ''
}

function openSchoolworkMenu(npc: SunnyTownNpc) {
  activeDialogueNpc.value = null
  activeDialogueLineIndex.value = 0
  activeShopNpc.value = null
  shopOpen.value = false
  activeSchoolworkNpc.value = npc
  schoolworkOpen.value = false
  schoolworkAssignment.value = null
  schoolworkAnswer.value = ''
  schoolworkError.value = ''
  schoolworkNotice.value = ''
  handleInputCancel()
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

function closeNpcOverlays() {
  closeDialogue()
  activeShopNpc.value = null
  shopOpen.value = false
  shopError.value = ''
  shopNotice.value = ''
  isPurchasing.value = false
  activeSchoolworkNpc.value = null
  schoolworkOpen.value = false
  schoolworkAssignment.value = null
  schoolworkAnswer.value = ''
  schoolworkError.value = ''
  schoolworkNotice.value = ''
  isLoadingSchoolwork.value = false
  isSubmittingSchoolwork.value = false
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
  if (!localSelf) {
    return
  }
  localSelf = simulatePlayer(localSelf, currentMovementInput(), 0)
}

function sendMove(force = false) {
  if (!socket || socket.readyState !== WebSocket.OPEN || !localSelf) {
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
  socket.send(JSON.stringify(message))
}

function renderLoop() {
  const now = performance.now()
  const deltaSeconds = lastRenderTime ? (now - lastRenderTime) / 1000 : 0
  lastRenderTime = now
  predictSelf(deltaSeconds)
  refreshNpcInteractionState()
  draw()
  animationFrame = window.requestAnimationFrame(renderLoop)
}

function draw() {
  const target = canvas.value
  if (!target) {
    return
  }

  const context = target.getContext('2d')
  if (!context) {
    return
  }

  const rect = target.getBoundingClientRect()
  const scale = window.devicePixelRatio || 1
  const width = Math.max(1, Math.floor(rect.width * scale))
  const height = Math.max(1, Math.floor(rect.height * scale))
  if (target.width !== width || target.height !== height) {
    target.width = width
    target.height = height
  }

  context.setTransform(scale, 0, 0, scale, 0, 0)
  context.clearRect(0, 0, rect.width, rect.height)

  const map = activeMap.value
  if (!map) {
    context.fillStyle = '#8fcf85'
    context.fillRect(0, 0, rect.width, rect.height)
    return
  }

  const renderedPlayers = renderedSunnyTownPlayers()
  const self = renderedPlayers.find((player) => player.id === selfId.value) || renderedPlayers[0]
  const worldWidth = map.width * map.tileSize
  const worldHeight = map.height * map.tileSize
  const cameraX = clamp((self?.x || worldWidth / 2) - rect.width / 2, 0, Math.max(0, worldWidth - rect.width))
  const cameraY = clamp((self?.y || worldHeight / 2) - rect.height / 2, 0, Math.max(0, worldHeight - rect.height))

  drawMap(context, map, cameraX, cameraY, rect.width, rect.height)
  for (const collectible of collectibles.value) {
    if (collectible.active) {
      drawCollectible(context, collectible, cameraX, cameraY)
    }
  }
  for (const node of resourceNodes.value) {
    drawResourceNode(context, node, cameraX, cameraY)
  }
  for (const npc of map.npcs) {
    drawNpc(context, npc, cameraX, cameraY)
  }
  for (const player of renderedPlayers) {
    drawPlayer(context, player, cameraX, cameraY)
  }
}

function refreshNpcInteractionState() {
  const nextNearbyNpc = nearestNpcToSelf()
  nearbyNpc.value = nextNearbyNpc
  if (activeDialogueNpc.value && nextNearbyNpc?.id !== activeDialogueNpc.value.id) {
    closeDialogue()
  }
  if (activeShopNpc.value && nextNearbyNpc?.id !== activeShopNpc.value.id) {
    closeNpcOverlays()
  }
  if (activeSchoolworkNpc.value && nextNearbyNpc?.id !== activeSchoolworkNpc.value.id) {
    closeNpcOverlays()
  }
}

function nearestNpcToSelf(): SunnyTownNpc | null {
  const map = activeMap.value
  const self = localSelf || players.value.find((player) => player.id === selfId.value)
  if (!map || !self) {
    return null
  }

  let nearest: SunnyTownNpc | null = null
  let nearestDistance = npcInteractionRadius
  for (const npc of map.npcs) {
    const distance = Math.hypot(self.x - npc.x, self.y - npc.y)
    if (distance <= nearestDistance) {
      nearest = npc
      nearestDistance = distance
    }
  }
  return nearest
}

function renderedSunnyTownPlayers(): SunnyTownPlayer[] {
  const selfSnapshot = players.value.find((player) => player.id === selfId.value)
  const remotePlayers = players.value.filter((player) => player.id !== selfId.value)
  const renderedRemotePlayers = smoothRemotePlayers(remotePlayers)
  if (!selfSnapshot) {
    return renderedRemotePlayers
  }

  if (!localSelf) {
    localSelf = { ...selfSnapshot }
  }
  if (!renderedSelf) {
    renderedSelf = { ...localSelf }
  }

  const target = localSelf
  renderedSelf.x = target.x
  renderedSelf.y = target.y
  renderedSelf.displayName = target.displayName
  renderedSelf.facing = target.facing
  renderedSelf.moving = target.moving
  renderedSelf.avatarId = target.avatarId
  renderedSelf.equipment = target.equipment ? { ...target.equipment } : undefined
  renderedSelf.lastProcessedSeq = target.lastProcessedSeq

  return [...renderedRemotePlayers, renderedSelf]
}

function recordRemoteSnapshots(snapshotPlayers: SunnyTownPlayer[], snapshotAt: number) {
  const remoteIds = new Set<string>()
  for (const player of snapshotPlayers) {
    if (player.id === selfId.value) {
      continue
    }

    remoteIds.add(player.id)
    const history = remotePlayerHistories.get(player.id) || []
    const latest = history[history.length - 1]
    if (latest && latest.at === snapshotAt) {
      latest.player = { ...player }
      continue
    }

    history.push({ at: snapshotAt, player: { ...player } })
    while (history.length > maxRemoteHistoryFrames) {
      history.shift()
    }
    remotePlayerHistories.set(player.id, history)
  }

  for (const playerId of remotePlayerHistories.keys()) {
    if (!remoteIds.has(playerId)) {
      remotePlayerHistories.delete(playerId)
    }
  }
}

function smoothRemotePlayers(remotePlayers: SunnyTownPlayer[]): SunnyTownPlayer[] {
  const renderAt = Date.now() - remoteInterpolationDelayMs
  return remotePlayers.map((target) => interpolateRemotePlayer(target, renderAt))
}

function interpolateRemotePlayer(target: SunnyTownPlayer, renderAt: number): SunnyTownPlayer {
  const history = remotePlayerHistories.get(target.id)
  if (!history || history.length === 0) {
    return { ...target }
  }
  const first = history[0]
  const latest = history[history.length - 1]
  if (!first || !latest) {
    return { ...target }
  }
  if (history.length === 1 || renderAt <= first.at) {
    return { ...first.player }
  }

  let before = first
  let after = latest
  for (let index = 1; index < history.length; index++) {
    const candidate = history[index]
    if (!candidate) {
      continue
    }
    if (candidate.at >= renderAt) {
      after = candidate
      break
    }
    before = candidate
  }

  if (renderAt >= after.at || after.at <= before.at) {
    return { ...after.player }
  }

  const progress = clamp((renderAt - before.at) / (after.at - before.at), 0, 1)
  return {
    ...after.player,
    x: before.player.x + (after.player.x - before.player.x) * progress,
    y: before.player.y + (after.player.y - before.player.y) * progress,
    facing: progress < 0.5 ? before.player.facing : after.player.facing,
    moving: before.player.moving || after.player.moving,
  }
}

function predictSelf(deltaSeconds: number) {
  if (!selfId.value || !connected.value || !activeMap.value) {
    return
  }

  const selfSnapshot = players.value.find((player) => player.id === selfId.value)
  if (!selfSnapshot) {
    return
  }
  if (!localSelf) {
    localSelf = { ...selfSnapshot }
  }

  const input = currentMovementInput()
  localSelf = simulatePlayer(localSelf, input, deltaSeconds)
  sendMove(false)
}

function syncLocalSelfFromSnapshot() {
  if (!selfId.value) {
    return
  }

  const selfSnapshot = players.value.find((player) => player.id === selfId.value)
  if (!selfSnapshot) {
    localSelf = null
    renderedSelf = null
    return
  }

  if (!localSelf) {
    localSelf = { ...selfSnapshot }
    return
  }

  localSelf.displayName = selfSnapshot.displayName
  localSelf.avatarId = selfSnapshot.avatarId
  localSelf.equipment = selfSnapshot.equipment ? { ...selfSnapshot.equipment } : undefined
  localSelf.lastProcessedSeq = selfSnapshot.lastProcessedSeq
}

function simulatePlayer(player: SunnyTownPlayer, input: MovementInput, deltaSeconds: number): SunnyTownPlayer {
  const map = activeMap.value
  if (!map) {
    return player
  }
  const result = { ...player }
  const vector = movementVector(input)
  if (!vector) {
    result.moving = false
    return result
  }

  const nextX = result.x + vector.x * playerSpeed * deltaSeconds
  const nextY = result.y + vector.y * playerSpeed * deltaSeconds
  if (!collides(map, nextX, result.y)) {
    result.x = clampPlayerX(map, nextX)
  }
  if (!collides(map, result.x, nextY)) {
    result.y = clampPlayerY(map, nextY)
  }
  result.facing = movementFacing(vector)
  result.moving = true
  return result
}

function currentMovementInput(): MovementInput {
  return {
    up: pressedDirections.has('up'),
    down: pressedDirections.has('down'),
    left: pressedDirections.has('left'),
    right: pressedDirections.has('right'),
  }
}

function movementVector(input: MovementInput): { x: number; y: number } | null {
  let x = Number(input.right) - Number(input.left)
  let y = Number(input.down) - Number(input.up)
  if (x === 0 && y === 0) {
    return null
  }

  const length = Math.hypot(x, y)
  x /= length
  y /= length
  return { x, y }
}

function movementFacing(vector: { x: number; y: number }): SunnyTownPlayer['facing'] {
  if (Math.abs(vector.x) > Math.abs(vector.y)) {
    return vector.x > 0 ? 'right' : 'left'
  }
  return vector.y > 0 ? 'down' : 'up'
}

function collides(map: SunnyTownMap, x: number, y: number): boolean {
  const playerRect = {
    x: x - playerSize / 2,
    y: y - playerSize / 2,
    width: playerSize,
    height: playerSize,
  }
  return map.blockedRects.some((blocked) => rectsOverlap(playerRect, blocked))
}

function clampPlayerX(map: SunnyTownMap, x: number): number {
  const maxX = map.width * map.tileSize - playerSize / 2
  return clamp(x, playerSize / 2, maxX)
}

function clampPlayerY(map: SunnyTownMap, y: number): number {
  const maxY = map.height * map.tileSize - playerSize / 2
  return clamp(y, playerSize / 2, maxY)
}

function rectsOverlap(
  first: { x: number; y: number; width: number; height: number },
  second: { x: number; y: number; width: number; height: number },
): boolean {
  return (
    first.x < second.x + second.width &&
    first.x + first.width > second.x &&
    first.y < second.y + second.height &&
    first.y + first.height > second.y
  )
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

function drawResourceNode(
  context: CanvasRenderingContext2D,
  node: SunnyTownResourceNode,
  cameraX: number,
  cameraY: number,
) {
  const x = node.x - cameraX
  const y = node.y - cameraY
  context.save()
  context.translate(x, y)
  context.fillStyle = node.active ? '#6b737b' : '#4f565d'
  context.strokeStyle = node.active ? '#343a40' : '#30343a'
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
  if (node.active) {
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
  const toolKey = player.equipment?.tool || ''
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
  if (!activeToolUse || activeToolUse.toolKey !== toolKey) {
    return null
  }
  const progress = (performance.now() - activeToolUse.startedAt) / activeToolUse.durationMs
  if (progress >= 1) {
    activeToolUse = null
    return null
  }
  return clamp(progress, 0, 1)
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

function movementDirectionForEvent(event: KeyboardEvent): MovementDirection | null {
  switch (event.code) {
    case 'ArrowUp':
    case 'KeyW':
      return 'up'
    case 'ArrowDown':
    case 'KeyS':
      return 'down'
    case 'ArrowLeft':
    case 'KeyA':
      return 'left'
    case 'ArrowRight':
    case 'KeyD':
      return 'right'
  }

  switch (event.key.toLowerCase()) {
    case 'arrowup':
    case 'w':
      return 'up'
    case 'arrowdown':
    case 's':
      return 'down'
    case 'arrowleft':
    case 'a':
      return 'left'
    case 'arrowright':
    case 'd':
      return 'right'
    default:
      return null
  }
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
  <section class="sunny-town-page">
    <div class="sunny-town-toolbar">
      <div>
        <p class="eyebrow">Pet</p>
        <h1>Sunny Town</h1>
      </div>
      <div class="sunny-town-toolbar__actions">
        <v-chip :color="connected ? 'success' : 'warning'" variant="tonal">
          {{ status }}
        </v-chip>
        <v-chip color="primary" variant="tonal">
          {{ playerCount }} online
        </v-chip>
        <v-chip color="warning" variant="tonal">
          {{ starBalance }} stars
        </v-chip>
        <v-btn color="primary" prepend-icon="mdi-arrow-left" variant="tonal" @click="backToPet">
          Back
        </v-btn>
      </div>
    </div>

    <v-alert v-if="error" class="mt-4" type="error" variant="tonal">
      {{ error }}
    </v-alert>

    <div class="sunny-town-stage">
      <canvas ref="canvas" aria-label="Sunny Town map" @pointerdown="handleCanvasPointerDown" />
      <div v-if="gameToast" class="sunny-town-toast" role="status">
        {{ gameToast }}
      </div>
      <div class="sunny-town-help">
        <v-icon icon="mdi-keyboard" size="small" />
          <span>Move with arrow keys or WASD - E inventory - F/click use tool</span>
      </div>
      <div v-if="nearbyNpc && !activeDialogueNpc && !activeSchoolworkNpc && !inventoryOpen" class="sunny-town-talk-hint">
        <v-icon icon="mdi-chat" size="small" />
        <span>F {{ nearbyNpc.name }}</span>
      </div>
      <div v-if="activeDialogueNpc" class="sunny-town-dialogue" role="dialog" :aria-label="activeDialogueNpc.name">
        <div class="sunny-town-dialogue__header">
          <strong>{{ activeDialogueNpc.name }}</strong>
          <span>{{ dialogueProgress }}</span>
        </div>
        <p>{{ activeDialogueLine }}</p>
        <div class="sunny-town-dialogue__footer">
          <span>F continue</span>
          <v-btn icon="mdi-close" size="x-small" variant="text" @click="closeDialogue" />
        </div>
      </div>
      <div v-if="activeShopNpc && !shopOpen" class="sunny-town-npc-menu" role="dialog" :aria-label="activeShopNpc.name">
        <strong>{{ activeShopNpc.name }}</strong>
        <p>{{ activeShopNpc.dialogue[0] }}</p>
        <div class="sunny-town-npc-menu__actions">
          <v-btn color="warning" prepend-icon="mdi-store" variant="flat" @click="openTrade">
            Trade
          </v-btn>
          <v-btn prepend-icon="mdi-close" variant="tonal" @click="closeNpcOverlays">
            Exit
          </v-btn>
        </div>
      </div>
      <div
        v-if="activeSchoolworkNpc && !schoolworkOpen"
        class="sunny-town-npc-menu"
        role="dialog"
        :aria-label="activeSchoolworkNpc.name"
      >
        <strong>{{ activeSchoolworkNpc.name }}</strong>
        <p>{{ activeSchoolworkNpc.dialogue[0] }}</p>
        <div class="sunny-town-npc-menu__actions">
          <v-btn color="primary" prepend-icon="mdi-school" variant="flat" @click="startSchoolwork">
            Do School Work
          </v-btn>
          <v-btn prepend-icon="mdi-close" variant="tonal" @click="closeNpcOverlays">
            Exit
          </v-btn>
        </div>
      </div>
      <div
        v-if="activeSchoolworkNpc && schoolworkOpen"
        class="sunny-town-schoolwork"
        role="dialog"
        :aria-label="`${activeSchoolworkNpc.name} school work`"
      >
        <div class="sunny-town-schoolwork__header">
          <div>
            <strong>{{ activeSchoolworkNpc.name }}</strong>
            <span>School Work</span>
          </div>
          <v-btn icon="mdi-close" size="x-small" variant="text" @click="closeNpcOverlays" />
        </div>
        <v-progress-linear
          v-if="isLoadingSchoolwork"
          class="mb-3"
          color="primary"
          indeterminate
        />
        <v-alert v-if="schoolworkError" class="mb-3" density="compact" type="error" variant="tonal">
          {{ schoolworkError }}
        </v-alert>
        <v-alert v-if="schoolworkNotice" class="mb-3" density="compact" type="success" variant="tonal">
          {{ schoolworkNotice }}
        </v-alert>
        <v-alert
          v-if="!isLoadingSchoolwork && !schoolworkAssignment && !schoolworkError"
          density="compact"
          type="success"
          variant="tonal"
        >
          You have finished all assignments.
        </v-alert>
        <v-form
          v-if="schoolworkAssignment"
          class="sunny-town-schoolwork__form"
          @submit.prevent="submitSchoolworkAnswer"
        >
          <div class="sunny-town-schoolwork__question">
            <v-chip color="primary" size="small" variant="tonal">
              {{ schoolworkAssignment.category }}
            </v-chip>
            <p>{{ schoolworkAssignment.prompt }}</p>
          </div>
          <v-textarea
            v-model="schoolworkAnswer"
            label="Your answer"
            rows="4"
            variant="outlined"
          />
          <v-btn
            :disabled="schoolworkAnswer.trim().length === 0"
            :loading="isSubmittingSchoolwork"
            color="primary"
            prepend-icon="mdi-send"
            type="submit"
            variant="flat"
          >
            Submit Answer
          </v-btn>
        </v-form>
      </div>
      <div v-if="activeShopNpc && shopOpen" class="sunny-town-shop" role="dialog" :aria-label="`${activeShopNpc.name} shop`">
        <div class="sunny-town-shop__header">
          <div>
            <strong>{{ activeShopNpc.name }}</strong>
            <span>{{ starBalance }} stars</span>
          </div>
          <v-btn icon="mdi-close" size="x-small" variant="text" @click="closeNpcOverlays" />
        </div>
        <v-alert v-if="shopError" class="mb-3" density="compact" type="error" variant="tonal">
          {{ shopError }}
        </v-alert>
        <v-alert v-if="shopNotice" class="mb-3" density="compact" type="success" variant="tonal">
          {{ shopNotice }}
        </v-alert>
        <div class="sunny-town-shop__columns">
          <section class="sunny-town-shop__column" aria-label="Your inventory">
            <h2>Your Inventory</h2>
            <div v-if="inventoryStore.items.length === 0" class="sunny-town-shop__empty">
              Nothing here yet.
            </div>
            <div v-for="item in inventoryStore.items" :key="item.key" class="sunny-town-shop__item">
              <span class="inventory-item__icon" :class="`inventory-item__icon--${item.key}`" aria-hidden="true" />
              <div>
                <p>{{ item.name }}</p>
                <small>{{ item.description }}</small>
              </div>
              <strong>{{ item.quantity }}</strong>
            </div>
          </section>
          <section class="sunny-town-shop__column" aria-label="Shop inventory">
            <h2>Shop Inventory</h2>
            <div v-for="item in shopItems" :key="item.itemKey" class="sunny-town-shop__item">
              <span class="inventory-item__icon" :class="`inventory-item__icon--${item.itemKey}`" aria-hidden="true" />
              <div>
                <p>{{ item.name }}</p>
                <small>{{ item.description }}</small>
                <small>{{ item.priceStars }} stars</small>
              </div>
              <v-btn
                color="warning"
                :disabled="starBalance < item.priceStars || isPurchasing"
                :loading="isPurchasing"
                size="small"
                variant="flat"
                @click="buyShopItem(item.itemKey)"
              >
                Buy
              </v-btn>
            </div>
          </section>
        </div>
      </div>
      <div v-if="inventoryOpen" class="sunny-town-inventory" role="dialog" aria-label="Inventory">
        <div class="sunny-town-inventory__header">
          <strong>Inventory</strong>
          <div class="sunny-town-inventory__actions">
            <v-btn
              :color="craftingPanelOpen ? 'warning' : undefined"
              :prepend-icon="craftingPanelOpen ? 'mdi-hammer-wrench' : 'mdi-hammer'"
              size="x-small"
              :variant="craftingPanelOpen ? 'flat' : 'tonal'"
              @click="toggleCraftingPanel"
            >
              Crafting
            </v-btn>
            <v-btn icon="mdi-close" size="x-small" variant="text" @click="inventoryOpen = false" />
          </div>
        </div>
        <v-alert v-if="inventoryStore.error" class="mb-3" density="compact" type="error" variant="tonal">
          {{ inventoryStore.error }}
        </v-alert>
        <section class="equipment-panel equipment-panel--dark" aria-label="Equipment">
          <div v-for="slot in inventoryStore.equipmentSlots" :key="slot.slot" class="equipment-slot">
            <div>
              <p class="summary-category">{{ slot.slot }}</p>
              <p class="inventory-item__name">{{ slot.item?.name || 'Empty' }}</p>
            </div>
            <v-btn
              v-if="slot.item"
              :loading="inventoryStore.isUpdatingEquipment"
              color="primary"
              size="x-small"
              variant="flat"
              @click="unequipInventorySlot(slot.slot)"
            >
              Unequip
            </v-btn>
          </div>
        </section>
        <div class="inventory-list inventory-list--compact">
          <div v-for="item in inventoryStore.unequippedItems" :key="item.key" class="inventory-item inventory-item--dark">
            <span class="inventory-item__icon" :class="`inventory-item__icon--${item.key}`" aria-hidden="true" />
            <div>
              <p class="inventory-item__name">{{ item.name }}</p>
              <p class="inventory-item__description">{{ item.description }}</p>
            </div>
            <v-btn
              v-if="item.equipSlot && !item.equipped"
              :loading="inventoryStore.isUpdatingEquipment"
              color="primary"
              size="x-small"
              variant="flat"
              @click="equipInventoryItem(item.key, item.equipSlot)"
            >
              Equip
            </v-btn>
            <strong v-else class="inventory-item__quantity">{{ item.quantity }}</strong>
          </div>
        </div>
        <section v-if="craftingPanelOpen" class="sunny-town-crafting" aria-label="Crafting">
          <div class="sunny-town-crafting__header">
            <strong>Crafting</strong>
            <v-switch
              v-model="showAllCraftingRecipes"
              color="warning"
              density="compact"
              hide-details
              inset
              label="All recipes"
            />
          </div>
          <v-progress-linear
            v-if="inventoryStore.isLoadingCrafting"
            class="mb-3"
            color="warning"
            indeterminate
          />
          <v-alert v-if="inventoryStore.craftingError" class="mb-3" density="compact" type="error" variant="tonal">
            {{ inventoryStore.craftingError }}
          </v-alert>
          <div v-if="visibleCraftingRecipes.length === 0 && !inventoryStore.isLoadingCrafting" class="sunny-town-crafting__empty">
            No recipes available.
          </div>
          <div
            v-for="recipe in visibleCraftingRecipes"
            :key="recipe.key"
            class="sunny-town-crafting__recipe"
            :class="{ 'sunny-town-crafting__recipe--disabled': !recipe.canCraft }"
          >
            <span class="inventory-item__icon" :class="`inventory-item__icon--${recipe.outputKey}`" aria-hidden="true" />
            <div>
              <p class="inventory-item__name">{{ recipe.name }}</p>
              <p class="inventory-item__description">{{ recipe.description }}</p>
              <div class="sunny-town-crafting__ingredients">
                <span
                  v-for="ingredient in recipe.ingredients"
                  :key="ingredient.itemKey"
                  :class="{ 'sunny-town-crafting__ingredient--missing': ingredient.owned < ingredient.required }"
                >
                  {{ ingredient.owned }}/{{ ingredient.required }} {{ ingredient.name }}
                </span>
              </div>
            </div>
            <v-btn
              :disabled="!recipe.canCraft || inventoryStore.isCrafting"
              :loading="inventoryStore.isCrafting"
              color="warning"
              size="x-small"
              variant="flat"
              @click="craftInventoryRecipe(recipe.key)"
            >
              Craft
            </v-btn>
          </div>
        </section>
      </div>
    </div>
  </section>
</template>
