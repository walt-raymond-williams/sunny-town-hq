import { authJson, jsonOptions } from './http'
import type { CraftRecipeResult, CraftingIngredient, CraftingRecipe } from '../types/inventory'
import { normalizeStudentInventorySlots, type StudentInventorySlotsResponse } from './inventoryApi'

interface CraftingIngredientResponse {
  itemKey?: string
  name?: string
  description?: string
  iconKey?: string
  maxStack?: number
  category?: string
  required?: number
  owned?: number
}

interface CraftingRecipeResponse {
  key?: string
  name?: string
  description?: string
  outputKey?: string
  outputName?: string
  outputIconKey?: string
  outputMaxStack?: number
  outputCategory?: string
  quantity?: number
  canCraft?: boolean
  ingredients?: CraftingIngredientResponse[]
}

interface CraftingRecipesResponse {
  recipes?: CraftingRecipeResponse[]
}

interface CraftRecipeResponse {
  inventory?: StudentInventorySlotsResponse
  recipes?: CraftingRecipeResponse[]
}

export async function getCraftingRecipes(): Promise<CraftingRecipe[]> {
  const response = await authJson<CraftingRecipesResponse>('/api/student/crafting/recipes')
  return (response.recipes || []).map(normalizeCraftingRecipe)
}

export async function craftStudentRecipe(recipeKey: string): Promise<CraftRecipeResult> {
  const response = await authJson<CraftRecipeResponse>(
    '/api/student/crafting/craft',
    jsonOptions('POST', { recipeKey }),
  )
  return {
    inventory: normalizeStudentInventorySlots(response.inventory || {}),
    recipes: (response.recipes || []).map(normalizeCraftingRecipe),
  }
}

export function normalizeCraftingRecipe(recipe: CraftingRecipeResponse): CraftingRecipe {
  return {
    key: recipe.key || '',
    name: recipe.name || '',
    description: recipe.description || '',
    outputKey: recipe.outputKey || '',
    outputName: recipe.outputName || recipe.name || '',
    outputIconKey: recipe.outputIconKey || recipe.outputKey || '',
    outputMaxStack: recipe.outputMaxStack ?? 0,
    outputCategory: recipe.outputCategory || '',
    quantity: recipe.quantity ?? 0,
    canCraft: recipe.canCraft ?? false,
    ingredients: (recipe.ingredients || []).map(normalizeCraftingIngredient),
  }
}

export function normalizeCraftingIngredient(ingredient: CraftingIngredientResponse): CraftingIngredient {
  return {
    itemKey: ingredient.itemKey || '',
    name: ingredient.name || '',
    description: ingredient.description || '',
    iconKey: ingredient.iconKey || ingredient.itemKey || '',
    maxStack: ingredient.maxStack ?? 0,
    category: ingredient.category || '',
    required: ingredient.required ?? 0,
    owned: ingredient.owned ?? 0,
  }
}
