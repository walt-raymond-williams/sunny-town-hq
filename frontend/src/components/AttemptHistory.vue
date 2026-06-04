<script setup>
import AttemptReviewGrid from './AttemptReviewGrid.vue'
import { attemptStatusColor, attemptStatusLabel } from '../domain/assignmentStatus'

defineProps({
  attempts: {
    type: Array,
    default: () => [],
  },
  emptyMessage: {
    type: String,
    default: 'No attempts yet.',
  },
  feedbackTitle: {
    type: String,
    default: 'Feedback',
  },
  showStudentName: {
    type: Boolean,
    default: false,
  },
  title: {
    type: String,
    default: 'Attempt History',
  },
})
</script>

<template>
  <section class="attempt-history">
    <h2>{{ title }}</h2>
    <v-alert v-if="attempts.length === 0" class="mt-3" type="info" variant="tonal">
      {{ emptyMessage }}
    </v-alert>
    <div v-for="attempt in attempts" v-else :key="attempt.id" class="attempt-card">
      <div class="question-title">
        <v-chip size="small" variant="tonal">
          Attempt {{ attempt.attempt_number }}
        </v-chip>
        <v-chip v-if="showStudentName" color="secondary" size="small" variant="tonal">
          {{ attempt.student_display_name }}
        </v-chip>
        <v-chip :color="attemptStatusColor(attempt)" size="small" variant="tonal">
          {{ attemptStatusLabel(attempt) }}
        </v-chip>
      </div>
      <AttemptReviewGrid
        class="mt-3"
        :boxes="[
          { title: showStudentName ? 'Student Answer' : 'Your Answer', text: attempt.submitted_answer },
          {
            title: feedbackTitle,
            text: attempt.feedback,
            emptyText: 'No feedback yet.',
            submitted: true,
          },
        ]"
      />
    </div>
  </section>
</template>
