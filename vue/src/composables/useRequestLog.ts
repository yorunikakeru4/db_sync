import { readonly, ref } from 'vue'
import type { RequestLogEntry } from '../types/api'

const entries = ref<RequestLogEntry[]>([])

export function useRequestLog() {
  const pushEntry = (entry: RequestLogEntry) => {
    entries.value = [entry, ...entries.value].slice(0, 20)
  }

  const clearEntries = () => {
    entries.value = []
  }

  return {
    entries: readonly(entries),
    pushEntry,
    clearEntries,
  }
}
