<script setup lang="ts">
import { computed } from 'vue'
import DataTable from '../components/DataTable.vue'
import FormSection from '../components/FormSection.vue'
import { useRequestLog } from '../composables/useRequestLog'
import type { RequestLogEntry } from '../types/api'

const logColumns: Array<keyof RequestLogEntry> = [
  'startedAt',
  'method',
  'url',
  'requestBody',
  'status',
  'durationMs',
  'responseBody',
  'errorMessage',
]
const { entries, clearEntries } = useRequestLog()

const tableRows = computed(() =>
  entries.value.map((entry) =>
    logColumns.reduce<Record<string, unknown>>((row, column) => {
      row[column] = entry[column] ?? ''
      return row
    }, {}),
  ),
)
</script>

<template>
  <section class="page-grid">
    <FormSection title="Requests log" description="Последние HTTP-запросы текущей сессии.">
      <button class="button" type="button" @click="clearEntries">Clear log</button>
    </FormSection>

    <div class="table-scroll">
      <DataTable :rows="tableRows" />
    </div>
  </section>
</template>
