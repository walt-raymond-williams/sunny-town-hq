import { authJson } from './http'

export interface CharacterProgressionSkill {
  key: string
  name: string
  description: string
  xp: number
  level: number
  currentLevelXp: number
  nextLevelXp: number
}

export interface CharacterProgression {
  characterId: number
  skills: CharacterProgressionSkill[]
}

interface CharacterProgressionSkillResponse {
  key?: string
  name?: string
  description?: string
  xp?: number
  level?: number
  currentLevelXp?: number
  nextLevelXp?: number
}

interface CharacterProgressionResponse {
  characterId?: number
  skills?: CharacterProgressionSkillResponse[]
}

export async function getCharacterProgression(): Promise<CharacterProgression> {
  const response = await authJson<CharacterProgressionResponse>('/api/student/sunny-town/progression')
  return {
    characterId: response.characterId ?? 0,
    skills: (response.skills || []).map((skill) => ({
      key: skill.key || '',
      name: skill.name || '',
      description: skill.description || '',
      xp: skill.xp ?? 0,
      level: skill.level ?? 1,
      currentLevelXp: skill.currentLevelXp ?? 0,
      nextLevelXp: skill.nextLevelXp ?? 100,
    })).filter((skill) => skill.key),
  }
}
