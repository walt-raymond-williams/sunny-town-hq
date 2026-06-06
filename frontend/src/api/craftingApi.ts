import { authJson, jsonOptions } from './http'
import type { CraftRecipeResult, CraftingIngredient, CraftingRecipe } from '../types/inventory'
import { normalizeInventoryItem, type StudentInventoryResponse } from './inventoryApi'

interface CraftingIngredientResponse {
  itemKey?: string
  name?: string
  description?: string
  required?: number
  owned?: number
}

interface CraftingRecipeResponse {
  key?: string
  name?: string
  description?: string
  outputKey?: string
  outputName?: string
  quantity?: number
  canCraft?: boolean
  ingredients?: CraftingIngredientResponse[]
}

interface CraftingRecipesResponse {
  recipes?: CraftingRecipeResponse[]
}

interface CraftRecipeResponse {
  inventory?: StudentInventoryResponse
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
    inventory: {
      items: (response.inventory?.items || []).map(normalizeInventoryItem).filter((item) => item.quantity > 0),
    },
    recipes: (response.recipes || []).map(normalizeCraftingRecipe),
  }
}

function normalizeCraftingRecipe(recipe: CraftingRecipeResponse): CraftingRecipe {
  return {
    key: recipe.key || '',
    name: recipe.name || '',
    description: recipe.description || '',
    outputKey: recipe.outputKey || '',
    outputName: recipe.outputName || recipe.name || '',
    quantity: recipe.quantity ?? 0,
    canCraft: recipe.canCraft ?? false,
    ingredients: (recipe.ingredients || []).map(normalizeCraftingIngredient),
  }
}

function normalizeCraftingIngredient(ingredient: CraftingIngredientResponse): CraftingIngredient {
  return {
    itemKey: ingredient.itemKey || '',
    name: ingredient.name || '',
    description: ingredient.description || '',
    required: ingredient.required ?? 0,
    owned: ingredient.owned ?? 0,
  }
}
