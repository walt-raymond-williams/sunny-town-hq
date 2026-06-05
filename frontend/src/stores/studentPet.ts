import { defineStore } from 'pinia'
import { create } from '@bufbuild/protobuf'
import { timestampDate } from '@bufbuild/protobuf/wkt'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import {
  ApplyGameResultRequestSchema,
  FeedPetRequestSchema,
  GetPetStateRequestSchema,
  type PetStateResponse,
  PetMood,
  PetService,
  PlayWithPetRequestSchema,
  PutPetToSleepRequestSchema,
  WakePetRequestSchema,
  WatchPetStateRequestSchema,
} from '../gen/hq/pet/v1/pet_pb'
import { connectAuthInterceptor } from '../auth'
import type { FallingStarsResult, PetMoodLabel } from '../types/pet'

export interface StudentPetState {
  id: number
  displayName: string
  cookies: number
  hunger: number
  happiness: number
  energy: number
  sleeping: boolean
  mood: PetMoodLabel
  updatedAt: string | null
  lastDecayAt: string | null
  isLoading: boolean
  isWatching: boolean
  error: string
}

const transport = createConnectTransport({
  baseUrl: window.location.origin,
  interceptors: [connectAuthInterceptor],
})
const petClient = createClient(PetService, transport)
let watchAbortController: AbortController | null = null

const moodLabels: Partial<Record<PetMood, PetMoodLabel>> = {
  [PetMood.IDLE]: 'idle',
  [PetMood.HAPPY]: 'happy',
  [PetMood.HUNGRY]: 'hungry',
  [PetMood.SAD]: 'sad',
  [PetMood.SLEEPING]: 'sleeping',
}

export const useStudentPetStore = defineStore('studentPet', {
  state: (): StudentPetState => ({
    id: 1,
    displayName: 'Student',
    cookies: 0,
    hunger: 50,
    happiness: 50,
    energy: 50,
    sleeping: false,
    mood: 'idle',
    updatedAt: null,
    lastDecayAt: null,
    isLoading: false,
    isWatching: false,
    error: '',
  }),
  actions: {
    applyProfile(body: PetStateResponse) {
      this.id = Number(body.userId)
      this.displayName = body.displayName
      this.cookies = body.cookies
      this.hunger = body.petState?.hunger ?? 50
      this.happiness = body.petState?.happiness ?? 50
      this.energy = body.petState?.energy ?? 50
      this.sleeping = body.petState?.sleeping ?? false
      this.mood = moodLabels[body.petState?.mood ?? PetMood.IDLE] || 'idle'
      this.updatedAt = body.petState?.updatedAt
        ? timestampDate(body.petState.updatedAt).toISOString()
        : null
      this.lastDecayAt = body.petState?.lastDecayAt
        ? timestampDate(body.petState.lastDecayAt).toISOString()
        : null
    },
    async loadProfile() {
      this.isLoading = true
      this.error = ''

      try {
        const body = await petClient.getPetState(create(GetPetStateRequestSchema))
        this.applyProfile(body)
      } catch (error) {
        this.error = errorMessage(error)
      } finally {
        this.isLoading = false
      }
    },
    startWatching() {
      if (watchAbortController) {
        return
      }

      const controller = new AbortController()
      watchAbortController = controller
      this.isWatching = true
      this.error = ''

      const watch = async () => {
        try {
          const stream = petClient.watchPetState(create(WatchPetStateRequestSchema), {
            signal: controller.signal,
          })

          for await (const body of stream) {
            this.applyProfile(body)
          }
        } catch (error) {
          if (!controller.signal.aborted) {
            this.error = errorMessage(error)
          }
        } finally {
          if (watchAbortController === controller) {
            watchAbortController = null
            this.isWatching = false
          }
        }
      }

      watch()
    },
    stopWatching() {
      if (watchAbortController) {
        watchAbortController.abort()
        watchAbortController = null
      }
      this.isWatching = false
    },
    async feedPet() {
      this.isLoading = true
      this.error = ''

      try {
        const body = await petClient.feedPet(create(FeedPetRequestSchema))
        this.applyProfile(body)
        return true
      } catch (error) {
        this.error = errorMessage(error)
        return false
      } finally {
        this.isLoading = false
      }
    },
    async playWithPet() {
      this.isLoading = true
      this.error = ''

      try {
        const body = await petClient.playWithPet(create(PlayWithPetRequestSchema))
        this.applyProfile(body)
        return true
      } catch (error) {
        this.error = errorMessage(error)
        return false
      } finally {
        this.isLoading = false
      }
    },
    async applyGameResult(result: FallingStarsResult) {
      this.isLoading = true
      this.error = ''

      try {
        const body = await petClient.applyGameResult(
          create(ApplyGameResultRequestSchema, {
            score: result.score,
          }),
        )
        this.applyProfile(body)
        return true
      } catch (error) {
        this.error = errorMessage(error)
        return false
      } finally {
        this.isLoading = false
      }
    },
    async putToSleep() {
      this.isLoading = true
      this.error = ''

      try {
        const body = await petClient.putPetToSleep(create(PutPetToSleepRequestSchema))
        this.applyProfile(body)
        return true
      } catch (error) {
        this.error = errorMessage(error)
        return false
      } finally {
        this.isLoading = false
      }
    },
    async wakePet() {
      this.isLoading = true
      this.error = ''

      try {
        const body = await petClient.wakePet(create(WakePetRequestSchema))
        this.applyProfile(body)
        return true
      } catch (error) {
        this.error = errorMessage(error)
        return false
      } finally {
        this.isLoading = false
      }
    },
  },
})

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
