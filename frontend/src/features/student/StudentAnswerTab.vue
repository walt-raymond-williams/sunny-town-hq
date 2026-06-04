<script setup>
import AttemptHistory from '../../components/AttemptHistory.vue'
import { studentCategoryOptions } from '../../domain/categories'
import { hasPreviousAttempts, previousAttempts } from '../../domain/assignmentStatus'

defineProps({
  isLoadingStudentAssignment: {
    type: Boolean,
    required: true,
  },
  isSubmittingStudentAnswer: {
    type: Boolean,
    required: true,
  },
  studentAssignment: {
    type: Object,
    default: null,
  },
  studentError: {
    type: String,
    default: '',
  },
  studentMessage: {
    type: String,
    default: '',
  },
})

const studentAnswer = defineModel('studentAnswer', { type: String, default: '' })
const studentCategoryFilter = defineModel('studentCategoryFilter', { type: String, default: 'ALL' })

defineEmits({
  categoryChange: () => true,
  submitAnswer: () => true,
})
</script>

<template>
  <v-select
    v-model="studentCategoryFilter"
    class="student-category-filter"
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

  <v-form v-else-if="studentAssignment" class="form-grid" @submit.prevent="$emit('submitAnswer')">
    <div class="student-question">
      <v-chip color="primary" size="small" variant="tonal">
        {{ studentAssignment.category }}
      </v-chip>
      <p>{{ studentAssignment.prompt }}</p>
    </div>

    <v-textarea v-model="studentAnswer" label="Your answer" rows="5" variant="outlined" />

    <v-btn :loading="isSubmittingStudentAnswer" color="secondary" size="large" type="submit">
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
