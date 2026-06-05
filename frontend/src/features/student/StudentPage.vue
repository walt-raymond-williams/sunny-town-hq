<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { logout } from '../../auth'
import { getMe } from '../../api/meApi'
import { useRouteAccess } from '../../composables/useRouteAccess'
import { useStudentAssignments } from '../../composables/useStudentAssignments'
import { useStudentPetUi } from '../../composables/useStudentPetUi'
import StudentAnswerTab from './StudentAnswerTab.vue'
import StudentGradesTab from './StudentGradesTab.vue'
import StudentPetTab from './StudentPetTab.vue'
import { useRouter } from 'vue-router'

type StudentTabName = 'answer' | 'grades' | 'pet'

const { ensureStudentAccess } = useRouteAccess()
const router = useRouter()
const {
  canFeedPet,
  canPlayWithPet,
  canPutPetToSleep,
  canWakePet,
  celebrateStudentAnswer,
  feedStudentPet,
  lastFallingStarsResult,
  playWithStudentPet,
  putStudentPetToSleep,
  resetStudentPetMood,
  studentPetStore,
  wakeStudentPet,
} = useStudentPetUi()
const {
  gradeSummaries,
  gradedAssignmentsByCategory,
  handleStudentCategoryChange,
  hasStudentGradedAssignments,
  isLoadingStudentAssignment,
  isLoadingStudentGrades,
  isSubmittingStudentAnswer,
  loadNextStudentAssignment,
  loadStudentGrades,
  resetStudentWork,
  studentAnswer,
  studentAssignment,
  studentCategoryFilter,
  studentError,
  studentGradesError,
  studentMessage,
  studentTab,
  submitStudentAnswer,
} = useStudentAssignments({
  onAnswerSubmitted: celebrateStudentAnswer,
})
const isLoggingOut = ref(false)

onMounted(async () => {
  const canAccess = await ensureStudentAccess()
  if (!canAccess) {
    return
  }

  await getMe()
  studentTab.value = 'answer'
  studentCategoryFilter.value = 'ALL'
  await studentPetStore.loadProfile()
  studentPetStore.startWatching()
  await loadNextStudentAssignment()
})

onBeforeUnmount(() => {
  studentPetStore.stopWatching()
  resetStudentPetMood()
})

async function handleStudentTabChange(tabName: StudentTabName) {
  if (tabName === 'answer') {
    await loadNextStudentAssignment()
  }

  if (tabName === 'grades') {
    await studentPetStore.loadProfile()
    await loadStudentGrades()
  }

  if (tabName === 'pet') {
    await studentPetStore.loadProfile()
  }
}

async function logoutStudent() {
  isLoggingOut.value = true
  resetStudentWork()

  try {
    studentPetStore.stopWatching()
    resetStudentPetMood()
    await logout(window.location.origin)
  } catch (error) {
    studentError.value = errorMessage(error)
  } finally {
    isLoggingOut.value = false
  }
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

function enterSunnyTown() {
  router.push('/student/pet/sunny-town')
}
</script>

<template>
  <v-card class="panel student-panel" elevation="8">
    <v-card-text>
      <div class="desk-header">
        <div>
          <p class="eyebrow">Student</p>
          <h1>Student Work</h1>
        </div>
        <v-btn
          :loading="isLoggingOut"
          color="primary"
          prepend-icon="mdi-logout"
          variant="tonal"
          @click="logoutStudent"
        >
          Logout
        </v-btn>
      </div>

      <v-tabs
        v-model="studentTab"
        class="mt-6"
        color="primary"
        @update:model-value="(value) => handleStudentTabChange(value as StudentTabName)"
      >
        <v-tab value="answer">Answer Questions</v-tab>
        <v-tab value="grades">View Grades</v-tab>
        <v-tab value="pet">Pet</v-tab>
      </v-tabs>

      <v-window v-model="studentTab" class="mt-6">
        <v-window-item value="answer">
          <StudentAnswerTab
            v-model:student-answer="studentAnswer"
            v-model:student-category-filter="studentCategoryFilter"
            :is-loading-student-assignment="isLoadingStudentAssignment"
            :is-submitting-student-answer="isSubmittingStudentAnswer"
            :student-assignment="studentAssignment"
            :student-error="studentError"
            :student-message="studentMessage"
            @category-change="handleStudentCategoryChange"
            @submit-answer="submitStudentAnswer"
          />
        </v-window-item>

        <v-window-item value="grades">
          <StudentGradesTab
            :grade-summaries="gradeSummaries"
            :graded-assignments-by-category="gradedAssignmentsByCategory"
            :has-student-graded-assignments="hasStudentGradedAssignments"
            :is-loading-student-grades="isLoadingStudentGrades"
            :student-grades-error="studentGradesError"
            @refresh="loadStudentGrades"
          />
        </v-window-item>

        <v-window-item value="pet">
          <StudentPetTab
            :can-feed-pet="canFeedPet"
            :can-play-with-pet="canPlayWithPet"
            :can-put-pet-to-sleep="canPutPetToSleep"
            :can-wake-pet="canWakePet"
            :last-falling-stars-result="lastFallingStarsResult"
            :student-pet-store="studentPetStore"
            @enter-sunny-town="enterSunnyTown"
            @feed="feedStudentPet"
            @play="playWithStudentPet"
            @sleep="putStudentPetToSleep"
            @wake="wakeStudentPet"
          />
        </v-window-item>
      </v-window>
    </v-card-text>
  </v-card>
</template>
