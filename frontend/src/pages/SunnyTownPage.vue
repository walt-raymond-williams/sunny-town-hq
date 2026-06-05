<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createSunnyTownSession } from '../api/sunnyTownApi'
import type {
  SunnyTownInputMessage,
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

interface PendingInput {
  seq: number
  input: MovementInput
  deltaSeconds: number
}

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
const tinyCorrectionDistance = 4
const snapCorrectionDistance = 48

const router = useRouter()
const canvas = ref<HTMLCanvasElement | null>(null)
const session = ref<SunnyTownSession | null>(null)
const players = ref<SunnyTownPlayer[]>([])
const selfId = ref('')
const status = ref('Entering Sunny Town...')
const error = ref('')
const connected = ref(false)

const pressedKeys = new Set<string>()
const visualPlayers = new Map<string, SunnyTownPlayer>()
const pendingInputs: PendingInput[] = []
let predictedSelf: SunnyTownPlayer | null = null
let renderedSelf: SunnyTownPlayer | null = null
let socket: WebSocket | null = null
let animationFrame = 0
let inputSeq = 0
let inputTimer = 0
let lastInputStateJson = ''
let lastPredictedInputJson = ''
let lastRenderTime = 0

const playerCount = computed(() => players.value.length)

onMounted(async () => {
  window.addEventListener('keydown', handleKeyDown)
  window.addEventListener('keyup', handleKeyUp)
  window.addEventListener('resize', handleResize)
  animationFrame = window.requestAnimationFrame(renderLoop)

  try {
    session.value = await createSunnyTownSession()
    status.value = 'Connecting...'
    await nextTick()
    connect(session.value)
    inputTimer = window.setInterval(sendInput, 75)
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : String(caught)
    status.value = 'Could not enter Sunny Town'
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeyDown)
  window.removeEventListener('keyup', handleKeyUp)
  window.removeEventListener('resize', handleResize)
  window.cancelAnimationFrame(animationFrame)
  window.clearInterval(inputTimer)
  socket?.close()
  socket = null
  pendingInputs.splice(0)
  predictedSelf = null
  renderedSelf = null
  visualPlayers.clear()
})

function connect(activeSession: SunnyTownSession) {
  const url = new URL(activeSession.websocketUrl)
  url.searchParams.set('token', activeSession.joinToken)
  socket = new WebSocket(url.toString())

  socket.addEventListener('open', () => {
    connected.value = true
    status.value = 'Connected'
    sendInput()
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
      reconcileSelf()
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
  if (!isMoveKey(event.key)) {
    return
  }
  event.preventDefault()
  pressedKeys.add(event.key.toLowerCase())
  sendInput()
}

function handleKeyUp(event: KeyboardEvent) {
  if (!isMoveKey(event.key)) {
    return
  }
  event.preventDefault()
  pressedKeys.delete(event.key.toLowerCase())
  sendInput()
}

function handleResize() {
  draw()
}

function sendInput() {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    return
  }

  const movementInput = currentMovementInput()
  const input: SunnyTownInputMessage = {
    type: 'input',
    seq: ++inputSeq,
    ...movementInput,
  }
  const inputStateJson = JSON.stringify(movementInput)
  const isMoving = isMovementInputActive(movementInput)
  if (!isMoving && inputStateJson === lastInputStateJson) {
    return
  }
  lastInputStateJson = inputStateJson
  pendingInputs.push({
    seq: input.seq,
    input: movementInput,
    deltaSeconds: 0,
  })
  socket.send(JSON.stringify(input))
}

function renderLoop() {
  const now = performance.now()
  const deltaSeconds = lastRenderTime ? (now - lastRenderTime) / 1000 : 0
  lastRenderTime = now
  predictSelf(deltaSeconds)
  draw(deltaSeconds)
  animationFrame = window.requestAnimationFrame(renderLoop)
}

function draw(deltaSeconds = 0) {
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

  const renderedPlayers = renderedSunnyTownPlayers(deltaSeconds)
  const self = renderedPlayers.find((player) => player.id === selfId.value) || renderedPlayers[0]
  const worldWidth = sunnyTownMap.width * sunnyTownMap.tileSize
  const worldHeight = sunnyTownMap.height * sunnyTownMap.tileSize
  const cameraX = clamp((self?.x || worldWidth / 2) - rect.width / 2, 0, Math.max(0, worldWidth - rect.width))
  const cameraY = clamp((self?.y || worldHeight / 2) - rect.height / 2, 0, Math.max(0, worldHeight - rect.height))

  drawMap(context, cameraX, cameraY, rect.width, rect.height)
  for (const player of renderedPlayers) {
    drawPlayer(context, player, cameraX, cameraY)
  }
}

function renderedSunnyTownPlayers(deltaSeconds: number): SunnyTownPlayer[] {
  const selfSnapshot = players.value.find((player) => player.id === selfId.value)
  const remotePlayers = players.value.filter((player) => player.id !== selfId.value)
  const renderedRemotePlayers = smoothRemotePlayers(remotePlayers, deltaSeconds)
  if (!selfSnapshot) {
    return renderedRemotePlayers
  }

  if (!renderedSelf) {
    renderedSelf = { ...(predictedSelf || selfSnapshot) }
  }

  const target = predictedSelf || selfSnapshot
  const correctionDistance = Math.hypot(target.x - renderedSelf.x, target.y - renderedSelf.y)
  if (correctionDistance <= tinyCorrectionDistance) {
    renderedSelf.x = target.x
    renderedSelf.y = target.y
  } else if (correctionDistance > snapCorrectionDistance) {
    renderedSelf.x = target.x
    renderedSelf.y = target.y
  } else {
    const smoothing = 1 - Math.exp(-24 * Math.max(0, deltaSeconds))
    renderedSelf.x += (target.x - renderedSelf.x) * smoothing
    renderedSelf.y += (target.y - renderedSelf.y) * smoothing
  }
  renderedSelf.displayName = target.displayName
  renderedSelf.facing = target.facing
  renderedSelf.moving = target.moving
  renderedSelf.avatarId = target.avatarId
  renderedSelf.lastProcessedSeq = target.lastProcessedSeq

  return [...renderedRemotePlayers, renderedSelf]
}

function smoothRemotePlayers(remotePlayers: SunnyTownPlayer[], deltaSeconds: number): SunnyTownPlayer[] {
  const targets = new Map(remotePlayers.map((player) => [player.id, player]))
  for (const playerId of visualPlayers.keys()) {
    if (!targets.has(playerId)) {
      visualPlayers.delete(playerId)
    }
  }

  const smoothing = 1 - Math.exp(-18 * Math.max(0, deltaSeconds))
  for (const target of remotePlayers) {
    const visual = visualPlayers.get(target.id)
    if (!visual) {
      visualPlayers.set(target.id, { ...target })
      continue
    }

    const distance = Math.hypot(target.x - visual.x, target.y - visual.y)
    const snapDistance = 160
    visual.x = distance > snapDistance ? target.x : visual.x + (target.x - visual.x) * smoothing
    visual.y = distance > snapDistance ? target.y : visual.y + (target.y - visual.y) * smoothing
    visual.displayName = target.displayName
    visual.facing = target.facing
    visual.moving = target.moving
    visual.avatarId = target.avatarId
  }

  return Array.from(visualPlayers.values())
}

function predictSelf(deltaSeconds: number) {
  if (!selfId.value || !connected.value) {
    return
  }

  const selfSnapshot = players.value.find((player) => player.id === selfId.value)
  if (!selfSnapshot) {
    return
  }
  if (!predictedSelf) {
    predictedSelf = { ...selfSnapshot }
  }

  const input = currentMovementInput()
  const inputStateJson = JSON.stringify(input)
  if (isMovementInputActive(input)) {
    predictedSelf = simulatePlayer(predictedSelf, input, deltaSeconds)
    const latestPendingInput = pendingInputs[pendingInputs.length - 1]
    if (latestPendingInput) {
      latestPendingInput.deltaSeconds += deltaSeconds
    }
    if (inputStateJson !== lastPredictedInputJson) {
      sendInput()
    }
  }
  lastPredictedInputJson = inputStateJson
}

function reconcileSelf() {
  if (!selfId.value) {
    return
  }

  const selfSnapshot = players.value.find((player) => player.id === selfId.value)
  if (!selfSnapshot) {
    predictedSelf = null
    renderedSelf = null
    return
  }

  const firstUnackedIndex = pendingInputs.findIndex((input) => input.seq > selfSnapshot.lastProcessedSeq)
  if (firstUnackedIndex === -1) {
    pendingInputs.splice(0)
  } else if (firstUnackedIndex > 0) {
    pendingInputs.splice(0, firstUnackedIndex)
  }

  let corrected = { ...selfSnapshot }
  for (const pendingInput of pendingInputs) {
    corrected = simulatePlayer(corrected, pendingInput.input, pendingInput.deltaSeconds)
  }
  predictedSelf = corrected

  if (!renderedSelf) {
    renderedSelf = { ...corrected }
  }
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
    up: pressedKeys.has('arrowup') || pressedKeys.has('w'),
    down: pressedKeys.has('arrowdown') || pressedKeys.has('s'),
    left: pressedKeys.has('arrowleft') || pressedKeys.has('a'),
    right: pressedKeys.has('arrowright') || pressedKeys.has('d'),
  }
}

function isMovementInputActive(input: MovementInput): boolean {
  return input.up || input.down || input.left || input.right
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

function isMoveKey(key: string): boolean {
  return ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'w', 'a', 's', 'd', 'W', 'A', 'S', 'D'].includes(key)
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
      <div class="sunny-town-help">
        <v-icon icon="mdi-keyboard" size="small" />
        <span>Move with arrow keys or WASD</span>
      </div>
    </div>
  </section>
</template>
