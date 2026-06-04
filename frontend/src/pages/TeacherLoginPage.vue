<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { loginForRole } from '../auth'
import { useRouteAccess } from '../composables/useRouteAccess'

const router = useRouter()
const { ensureTeacherAccess, showSplash } = useRouteAccess()
const password = ref('')
const loginError = ref('')

onMounted(async () => {
  await ensureTeacherAccess({ redirectToTeacher: true })
})

async function loginTeacher() {
  loginError.value = ''

  try {
    await loginForRole('teacher', `${window.location.origin}/teacher`)
    await router.push({ name: 'teacher' })
  } catch (error) {
    loginError.value = error.message
  }
}
</script>

<template>
  <v-card class="panel" elevation="8">
    <v-card-text>
      <v-btn
        class="mb-4"
        color="primary"
        prepend-icon="mdi-arrow-left"
        variant="text"
        @click="showSplash"
      >
        Back
      </v-btn>
      <p class="eyebrow">Teacher</p>
      <h1>Teacher Login</h1>
      <v-form class="form-grid" @submit.prevent="loginTeacher">
        <v-text-field
          v-model="password"
          autocomplete="current-password"
          label="Password"
          type="password"
          variant="outlined"
        />
        <v-btn color="secondary" size="large" type="submit">Enter</v-btn>
      </v-form>
      <v-alert v-if="loginError" class="mt-5" type="error" variant="tonal">
        {{ loginError }}
      </v-alert>
    </v-card-text>
  </v-card>
</template>
