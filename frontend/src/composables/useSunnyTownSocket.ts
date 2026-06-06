import { ref } from 'vue'
import { createSunnyTownSession } from '../api/sunnyTownApi'
import type { SunnyTownServerMessage, SunnyTownSession } from '../types/sunnyTown'

interface SunnyTownSocketOptions {
  onMessage: (message: SunnyTownServerMessage) => void
  onSession: (session: SunnyTownSession) => void | Promise<void>
}

const reconnectInitialDelayMs = 500
const reconnectMaxDelayMs = 8_000

export function useSunnyTownSocket(options: SunnyTownSocketOptions) {
  const session = ref<SunnyTownSession | null>(null)
  const status = ref('Entering Sunny Town...')
  const error = ref('')
  const connected = ref(false)

  let socket: WebSocket | null = null
  let reconnectTimer = 0
  let reconnectAttempts = 0
  let shuttingDown = false

  async function start() {
    shuttingDown = false
    try {
      const nextSession = await createSunnyTownSession()
      session.value = nextSession
      await options.onSession(nextSession)
      status.value = 'Connecting...'
      connect(nextSession)
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : String(caught)
      status.value = 'Could not enter Sunny Town'
    }
  }

  function stop() {
    shuttingDown = true
    clearReconnectTimer()
    socket?.close()
    socket = null
    connected.value = false
  }

  function send(message: string): boolean {
    if (!isOpen()) {
      return false
    }
    socket?.send(message)
    return true
  }

  function isOpen(): boolean {
    return socket?.readyState === WebSocket.OPEN
  }

  function connect(activeSession: SunnyTownSession) {
    const url = new URL(activeSession.websocketUrl)
    url.searchParams.set('token', activeSession.joinToken)
    const nextSocket = new WebSocket(url.toString())
    socket = nextSocket

    nextSocket.addEventListener('open', () => {
      if (socket !== nextSocket) {
        return
      }
      reconnectAttempts = 0
      connected.value = true
      status.value = 'Connected'
      error.value = ''
    })

    nextSocket.addEventListener('message', (event) => {
      if (socket !== nextSocket) {
        return
      }
      const message = parseServerMessage(event.data)
      if (message) {
        options.onMessage(message)
      }
    })

    nextSocket.addEventListener('close', () => {
      if (socket !== nextSocket) {
        return
      }
      connected.value = false
      socket = null
      if (shuttingDown) {
        status.value = 'Disconnected'
        return
      }
      scheduleReconnect()
    })

    nextSocket.addEventListener('error', () => {
      if (socket !== nextSocket) {
        return
      }
      error.value = 'Sunny Town connection failed'
    })
  }

  function scheduleReconnect() {
    if (shuttingDown || reconnectTimer) {
      return
    }

    const delay = Math.min(reconnectMaxDelayMs, reconnectInitialDelayMs * 2 ** reconnectAttempts)
    reconnectAttempts += 1
    status.value = 'Reconnecting...'
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = 0
      void reconnectSunnyTown()
    }, delay)
  }

  async function reconnectSunnyTown() {
    if (shuttingDown) {
      return
    }

    try {
      const nextSession = await createSunnyTownSession()
      session.value = nextSession
      await options.onSession(nextSession)
      connect(nextSession)
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : String(caught)
      scheduleReconnect()
    }
  }

  function clearReconnectTimer() {
    if (reconnectTimer) {
      window.clearTimeout(reconnectTimer)
      reconnectTimer = 0
    }
  }

  function parseServerMessage(data: unknown): SunnyTownServerMessage | null {
    if (typeof data !== 'string') {
      error.value = 'Sunny Town sent an unsupported message'
      return null
    }

    try {
      return JSON.parse(data) as SunnyTownServerMessage
    } catch {
      error.value = 'Sunny Town sent an invalid message'
      return null
    }
  }

  return {
    connected,
    error,
    isOpen,
    send,
    session,
    start,
    status,
    stop,
  }
}
