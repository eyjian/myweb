<script setup lang="ts">
import { ref, shallowRef } from 'vue'
import { Codemirror } from 'vue-codemirror'
import { sql } from '@codemirror/lang-sql'
import { oneDark } from '@codemirror/theme-one-dark'
import { keymap } from '@codemirror/view'
import type { Extension } from '@codemirror/state'

const emit = defineEmits<{
  execute: [sql: string]
}>()

const code = ref('')

const runShortcut = (): boolean => {
  if (code.value.trim()) {
    emit('execute', code.value)
  }
  return true
}

const extensions: Extension[] = [
  sql(),
  oneDark,
  keymap.of([
    { key: 'Ctrl-Enter', run: runShortcut },
    { key: 'Cmd-Enter', run: runShortcut },
  ]),
]

const extRef = shallowRef(extensions)

function setCode(value: string) {
  code.value = value
}

function clearCode() {
  code.value = ''
}

function getCode(): string {
  return code.value
}

defineExpose({ setCode, clearCode, getCode })
</script>

<template>
  <div class="sql-editor h-full">
    <Codemirror
      v-model="code"
      :extensions="extRef"
      :style="{ height: '100%' }"
      placeholder="Type SQL here... (Ctrl+Enter to execute)"
      :tab-size="2"
    />
  </div>
</template>

<style scoped>
.sql-editor :deep(.cm-editor) {
  height: 100%;
  font-size: 14px;
}
.sql-editor :deep(.cm-scroller) {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}
</style>
