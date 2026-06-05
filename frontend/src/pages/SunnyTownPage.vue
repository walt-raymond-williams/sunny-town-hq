<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createSunnyTownSession } from '../api/sunnyTownApi'
import type {
  SunnyTownCollectible,
  SunnyTownMoveMessage,
  SunnyTownPlayer,
  SunnyTownServerMessage,
  SunnyTownSession,
} from '../types/sunnyTown'

interface SunnyTownMap {
  tileSize: number
  width: number
  height: number
  blockedRects: Array<{ x: number; y: number; width: number; height: number }>
}

interface MovementInput {
  up: boolean
  down: boolean
  left: boolean
  right: boolean
}

type MovementDirection = keyof MovementInput

const sunnyTownMap: SunnyTownMap = {
  tileSize: 32,
  width: 40,
  height: 30,
  blockedRects: [
    { x: 0, y: 0, width: 1280, height: 32 },
    { x: 0, y: 928, width: 1280, height: 32 },
    { x: 0, y: 0, width: 32, height: 960 },
    { x: 1248, y: 0, width: 32, height: 960 },
    { x: 160, y: 128, width: 224, height: 160 },
    { x: 832, y: 128, width: 224, height: 160 },
    { x: 512, y: 672, width: 256, height: 96 },
  ],
}

const playerSpeed = 150
const playerSize = 28
const moveSendIntervalMs = 50
const remoteInterpolationDelayMs = 150
const maxRemoteHistoryFrames = 12
const movementInputEventOptions = { capture: true }

const router = useRouter()
const canvas = ref<HTMLCanvasElement | null>(null)
const session = ref<SunnyTownSession | null>(null)
const players = ref<SunnyTownPlayer[]>([])
const collectibles = ref<SunnyTownCollectible[]>([])
const selfId = ref('')
const status = ref('Entering Sunny Town...')
const error = ref('')
const connected = ref(false)
const starBalance = ref(0)
const gameToast = ref('')

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

const playerCount = computed(() => players.value.length)

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
  window.removeEventListener('keydown', handleKeyDown, movementInputEventOptions)
  window.removeEventListener('keyup', handleKeyUp, movementInputEventOptions)
  window.removeEventListener('blur', handleInputCancel)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  window.removeEventListener('resize', handleResize)
  window.cancelAnimationFrame(animationFrame)
  socket?.close()
  socket = null
  localSelf = null
  renderedSelf = null
  remotePlayerHistories.clear()
})

function connect(activeSession: SunnyTownSession) {
  const url = new URL(activeSession.websocketUrl)
  url.searchParams.set('token', activeSession.joinToken)
  socket = new WebSocket(url.toString())

  socket.addEventListener('open', () => {
    connected.value = true
    status.value = 'Connected'
  })

  socket.addEventListener('message', (event) => {
    const message = parseServerMessage(event.data)
    if (!message) {
      return
    }
    if (message.type === 'hello') {
      selfId.value = message.selfId || ''
      return
    }
    if (message.type === 'snapshot') {
      players.value = message.players || []
      collectibles.value = message.collectibles || []
      recordRemoteSnapshots(players.value, message.serverTimeMs || Date.now())
      syncLocalSelfFromSnapshot()
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
    if (message.type === 'error') {
      error.value = message.code || 'Sunny Town received an invalid message'
    }
  })

  socket.addEventListener('close', () => {
    connected.value = false
    status.value = 'Disconnected'
  })

  socket.addEventListener('error', () => {
    error.value = 'Sunny Town connection failed'
  })
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

  const renderedPlayers = renderedSunnyTownPlayers()
  const self = renderedPlayers.find((player) => player.id === selfId.value) || renderedPlayers[0]
  const worldWidth = sunnyTownMap.width * sunnyTownMap.tileSize
  const worldHeight = sunnyTownMap.height * sunnyTownMap.tileSize
  const cameraX = clamp((self?.x || worldWidth / 2) - rect.width / 2, 0, Math.max(0, worldWidth - rect.width))
  const cameraY = clamp((self?.y || worldHeight / 2) - rect.height / 2, 0, Math.max(0, worldHeight - rect.height))

  drawMap(context, cameraX, cameraY, rect.width, rect.height)
  for (const collectible of collectibles.value) {
    if (collectible.active) {
      drawCollectible(context, collectible, cameraX, cameraY)
    }
  }
  for (const player of renderedPlayers) {
    drawPlayer(context, player, cameraX, cameraY)
  }
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
  if (!selfId.value || !connected.value) {
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
  localSelf.lastProcessedSeq = selfSnapshot.lastProcessedSeq
}

function simulatePlayer(player: SunnyTownPlayer, input: MovementInput, deltaSeconds: number): SunnyTownPlayer {
  const result = { ...player }
  const vector = movementVector(input)
  if (!vector) {
    result.moving = false
    return result
  }

  const nextX = result.x + vector.x * playerSpeed * deltaSeconds
  const nextY = result.y + vector.y * playerSpeed * deltaSeconds
  if (!collides(nextX, result.y)) {
    result.x = clampPlayerX(nextX)
  }
  if (!collides(result.x, nextY)) {
    result.y = clampPlayerY(nextY)
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

function collides(x: number, y: number): boolean {
  const playerRect = {
    x: x - playerSize / 2,
    y: y - playerSize / 2,
    width: playerSize,
    height: playerSize,
  }
  return sunnyTownMap.blockedRects.some((blocked) => rectsOverlap(playerRect, blocked))
}

function clampPlayerX(x: number): number {
  const maxX = sunnyTownMap.width * sunnyTownMap.tileSize - playerSize / 2
  return clamp(x, playerSize / 2, maxX)
}

function clampPlayerY(y: number): number {
  const maxY = sunnyTownMap.height * sunnyTownMap.tileSize - playerSize / 2
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

function drawMap(context: CanvasRenderingContext2D, cameraX: number, cameraY: number, width: number, height: number) {
  context.fillStyle = '#8fcf85'
  context.fillRect(0, 0, width, height)

  context.fillStyle = '#d6bd79'
  context.fillRect(0 - cameraX, 420 - cameraY, sunnyTownMap.width * sunnyTownMap.tileSize, 124)
  context.fillRect(570 - cameraX, 0 - cameraY, 140, sunnyTownMap.height * sunnyTownMap.tileSize)

  context.strokeStyle = 'rgba(255, 255, 255, 0.18)'
  context.lineWidth = 1
  for (let x = -cameraX % sunnyTownMap.tileSize; x < width; x += sunnyTownMap.tileSize) {
    context.beginPath()
    context.moveTo(x, 0)
    context.lineTo(x, height)
    context.stroke()
  }
  for (let y = -cameraY % sunnyTownMap.tileSize; y < height; y += sunnyTownMap.tileSize) {
    context.beginPath()
    context.moveTo(0, y)
    context.lineTo(width, y)
    context.stroke()
  }

  for (const blocked of sunnyTownMap.blockedRects) {
    context.fillStyle = blocked.width > 400 || blocked.height > 400 ? '#4f8a5b' : '#7e6b52'
    context.fillRect(blocked.x - cameraX, blocked.y - cameraY, blocked.width, blocked.height)
  }
}

function drawPlayer(context: CanvasRenderingContext2D, player: SunnyTownPlayer, cameraX: number, cameraY: number) {
  const x = player.x - cameraX
  const y = player.y - cameraY
  const isSelf = player.id === selfId.value

  context.fillStyle = 'rgba(0, 0, 0, 0.18)'
  context.beginPath()
  context.ellipse(x, y + 16, 18, 7, 0, 0, Math.PI * 2)
  context.fill()

  context.fillStyle = isSelf ? '#27746f' : '#5c6bc0'
  context.beginPath()
  context.arc(x, y, 16, 0, Math.PI * 2)
  context.fill()

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
      <canvas ref="canvas" aria-label="Sunny Town map" />
      <div v-if="gameToast" class="sunny-town-toast" role="status">
        {{ gameToast }}
      </div>
      <div class="sunny-town-help">
        <v-icon icon="mdi-keyboard" size="small" />
        <span>Move with arrow keys or WASD</span>
      </div>
    </div>
  </section>
</template>
