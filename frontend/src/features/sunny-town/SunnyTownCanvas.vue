<script setup lang="ts">
import { onMounted, ref } from 'vue'

const emit = defineEmits<{
  pointerDown: [event: PointerEvent]
  pointerLeave: []
  pointerMove: [event: PointerEvent]
  ready: [canvas: HTMLCanvasElement]
}>()

const canvas = ref<HTMLCanvasElement | null>(null)

onMounted(() => {
  if (canvas.value) {
    emit('ready', canvas.value)
  }
})
</script>

<template>
  <canvas
    ref="canvas"
    aria-label="Sunny Town map"
    data-testid="sunny-town-canvas"
    @pointerdown="emit('pointerDown', $event)"
    @pointermove="emit('pointerMove', $event)"
    @pointerleave="emit('pointerLeave')"
  />
</template>
