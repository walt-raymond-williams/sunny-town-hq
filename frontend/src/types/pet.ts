export type PetMoodLabel = 'idle' | 'happy' | 'hungry' | 'sad' | 'sleeping'
export type PetUiMood = PetMoodLabel | 'eating'

export interface FallingStarsResult {
  score: number
  won: boolean
  happinessDelta: number
  energyDelta: number
}

export interface FallingObject {
  id: string
  type: 'star' | 'badStar'
  x: number
  y: number
  speed: number
}
