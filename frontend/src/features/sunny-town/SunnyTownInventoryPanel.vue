<script setup lang="ts">
import { computed, ref } from 'vue'
import { useStudentInventoryStore } from '../../stores/studentInventory'
import type { EquipmentSlot } from '../../types/inventory'
import SunnyTownInventorySlot from './SunnyTownInventorySlot.vue'

const props = defineProps<{
  craftingPanelOpen: boolean
  selectedHotbarIndex: number
  showAllCraftingRecipes: boolean
}>()

const emit = defineEmits<{
  assignHotbar: [itemKey: string]
  clearHotbar: []
  close: []
  craftRecipe: [recipeKey: string]
  equipItem: [itemKey: string, slot: EquipmentSlot | '']
  equipInventorySlotDrop: [slot: EquipmentSlot]
  selectHotbarSlot: [index: number]
  toggleCraftingPanel: []
  unequipSlot: [slot: EquipmentSlot]
  updateShowAllCraftingRecipes: [value: boolean]
}>()

const inventoryStore = useStudentInventoryStore()
const selectedInventorySlotIndex = ref<number | null>(null)
const selectedHotbarItem = computed(() => inventoryStore.hotbarSlots[props.selectedHotbarIndex]?.item || null)
const selectedInventorySlot = computed(() => {
  if (selectedInventorySlotIndex.value === null) {
    return inventoryStore.inventorySlots.find((slot) => slot.item) || null
  }
  return inventoryStore.inventorySlots.find((slot) => slot.slotIndex === selectedInventorySlotIndex.value) || null
})
const selectedInventoryItem = computed(() => selectedInventorySlot.value?.item || null)
const visibleCraftingRecipes = computed(() => (
  props.showAllCraftingRecipes ? inventoryStore.knownCraftingRecipes : inventoryStore.craftableRecipes
))

function handleInventorySlotDragStart(slotIndex: number, event: DragEvent) {
  if (!inventoryStore.startInventorySlotDrag(slotIndex)) {
    event.preventDefault()
  }
}

async function handleInventorySlotDrop(slotIndex: number) {
  const moved = await inventoryStore.dropInventorySlot(slotIndex)
  if (moved) {
    selectedInventorySlotIndex.value = slotIndex
  }
}

async function handleHotbarSlotDrop(slot: number) {
  await inventoryStore.dropInventorySlotOnHotbar(slot)
}

function handleEquipmentSlotDrop(slot: EquipmentSlot) {
  emit('equipInventorySlotDrop', slot)
}
</script>

<template>
  <div class="sunny-town-inventory-tray" data-testid="sunny-town-inventory-panel" role="dialog" aria-label="Inventory">
    <section v-if="craftingPanelOpen" class="sunny-town-crafting" aria-label="Crafting">
      <div class="sunny-town-crafting__header">
        <strong>Crafting</strong>
        <v-switch
          :model-value="showAllCraftingRecipes"
          color="warning"
          density="compact"
          hide-details
          inset
          label="All recipes"
          @update:model-value="emit('updateShowAllCraftingRecipes', Boolean($event))"
        />
      </div>
      <v-progress-linear
        v-if="inventoryStore.isLoadingCrafting"
        class="mb-3"
        color="warning"
        indeterminate
      />
      <v-alert v-if="inventoryStore.craftingError" class="mb-3" density="compact" type="error" variant="tonal">
        {{ inventoryStore.craftingError }}
      </v-alert>
      <div v-if="visibleCraftingRecipes.length === 0 && !inventoryStore.isLoadingCrafting" class="sunny-town-crafting__empty">
        No recipes available.
      </div>
      <div
        v-for="recipe in visibleCraftingRecipes"
        :key="recipe.key"
        class="sunny-town-crafting__recipe"
        :class="{ 'sunny-town-crafting__recipe--disabled': !recipe.canCraft }"
      >
        <span class="inventory-item__icon" :class="`inventory-item__icon--${recipe.outputKey}`" aria-hidden="true" />
        <div>
          <p class="inventory-item__name">{{ recipe.name }}</p>
          <p class="inventory-item__description">{{ recipe.description }}</p>
          <div class="sunny-town-crafting__ingredients">
            <span
              v-for="ingredient in recipe.ingredients"
              :key="ingredient.itemKey"
              :class="{ 'sunny-town-crafting__ingredient--missing': ingredient.owned < ingredient.required }"
            >
              {{ ingredient.owned }}/{{ ingredient.required }} {{ ingredient.name }}
            </span>
          </div>
        </div>
        <v-btn
          :disabled="!recipe.canCraft || inventoryStore.isCrafting"
          :loading="inventoryStore.isCrafting"
          color="warning"
          size="x-small"
          variant="flat"
          @click="emit('craftRecipe', recipe.key)"
        >
          Craft
        </v-btn>
      </div>
    </section>
    <div class="sunny-town-inventory">
      <div class="sunny-town-inventory__header">
        <strong>Inventory</strong>
        <div class="sunny-town-inventory__actions">
          <v-btn
            :color="craftingPanelOpen ? 'warning' : undefined"
            :prepend-icon="craftingPanelOpen ? 'mdi-hammer-wrench' : 'mdi-hammer'"
            size="x-small"
            :variant="craftingPanelOpen ? 'flat' : 'tonal'"
            @click="emit('toggleCraftingPanel')"
          >
            Crafting
          </v-btn>
          <v-btn icon="mdi-close" size="x-small" variant="text" @click="emit('close')" />
        </div>
      </div>
      <v-alert v-if="inventoryStore.error" class="mb-3" density="compact" type="error" variant="tonal">
        {{ inventoryStore.error }}
      </v-alert>
      <section class="equipment-panel equipment-panel--dark" aria-label="Equipment">
        <div v-for="slot in inventoryStore.equipmentSlots" :key="slot.slot" class="equipment-slot">
          <SunnyTownInventorySlot
            :draggable-enabled="false"
            :item="slot.item"
            :invalid-drop="inventoryStore.invalidEquipmentDropSlot === slot.slot"
            :pending="inventoryStore.pendingEquipmentDropSlot === slot.slot"
            :slot-label="slot.slot.slice(0, 1).toUpperCase()"
            :tooltip="false"
            variant="compact"
            @drag-end="inventoryStore.cancelInventorySlotDrag()"
            @drag-leave="inventoryStore.clearEquipmentDropTarget(slot.slot)"
            @drag-over="inventoryStore.setEquipmentDropTarget(slot.slot)"
            @drop="handleEquipmentSlotDrop(slot.slot)"
          />
          <div class="equipment-slot__summary">
            <p class="summary-category">{{ slot.slot }}</p>
            <p class="inventory-item__name">{{ slot.item?.name || 'Empty' }}</p>
          </div>
          <v-btn
            v-if="slot.item"
            :loading="inventoryStore.isUpdatingEquipment"
            color="primary"
            size="x-small"
            variant="flat"
            @click="emit('unequipSlot', slot.slot)"
          >
            Unequip
          </v-btn>
        </div>
      </section>
      <div class="sunny-town-inventory-grid" aria-label="Inventory slots">
        <SunnyTownInventorySlot
          v-for="slot in inventoryStore.inventorySlots"
          :key="slot.slotIndex"
          :item="slot.item"
          :invalid-drop="inventoryStore.invalidInventoryDropSlotIndex === slot.slotIndex"
          :pending="inventoryStore.pendingInventoryMoveSourceIndex === slot.slotIndex || inventoryStore.pendingInventoryMoveDestinationIndex === slot.slotIndex"
          :quantity="slot.item?.quantity"
          :selected="selectedInventorySlot?.slotIndex === slot.slotIndex"
          :slot-label="String(slot.slotIndex + 1)"
          @drag-end="inventoryStore.cancelInventorySlotDrag()"
          @drag-leave="inventoryStore.clearInventorySlotDropTarget(slot.slotIndex)"
          @drag-over="inventoryStore.setInventorySlotDropTarget(slot.slotIndex)"
          @drag-start="handleInventorySlotDragStart(slot.slotIndex, $event)"
          @drop="handleInventorySlotDrop(slot.slotIndex)"
          @select="selectedInventorySlotIndex = slot.slotIndex"
        />
      </div>
      <section class="sunny-town-selected-item" aria-label="Selected inventory item">
        <template v-if="selectedInventoryItem">
          <span class="inventory-item__icon" :class="`inventory-item__icon--${selectedInventoryItem.iconKey || selectedInventoryItem.key}`" aria-hidden="true" />
          <div>
            <p class="inventory-item__name">{{ selectedInventoryItem.name }}</p>
            <p class="inventory-item__description">{{ selectedInventoryItem.description }}</p>
          </div>
          <div class="sunny-town-inventory__item-actions">
            <v-btn
              v-if="selectedInventoryItem.equipSlot && !selectedInventoryItem.equipped"
              :loading="inventoryStore.isUpdatingEquipment"
              color="primary"
              size="x-small"
              variant="flat"
              @click="emit('equipItem', selectedInventoryItem.key, selectedInventoryItem.equipSlot)"
            >
              Wear
            </v-btn>
            <v-btn
              :loading="inventoryStore.isUpdatingHotbar"
              color="warning"
              size="x-small"
              variant="tonal"
              @click="emit('assignHotbar', selectedInventoryItem.key)"
            >
              Slot {{ selectedHotbarIndex + 1 }}
            </v-btn>
            <strong class="inventory-item__quantity">{{ selectedInventoryItem.quantity }}</strong>
          </div>
        </template>
        <p v-else class="inventory-item__description">Select an item slot.</p>
      </section>
      <section class="sunny-town-hotbar-editor" aria-label="Hotbar slots">
        <div class="sunny-town-hotbar-editor__summary">
          <p class="inventory-item__name">Slot {{ selectedHotbarIndex + 1 }}</p>
          <p class="inventory-item__description">{{ selectedHotbarItem?.name || 'Empty' }}</p>
        </div>
        <div class="sunny-town-hotbar-editor__slots">
          <SunnyTownInventorySlot
            v-for="(slot, index) in inventoryStore.hotbarSlots"
            :key="slot.slot"
            :draggable-enabled="false"
            :item="slot.item"
            :invalid-drop="inventoryStore.invalidHotbarDropSlot === slot.slot"
            :pending="inventoryStore.pendingHotbarDropSlot === slot.slot"
            :quantity="slot.item?.quantity"
            :selected="selectedHotbarIndex === index"
            :slot-label="String(slot.slot)"
            :tooltip="false"
            variant="compact"
            @click="emit('selectHotbarSlot', index)"
            @drag-end="inventoryStore.cancelInventorySlotDrag()"
            @drag-leave="inventoryStore.clearHotbarDropTarget(slot.slot)"
            @drag-over="inventoryStore.setHotbarDropTarget(slot.slot)"
            @drop="handleHotbarSlotDrop(slot.slot)"
          />
        </div>
        <v-btn
          :disabled="!selectedHotbarItem"
          :loading="inventoryStore.isUpdatingHotbar"
          prepend-icon="mdi-close"
          size="x-small"
          variant="tonal"
          @click="emit('clearHotbar')"
        >
          Clear
        </v-btn>
      </section>
    </div>
  </div>
</template>
