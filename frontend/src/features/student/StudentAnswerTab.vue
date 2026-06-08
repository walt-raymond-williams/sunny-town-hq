<script setup lang="ts">
import AttemptHistory from '../../components/AttemptHistory.vue'
import { studentCategoryOptions } from '../../domain/categories'
import { hasPreviousAttempts, previousAttempts } from '../../domain/assignmentStatus'
import type { Assignment, StudentCategoryFilter } from '../../types/assignment'

withDefaults(
  defineProps<{
    isLoadingStudentAssignment: boolean
    isSubmittingStudentAnswer: boolean
    studentAssignment: Assignment | null
    studentError?: string
    studentMessage?: string
  }>(),
  {
    studentError: '',
    studentMessage: '',
  },
)

const studentAnswer = defineModel('studentAnswer', { type: String, default: '' })
const studentCategoryFilter = defineModel<StudentCategoryFilter>('studentCategoryFilter', {
  default: 'ALL',
})

defineEmits({
  categoryChange: () => true,
  submitAnswer: () => true,
})
</script>

<template>
  <v-select
    v-model="studentCategoryFilter"
    class="student-category-filter"
    data-testid="student-category-filter"
    density="comfortable"
    hide-details
    item-title="title"
    item-value="value"
    :items="studentCategoryOptions"
    label="Question category"
    variant="outlined"
    @update:model-value="$emit('categoryChange')"
  />

  <v-progress-linear
    v-if="isLoadingStudentAssignment"
    class="mt-6"
    color="primary"
    indeterminate
  />

  <v-alert
    v-else-if="!studentAssignment && !studentError"
    class="mt-6"
    type="success"
    variant="tonal"
  >
    You have finished all assignments
  </v-alert>

  <v-form
    v-else-if="studentAssignment"
    class="form-grid"
    data-testid="student-answer-form"
    @submit.prevent="$emit('submitAnswer')"
  >
    <div class="student-question" data-testid="student-assignment-prompt">
      <v-chip color="primary" size="small" variant="tonal">
        {{ studentAssignment.category }}
      </v-chip>
      <p>{{ studentAssignment.prompt }}</p>
    </div>

    <v-textarea
      v-model="studentAnswer"
      data-testid="student-answer-input"
      label="Your answer"
      rows="5"
      variant="outlined"
    />

    <v-btn
      :loading="isSubmittingStudentAnswer"
      color="secondary"
      data-testid="assignment-submit-button"
      size="large"
      type="submit"
    >
      Submit Answer
    </v-btn>
  </v-form>

  <AttemptHistory
    v-if="studentAssignment && hasPreviousAttempts(studentAssignment)"
    :attempts="previousAttempts(studentAssignment)"
    title="Previous Attempts"
  />

  <v-alert v-if="studentMessage" class="mt-5" type="success" variant="tonal">
    {{ studentMessage }}
  </v-alert>
  <v-alert v-if="studentError" class="mt-5" type="error" variant="tonal">
    {{ studentError }}
  </v-alert>
</template>
