import { computed, ref } from 'vue'
import type { Assignment } from '../types/assignment'
import type { SunnyTownNpc, SunnyTownPlayer } from '../types/sunnyTown'

export interface NpcInteractionResult {
  handled: boolean
  openedSchoolwork: boolean
}

export function useSunnyTownNpcInteractions() {
  const nearbyNpc = ref<SunnyTownNpc | null>(null)
  const activeDialogueNpc = ref<SunnyTownNpc | null>(null)
  const activeDialogueLineIndex = ref(0)
  const activeShopNpc = ref<SunnyTownNpc | null>(null)
  const shopOpen = ref(false)
  const shopError = ref('')
  const shopNotice = ref('')
  const isPurchasing = ref(false)
  const activeSchoolworkNpc = ref<SunnyTownNpc | null>(null)
  const schoolworkOpen = ref(false)
  const schoolworkAssignment = ref<Assignment | null>(null)
  const schoolworkAnswer = ref('')
  const schoolworkError = ref('')
  const schoolworkNotice = ref('')
  const isLoadingSchoolwork = ref(false)
  const isSubmittingSchoolwork = ref(false)

  const activeDialogueLine = computed(() => activeDialogueNpc.value?.dialogue[activeDialogueLineIndex.value] || '')
  const dialogueProgress = computed(() => {
    if (!activeDialogueNpc.value) {
      return ''
    }
    return `${activeDialogueLineIndex.value + 1}/${activeDialogueNpc.value.dialogue.length}`
  })

  function hasActiveOverlay(): boolean {
    return Boolean(activeDialogueNpc.value || activeShopNpc.value || activeSchoolworkNpc.value)
  }

  function closeDialogue() {
    activeDialogueNpc.value = null
    activeDialogueLineIndex.value = 0
  }

  function openShopMenu(npc: SunnyTownNpc) {
    closeDialogue()
    activeSchoolworkNpc.value = null
    schoolworkOpen.value = false
    activeShopNpc.value = npc
    shopOpen.value = false
    shopError.value = ''
    shopNotice.value = ''
  }

  function openSchoolworkMenu(npc: SunnyTownNpc) {
    closeDialogue()
    activeShopNpc.value = null
    shopOpen.value = false
    activeSchoolworkNpc.value = npc
    schoolworkOpen.value = false
    schoolworkAssignment.value = null
    schoolworkAnswer.value = ''
    schoolworkError.value = ''
    schoolworkNotice.value = ''
  }

  function closeOverlays() {
    closeDialogue()
    activeShopNpc.value = null
    shopOpen.value = false
    shopError.value = ''
    shopNotice.value = ''
    isPurchasing.value = false
    activeSchoolworkNpc.value = null
    schoolworkOpen.value = false
    schoolworkAssignment.value = null
    schoolworkAnswer.value = ''
    schoolworkError.value = ''
    schoolworkNotice.value = ''
    isLoadingSchoolwork.value = false
    isSubmittingSchoolwork.value = false
  }

  function interactWith(npc: SunnyTownNpc | null): NpcInteractionResult {
    if (activeSchoolworkNpc.value || activeShopNpc.value) {
      return { handled: true, openedSchoolwork: false }
    }
    if (activeDialogueNpc.value) {
      if (activeDialogueLineIndex.value < activeDialogueNpc.value.dialogue.length - 1) {
        activeDialogueLineIndex.value++
      } else {
        closeDialogue()
      }
      return { handled: true, openedSchoolwork: false }
    }

    if (!npc) {
      return { handled: false, openedSchoolwork: false }
    }
    if (npc.shop) {
      openShopMenu(npc)
      return { handled: true, openedSchoolwork: false }
    }
    if (npc.activity?.type === 'schoolwork') {
      openSchoolworkMenu(npc)
      return { handled: true, openedSchoolwork: true }
    }

    activeDialogueNpc.value = npc
    activeDialogueLineIndex.value = 0
    return { handled: true, openedSchoolwork: false }
  }

  function refreshNearby(nextNearbyNpc: SunnyTownNpc | null) {
    nearbyNpc.value = nextNearbyNpc
    if (activeDialogueNpc.value && nextNearbyNpc?.id !== activeDialogueNpc.value.id) {
      closeDialogue()
    }
    if (activeShopNpc.value && nextNearbyNpc?.id !== activeShopNpc.value.id) {
      closeOverlays()
    }
    if (activeSchoolworkNpc.value && nextNearbyNpc?.id !== activeSchoolworkNpc.value.id) {
      closeOverlays()
    }
  }

  return {
    activeDialogueLine,
    activeDialogueLineIndex,
    activeDialogueNpc,
    activeSchoolworkNpc,
    activeShopNpc,
    closeDialogue,
    closeOverlays,
    dialogueProgress,
    hasActiveOverlay,
    interactWith,
    isLoadingSchoolwork,
    isPurchasing,
    isSubmittingSchoolwork,
    nearbyNpc,
    openSchoolworkMenu,
    openShopMenu,
    refreshNearby,
    schoolworkAnswer,
    schoolworkAssignment,
    schoolworkError,
    schoolworkNotice,
    schoolworkOpen,
    shopError,
    shopNotice,
    shopOpen,
  }
}

export function nearestSunnyTownNpc(
  npcs: SunnyTownNpc[],
  self: SunnyTownPlayer | null | undefined,
  radius: number,
): SunnyTownNpc | null {
  if (!self) {
    return null
  }

  let nearest: SunnyTownNpc | null = null
  let nearestDistance = radius
  for (const npc of npcs) {
    const distance = Math.hypot(self.x - npc.x, self.y - npc.y)
    if (distance <= nearestDistance) {
      nearest = npc
      nearestDistance = distance
    }
  }
  return nearest
}
