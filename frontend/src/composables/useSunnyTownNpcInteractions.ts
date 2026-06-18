import { computed, ref, type Ref } from 'vue'
import { getNextStudentAssignment, submitStudentAnswer as submitStudentAnswerRequest } from '../api/studentAssignmentsApi'
import { getShopStock, purchaseShopItem as purchaseShopItemRequest } from '../api/shopApi'
import type { Assignment } from '../types/assignment'
import type { StudentInventory } from '../types/inventory'
import type { SunnyTownNpc, SunnyTownPlayer } from '../types/sunnyTown'

export interface NpcInteractionResult {
  handled: boolean
  openedSchoolwork: boolean
}

interface ShopPurchaseRequest {
  shopId: string
  itemKey: string
  quantity: number
}

interface ShopPurchaseResult {
  starBalance: number
  inventory: StudentInventory
}

interface ShopStockResult {
  shopId: string
  items: Array<{
    itemKey: string
    quantity: number
    capacity: number
  }>
}

interface SunnyTownNpcInteractionOptions {
  loadInventory?: () => Promise<void>
  loadShopStock?: (shopId: string) => Promise<ShopStockResult>
  loadNextAssignment?: () => Promise<Assignment | null>
  purchaseShopItem?: (purchase: ShopPurchaseRequest) => Promise<ShopPurchaseResult>
  setInventoryItems?: (items: StudentInventory['items']) => void
  starBalance?: Ref<number>
  submitStudentAnswer?: (assignmentId: number, answer: string) => Promise<unknown>
}

export function useSunnyTownNpcInteractions(options: SunnyTownNpcInteractionOptions = {}) {
  const nearbyNpc = ref<SunnyTownNpc | null>(null)
  const activeDialogueNpc = ref<SunnyTownNpc | null>(null)
  const activeDialogueLineIndex = ref(0)
  const activeShopNpc = ref<SunnyTownNpc | null>(null)
  const shopOpen = ref(false)
  const shopError = ref('')
  const shopNotice = ref('')
  const shopStock = ref<Record<string, number>>({})
  const shopStockCapacity = ref<Record<string, number>>({})
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
    shopStock.value = {}
    shopStockCapacity.value = {}
  }

  function openSchoolworkMenu(npc: SunnyTownNpc) {
    closeDialogue()
    activeShopNpc.value = null
    shopOpen.value = false
    shopStock.value = {}
    shopStockCapacity.value = {}
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
    shopStock.value = {}
    shopStockCapacity.value = {}
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

  async function startSchoolwork() {
    if (!activeSchoolworkNpc.value || isLoadingSchoolwork.value) {
      return
    }
    schoolworkOpen.value = true
    schoolworkAssignment.value = null
    schoolworkAnswer.value = ''
    schoolworkError.value = ''
    schoolworkNotice.value = ''
    isLoadingSchoolwork.value = true

    try {
      schoolworkAssignment.value = await loadNextAssignment()
    } catch (caught) {
      schoolworkError.value = errorMessage(caught)
    } finally {
      isLoadingSchoolwork.value = false
    }
  }

  async function openTrade() {
    const shop = activeShopNpc.value?.shop
    if (!shop) {
      return
    }
    shopOpen.value = true
    shopError.value = ''
    shopNotice.value = ''
    try {
      await Promise.all([
        options.loadInventory?.(),
        refreshShopStock(shop.id),
      ])
    } catch (caught) {
      shopError.value = errorMessage(caught)
    }
  }

  async function buyShopItem(itemKey: string) {
    const shop = activeShopNpc.value?.shop
    if (!shop || isPurchasing.value) {
      return
    }
    isPurchasing.value = true
    shopError.value = ''
    shopNotice.value = ''

    try {
      const purchase = await purchaseItem({
        shopId: shop.id,
        itemKey,
        quantity: 1,
      })
      if (options.starBalance) {
        options.starBalance.value = purchase.starBalance
      }
      options.setInventoryItems?.(purchase.inventory.items)
      await refreshShopStock(shop.id)
      shopNotice.value = 'Purchased.'
    } catch (caught) {
      shopError.value = errorMessage(caught)
    } finally {
      isPurchasing.value = false
    }
  }

  async function submitSchoolworkAnswer() {
    if (!schoolworkAssignment.value || isSubmittingSchoolwork.value) {
      return
    }

    isSubmittingSchoolwork.value = true
    schoolworkError.value = ''
    schoolworkNotice.value = ''

    try {
      await submitAnswer(schoolworkAssignment.value.id, schoolworkAnswer.value.trim())
      schoolworkNotice.value = 'Answer submitted.'
      schoolworkAnswer.value = ''
      schoolworkAssignment.value = await loadNextAssignment()
    } catch (caught) {
      schoolworkError.value = errorMessage(caught)
    } finally {
      isSubmittingSchoolwork.value = false
    }
  }

  async function loadNextAssignment(): Promise<Assignment | null> {
    if (options.loadNextAssignment) {
      return options.loadNextAssignment()
    }
    return getNextStudentAssignment()
  }

  async function submitAnswer(assignmentId: number, answer: string): Promise<void> {
    if (options.submitStudentAnswer) {
      await options.submitStudentAnswer(assignmentId, answer)
      return
    }
    await submitStudentAnswerRequest(assignmentId, answer)
  }

  async function purchaseItem(purchase: ShopPurchaseRequest): Promise<ShopPurchaseResult> {
    if (options.purchaseShopItem) {
      return options.purchaseShopItem(purchase)
    }
    return purchaseShopItemRequest(purchase)
  }

  async function refreshShopStock(shopId: string): Promise<void> {
    const stock = await loadShopStock(shopId)
    shopStock.value = Object.fromEntries(stock.items.map((item) => [item.itemKey, item.quantity]))
    shopStockCapacity.value = Object.fromEntries(stock.items.map((item) => [item.itemKey, item.capacity]))
  }

  async function loadShopStock(shopId: string): Promise<ShopStockResult> {
    if (options.loadShopStock) {
      return options.loadShopStock(shopId)
    }
    return getShopStock(shopId)
  }

  return {
    activeDialogueLine,
    activeDialogueLineIndex,
    activeDialogueNpc,
    activeSchoolworkNpc,
    activeShopNpc,
    closeDialogue,
    closeOverlays,
    buyShopItem,
    dialogueProgress,
    hasActiveOverlay,
    interactWith,
    isLoadingSchoolwork,
    isPurchasing,
    isSubmittingSchoolwork,
    nearbyNpc,
    openTrade,
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
    shopStock,
    shopStockCapacity,
    startSchoolwork,
    submitSchoolworkAnswer,
  }
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
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
