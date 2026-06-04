import { defineStore } from 'pinia'
import { create } from '@bufbuild/protobuf'
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import {
  ApplyGameResultRequestSchema,
  FeedPetRequestSchema,
  GetPetStateRequestSchema,
  PetMood,
  PetService,
  PlayWithPetRequestSchema,
  PutPetToSleepRequestSchema,
  WakePetRequestSchema,
  WatchPetStateRequestSchema,
} from '../gen/hq/pet/v1/pet_pb'

const transport = createConnectTransport({
  baseUrl: window.location.origin,
})
const petClient = createClient(PetService, transport)
let watchAbortController = null

const moodLabels = {
  [PetMood.IDLE]: 'idle',
  [PetMood.HAPPY]: 'happy',
  [PetMood.HUNGRY]: 'hungry',
  [PetMood.SAD]: 'sad',
  [PetMood.SLEEPING]: 'sleeping',
}

export const useStudentPetStore = defineStore('studentPet', {
  state: () => ({
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
    applyProfile(body) {
      this.id = Number(body.userId)
      this.displayName = body.displayName
      this.cookies = body.cookies
      this.hunger = body.petState?.hunger ?? 50
      this.happiness = body.petState?.happiness ?? 50
      this.energy = body.petState?.energy ?? 50
      this.sleeping = body.petState?.sleeping ?? false
      this.mood = moodLabels[body.petState?.mood] || 'idle'
      this.updatedAt = body.petState?.updatedAt?.toDate?.().toISOString?.() || null
      this.lastDecayAt = body.petState?.lastDecayAt?.toDate?.().toISOString?.() || null
    },
    async loadProfile() {
      this.isLoading = true
      this.error = ''

      try {
        const body = await petClient.getPetState(create(GetPetStateRequestSchema))
        this.applyProfile(body)
      } catch (error) {
        this.error = error.message
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
            this.error = error.message
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
        this.error = error.message
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
        this.error = error.message
        return false
      } finally {
        this.isLoading = false
      }
    },
    async applyGameResult(result) {
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
        this.error = error.message
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
        this.error = error.message
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
        this.error = error.message
        return false
      } finally {
        this.isLoading = false
      }
    },
  },
})
