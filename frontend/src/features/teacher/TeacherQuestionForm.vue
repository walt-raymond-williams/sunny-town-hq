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
  <v-form class="form-grid create-question-form" @submit.prevent="$emit('save')">
    <v-select v-model="form.category" :items="categories" label="Category" variant="outlined" />
    <v-textarea
      v-model="form.prompt"
      label="Prompt"
      placeholder="What is 7 + 5?"
      rows="4"
      variant="outlined"
    />
    <v-textarea
      v-model="form.expected_answer"
      label="Expected answer"
      placeholder="12"
      rows="3"
      variant="outlined"
    />
    <v-btn
      :loading="isSaving"
      color="secondary"
      prepend-icon="mdi-content-save"
      size="large"
      type="submit"
    >
      Save Assignment
    </v-btn>
  </v-form>
</template>
