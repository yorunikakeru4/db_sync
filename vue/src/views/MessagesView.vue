<script setup lang="ts">
import { reactive, ref } from 'vue'
import { createMessage, getMessages } from '../api/messages'
import DataTable from '../components/DataTable.vue'
import DetailCard from '../components/DetailCard.vue'
import ErrorAlert from '../components/ErrorAlert.vue'
import FormSection from '../components/FormSection.vue'

const createForm = reactive({
  user_id: '',
  subject: '',
  content: '',
  date_sent: '',
})

const getForm = reactive({
  value: '',
})

const created = ref<Record<string, unknown> | null>(null)
const rows = ref<Record<string, unknown>[]>([])
const errorMessage = ref<string | null>(null)
const loading = ref(false)

const normalizeRows = (value: unknown): Record<string, unknown>[] => {
  if (Array.isArray(value)) {
    return value.filter((item): item is Record<string, unknown> => item !== null && typeof item === 'object')
  }

  if (value !== null && typeof value === 'object') {
    return [value as Record<string, unknown>]
  }

  return []
}

async function handleCreate() {
  loading.value = true
  errorMessage.value = null

  try {
    created.value = await createMessage({ ...createForm })
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Create message failed'
  } finally {
    loading.value = false
  }
}

async function handleGet() {
  loading.value = true
  errorMessage.value = null

  try {
    rows.value = normalizeRows(await getMessages(getForm.value))
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Get messages failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="page-grid">
    <FormSection title="Create message" description="Создание сообщения через write endpoint.">
      <form class="form-grid" @submit.prevent="handleCreate">
        <label class="field">
          <span>User ID</span>
          <input v-model="createForm.user_id" type="text" required />
        </label>
        <label class="field">
          <span>Subject</span>
          <input v-model="createForm.subject" type="text" required />
        </label>
        <label class="field">
          <span>Content</span>
          <textarea v-model="createForm.content" rows="4" required />
        </label>
        <label class="field">
          <span>Date sent</span>
          <input v-model="createForm.date_sent" type="datetime-local" required />
        </label>
        <button class="button" :disabled="loading">Create</button>
      </form>
    </FormSection>

    <FormSection title="Get messages" description="Чтение сообщений через current read endpoint helper.">
      <form class="form-grid" @submit.prevent="handleGet">
        <label class="field">
          <span>Lookup value</span>
          <input v-model="getForm.value" type="text" required />
        </label>
        <button class="button" :disabled="loading">Get</button>
      </form>
    </FormSection>

    <ErrorAlert :message="errorMessage" />
    <DetailCard :value="created" />
    <DataTable :rows="rows" />
  </section>
</template>
