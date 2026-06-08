<script setup lang="ts">
import type { Assignment } from '../../types/assignment'
import type { SunnyTownNpc } from '../../types/sunnyTown'

defineProps<{
  answer: string
  assignment: Assignment | null
  isLoading: boolean
  isSubmitting: boolean
  notice: string
  npc: SunnyTownNpc
  open: boolean
  schoolworkError: string
}>()

defineEmits<{
  close: []
  start: []
  submit: []
  updateAnswer: [answer: string]
}>()
</script>

<template>
  <div v-if="!open" class="sunny-town-npc-menu" data-testid="sunny-town-npc-menu" role="dialog" :aria-label="npc.name">
    <strong>{{ npc.name }}</strong>
    <p>{{ npc.dialogue[0] }}</p>
    <div class="sunny-town-npc-menu__actions">
      <v-btn color="primary" data-testid="sunny-town-start-schoolwork-button" prepend-icon="mdi-school" variant="flat" @click="$emit('start')">
        Do School Work
      </v-btn>
      <v-btn prepend-icon="mdi-close" variant="tonal" @click="$emit('close')">
        Exit
      </v-btn>
    </div>
  </div>
  <div v-else class="sunny-town-schoolwork" data-testid="sunny-town-schoolwork-panel" role="dialog" :aria-label="`${npc.name} school work`">
    <div class="sunny-town-schoolwork__header">
      <div>
        <strong>{{ npc.name }}</strong>
        <span>School Work</span>
      </div>
      <v-btn icon="mdi-close" size="x-small" variant="text" @click="$emit('close')" />
    </div>
    <v-progress-linear
      v-if="isLoading"
      class="mb-3"
      color="primary"
      indeterminate
    />
    <v-alert v-if="schoolworkError" class="mb-3" density="compact" type="error" variant="tonal">
      {{ schoolworkError }}
    </v-alert>
    <v-alert v-if="notice" class="mb-3" density="compact" type="success" variant="tonal">
      {{ notice }}
    </v-alert>
    <v-alert
      v-if="!isLoading && !assignment && !schoolworkError"
      density="compact"
      type="success"
      variant="tonal"
    >
      You have finished all assignments.
    </v-alert>
    <v-form
      v-if="assignment"
      class="sunny-town-schoolwork__form"
      @submit.prevent="$emit('submit')"
    >
      <div class="sunny-town-schoolwork__question">
        <v-chip color="primary" size="small" variant="tonal">
          {{ assignment.category }}
        </v-chip>
        <p>{{ assignment.prompt }}</p>
      </div>
      <v-textarea
        :model-value="answer"
        data-testid="sunny-town-schoolwork-answer-input"
        label="Your answer"
        rows="4"
        variant="outlined"
        @update:model-value="$emit('updateAnswer', String($event))"
      />
      <v-btn
        :disabled="answer.trim().length === 0"
        :loading="isSubmitting"
        color="primary"
        data-testid="sunny-town-schoolwork-submit-button"
        prepend-icon="mdi-send"
        type="submit"
        variant="flat"
      >
        Submit Answer
      </v-btn>
    </v-form>
  </div>
</template>
