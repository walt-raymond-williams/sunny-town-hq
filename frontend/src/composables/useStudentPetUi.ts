import { computed, ref } from 'vue'
import { useStudentPetStore } from '../stores/studentPet'
import type { FallingStarsResult, PetUiMood } from '../types/pet'

const studentPetMood = ref<PetUiMood>('idle')
const isFallingStarsVisible = ref(false)
const lastFallingStarsResult = ref<FallingStarsResult | null>(null)
let studentPetMoodTimer: number | undefined

export function useStudentPetUi() {
  const studentPetStore = useStudentPetStore()

  const canFeedPet = computed(() => studentPetStore.cookies > 0 && studentPetStore.hunger < 100)
  const canPlayWithPet = computed(() => !studentPetStore.sleeping && studentPetStore.energy >= 10)
  const canPutPetToSleep = computed(() => !studentPetStore.sleeping)
  const canWakePet = computed(() => studentPetStore.sleeping)
  const petAvatarMood = computed(() =>
    studentPetMood.value === 'idle' ? studentPetStore.mood : studentPetMood.value,
  )

  function resetStudentPetMood() {
    window.clearTimeout(studentPetMoodTimer)
    studentPetMood.value = 'idle'
  }

  function celebrateStudentAnswer() {
    window.clearTimeout(studentPetMoodTimer)
    studentPetMood.value = 'happy'
    studentPetMoodTimer = window.setTimeout(() => {
      studentPetMood.value = 'idle'
    }, 2600)
  }

  function playPetEatingAnimation() {
    window.clearTimeout(studentPetMoodTimer)
    studentPetMood.value = 'eating'
    studentPetMoodTimer = window.setTimeout(() => {
      studentPetMood.value = 'idle'
    }, 2600)
  }

  async function feedStudentPet() {
    const wasFed = await studentPetStore.feedPet()
    if (wasFed) {
      playPetEatingAnimation()
    }
  }

  function playWithStudentPet() {
    lastFallingStarsResult.value = null
    isFallingStarsVisible.value = true
  }

  async function handleFallingStarsComplete(result: FallingStarsResult) {
    isFallingStarsVisible.value = false
    const wasApplied = await studentPetStore.applyGameResult(result)
    if (!wasApplied) {
      return
    }

    lastFallingStarsResult.value = result

    if (result.won) {
      celebrateStudentAnswer()
    }
  }

  function handleFallingStarsQuit() {
    isFallingStarsVisible.value = false
  }

  async function putStudentPetToSleep() {
    await studentPetStore.putToSleep()
  }

  async function wakeStudentPet() {
    await studentPetStore.wakePet()
  }

  return {
    canFeedPet,
    canPlayWithPet,
    canPutPetToSleep,
    canWakePet,
    celebrateStudentAnswer,
    feedStudentPet,
    handleFallingStarsComplete,
    handleFallingStarsQuit,
    isFallingStarsVisible,
    lastFallingStarsResult,
    petAvatarMood,
    playWithStudentPet,
    putStudentPetToSleep,
    resetStudentPetMood,
    studentPetStore,
    wakeStudentPet,
  }
}
