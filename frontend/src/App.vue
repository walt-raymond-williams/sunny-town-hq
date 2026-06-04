<script setup>
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import FallingStarsGame from './components/pet/FallingStarsGame.vue'
import StudentPet from './components/StudentPet.vue'
import { useStudentPetUi } from './composables/useStudentPetUi'

const route = useRoute()
const routeName = computed(() => route.name || 'splash')
const {
  handleFallingStarsComplete,
  handleFallingStarsQuit,
  isFallingStarsVisible,
  petAvatarMood,
} = useStudentPetUi()
</script>

<template>
  <v-app>
    <v-main>
      <v-container class="app-container" fluid>
        <RouterView />

        <div v-if="isFallingStarsVisible" class="game-overlay" role="dialog" aria-modal="true">
          <FallingStarsGame
            @complete="handleFallingStarsComplete"
            @quit="handleFallingStarsQuit"
          />
        </div>

        <StudentPet v-if="routeName === 'student'" :mood="petAvatarMood" />
      </v-container>
    </v-main>
  </v-app>
</template>
