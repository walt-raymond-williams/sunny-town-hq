<script setup lang="ts">
import { categories } from '../../domain/categories'
import type { CreateAssignmentPayload } from '../../types/assignment'

defineProps<{
  isSaving: boolean
}>()

const form = defineModel<CreateAssignmentPayload>('form', { required: true })

defineEmits({
  save: () => true,
})
</script>

<template>
  <v-form
    class="form-grid create-question-form"
    data-testid="assignment-create-form"
    @submit.prevent="$emit('save')"
  >
    <v-select
      v-model="form.category"
      :items="categories"
      data-testid="assignment-category-input"
      label="Category"
      variant="outlined"
    />
    <v-textarea
      v-model="form.prompt"
      data-testid="assignment-prompt-input"
      label="Prompt"
      placeholder="What is 7 + 5?"
      rows="4"
      variant="outlined"
    />
    <v-textarea
      v-model="form.expected_answer"
      data-testid="assignment-expected-answer-input"
      label="Expected answer"
      placeholder="12"
      rows="3"
      variant="outlined"
    />
    <v-btn
      :loading="isSaving"
      color="secondary"
      data-testid="assignment-save-button"
      prepend-icon="mdi-content-save"
      size="large"
      type="submit"
    >
      Save Assignment
    </v-btn>
  </v-form>
</template>
