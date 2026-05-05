<script setup lang="ts">
defineProps<{
  connected: boolean
  host?: string
  database?: string
  user?: string
}>()
</script>

<template>
  <div class="status-bar flex items-center gap-3 px-4 py-1.5 bg-gray-900 border-t border-gray-700 text-sm">
    <span class="flex items-center gap-1.5">
      <span :class="connected ? 'bg-green-400' : 'bg-red-400'" class="w-2 h-2 rounded-full inline-block"></span>
      <span class="text-gray-300">{{ connected ? 'Connected' : 'Disconnected' }}</span>
    </span>
    <span v-if="host" class="text-gray-400">{{ user }}@{{ host }}</span>
    <span v-if="database" class="text-blue-400">{{ database }}</span>
  </div>
</template>
