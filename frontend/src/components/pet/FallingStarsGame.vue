<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { FallingObject, FallingStarsResult } from '../../types/pet'

const emit = defineEmits<{
  complete: [result: FallingStarsResult]
  quit: []
}>()

const targetScore = 10
const roundLength = 30
const playAreaRef = ref<HTMLElement | null>(null)
const gameState = ref<'idle' | 'countdown' | 'playing' | 'complete'>('idle')
const score = ref(0)
const starsCollected = ref(0)
const timeRemaining = ref(roundLength)
const petX = ref(50)
const fallingObjects = ref<FallingObject[]>([])
const countdownValue = ref(3)
const result = ref<FallingStarsResult | null>(null)
const feedback = ref('')

let animationFrame = 0
let lastFrameTime = 0
let spawnTimer = 0
let countdownTimer = 3
let movementDirection = 0
let objectId = 0
let feedbackTimer = 0
let completionEmitted = false
let roundId = ''

const scoreLabel = computed(() => `Score: ${score.value} / ${targetScore}`)
const hasActiveRound = computed(() => gameState.value === 'countdown' || gameState.value === 'playing')

function startGame() {
  stopLoop()
  roundId = createRoundId()
  score.value = 0
  starsCollected.value = 0
  timeRemaining.value = roundLength
  petX.value = 50
  fallingObjects.value = []
  countdownValue.value = 3
  countdownTimer = 3
  spawnTimer = 0
  movementDirection = 0
  result.value = null
  feedback.value = ''
  feedbackTimer = 0
  completionEmitted = false
  gameState.value = 'countdown'
  window.addEventListener('keydown', handleKeyDown)
  window.addEventListener('keyup', handleKeyUp)
  lastFrameTime = performance.now()
  animationFrame = window.requestAnimationFrame(tick)
}

function tick(timestamp: number) {
  const deltaSeconds = Math.min((timestamp - lastFrameTime) / 1000, 0.05)
  lastFrameTime = timestamp

  if (gameState.value === 'countdown') {
    countdownTimer -= deltaSeconds
    countdownValue.value = Math.max(1, Math.ceil(countdownTimer))

    if (countdownTimer <= 0) {
      gameState.value = 'playing'
      timeRemaining.value = roundLength
      spawnObject()
    }
  } else if (gameState.value === 'playing') {
    updatePlaying(deltaSeconds)
  }

  if (hasActiveRound.value) {
    animationFrame = window.requestAnimationFrame(tick)
  }
}

function updatePlaying(deltaSeconds: number) {
  timeRemaining.value = Math.max(0, timeRemaining.value - deltaSeconds)
  petX.value = clamp(petX.value + movementDirection * 58 * deltaSeconds, 6, 94)

  spawnTimer -= deltaSeconds
  if (spawnTimer <= 0) {
    spawnObject()
    spawnTimer = randomBetween(0.55, 0.9)
  }

  fallingObjects.value = fallingObjects.value
    .map((object) => ({
      ...object,
      y: object.y + object.speed * deltaSeconds,
    }))
    .filter((object) => {
      if (object.y > 104) {
        return false
      }

      if (isCaught(object)) {
        applyCatch(object)
        return false
      }

      return true
    })

  if (feedbackTimer > 0) {
    feedbackTimer -= deltaSeconds
    if (feedbackTimer <= 0) {
      feedback.value = ''
    }
  }

  if (timeRemaining.value <= 0) {
    completeGame()
  }
}

function spawnObject() {
  const type = Math.random() < 0.8 ? 'star' : 'badStar'
  fallingObjects.value.push({
    id: `${Date.now()}-${objectId++}`,
    type,
    x: randomBetween(7, 93),
    y: -8,
    speed: randomBetween(18, 31),
  })
}

function isCaught(object: FallingObject): boolean {
  const nearPetX = Math.abs(object.x - petX.value) <= 9
  return nearPetX && object.y >= 76 && object.y <= 92
}

function applyCatch(object: FallingObject) {
  if (object.type === 'star') {
    score.value += 1
    starsCollected.value += 1
    feedback.value = 'sparkle'
  } else {
    score.value = Math.max(0, score.value - 1)
    feedback.value = 'bad'
  }
  feedbackTimer = 0.22
}

function completeGame() {
  stopLoop()
  gameState.value = 'complete'
  movementDirection = 0
  timeRemaining.value = 0
  fallingObjects.value = []

  const won = score.value >= targetScore
  const gameResult = {
    roundId,
    score: score.value,
    starsCollected: starsCollected.value,
    won,
    happinessDelta: won ? Math.min(score.value * 2, 20) : Math.min(score.value, 8),
    energyDelta: -5,
  }
  result.value = gameResult

  if (!completionEmitted) {
    completionEmitted = true
    emit('complete', gameResult)
  }
}

function quitGame() {
  stopLoop()
  movementDirection = 0
  fallingObjects.value = []
  gameState.value = 'idle'

  if (!completionEmitted) {
    emit('quit')
  }
}

function handleKeyDown(event: KeyboardEvent) {
  if (!hasActiveRound.value) {
    return
  }

  if (event.key === 'ArrowLeft' || event.key.toLowerCase() === 'a') {
    movementDirection = -1
    event.preventDefault()
  }

  if (event.key === 'ArrowRight' || event.key.toLowerCase() === 'd') {
    movementDirection = 1
    event.preventDefault()
  }
}

function handleKeyUp(event: KeyboardEvent) {
  if (
    event.key === 'ArrowLeft' ||
    event.key === 'ArrowRight' ||
    event.key.toLowerCase() === 'a' ||
    event.key.toLowerCase() === 'd'
  ) {
    movementDirection = 0
  }
}

function handlePointerDown(event: PointerEvent) {
  if (!hasActiveRound.value || !playAreaRef.value) {
    return
  }

  const bounds = playAreaRef.value.getBoundingClientRect()
  movementDirection = event.clientX < bounds.left + bounds.width / 2 ? -1 : 1
}

function handlePointerMove(event: PointerEvent) {
  if (movementDirection === 0 || !hasActiveRound.value || !playAreaRef.value) {
    return
  }

  const bounds = playAreaRef.value.getBoundingClientRect()
  movementDirection = event.clientX < bounds.left + bounds.width / 2 ? -1 : 1
}

function stopMoving() {
  movementDirection = 0
}

function stopLoop() {
  if (animationFrame) {
    window.cancelAnimationFrame(animationFrame)
    animationFrame = 0
  }
  window.removeEventListener('keydown', handleKeyDown)
  window.removeEventListener('keyup', handleKeyUp)
}

function randomBetween(min: number, max: number): number {
  return min + Math.random() * (max - min)
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}

function createRoundId(): string {
  if (window.crypto?.randomUUID) {
    return window.crypto.randomUUID()
  }

  if (!window.crypto?.getRandomValues) {
    return `${Date.now()}-${Math.random().toString(16).slice(2)}`
  }

  const random = new Uint32Array(4)
  window.crypto.getRandomValues(random)
  return Array.from(random, (value) => value.toString(16).padStart(8, '0')).join('')
}

onBeforeUnmount(() => {
  stopLoop()
})

onMounted(() => {
  startGame()
})
</script>

<template>
  <section class="falling-stars-game" aria-label="Falling Stars mini-game">
    <div class="falling-stars-game__header">
      <div>
        <p class="summary-category">Mini-game</p>
        <h2>Falling Stars</h2>
      </div>
      <div class="falling-stars-game__stats" aria-live="polite">
        <strong>{{ scoreLabel }}</strong>
        <span>Time: {{ Math.ceil(timeRemaining) }}s</span>
      </div>
      <v-btn
        color="secondary"
        prepend-icon="mdi-close"
        variant="tonal"
        @click.stop="quitGame"
      >
        Quit
      </v-btn>
    </div>

    <div
      ref="playAreaRef"
      class="falling-stars-game__area"
      :class="{
        'falling-stars-game__area--bad': feedback === 'bad',
        'falling-stars-game__area--complete': gameState === 'complete',
      }"
      tabindex="0"
      @pointerdown.prevent="handlePointerDown"
      @pointermove.prevent="handlePointerMove"
      @pointerup="stopMoving"
      @pointercancel="stopMoving"
      @pointerleave="stopMoving"
    >
      <div class="falling-stars-game__zone falling-stars-game__zone--left" aria-hidden="true" />
      <div class="falling-stars-game__zone falling-stars-game__zone--right" aria-hidden="true" />

      <span
        v-for="object in fallingObjects"
        :key="object.id"
        class="falling-stars-game__object"
        :class="`falling-stars-game__object--${object.type}`"
        :style="{ left: `${object.x}%`, top: `${object.y}%` }"
        aria-hidden="true"
      >
        {{ object.type === 'star' ? '⭐' : '💥' }}
      </span>

      <div
        class="falling-stars-game__pet"
        :class="{ 'falling-stars-game__pet--hit': feedback === 'bad' }"
        :style="{ left: `${petX}%` }"
        aria-hidden="true"
      >
        <span class="falling-stars-game__pet-face">●‿●</span>
      </div>

      <div v-if="feedback === 'sparkle'" class="falling-stars-game__sparkle" aria-hidden="true">
        +1
      </div>

      <div v-if="gameState === 'countdown'" class="falling-stars-game__overlay">
        <strong class="falling-stars-game__countdown">{{ countdownValue }}</strong>
      </div>
    </div>
  </section>
</template>

<style scoped>
.falling-stars-game {
  background: #ffffff;
  border: 1px solid #d9e1e4;
  border-radius: 8px;
  display: grid;
  gap: 14px;
  padding: 16px;
  width: min(100%, 980px);
}

.falling-stars-game__header {
  align-items: center;
  display: flex;
  gap: 12px;
  justify-content: space-between;
}

.falling-stars-game__stats {
  color: #46535f;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: flex-end;
}

.falling-stars-game__stats strong {
  color: #17212b;
}

.falling-stars-game__area {
  background:
    radial-gradient(circle at 12% 18%, rgba(255, 255, 255, 0.9) 0 2px, transparent 3px),
    radial-gradient(circle at 76% 12%, rgba(255, 255, 255, 0.8) 0 2px, transparent 3px),
    linear-gradient(180deg, #1f3959 0%, #315977 58%, #d8f1f0 100%);
  border: 2px solid #17212b;
  border-radius: 8px;
  height: min(72vh, 620px);
  min-height: 320px;
  overflow: hidden;
  position: relative;
  touch-action: none;
}

.falling-stars-game__area--bad {
  animation: game-flash 0.2s ease-in-out;
}

.falling-stars-game__area--complete {
  background:
    radial-gradient(circle at 22% 18%, rgba(255, 255, 255, 0.9) 0 2px, transparent 3px),
    linear-gradient(180deg, #28486b 0%, #5b8799 62%, #d8f1f0 100%);
}

.falling-stars-game__zone {
  bottom: 0;
  opacity: 0.08;
  position: absolute;
  top: 0;
  width: 50%;
}

.falling-stars-game__zone--left {
  background: #ffffff;
  left: 0;
}

.falling-stars-game__zone--right {
  background: #f0b22b;
  right: 0;
}

.falling-stars-game__object {
  font-size: 30px;
  line-height: 1;
  position: absolute;
  transform: translate(-50%, -50%);
  user-select: none;
  z-index: 2;
}

.falling-stars-game__object--badStar {
  filter: drop-shadow(0 0 4px rgba(159, 45, 45, 0.85));
}

.falling-stars-game__pet {
  align-items: center;
  background: linear-gradient(145deg, #6bcf9f, #48a97d);
  border: 3px solid #17212b;
  border-radius: 48% 52% 42% 46%;
  bottom: 18px;
  box-shadow: 0 8px 0 rgba(23, 33, 43, 0.12) inset;
  color: #17212b;
  display: flex;
  height: 68px;
  justify-content: center;
  position: absolute;
  transform: translateX(-50%);
  width: 78px;
  z-index: 3;
}

.falling-stars-game__pet::after {
  background: #f9f2cf;
  border: 2px solid rgba(23, 33, 43, 0.6);
  border-radius: 50%;
  bottom: 8px;
  content: '';
  height: 18px;
  position: absolute;
  width: 24px;
}

.falling-stars-game__pet-face {
  font-size: 16px;
  font-weight: 900;
  margin-bottom: 18px;
  position: relative;
  z-index: 1;
}

.falling-stars-game__pet--hit {
  animation: pet-hit 0.2s ease-in-out;
}

.falling-stars-game__sparkle {
  bottom: 102px;
  color: #fff8df;
  font-size: 22px;
  font-weight: 900;
  left: 50%;
  position: absolute;
  text-shadow: 0 2px 4px rgba(23, 33, 43, 0.45);
  transform: translateX(-50%);
  z-index: 4;
}

.falling-stars-game__overlay {
  align-items: center;
  background: rgba(247, 250, 249, 0.9);
  color: #17212b;
  display: flex;
  flex-direction: column;
  gap: 10px;
  inset: 0;
  justify-content: center;
  padding: 20px;
  position: absolute;
  text-align: center;
  z-index: 5;
}

.falling-stars-game__overlay h3,
.falling-stars-game__overlay p {
  margin: 0;
}

.falling-stars-game__overlay p {
  max-width: 420px;
}

.falling-stars-game__countdown {
  font-size: 72px;
  line-height: 1;
}

@keyframes pet-hit {
  0%,
  100% {
    transform: translateX(-50%);
  }

  50% {
    transform: translateX(-50%) rotate(8deg);
  }
}

@keyframes game-flash {
  50% {
    border-color: #9f2d2d;
    box-shadow: inset 0 0 0 999px rgba(159, 45, 45, 0.12);
  }
}

@media (max-width: 640px) {
  .falling-stars-game__header {
    align-items: flex-start;
    flex-direction: column;
  }

  .falling-stars-game__stats {
    justify-content: flex-start;
  }

  .falling-stars-game__area {
    min-height: 300px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .falling-stars-game__area--bad,
  .falling-stars-game__pet--hit {
    animation: none;
  }
}
</style>
