<script setup lang="ts">
import type { SunnyTownNpc } from '../../types/sunnyTown'

defineProps<{
  line: string
  npc: SunnyTownNpc
  progress: string
}>()

defineEmits<{
  close: []
}>()
</script>

<template>
  <div class="sunny-town-dialogue" role="dialog" :aria-label="npc.name">
    <div class="sunny-town-dialogue__header">
      <strong>{{ npc.name }}</strong>
      <span>{{ progress }}</span>
    </div>
    <p>{{ line }}</p>
    <div class="sunny-town-dialogue__footer">
      <span>F continue</span>
      <v-btn icon="mdi-close" size="x-small" variant="text" @click="$emit('close')" />
    </div>
  </div>
</template>
