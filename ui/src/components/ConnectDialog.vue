<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{
  connect: [dsn: string]
  close: []
}>()

const host = ref('127.0.0.1')
const port = ref(3306)
const user = ref('root')
const password = ref('')
const database = ref('')
const connecting = ref(false)

function onSubmit() {
  connecting.value = true
  const dsn = `${user.value}:${password.value}@tcp(${host.value}:${port.value})/${database.value}`
  emit('connect', dsn)
}

defineExpose({ setConnecting: (v: boolean) => { connecting.value = v } })
</script>

<template>
  <div class="fixed inset-0 bg-black/60 flex items-center justify-center z-50" @click.self="$emit('close')">
    <div class="bg-gray-800 rounded-lg shadow-xl p-6 w-96">
      <h2 class="text-lg font-semibold text-white mb-4">Connect to MySQL</h2>
      <form @submit.prevent="onSubmit" class="space-y-3">
        <div class="flex gap-3">
          <div class="flex-1">
            <label class="text-xs text-gray-400">Host</label>
            <input v-model="host" class="w-full px-3 py-1.5 bg-gray-700 text-white rounded border border-gray-600 focus:border-blue-500 focus:outline-none" />
          </div>
          <div class="w-24">
            <label class="text-xs text-gray-400">Port</label>
            <input v-model.number="port" type="number" class="w-full px-3 py-1.5 bg-gray-700 text-white rounded border border-gray-600 focus:border-blue-500 focus:outline-none" />
          </div>
        </div>
        <div>
          <label class="text-xs text-gray-400">User</label>
          <input v-model="user" class="w-full px-3 py-1.5 bg-gray-700 text-white rounded border border-gray-600 focus:border-blue-500 focus:outline-none" />
        </div>
        <div>
          <label class="text-xs text-gray-400">Password</label>
          <input v-model="password" type="password" class="w-full px-3 py-1.5 bg-gray-700 text-white rounded border border-gray-600 focus:border-blue-500 focus:outline-none" />
        </div>
        <div>
          <label class="text-xs text-gray-400">Database</label>
          <input v-model="database" class="w-full px-3 py-1.5 bg-gray-700 text-white rounded border border-gray-600 focus:border-blue-500 focus:outline-none" />
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <button type="button" @click="$emit('close')" class="px-4 py-1.5 text-gray-400 hover:text-white">Cancel</button>
          <button type="submit" :disabled="connecting" class="px-4 py-1.5 bg-blue-600 hover:bg-blue-500 text-white rounded disabled:opacity-50">
            {{ connecting ? 'Connecting...' : 'Connect' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
