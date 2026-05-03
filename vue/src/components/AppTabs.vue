<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'

const route = useRoute()

const tabs = [
  { to: '/users', label: 'Users' },
  { to: '/messages', label: 'Messages' },
  { to: '/emails', label: 'Emails' },
  { to: '/requests', label: 'Requests' },
]

const isActive = (path: string) => route.path === path
</script>

<template>
  <nav class="tabs">
    <RouterLink
      v-for="tab in tabs"
      :key="tab.to"
      :to="tab.to"
      class="tabs__link"
      :class="{ 'tabs__link--active': isActive(tab.to) }"
    >
      {{ tab.label }}
    </RouterLink>
  </nav>
</template>
