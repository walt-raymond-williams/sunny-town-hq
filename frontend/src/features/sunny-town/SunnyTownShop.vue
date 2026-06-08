<script setup lang="ts">
import { useStudentInventoryStore } from '../../stores/studentInventory'
import type { SunnyTownNpc } from '../../types/sunnyTown'

defineProps<{
  isPurchasing: boolean
  npc: SunnyTownNpc
  open: boolean
  shopError: string
  shopNotice: string
  shopStock: Record<string, number>
  shopStockCapacity: Record<string, number>
  starBalance: number
}>()

defineEmits<{
  buy: [itemKey: string]
  close: []
  openTrade: []
}>()

const inventoryStore = useStudentInventoryStore()
</script>

<template>
  <div v-if="!open" class="sunny-town-npc-menu" data-testid="sunny-town-npc-menu" role="dialog" :aria-label="npc.name">
    <strong>{{ npc.name }}</strong>
    <p>{{ npc.dialogue[0] }}</p>
    <div class="sunny-town-npc-menu__actions">
      <v-btn color="warning" data-testid="sunny-town-open-shop-button" prepend-icon="mdi-store" variant="flat" @click="$emit('openTrade')">
        Trade
      </v-btn>
      <v-btn prepend-icon="mdi-close" variant="tonal" @click="$emit('close')">
        Exit
      </v-btn>
    </div>
  </div>
  <div v-else class="sunny-town-shop" data-testid="sunny-town-shop-panel" role="dialog" :aria-label="`${npc.name} shop`">
    <div class="sunny-town-shop__header">
      <div>
        <strong>{{ npc.name }}</strong>
        <span>{{ starBalance }} stars</span>
      </div>
      <v-btn icon="mdi-close" size="x-small" variant="text" @click="$emit('close')" />
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
        <div v-for="item in npc.shop?.items || []" :key="item.itemKey" class="sunny-town-shop__item" :data-testid="`shop-item-${item.itemKey}`">
          <span class="inventory-item__icon" :class="`inventory-item__icon--${item.itemKey}`" aria-hidden="true" />
          <div>
            <p>{{ item.name }}</p>
            <small>{{ item.description }}</small>
            <small>{{ item.priceStars }} stars</small>
            <small>{{ shopStock[item.itemKey] ?? 0 }} / {{ shopStockCapacity[item.itemKey] ?? 0 }} in stock</small>
          </div>
          <v-btn
            color="warning"
            :disabled="starBalance < item.priceStars || (shopStock[item.itemKey] ?? 0) < 1 || isPurchasing"
            :loading="isPurchasing"
            size="small"
            :data-testid="`buy-${item.itemKey}-button`"
            variant="flat"
            @click="$emit('buy', item.itemKey)"
          >
            Buy
          </v-btn>
        </div>
      </section>
    </div>
  </div>
</template>
