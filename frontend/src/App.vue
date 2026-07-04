<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import FallingStarsGame from './components/pet/FallingStarsGame.vue'
import StudentPet from './components/StudentPet.vue'
import { useStudentPetUi } from './composables/useStudentPetUi'

const route = useRoute()
const routeName = computed(() => route.name || 'splash')
const isSunnyTownRoute = computed(() => (
  routeName.value === 'sunny-town' ||
  routeName.value === 'sunny-town-canvas' ||
  routeName.value === 'sunny-town-godot'
))
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
      <v-container
        class="app-container"
        :class="{ 'app-container--sunny-town': isSunnyTownRoute }"
        fluid
      >
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
