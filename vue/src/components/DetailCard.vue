<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  value: Record<string, unknown> | null
}>()

const entries = computed(() => Object.entries(props.value ?? {}))
</script>

<template>
  <section v-if="value" class="panel">
    <h3>Result</h3>
    <dl class="detail-list">
      <template v-for="[key, entry] in entries" :key="key">
        <dt>{{ key }}</dt>
        <dd>{{ typeof entry === 'object' ? JSON.stringify(entry) : String(entry) }}</dd>
      </template>
    </dl>
  </section>
</template>
