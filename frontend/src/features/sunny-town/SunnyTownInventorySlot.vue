<script setup lang="ts">
import { computed } from 'vue'
import type { InventorySlotItem } from '../../types/inventory'

const props = withDefaults(defineProps<{
  disabled?: boolean
  draggableEnabled?: boolean
  invalidDrop?: boolean
  item: InventorySlotItem | null
  pending?: boolean
  quantity?: number
  selected?: boolean
  slotLabel?: string
  tooltip?: boolean
  variant?: 'default' | 'compact'
}>(), {
  disabled: false,
  draggableEnabled: true,
  invalidDrop: false,
  pending: false,
  quantity: undefined,
  selected: false,
  slotLabel: '',
  tooltip: true,
  variant: 'default',
})

const emit = defineEmits<{
  click: []
  dragEnd: []
  dragLeave: []
  dragOver: [event: DragEvent]
  dragStart: [event: DragEvent]
  drop: [event: DragEvent]
  select: []
}>()

const iconClass = computed(() => {
  const iconKey = props.item?.iconKey || props.item?.key || ''
  return iconKey ? `inventory-item__icon--${iconKey}` : ''
})

const stackQuantity = computed(() => props.quantity ?? props.item?.quantity ?? 0)
const showQuantity = computed(() => Boolean(props.item) && stackQuantity.value !== 1)
const tooltipText = computed(() => {
  if (!props.item || !props.tooltip) {
    return ''
  }
  return props.item.description ? `${props.item.name}: ${props.item.description}` : props.item.name
})
const accessibleLabel = computed(() => {
  if (!props.item) {
    return props.slotLabel ? `${props.slotLabel}, empty` : 'Empty inventory slot'
  }
  const quantityText = showQuantity.value ? `, quantity ${stackQuantity.value}` : ''
  return props.slotLabel ? `${props.slotLabel}, ${props.item.name}${quantityText}` : `${props.item.name}${quantityText}`
})

function handleDragStart(event: DragEvent) {
  if (props.disabled || !props.draggableEnabled || !props.item) {
    event.preventDefault()
    return
  }
  event.dataTransfer?.setData('text/plain', props.item.key)
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
  emit('dragStart', event)
}

function handleDragOver(event: DragEvent) {
  if (props.disabled) {
    return
  }
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
  emit('dragOver', event)
}

function handleDrop(event: DragEvent) {
  if (props.disabled) {
    return
  }
  event.preventDefault()
  emit('drop', event)
}
</script>

<template>
  <button
    class="sunny-town-inventory-slot"
    :class="{
      'sunny-town-inventory-slot--compact': variant === 'compact',
      'sunny-town-inventory-slot--disabled': disabled,
      'sunny-town-inventory-slot--empty': !item,
      'sunny-town-inventory-slot--invalid-drop': invalidDrop,
      'sunny-town-inventory-slot--occupied': item,
      'sunny-town-inventory-slot--pending': pending,
      'sunny-town-inventory-slot--selected': selected,
    }"
    type="button"
    :aria-disabled="disabled"
    :aria-label="accessibleLabel"
    :draggable="Boolean(item) && draggableEnabled && !disabled"
    :disabled="disabled"
    :title="tooltipText"
    @click="emit('click'); emit('select')"
    @dragend="emit('dragEnd')"
    @dragleave="emit('dragLeave')"
    @dragover="handleDragOver"
    @dragstart="handleDragStart"
    @drop="handleDrop"
  >
    <span v-if="slotLabel" class="sunny-town-inventory-slot__label">{{ slotLabel }}</span>
    <span v-if="item" class="inventory-item__icon sunny-town-inventory-slot__icon" :class="iconClass" aria-hidden="true" />
    <span v-else class="sunny-town-inventory-slot__empty" aria-hidden="true" />
    <strong v-if="showQuantity" class="sunny-town-inventory-slot__quantity">{{ stackQuantity }}</strong>
    <span v-if="pending" class="sunny-town-inventory-slot__pending" aria-hidden="true" />
    <span v-if="item && tooltip" class="sunny-town-inventory-slot__tooltip" role="tooltip">
      <strong>{{ item.name }}</strong>
      <span v-if="item.description">{{ item.description }}</span>
    </span>
  </button>
</template>
