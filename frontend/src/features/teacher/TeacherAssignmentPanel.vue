<script setup>
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

defineProps({
  assignment: {
    type: Object,
    required: true,
  },
  gradingForm: {
    type: Object,
    default: null,
  },
  isResettingAssignment: {
    type: Boolean,
    required: true,
  },
  isSavingGrade: {
    type: Boolean,
    required: true,
  },
})

defineEmits({
  delete: () => true,
  reset: () => true,
  saveGrade: (_passed) => true,
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
        <v-chip v-if="currentAttempt(assignment)" size="small" variant="tonal">
          Attempt {{ currentAttempt(assignment).attempt_number }}
        </v-chip>
        <v-chip
          v-if="currentAttempt(assignment)"
          color="secondary"
          size="small"
          variant="tonal"
        >
          {{ currentAttempt(assignment).student_display_name }}
        </v-chip>
        <span>{{ assignment.prompt }}</span>
      </div>
      <p v-if="currentAttempt(assignment)" class="answer-preview">
        {{ answerPreview(currentAttempt(assignment).submitted_answer) }}
      </p>
    </div>
  </v-expansion-panel-title>
  <v-expansion-panel-text>
    <AttemptReviewGrid
      :boxes="[
        { title: 'Expected Answer', text: assignment.expected_answer },
        currentAttempt(assignment)
          ? {
              title: `${currentAttempt(assignment).student_display_name}'s Answer`,
              text: currentAttempt(assignment).submitted_answer,
              submitted: true,
            }
          : null,
      ].filter(Boolean)"
    />

    <v-form v-if="currentAttempt(assignment) && gradingForm" class="form-grid" @submit.prevent>
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
            :key="option.value"
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
