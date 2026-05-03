<script setup lang="ts">
import AppTabs from './components/AppTabs.vue'
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <div>
        <p class="eyebrow">CQRS</p>
        <h1>CQRS Console</h1>
        <p class="subtitle">Формы для write/read endpoint'ов и журнал запросов.</p>
      </div>
      <AppTabs />
    </header>

    <main class="app-main">
      <router-view />
    </main>
  </div>
</template>
