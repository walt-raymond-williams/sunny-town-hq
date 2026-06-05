<script setup lang="ts">
import { petStats } from '../../domain/categories'
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
      {{ lastFallingStarsResult.energyDelta }}.
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
</template>
