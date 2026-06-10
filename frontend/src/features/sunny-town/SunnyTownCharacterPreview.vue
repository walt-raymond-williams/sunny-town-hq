<script setup lang="ts">
import { computed } from 'vue'
import type { CharacterProgressionSkill } from '../../api/characterProgressionApi'
import type { EquipmentSlot, EquippedSlot } from '../../types/inventory'
import SunnyTownInventorySlot from './SunnyTownInventorySlot.vue'

const props = defineProps<{
  equipment: Record<EquipmentSlot, string>
  equipmentSlots: EquippedSlot[]
  invalidDropSlot: EquipmentSlot | null
  isUpdatingEquipment: boolean
  pendingDropSlot: EquipmentSlot | null
  progressionError: string
  progressionLoading: boolean
  skills: CharacterProgressionSkill[]
}>()

const emit = defineEmits<{
  cancelDrag: []
  clearDropTarget: [slot: EquipmentSlot]
  dropEquipment: [slot: EquipmentSlot]
  setDropTarget: [slot: EquipmentSlot]
  unequip: [slot: EquipmentSlot]
}>()

const miningSkill = computed(() => props.skills.find((skill) => skill.key === 'mining') || null)
const miningProgress = computed(() => {
  const skill = miningSkill.value
  if (!skill || skill.nextLevelXp < 1) {
    return 0
  }
  return Math.min(100, Math.round((skill.currentLevelXp / skill.nextLevelXp) * 100))
})
</script>

<template>
  <div class="sunny-town-character-preview" aria-label="Character preview">
    <div class="sunny-town-character-preview__equipment" aria-label="Equipment slots">
      <div v-for="slot in equipmentSlots" :key="slot.slot" class="equipment-slot equipment-slot--rail">
        <SunnyTownInventorySlot
          :draggable-enabled="false"
          :item="slot.item"
          :invalid-drop="invalidDropSlot === slot.slot"
          :pending="pendingDropSlot === slot.slot"
          :slot-label="slot.slot.slice(0, 1).toUpperCase()"
          :tooltip="false"
          variant="compact"
          @drag-end="emit('cancelDrag')"
          @drag-leave="emit('clearDropTarget', slot.slot)"
          @drag-over="emit('setDropTarget', slot.slot)"
          @drop="emit('dropEquipment', slot.slot)"
        />
        <div class="equipment-slot__summary">
          <p class="summary-category">{{ slot.slot }}</p>
          <p class="inventory-item__name">{{ slot.item?.name || 'Empty' }}</p>
        </div>
        <v-btn
          v-if="slot.item"
          :loading="isUpdatingEquipment"
          color="primary"
          size="x-small"
          variant="flat"
          @click="emit('unequip', slot.slot)"
        >
          Unequip
        </v-btn>
      </div>
    </div>
    <div class="sunny-town-character-preview__avatar-wrap">
      <svg
        class="sunny-town-character-preview__avatar"
        viewBox="0 0 128 136"
        role="img"
        aria-label="Equipped character preview"
      >
        <ellipse cx="64" cy="116" rx="36" ry="12" fill="rgba(0, 0, 0, 0.2)" />
        <g v-if="props.equipment.tool === 'pickaxe'" class="sunny-town-character-preview__tool">
          <line x1="86" y1="76" x2="102" y2="116" stroke="#7b4b24" stroke-width="8" stroke-linecap="round" />
          <path d="M86 72 Q104 56 120 72" fill="none" stroke="#5f6b75" stroke-width="8" stroke-linecap="round" />
        </g>
        <circle
          cx="64"
          cy="74"
          r="34"
          :fill="props.equipment.gear === 'sunny_hoodie' ? '#f06f38' : '#27746f'"
        />
        <g v-if="props.equipment.gear === 'sunny_hoodie'">
          <rect x="42" y="78" width="44" height="18" rx="6" fill="#2f7d72" />
          <line x1="64" y1="78" x2="64" y2="96" stroke="#f4d48e" stroke-width="4" stroke-linecap="round" />
        </g>
        <g v-if="props.equipment.accessory === 'star_cap'">
          <ellipse cx="64" cy="44" rx="30" ry="11" fill="#f2c84b" />
          <rect x="45" y="27" width="38" height="18" rx="5" fill="#365d9f" />
          <path
            d="M64 29 L70 39 L82 39 L72 46 L76 58 L64 51 L52 58 L56 46 L46 39 L58 39 Z"
            fill="#ffffff"
          />
        </g>
        <circle cx="53" cy="66" r="5" fill="#ffffff" />
        <circle cx="75" cy="66" r="5" fill="#ffffff" />
      </svg>
    </div>
    <div class="sunny-town-character-preview__sheet" aria-label="Character growth">
      <p class="sunny-town-character-preview__title">Character</p>
      <p v-if="progressionError" class="sunny-town-character-preview__error">{{ progressionError }}</p>
      <div class="sunny-town-character-preview__stats" aria-label="Stats-ready character area">
        <div class="sunny-town-character-preview__stat">
          <span>Stats</span>
        </div>
        <div class="sunny-town-character-preview__stat">
          <span v-if="progressionLoading">Mining</span>
          <template v-else-if="miningSkill">
            <span>{{ miningSkill.name }} Lv {{ miningSkill.level }}</span>
            <strong>{{ miningSkill.currentLevelXp }}/{{ miningSkill.nextLevelXp }}</strong>
          </template>
          <span v-else>Mining</span>
          <span
            class="sunny-town-character-preview__rail"
            :class="{ 'sunny-town-character-preview__rail--filled': miningSkill }"
            :style="{ '--progress': `${miningProgress}%` }"
            aria-hidden="true"
          />
        </div>
        <div class="sunny-town-character-preview__stat">
          <span>Traits</span>
        </div>
      </div>
    </div>
  </div>
</template>
