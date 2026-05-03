<script setup lang="ts">
defineProps<{
  message: string | null
}>()
</script>

<template>
  <div v-if="message" class="error-alert">
    {{ message }}
  </div>
</template>
