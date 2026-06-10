import { defineStore } from 'pinia'
import { getCharacterProgression, type CharacterProgressionSkill } from '../api/characterProgressionApi'

interface CharacterProgressionState {
  characterId: number
  skills: CharacterProgressionSkill[]
  isLoading: boolean
  error: string
}

export const useCharacterProgressionStore = defineStore('characterProgression', {
  state: (): CharacterProgressionState => ({
    characterId: 0,
    skills: [],
    isLoading: false,
    error: '',
  }),
  getters: {
    miningSkill: (state) => state.skills.find((skill) => skill.key === 'mining') || null,
  },
  actions: {
    async loadProgression() {
      this.isLoading = true
      this.error = ''

      try {
        const progression = await getCharacterProgression()
        this.characterId = progression.characterId
        this.skills = progression.skills
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error)
      } finally {
        this.isLoading = false
      }
    },
  },
})
