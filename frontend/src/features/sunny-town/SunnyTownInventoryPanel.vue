<script setup lang="ts">
import { computed } from 'vue'
import { useStudentInventoryStore } from '../../stores/studentInventory'
import type { EquipmentSlot } from '../../types/inventory'

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
  toggleCraftingPanel: []
  unequipSlot: [slot: EquipmentSlot]
  updateShowAllCraftingRecipes: [value: boolean]
}>()

const inventoryStore = useStudentInventoryStore()
const selectedHotbarItem = computed(() => inventoryStore.hotbarSlots[props.selectedHotbarIndex]?.item || null)
const visibleCraftingRecipes = computed(() => (
  props.showAllCraftingRecipes ? inventoryStore.knownCraftingRecipes : inventoryStore.craftableRecipes
))
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
          <div>
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
      <div class="inventory-list inventory-list--compact">
        <div v-for="item in inventoryStore.items" :key="item.key" class="inventory-item inventory-item--dark">
          <span class="inventory-item__icon" :class="`inventory-item__icon--${item.key}`" aria-hidden="true" />
          <div>
            <p class="inventory-item__name">{{ item.name }}</p>
            <p class="inventory-item__description">{{ item.description }}</p>
          </div>
          <div class="sunny-town-inventory__item-actions">
            <v-btn
              v-if="item.equipSlot && !item.equipped"
              :loading="inventoryStore.isUpdatingEquipment"
              color="primary"
              size="x-small"
              variant="flat"
              @click="emit('equipItem', item.key, item.equipSlot)"
            >
              Wear
            </v-btn>
            <v-btn
              :loading="inventoryStore.isUpdatingHotbar"
              color="warning"
              size="x-small"
              variant="tonal"
              @click="emit('assignHotbar', item.key)"
            >
              Slot {{ selectedHotbarIndex + 1 }}
            </v-btn>
            <strong class="inventory-item__quantity">{{ item.quantity }}</strong>
          </div>
        </div>
      </div>
      <section class="sunny-town-hotbar-editor" aria-label="Selected hotbar slot">
        <div>
          <p class="inventory-item__name">Slot {{ selectedHotbarIndex + 1 }}</p>
          <p class="inventory-item__description">{{ selectedHotbarItem?.name || 'Empty' }}</p>
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
