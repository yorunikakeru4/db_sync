<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  rows: Record<string, unknown>[]
}>()

const columns = computed(() => {
  const first = props.rows[0]
  return first ? Object.keys(first) : []
})
</script>

<template>
  <section class="panel">
    <h3>Results</h3>
    <table v-if="rows.length" class="data-table">
      <thead>
        <tr>
          <th v-for="column in columns" :key="column">{{ column }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, rowIndex) in rows" :key="rowIndex">
          <td v-for="column in columns" :key="column">
            {{ typeof row[column] === 'object' ? JSON.stringify(row[column]) : String(row[column] ?? '') }}
          </td>
        </tr>
      </tbody>
    </table>
    <p v-else>Нет данных.</p>
  </section>
</template>
