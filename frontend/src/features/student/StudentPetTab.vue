<script setup lang="ts">
import { ref } from 'vue'
import { petStats } from '../../domain/categories'
import { useStudentInventoryStore } from '../../stores/studentInventory'
import type { useStudentPetStore } from '../../stores/studentPet'
import type { FallingStarsResult } from '../../types/pet'

defineProps<{
  canFeedPet: boolean
  canPlayWithPet: boolean
  canPutPetToSleep: boolean
  canWakePet: boolean
  lastFallingStarsResult: FallingStarsResult | null
  studentPetStore: ReturnType<typeof useStudentPetStore>
}>()

defineEmits({
  enterSunnyTown: () => true,
  feed: () => true,
  play: () => true,
  sleep: () => true,
  wake: () => true,
})

const inventoryStore = useStudentInventoryStore()
const inventoryOpen = ref(false)

async function openInventory() {
  inventoryOpen.value = true
  await inventoryStore.loadInventory()
}

async function equipFromInventory(itemKey: string, slot: 'gear' | 'accessory' | '') {
  if (!slot) {
    return
  }
  await inventoryStore.equipItem(slot, itemKey)
}
</script>

<template>
  <div class="list-header">
    <h2>Pet</h2>
    <div class="actions">
      <v-btn
        color="success"
        prepend-icon="mdi-map"
        variant="flat"
        @click="$emit('enterSunnyTown')"
      >
        Sunny Town
      </v-btn>
      <v-btn
        :loading="inventoryStore.isLoading"
        color="secondary"
        prepend-icon="mdi-bag-personal"
        variant="tonal"
        @click="openInventory"
      >
        Inventory
      </v-btn>
      <v-btn
        :loading="studentPetStore.isLoading"
        color="primary"
        prepend-icon="mdi-refresh"
        variant="tonal"
        @click="studentPetStore.loadProfile"
      >
        Refresh
      </v-btn>
    </div>
  </div>

  <v-alert v-if="studentPetStore.error" class="mt-5" type="error" variant="tonal">
    {{ studentPetStore.error }}
  </v-alert>

  <section class="pet-dashboard">
    <div class="cookie-display">
      <span class="cookie-display__icon" aria-hidden="true" />
      <div>
        <p class="summary-category">Cookies</p>
        <p class="cookie-display__count">{{ studentPetStore.cookies }}</p>
      </div>
      <v-divider vertical />
      <div>
        <p class="summary-category">Stars</p>
        <p class="cookie-display__count">{{ studentPetStore.starBalance }}</p>
      </div>
      <div class="pet-actions">
        <v-chip :color="studentPetStore.sleeping ? 'primary' : 'success'" size="small" variant="tonal">
          {{ studentPetStore.sleeping ? 'Sleeping' : 'Awake' }}
        </v-chip>
        <v-btn
          :disabled="!canFeedPet"
          :loading="studentPetStore.isLoading"
          color="secondary"
          prepend-icon="mdi-cookie"
          variant="flat"
          @click="$emit('feed')"
        >
          Feed Pet
        </v-btn>
        <v-btn
          :disabled="!canPlayWithPet"
          color="success"
          prepend-icon="mdi-controller"
          variant="tonal"
          @click="$emit('play')"
        >
          Play
        </v-btn>
        <v-btn
          :disabled="!canPutPetToSleep"
          :loading="studentPetStore.isLoading"
          color="primary"
          prepend-icon="mdi-sleep"
          variant="tonal"
          @click="$emit('sleep')"
        >
          Sleep
        </v-btn>
        <v-btn
          :disabled="!canWakePet"
          :loading="studentPetStore.isLoading"
          color="warning"
          prepend-icon="mdi-weather-sunny"
          variant="tonal"
          @click="$emit('wake')"
        >
          Wake
        </v-btn>
      </div>
    </div>

    <v-alert v-if="studentPetStore.cookies === 0" type="info" variant="tonal">
      Earn cookies by passing assignments.
    </v-alert>
    <v-alert v-else-if="studentPetStore.hunger >= 100" type="success" variant="tonal">
      Your pet is full.
    </v-alert>

    <v-alert
      v-if="lastFallingStarsResult"
      :type="lastFallingStarsResult.won ? 'success' : 'info'"
      variant="tonal"
    >
      {{
        lastFallingStarsResult.won
          ? 'Great job! Your pet loved playing with the falling stars.'
          : 'Almost! Catch more stars next time to make your pet happier.'
      }}
      Final score: {{ lastFallingStarsResult.score }} / 10. Happiness gained:
      +{{ lastFallingStarsResult.happinessDelta }}. Energy spent:
      {{ lastFallingStarsResult.energyDelta }}. Wallet stars gained:
      +{{ lastFallingStarsResult.starsCollected }}.
    </v-alert>

    <div class="pet-stat-list">
      <div v-for="stat in petStats" :key="stat.key" class="pet-stat-row">
        <div class="pet-stat-row__header">
          <div class="question-title">
            <v-icon :color="stat.color" :icon="stat.icon" size="small" />
            <span>{{ stat.label }}</span>
          </div>
          <strong>{{ studentPetStore[stat.key] }}</strong>
        </div>
        <v-progress-linear
          :color="stat.color"
          :model-value="studentPetStore[stat.key]"
          height="12"
          rounded
        />
      </div>
    </div>
  </section>

  <v-dialog v-model="inventoryOpen" max-width="420">
    <v-card>
      <v-card-title class="d-flex align-center justify-space-between">
        <span>Inventory</span>
        <v-btn icon="mdi-close" size="small" variant="text" @click="inventoryOpen = false" />
      </v-card-title>
      <v-card-text>
        <v-alert v-if="inventoryStore.error" class="mb-4" type="error" variant="tonal">
          {{ inventoryStore.error }}
        </v-alert>
        <section class="equipment-panel mb-4" aria-label="Equipment">
          <div v-for="slot in inventoryStore.equipmentSlots" :key="slot.slot" class="equipment-slot">
            <div>
              <p class="summary-category">{{ slot.slot }}</p>
              <p class="inventory-item__name">{{ slot.item?.name || 'Empty' }}</p>
            </div>
            <v-btn
              v-if="slot.item"
              :loading="inventoryStore.isUpdatingEquipment"
              color="secondary"
              size="small"
              variant="tonal"
              @click="inventoryStore.unequipItem(slot.slot)"
            >
              Unequip
            </v-btn>
          </div>
        </section>
        <div class="inventory-list">
          <div v-for="item in inventoryStore.items" :key="item.key" class="inventory-item">
            <span class="inventory-item__icon" :class="`inventory-item__icon--${item.key}`" aria-hidden="true" />
            <div>
              <p class="inventory-item__name">{{ item.name }}</p>
              <p class="inventory-item__description">{{ item.description }}</p>
            </div>
            <v-btn
              v-if="item.equipSlot && !item.equipped"
              :loading="inventoryStore.isUpdatingEquipment"
              color="primary"
              size="small"
              variant="flat"
              @click="equipFromInventory(item.key, item.equipSlot)"
            >
              Equip
            </v-btn>
            <v-btn
              v-else-if="item.equipped && item.equipSlot"
              :loading="inventoryStore.isUpdatingEquipment"
              color="secondary"
              size="small"
              variant="tonal"
              @click="inventoryStore.unequipItem(item.equipSlot)"
            >
              Unequip
            </v-btn>
            <strong v-else class="inventory-item__quantity">{{ item.quantity }}</strong>
          </div>
        </div>
      </v-card-text>
    </v-card>
  </v-dialog>
</template>
