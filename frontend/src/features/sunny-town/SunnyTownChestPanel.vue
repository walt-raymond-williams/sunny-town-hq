<script setup lang="ts">
import type { SunnyTownWorldObject } from '../../types/sunnyTown'

defineProps<{
  chest: SunnyTownWorldObject
  error: string
  isLoading: boolean
  itemKey: string
  quantity: number
  capacity: number
}>()

defineEmits<{
  close: []
}>()
</script>

<template>
  <div class="sunny-town-chest-panel" data-testid="sunny-town-chest-panel" role="dialog" :aria-label="chest.name || 'Storage chest'">
    <div class="sunny-town-chest-panel__header">
      <div>
        <strong>{{ chest.name || 'Storage Chest' }}</strong>
        <span>Output storage</span>
      </div>
      <v-btn icon="mdi-close" size="x-small" variant="text" @click="$emit('close')" />
    </div>
    <v-alert v-if="error" class="mb-3" density="compact" type="error" variant="tonal">
      {{ error }}
    </v-alert>
    <div v-if="isLoading" class="sunny-town-chest-panel__empty">
      Checking stock...
    </div>
    <div v-else class="sunny-town-chest-panel__item" :data-testid="`chest-item-${itemKey}`">
      <span class="inventory-item__icon" :class="`inventory-item__icon--${itemKey}`" aria-hidden="true" />
      <div>
        <p>{{ itemKey || 'Stored item' }}</p>
        <small>{{ quantity }} / {{ capacity }} stored</small>
      </div>
    </div>
  </div>
</template>
