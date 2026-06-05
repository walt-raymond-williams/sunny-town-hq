<script setup lang="ts">
import { computed } from 'vue'
import AttemptHistory from '../../components/AttemptHistory.vue'
import AttemptReviewGrid from '../../components/AttemptReviewGrid.vue'
import { passFailOptions } from '../../domain/categories'
import {
  answerPreview,
  attemptHistory,
  currentAttempt,
  statusColor,
  statusLabel,
} from '../../domain/assignmentStatus'
import type { AttemptReviewBox } from '../../components/AttemptReviewGrid.vue'
import type { Assignment, GradingForm } from '../../types/assignment'

const props = defineProps<{
  assignment: Assignment
  gradingForm: GradingForm | undefined
  isResettingAssignment: boolean
  isSavingGrade: boolean
}>()

defineEmits<{
  delete: []
  reset: []
  saveGrade: [passed: boolean]
}>()

const activeAttempt = computed(() => currentAttempt(props.assignment))
const reviewBoxes = computed<AttemptReviewBox[]>(() => {
  const boxes: AttemptReviewBox[] = [
    { title: 'Expected Answer', text: props.assignment.expected_answer },
  ]

  if (activeAttempt.value) {
    boxes.push({
      title: `${activeAttempt.value.student_display_name}'s Answer`,
      text: activeAttempt.value.submitted_answer,
      submitted: true,
    })
  }

  return boxes
})
</script>

<template>
  <v-expansion-panel-title>
    <div class="grading-title">
      <div class="question-title">
        <v-chip color="primary" size="small" variant="tonal">
          {{ assignment.category }}
        </v-chip>
        <v-chip :color="statusColor(assignment)" size="small" variant="tonal">
          {{ statusLabel(assignment) }}
        </v-chip>
        <v-chip v-if="activeAttempt" size="small" variant="tonal">
          Attempt {{ activeAttempt.attempt_number }}
        </v-chip>
        <v-chip v-if="activeAttempt" color="secondary" size="small" variant="tonal">
          {{ activeAttempt.student_display_name }}
        </v-chip>
        <span>{{ assignment.prompt }}</span>
      </div>
      <p v-if="activeAttempt" class="answer-preview">
        {{ answerPreview(activeAttempt.submitted_answer) }}
      </p>
    </div>
  </v-expansion-panel-title>
  <v-expansion-panel-text>
    <AttemptReviewGrid :boxes="reviewBoxes" />

    <v-form v-if="activeAttempt && gradingForm" class="form-grid" @submit.prevent>
      <v-textarea
        v-model="gradingForm.feedback"
        label="Feedback"
        placeholder="Good explanation."
        rows="3"
        variant="outlined"
      />
      <div class="actions">
        <v-btn-toggle
          v-model="gradingForm.passed"
          class="pass-fail-toggle"
          color="primary"
          mandatory
          variant="outlined"
        >
          <v-btn
            v-for="option in passFailOptions"
            :key="option.label"
            :color="option.value ? 'success' : 'error'"
            :loading="isSavingGrade && gradingForm.passed === option.value"
            :value="option.value"
            size="large"
            @click="$emit('saveGrade', option.value)"
          >
            {{ option.label }}
          </v-btn>
        </v-btn-toggle>
        <v-btn
          :loading="isResettingAssignment"
          color="warning"
          prepend-icon="mdi-restore"
          size="large"
          variant="flat"
          @click="$emit('reset')"
        >
          Reset Problem
        </v-btn>
      </div>
    </v-form>

    <AttemptHistory
      :attempts="attemptHistory(assignment)"
      feedback-title="Feedback"
      show-student-name
      title="Attempt History"
    />

    <v-btn
      class="mt-4"
      color="error"
      prepend-icon="mdi-delete"
      variant="flat"
      @click="$emit('delete')"
    >
      Delete
    </v-btn>
  </v-expansion-panel-text>
</template>
