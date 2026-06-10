import { authJson, jsonOptions } from './http'
import type { StudentInventory } from '../types/inventory'
import { normalizeInventoryItem, type StudentInventoryResponse } from './inventoryApi'

interface ShopPurchaseResponse {
  starBalance?: number
  inventory?: StudentInventoryResponse
}

interface ShopStockResponse {
  shopId?: string
  items?: ShopStockItemResponse[]
}

interface ShopStockItemResponse {
  itemKey?: string
  quantity?: number
  capacity?: number
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

export interface ShopStock {
  shopId: string
  items: ShopStockItem[]
}

export interface ShopStockItem {
  itemKey: string
  quantity: number
  capacity: number
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

export async function getShopStock(shopId: string): Promise<ShopStock> {
  const response = await authJson<ShopStockResponse>(
    `/api/student/shop/stock?shop_id=${encodeURIComponent(shopId)}`,
  )
  return {
    shopId: response.shopId || shopId,
    items: (response.items || []).map((item) => ({
      itemKey: item.itemKey || '',
      quantity: item.quantity ?? 0,
      capacity: item.capacity ?? 0,
    })),
  }
}
