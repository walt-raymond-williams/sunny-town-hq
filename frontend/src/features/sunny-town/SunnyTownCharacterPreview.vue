<script setup lang="ts">
import type { EquipmentSlot } from '../../types/inventory'

const props = defineProps<{
  equipment: Record<EquipmentSlot, string>
}>()
</script>

<template>
  <div class="sunny-town-character-preview" aria-label="Character preview">
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
    <div class="sunny-town-character-preview__meta" aria-hidden="true">
      <span />
      <span />
      <span />
    </div>
  </div>
</template>
