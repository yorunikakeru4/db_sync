<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  rows: Record<string, unknown>[]
}>()

const priorityColumns = ['id', 'user_id', 'contact_id', 'message_id', 'email', 'value', 'subject', 'status', 'method']

const formatValue = (value: unknown): string => {
  if (value === null || value === undefined || value === '') {
    return '—'
  }

  if (typeof value === 'object') {
    return JSON.stringify(value, null, 2)
  }

  return String(value)
}

const rowsWithEntries = computed(() =>
  props.rows.map((row) => {
    const keys = Object.keys(row).sort((left, right) => {
      const leftIndex = priorityColumns.indexOf(left)
      const rightIndex = priorityColumns.indexOf(right)

      if (leftIndex !== -1 || rightIndex !== -1) {
        if (leftIndex === -1) {
          return 1
        }
        if (rightIndex === -1) {
          return -1
        }
        return leftIndex - rightIndex
      }

      return left.localeCompare(right)
    })

    return keys.map((key) => ({
      key,
      value: row[key],
      formattedValue: formatValue(row[key]),
      isStructured: typeof row[key] === 'object' && row[key] !== null,
    }))
  }),
)
</script>

<template>
  <section class="panel">
    <div class="records-header">
      <h3>Results</h3>
      <span class="records-count">{{ rows.length }} items</span>
    </div>
    <div v-if="rows.length" class="records-grid">
      <article v-for="(entries, rowIndex) in rowsWithEntries" :key="rowIndex" class="record-card">
        <div class="record-card__index">#{{ rowIndex + 1 }}</div>
        <dl class="record-list">
          <template v-for="entry in entries" :key="entry.key">
            <dt>{{ entry.key }}</dt>
            <dd>
              <pre v-if="entry.isStructured" class="record-value record-value--structured">{{ entry.formattedValue }}</pre>
              <span v-else class="record-value">{{ entry.formattedValue }}</span>
            </dd>
          </template>
        </dl>
      </article>
    </div>
    <p v-else class="empty-state">Нет данных.</p>
  </section>
</template>
