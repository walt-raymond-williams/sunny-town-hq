<script setup lang="ts">
import AttemptReviewGrid from '../../components/AttemptReviewGrid.vue'
import {
  categoryPercentLabel,
  gradedAttempts,
  latestGradedAttempt,
  passFailColor,
  passFailLabel,
} from '../../domain/assignmentStatus'
import type { GradeSummary, GradedAssignmentGroup } from '../../types/assignment'

withDefaults(
  defineProps<{
    gradeSummaries: GradeSummary[]
    gradedAssignmentsByCategory: GradedAssignmentGroup[]
    hasStudentGradedAssignments: boolean
    isLoadingStudentGrades: boolean
    studentGradesError?: string
  }>(),
  {
    studentGradesError: '',
  },
)

defineEmits({
  refresh: () => true,
})
</script>

<template>
  <div class="list-header">
    <h2>Grade Summary</h2>
    <v-btn
      :loading="isLoadingStudentGrades"
      color="primary"
      prepend-icon="mdi-refresh"
      variant="tonal"
      @click="$emit('refresh')"
    >
      Refresh
    </v-btn>
  </div>

  <v-progress-linear v-if="isLoadingStudentGrades" class="mt-3" color="primary" indeterminate />

  <v-alert v-if="studentGradesError" class="mt-5" type="error" variant="tonal">
    {{ studentGradesError }}
  </v-alert>

  <div class="summary-grid">
    <v-card
      v-for="summary in gradeSummaries"
      :key="summary.category"
      class="summary-card"
      variant="outlined"
    >
      <v-card-text>
        <p class="summary-category">{{ summary.category }}</p>
        <p class="summary-percent">{{ categoryPercentLabel(summary) }}</p>
        <p class="summary-count">{{ summary.passedCount }} of {{ summary.total }} passed</p>
      </v-card-text>
    </v-card>
  </div>

  <v-alert
    v-if="!isLoadingStudentGrades && !hasStudentGradedAssignments"
    class="mt-5"
    type="info"
    variant="tonal"
  >
    No graded assignments yet.
  </v-alert>

  <section
    v-for="group in gradedAssignmentsByCategory"
    v-else
    :key="group.category"
    class="graded-section"
  >
    <h2>{{ group.category }}</h2>
    <v-alert v-if="group.assignments.length === 0" class="mt-3" type="info" variant="tonal">
      No graded {{ group.category.toLowerCase() }} assignments yet.
    </v-alert>

    <v-expansion-panels v-else class="mt-3" variant="accordion">
      <v-expansion-panel v-for="assignment in group.assignments" :key="assignment.id">
        <v-expansion-panel-title>
          <div class="question-title">
            <v-chip
              :color="passFailColor(latestGradedAttempt(assignment)?.passed)"
              size="small"
              variant="tonal"
            >
              {{ assignment.category }}
            </v-chip>
            <v-chip
              :color="passFailColor(latestGradedAttempt(assignment)?.passed)"
              size="small"
              variant="tonal"
            >
              {{ passFailLabel(latestGradedAttempt(assignment)?.passed) }}
            </v-chip>
            <span>{{ assignment.prompt }}</span>
          </div>
        </v-expansion-panel-title>
        <v-expansion-panel-text>
          <div v-for="attempt in gradedAttempts(assignment)" :key="attempt.id" class="attempt-card">
            <div class="question-title">
              <v-chip size="small" variant="tonal">
                Attempt {{ attempt.attempt_number }}
              </v-chip>
              <v-chip :color="passFailColor(attempt.passed)" size="small" variant="tonal">
                {{ passFailLabel(attempt.passed) }}
              </v-chip>
            </div>
            <AttemptReviewGrid
              class="mt-3"
              :boxes="[
                { title: 'Your Answer', text: attempt.submitted_answer },
                { title: 'Expected Answer', text: assignment.expected_answer },
                {
                  title: 'Feedback',
                  text: attempt.feedback,
                  emptyText: 'No feedback yet.',
                  submitted: true,
                },
              ]"
            />
          </div>
        </v-expansion-panel-text>
      </v-expansion-panel>
    </v-expansion-panels>
  </section>
</template>
