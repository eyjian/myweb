<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  columns: string[]
  rows: unknown[][]
  duration?: string
  error?: string
}>()

const hasResult = computed(() => props.columns.length > 0 || props.rows.length > 0)
</script>

<template>
  <div class="result-table flex flex-col h-full overflow-auto">
    <div v-if="error" class="p-3 text-red-400 bg-red-900/20 rounded">
      {{ error }}
    </div>
    <div v-else-if="hasResult" class="overflow-auto flex-1">
      <table class="w-full text-sm border-collapse">
        <thead class="sticky top-0 bg-gray-800">
          <tr>
            <th
              v-for="col in columns"
              :key="col"
              class="px-3 py-2 text-left text-gray-300 border-b border-gray-700 whitespace-nowrap"
            >
              {{ col }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, i) in rows" :key="i" class="hover:bg-gray-800/50">
            <td
              v-for="(cell, j) in row"
              :key="j"
              class="px-3 py-1.5 border-b border-gray-800 whitespace-nowrap"
              :class="{ 'text-gray-500 italic': cell === null }"
            >
              {{ cell === null ? 'NULL' : cell }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="flex items-center justify-center h-full text-gray-500">
      No results yet. Execute a query to see results.
    </div>
    <div v-if="duration" class="px-3 py-1 text-xs text-gray-400 border-t border-gray-700">
      {{ rows.length }} row(s) in {{ duration }}
    </div>
  </div>
</template>
