<script setup lang="ts">
import { computed, ref } from 'vue'
import { useStudentInventoryStore } from '../../stores/studentInventory'
import type { ContainerInventorySlots, InventoryStorageRef } from '../../types/inventory'
import type { SunnyTownWorldObject } from '../../types/sunnyTown'
import SunnyTownInventorySlot from './SunnyTownInventorySlot.vue'

const props = defineProps<{
  canDeposit: boolean
  canWithdraw: boolean
  chest: SunnyTownWorldObject
  container: ContainerInventorySlots | null
  error: string
  isLoading: boolean
  isTransferring: boolean
}>()

const emit = defineEmits<{
  close: []
  transfer: [source: InventoryStorageRef, destination: InventoryStorageRef]
}>()

type DraggedStorage = {
  kind: 'player_inventory' | 'container'
  slotIndex: number
}

const inventoryStore = useStudentInventoryStore()
const draggedStorage = ref<DraggedStorage | null>(null)
const selectedPlayerSlotIndex = ref<number | null>(null)
const selectedContainerSlotIndex = ref<number | null>(null)
const invalidPlayerSlotIndex = ref<number | null>(null)
const invalidContainerSlotIndex = ref<number | null>(null)
const pendingSource = ref<DraggedStorage | null>(null)
const pendingDestination = ref<DraggedStorage | null>(null)

const storageLabel = computed(() => {
  if (props.chest.storageRole === 'input') {
    return 'Input storage'
  }
  if (props.chest.storageRole === 'output') {
    return 'Output storage'
  }
  return 'Shared storage'
})
const selectedPlayerSlot = computed(() => {
  if (selectedPlayerSlotIndex.value === null) {
    return inventoryStore.inventorySlots.find((slot) => slot.item) || null
  }
  return inventoryStore.inventorySlots.find((slot) => slot.slotIndex === selectedPlayerSlotIndex.value) || null
})
const selectedContainerSlot = computed(() => {
  if (selectedContainerSlotIndex.value === null) {
    return props.container?.slots.find((slot) => slot.item) || null
  }
  return props.container?.slots.find((slot) => slot.slotIndex === selectedContainerSlotIndex.value) || null
})

function startPlayerDrag(slotIndex: number, event: DragEvent) {
  const slot = inventoryStore.inventorySlots.find((candidate) => candidate.slotIndex === slotIndex)
  if (!slot?.item || !props.canDeposit || props.isTransferring) {
    invalidPlayerSlotIndex.value = slotIndex
    event.preventDefault()
    return
  }
  draggedStorage.value = { kind: 'player_inventory', slotIndex }
  invalidPlayerSlotIndex.value = null
  invalidContainerSlotIndex.value = null
}

function startContainerDrag(slotIndex: number, event: DragEvent) {
  const slot = props.container?.slots.find((candidate) => candidate.slotIndex === slotIndex)
  if (!slot?.item || !props.canWithdraw || props.isTransferring) {
    invalidContainerSlotIndex.value = slotIndex
    event.preventDefault()
    return
  }
  draggedStorage.value = { kind: 'container', slotIndex }
  invalidPlayerSlotIndex.value = null
  invalidContainerSlotIndex.value = null
}

function setPlayerDropTarget(slotIndex: number) {
  invalidPlayerSlotIndex.value = draggedStorage.value?.kind === 'container' && props.canWithdraw ? null : slotIndex
}

function setContainerDropTarget(slotIndex: number) {
  invalidContainerSlotIndex.value = draggedStorage.value?.kind === 'player_inventory' && props.canDeposit ? null : slotIndex
}

function clearPlayerDropTarget(slotIndex: number) {
  if (invalidPlayerSlotIndex.value === slotIndex) {
    invalidPlayerSlotIndex.value = null
  }
}

function clearContainerDropTarget(slotIndex: number) {
  if (invalidContainerSlotIndex.value === slotIndex) {
    invalidContainerSlotIndex.value = null
  }
}

function dropOnPlayer(slotIndex: number) {
  const source = draggedStorage.value
  if (!source || source.kind !== 'container' || !props.canWithdraw) {
    invalidPlayerSlotIndex.value = slotIndex
    clearDrag()
    return
  }
  transfer(source, { kind: 'player_inventory', slotIndex })
  selectedPlayerSlotIndex.value = slotIndex
}

function dropOnContainer(slotIndex: number) {
  const source = draggedStorage.value
  if (!source || source.kind !== 'player_inventory' || !props.canDeposit) {
    invalidContainerSlotIndex.value = slotIndex
    clearDrag()
    return
  }
  transfer(source, { kind: 'container', containerId: props.container?.containerId || '', slotIndex })
  selectedContainerSlotIndex.value = slotIndex
}

function transfer(source: DraggedStorage, destination: InventoryStorageRef) {
  pendingSource.value = source
  pendingDestination.value = { kind: destination.kind, slotIndex: destination.slotIndex }
  emit('transfer', source.kind === 'container'
    ? { kind: 'container', containerId: props.container?.containerId || '', slotIndex: source.slotIndex }
    : { kind: 'player_inventory', slotIndex: source.slotIndex }, destination)
  clearDrag()
}

function clearDrag() {
  draggedStorage.value = null
  invalidPlayerSlotIndex.value = null
  invalidContainerSlotIndex.value = null
}

function isPending(kind: DraggedStorage['kind'], slotIndex: number): boolean {
  if (!props.isTransferring) {
    return false
  }
  return (pendingSource.value?.kind === kind && pendingSource.value.slotIndex === slotIndex) ||
    (pendingDestination.value?.kind === kind && pendingDestination.value.slotIndex === slotIndex)
}
</script>

<template>
  <div class="sunny-town-chest-panel" data-testid="sunny-town-chest-panel" role="dialog" :aria-label="chest.name || 'Storage chest'">
    <div class="sunny-town-chest-panel__header">
      <div>
        <strong>{{ chest.name || 'Storage Chest' }}</strong>
        <span>{{ storageLabel }}</span>
      </div>
      <v-btn icon="mdi-close" size="x-small" variant="text" @click="$emit('close')" />
    </div>
    <v-alert v-if="error" class="mb-3" density="compact" type="error" variant="tonal">
      {{ error }}
    </v-alert>
    <div v-if="isLoading" class="sunny-town-chest-panel__empty">
      Opening storage...
    </div>
    <div v-else class="sunny-town-chest-panel__grids">
      <section class="sunny-town-chest-panel__storage" aria-label="Player inventory">
        <div class="sunny-town-chest-panel__storage-header">
          <strong>Inventory</strong>
          <span v-if="canDeposit">Drag items to storage</span>
          <span v-else>Read-only from here</span>
        </div>
        <div class="sunny-town-inventory-grid sunny-town-chest-panel__grid">
          <SunnyTownInventorySlot
            v-for="slot in inventoryStore.inventorySlots"
            :key="slot.slotIndex"
            :draggable-enabled="canDeposit"
            :item="slot.item"
            :invalid-drop="invalidPlayerSlotIndex === slot.slotIndex"
            :pending="isPending('player_inventory', slot.slotIndex)"
            :quantity="slot.item?.quantity"
            :selected="selectedPlayerSlot?.slotIndex === slot.slotIndex"
            :slot-label="String(slot.slotIndex + 1)"
            @drag-end="clearDrag"
            @drag-leave="clearPlayerDropTarget(slot.slotIndex)"
            @drag-over="setPlayerDropTarget(slot.slotIndex)"
            @drag-start="startPlayerDrag(slot.slotIndex, $event)"
            @drop="dropOnPlayer(slot.slotIndex)"
            @select="selectedPlayerSlotIndex = slot.slotIndex"
          />
        </div>
      </section>
      <section class="sunny-town-chest-panel__storage" aria-label="Container storage">
        <div class="sunny-town-chest-panel__storage-header">
          <strong>Chest</strong>
          <span v-if="canWithdraw">Drag items to inventory</span>
          <span v-else-if="canDeposit">Deposit only</span>
          <span v-else>Read-only</span>
        </div>
        <div class="sunny-town-inventory-grid sunny-town-chest-panel__grid">
          <SunnyTownInventorySlot
            v-for="slot in container?.slots || []"
            :key="slot.slotIndex"
            :draggable-enabled="canWithdraw"
            :item="slot.item"
            :invalid-drop="invalidContainerSlotIndex === slot.slotIndex"
            :pending="isPending('container', slot.slotIndex)"
            :quantity="slot.item?.quantity"
            :selected="selectedContainerSlot?.slotIndex === slot.slotIndex"
            :slot-label="String(slot.slotIndex + 1)"
            @drag-end="clearDrag"
            @drag-leave="clearContainerDropTarget(slot.slotIndex)"
            @drag-over="setContainerDropTarget(slot.slotIndex)"
            @drag-start="startContainerDrag(slot.slotIndex, $event)"
            @drop="dropOnContainer(slot.slotIndex)"
            @select="selectedContainerSlotIndex = slot.slotIndex"
          />
        </div>
      </section>
    </div>
  </div>
</template>
