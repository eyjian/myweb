import { ref, onUnmounted } from 'vue'

export interface WSResultStart {
  type: 'result_start'
  columns: string[]
  format: string
}

export interface WSResultRow {
  type: 'result_row'
  row: unknown[]
}

export interface WSResultEnd {
  type: 'result_end'
  rows: number
  affected: number
  time: string
}

export interface WSError {
  type: 'error'
  message: string
}

export interface WSConnected {
  type: 'connected'
  host: string
  database: string
  user: string
}

export interface WSInfo {
  type: 'info'
  message: string
}

export type WSMessage = WSResultStart | WSResultRow | WSResultEnd | WSError | WSConnected | WSInfo

export function useWebSocket() {
  const ws = ref<WebSocket | null>(null)
  const connected = ref(false)
  const lastError = ref('')
  const onMessage = ref<(msg: WSMessage) => void>(() => {})

  function connect() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${location.host}/ws`
    const socket = new WebSocket(url)

    socket.onopen = () => {
      connected.value = true
      lastError.value = ''
    }

    socket.onclose = () => {
      connected.value = false
      setTimeout(connect, 3000)
    }

    socket.onerror = () => {
      lastError.value = 'WebSocket connection error'
    }

    socket.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data)
        onMessage.value(msg)
      } catch (e) {
        console.error('Failed to parse WebSocket message', e)
      }
    }

    ws.value = socket
  }

  function send(type: string, data: Record<string, unknown> = {}) {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify({ type, ...data }))
    }
  }

  function sendQuery(sql: string) {
    send('query', { sql })
  }

  function sendCancel() {
    send('cancel')
  }

  connect()

  onUnmounted(() => {
    if (ws.value) {
      ws.value.close()
    }
  })

  return { connected, lastError, onMessage, sendQuery, sendCancel }
}
