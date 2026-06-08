<script setup lang="ts">
import type { HotbarSlot } from '../../types/inventory'

defineProps<{
  connected: boolean
  error: string
  gameToast: string
  hotbarSlots: HotbarSlot[]
  playerCount: number
  selectedHotbarIndex: number
  starBalance: number
  status: string
}>()

defineEmits<{
  back: []
  selectHotbarSlot: [index: number]
}>()
</script>

<template>
  <section class="sunny-town-page" data-testid="sunny-town-page">
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
        <v-btn color="primary" prepend-icon="mdi-arrow-left" variant="tonal" @click="$emit('back')">
          Back
        </v-btn>
      </div>
    </div>

    <v-alert v-if="error" class="mt-4" type="error" variant="tonal">
      {{ error }}
    </v-alert>

    <div class="sunny-town-stage">
      <slot name="canvas" />
      <div v-if="gameToast" class="sunny-town-toast" role="status">
        {{ gameToast }}
      </div>
      <div class="sunny-town-help">
        <v-icon icon="mdi-keyboard" size="small" />
        <span>Move with arrow keys or WASD - 1-5 select - E inventory - F/click use</span>
      </div>
      <div class="sunny-town-hotbar" aria-label="Hotbar">
        <button
          v-for="(slot, index) in hotbarSlots"
          :key="slot.slot"
          class="sunny-town-hotbar__slot"
          :class="{
            'sunny-town-hotbar__slot--selected': selectedHotbarIndex === index,
            'sunny-town-hotbar__slot--empty-item': slot.item && slot.item.quantity < 1,
          }"
          type="button"
          @click="$emit('selectHotbarSlot', index)"
        >
          <span class="sunny-town-hotbar__number">{{ slot.slot }}</span>
          <span
            v-if="slot.item"
            class="inventory-item__icon"
            :class="`inventory-item__icon--${slot.item.key}`"
            aria-hidden="true"
          />
          <span v-else class="sunny-town-hotbar__empty" aria-hidden="true" />
          <strong v-if="slot.item && (slot.item.quantity > 1 || slot.item.quantity < 1)" class="sunny-town-hotbar__quantity">
            {{ slot.item.quantity }}
          </strong>
        </button>
      </div>
      <slot />
    </div>
  </section>
</template>
