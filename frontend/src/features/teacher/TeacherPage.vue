<script setup>
import { onMounted, ref } from 'vue'
import { logout } from '../../auth'
import { getMe } from '../../api/meApi'
import { useRouteAccess } from '../../composables/useRouteAccess'
import { useTeacherAssignments } from '../../composables/useTeacherAssignments'
import { teacherFilters } from '../../domain/categories'
import TeacherAssignmentPanel from './TeacherAssignmentPanel.vue'
import TeacherQuestionForm from './TeacherQuestionForm.vue'

const { ensureTeacherAccess } = useRouteAccess()
const {
  assignmentError,
  assignmentMessage,
  deleteAssignment,
  expandedAssignmentId,
  filteredAssignments,
  form,
  gradingError,
  gradingForms,
  gradingMessage,
  handleExpandedAssignmentChange,
  handleTeacherStudentChange,
  hasAssignments,
  hasFilteredAssignments,
  isCreatingQuestion,
  isLoadingAssignments,
  isResettingAssignment,
  isSaving,
  isSavingGrade,
  loadAssignments,
  loadStudents,
  resetAssignment,
  resetTeacherWork,
  saveAssignment,
  saveGrade,
  setAssignmentPanelRef,
  teacherFilter,
  teacherStudentFilter,
  teacherStudentOptions,
} = useTeacherAssignments()
const isLoggingOut = ref(false)

onMounted(async () => {
  const canAccess = await ensureTeacherAccess()
  if (!canAccess) {
    return
  }

  await getMe()
  teacherFilter.value = 'needs-review'
  teacherStudentFilter.value = 'ALL'
  await loadStudents()
  await loadAssignments()
})

async function logoutTeacher() {
  isLoggingOut.value = true
  resetTeacherWork()

  try {
    await logout(window.location.origin)
  } catch (error) {
    gradingError.value = error.message
  } finally {
    isLoggingOut.value = false
  }
}
</script>

<template>
  <v-card class="panel teacher-panel" elevation="8">
    <v-card-text>
      <div class="desk-header">
        <div>
          <p class="eyebrow">Teacher</p>
          <h1>Teacher Desk</h1>
        </div>
        <v-btn
          :loading="isLoggingOut"
          color="primary"
          prepend-icon="mdi-logout"
          variant="tonal"
          @click="logoutTeacher"
        >
          Logout
        </v-btn>
      </div>

      <section class="teacher-workspace">
        <v-btn
          :prepend-icon="isCreatingQuestion ? 'mdi-chevron-up' : 'mdi-plus'"
          color="secondary"
          variant="flat"
          @click="isCreatingQuestion = !isCreatingQuestion"
        >
          New Question
        </v-btn>

        <v-expand-transition>
          <TeacherQuestionForm
            v-if="isCreatingQuestion"
            v-model:form="form"
            :is-saving="isSaving"
            @save="saveAssignment"
          />
        </v-expand-transition>

        <v-alert v-if="assignmentMessage" class="mt-5" type="success" variant="tonal">
          {{ assignmentMessage }}
        </v-alert>
        <v-alert v-if="assignmentError" class="mt-5" type="error" variant="tonal">
          {{ assignmentError }}
        </v-alert>
        <v-alert v-if="gradingMessage" class="mt-5" type="success" variant="tonal">
          {{ gradingMessage }}
        </v-alert>
        <v-alert v-if="gradingError" class="mt-5" type="error" variant="tonal">
          {{ gradingError }}
        </v-alert>

        <div class="list-header">
          <h2>Questions</h2>
          <v-btn
            :loading="isLoadingAssignments"
            color="primary"
            prepend-icon="mdi-refresh"
            variant="tonal"
            @click="loadAssignments"
          >
            Refresh
          </v-btn>
        </div>

        <v-btn-toggle
          v-model="teacherFilter"
          class="filter-toggle"
          color="primary"
          divided
          mandatory
          variant="outlined"
        >
          <v-btn v-for="filter in teacherFilters" :key="filter.value" :value="filter.value">
            {{ filter.label }}
          </v-btn>
        </v-btn-toggle>

        <v-select
          v-model="teacherStudentFilter"
          :items="teacherStudentOptions"
          class="student-filter"
          density="comfortable"
          item-title="title"
          item-value="value"
          label="Student"
          variant="outlined"
          @update:model-value="handleTeacherStudentChange"
        />

        <v-progress-linear v-if="isLoadingAssignments" class="mt-3" color="primary" indeterminate />

        <v-alert v-else-if="!hasAssignments" class="mt-4" type="info" variant="tonal">
          No questions yet.
        </v-alert>

        <v-alert v-else-if="!hasFilteredAssignments" class="mt-4" type="info" variant="tonal">
          No questions match this filter.
        </v-alert>

        <v-expansion-panels
          v-else
          v-model="expandedAssignmentId"
          class="mt-4"
          variant="accordion"
          @update:model-value="handleExpandedAssignmentChange"
        >
          <v-expansion-panel
            v-for="assignment in filteredAssignments"
            :key="assignment.id"
            :ref="(element) => setAssignmentPanelRef(assignment.id, element)"
            :value="assignment.id"
          >
            <TeacherAssignmentPanel
              :assignment="assignment"
              :grading-form="gradingForms[assignment.id]"
              :is-resetting-assignment="isResettingAssignment"
              :is-saving-grade="isSavingGrade"
              @delete="deleteAssignment(assignment)"
              @reset="resetAssignment(assignment)"
              @save-grade="(passed) => saveGrade(assignment, passed)"
            />
          </v-expansion-panel>
        </v-expansion-panels>
      </section>
    </v-card-text>
  </v-card>
</template>
