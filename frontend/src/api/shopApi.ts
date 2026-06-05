import { authJson, jsonOptions } from './http'
import type { StudentInventory } from '../types/inventory'
import type { InventoryItemResponse, StudentInventoryResponse } from './inventoryApi'

interface ShopPurchaseResponse {
  starBalance?: number
  inventory?: StudentInventoryResponse
}

export interface ShopPurchase {
  shopId: string
  itemKey: string
  quantity: number
}

export interface PurchasedShopItem {
  starBalance: number
  inventory: StudentInventory
}

export async function purchaseShopItem(purchase: ShopPurchase): Promise<PurchasedShopItem> {
  const response = await authJson<ShopPurchaseResponse>(
    '/api/student/shop/purchase',
    jsonOptions('POST', purchase),
  )
  return {
    starBalance: response.starBalance ?? 0,
    inventory: {
      items: (response.inventory?.items || []).map(normalizeInventoryItem).filter((item) => item.quantity > 0),
    },
  }
}

function normalizeInventoryItem(item: InventoryItemResponse) {
  return {
    key: item.key || '',
    name: item.name || '',
    description: item.description || '',
    quantity: item.quantity ?? 0,
  }
}
