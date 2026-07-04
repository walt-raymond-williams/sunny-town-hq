<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useRouteAccess } from '../../composables/useRouteAccess'

const exportPath = '/godot/sunny-town/index.html'

const router = useRouter()
const { ensureStudentAccess } = useRouteAccess()
const exportState = ref<'checking' | 'available' | 'missing'>('checking')

onMounted(async () => {
  const canAccess = await ensureStudentAccess()
  if (!canAccess) {
    return
  }
  await checkExport()
})

async function checkExport() {
  try {
    const response = await fetch(`${exportPath}?sunnyTownGodotProbe=1`, {
      cache: 'no-store',
    })
    const contentType = response.headers.get('content-type') || ''
    const body = await response.text()
    exportState.value = (
      response.ok &&
      contentType.includes('text/html') &&
      body.toLowerCase().includes('godot')
    ) ? 'available' : 'missing'
  } catch {
    exportState.value = 'missing'
  }
}

function backToPet() {
  router.push('/student')
}

function openCanvasFallback() {
  router.push('/student/pet/sunny-town/canvas')
}
</script>

<template>
  <section class="sunny-town-page sunny-town-godot-page" aria-label="Sunny Town Godot">
    <div class="sunny-town-stage sunny-town-godot-stage">
      <iframe
        v-if="exportState === 'available'"
        class="sunny-town-godot-frame"
        title="Sunny Town Godot"
        :src="exportPath"
        allow="autoplay; fullscreen; gamepad; screen-wake-lock"
      />
      <div v-else class="sunny-town-godot-placeholder">
        <div>
          <p class="eyebrow">Godot Web</p>
          <h1>Sunny Town Godot</h1>
          <p v-if="exportState === 'checking'">Checking export...</p>
          <p v-else>Godot export missing at {{ exportPath }}.</p>
        </div>
      </div>
    </div>

    <div class="sunny-town-toolbar">
      <div>
        <p class="eyebrow">Sunny Town</p>
        <h1>Godot Client</h1>
      </div>
      <div class="sunny-town-toolbar__actions">
        <v-btn variant="tonal" color="white" prepend-icon="mdi-gamepad-variant" @click="openCanvasFallback">
          Canvas
        </v-btn>
        <v-btn variant="tonal" color="white" prepend-icon="mdi-arrow-left" @click="backToPet">
          Back
        </v-btn>
      </div>
    </div>
  </section>
</template>
