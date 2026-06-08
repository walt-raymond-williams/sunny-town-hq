<script setup lang="ts">
import { hasRole, keycloak } from '../auth'
import { useRouteAccess } from '../composables/useRouteAccess'

const { showStudent, showTeacherLogin } = useRouteAccess()
</script>

<template>
  <v-card class="panel" data-testid="app-shell" elevation="8">
    <v-card-text>
      <p class="eyebrow">HQ</p>
      <h1>Headquarters</h1>
      <p class="lead">Choose how you want to use HQ today.</p>
      <div class="actions">
        <v-btn
          v-if="!keycloak.authenticated || hasRole('student')"
          color="primary"
          data-testid="student-entry-button"
          variant="tonal"
          size="large"
          @click="showStudent"
        >
          Student
        </v-btn>
        <v-btn
          v-if="!keycloak.authenticated || hasRole('teacher')"
          color="secondary"
          data-testid="teacher-entry-button"
          size="large"
          @click="showTeacherLogin"
        >
          Teacher
        </v-btn>
      </div>
    </v-card-text>
  </v-card>
</template>
