<script setup lang="ts">
import { ref, onMounted } from 'vue'
import SqlEditor from './components/SqlEditor.vue'
import ResultTable from './components/ResultTable.vue'
import StatusBar from './components/StatusBar.vue'
import ConnectDialog from './components/ConnectDialog.vue'
import { useApi } from './composables/useApi'
import { useWebSocket } from './composables/useWebSocket'
import type { WSMessage } from './composables/useWebSocket'

const { getStatus, connect } = useApi()
const { onMessage, sendQuery, sendCancel } = useWebSocket()

const connected = ref(false)
const host = ref('')
const database = ref('')
const user = ref('')
const showConnect = ref(false)

// Query result state
const resultColumns = ref<string[]>([])
const resultRows = ref<unknown[][]>([])
const resultDuration = ref('')
const resultError = ref('')

const editorRef = ref<InstanceType<typeof SqlEditor> | null>(null)

// Load initial status
onMounted(async () => {
  try {
    const status = await getStatus()
    connected.value = status.connected
    host.value = status.host || ''
    database.value = status.database || ''
    user.value = status.user || ''
  } catch {
    connected.value = false
  }
})

// Handle WebSocket messages
onMessage.value = (msg: WSMessage) => {
  switch (msg.type) {
    case 'result_start':
      resultColumns.value = msg.columns
      resultRows.value = []
      resultError.value = ''
      break
    case 'result_row':
      resultRows.value = [...resultRows.value, msg.row]
      break
    case 'result_end':
      resultDuration.value = msg.time
      break
    case 'error':
      resultError.value = msg.message
      break
    case 'connected':
      connected.value = true
      host.value = msg.host
      database.value = msg.database
      user.value = msg.user
      break
    case 'info':
      break
  }
}

function executeQuery(sql: string) {
  resultError.value = ''
  resultColumns.value = []
  resultRows.value = []
  resultDuration.value = ''
  sendQuery(sql)
}

async function handleConnect(dsn: string) {
  try {
    const status = await connect(dsn)
    connected.value = status.connected
    host.value = status.host || ''
    database.value = status.database || ''
    user.value = status.user || ''
    showConnect.value = false
  } catch (err: unknown) {
    if (err instanceof Error) {
      resultError.value = err.message
    }
  }
}

function handleCancel() {
  sendCancel()
}

function handleRun() {
  if (editorRef.value) {
    const sql = editorRef.value.getCode()
    if (sql.trim()) {
      executeQuery(sql)
    }
  }
}
</script>

<template>
  <div class="h-screen flex flex-col bg-gray-900 text-gray-100">
    <!-- Top bar -->
    <header class="flex items-center justify-between px-4 py-2 bg-gray-800 border-b border-gray-700">
      <div class="flex items-center gap-2">
        <span class="text-blue-400 font-bold">myweb</span>
        <span v-if="database" class="text-gray-400">{{ database }}@{{ host }}</span>
      </div>
      <button
        @click="showConnect = true"
        class="px-3 py-1 text-sm bg-blue-600 hover:bg-blue-500 rounded"
      >
        {{ connected ? 'Switch Connection' : 'Connect' }}
      </button>
    </header>

    <!-- Main content -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- SQL Editor -->
      <div class="h-48 border-b border-gray-700">
        <SqlEditor ref="editorRef" @execute="executeQuery" />
      </div>

      <!-- Action bar -->
      <div class="flex items-center gap-2 px-4 py-1.5 bg-gray-800 border-b border-gray-700">
        <button @click="handleRun" class="px-3 py-1 text-sm bg-green-600 hover:bg-green-500 rounded flex items-center gap-1">
          ▶ Run
        </button>
        <button @click="handleCancel" class="px-3 py-1 text-sm bg-gray-700 hover:bg-gray-600 rounded flex items-center gap-1">
          ⏹ Cancel
        </button>
      </div>

      <!-- Result area -->
      <div class="flex-1 overflow-hidden">
        <ResultTable
          :columns="resultColumns"
          :rows="resultRows"
          :duration="resultDuration"
          :error="resultError"
        />
      </div>
    </div>

    <!-- Status bar -->
    <StatusBar
      :connected="connected"
      :host="host"
      :database="database"
      :user="user"
    />

    <!-- Connect dialog -->
    <ConnectDialog
      v-if="showConnect"
      @connect="handleConnect"
      @close="showConnect = false"
    />
  </div>
</template>
