<script setup lang="ts">
import { onMounted, onUnmounted, reactive, ref } from 'vue'
import { createMessage, deleteMessageById, listMessages, updateMessageById } from '../api/messages'
import DataTable from '../components/DataTable.vue'
import DetailCard from '../components/DetailCard.vue'
import ErrorAlert from '../components/ErrorAlert.vue'
import FormSection from '../components/FormSection.vue'

const createForm = reactive({
  external_id: '',
  sender_id: 0,
  receiver_id: 0,
  subject: '',
  text: '',
})

const updateForm = reactive({
  id: '',
  external_id: '',
  sender_id: 0,
  receiver_id: 0,
  subject: '',
  text: '',
})

const deleteForm = reactive({
  id: '',
})

const created = ref<Record<string, unknown> | null>(null)
const rows = ref<Record<string, unknown>[]>([])
const errorMessage = ref<string | null>(null)
const loading = ref(false)
let pollTimer: number | undefined

async function refreshMessages() {
  try {
    rows.value = (await listMessages()) ?? []
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'List messages failed'
  }
}

async function handleCreate() {
  loading.value = true
  errorMessage.value = null

  try {
    created.value = await createMessage({
      ...createForm,
      sender_id: Number(createForm.sender_id),
      receiver_id: Number(createForm.receiver_id),
    })
    await refreshMessages()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Create message failed'
  } finally {
    loading.value = false
  }
}

async function handleDelete() {
  loading.value = true
  errorMessage.value = null

  try {
    await deleteMessageById(deleteForm.id)
    created.value = null
    await refreshMessages()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Delete message failed'
  } finally {
    loading.value = false
  }
}

async function handleUpdate() {
  loading.value = true
  errorMessage.value = null

  try {
    created.value = await updateMessageById({
      id: updateForm.id,
      external_id: updateForm.external_id,
      sender_id: Number(updateForm.sender_id),
      receiver_id: Number(updateForm.receiver_id),
      subject: updateForm.subject,
      text: updateForm.text,
    })
    await refreshMessages()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Update message failed'
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await refreshMessages()
  pollTimer = window.setInterval(() => {
    void refreshMessages()
  }, 5000)
})

onUnmounted(() => {
  if (pollTimer !== undefined) {
    window.clearInterval(pollTimer)
  }
})
</script>

<template>
  <section class="page-grid">
    <FormSection title="Create message" description="Создание сообщения через write endpoint.">
      <form class="form-grid" @submit.prevent="handleCreate">
        <label class="field">
          <span>External ID</span>
          <input v-model="createForm.external_id" type="text" required />
        </label>
        <label class="field">
          <span>Sender ID</span>
          <input v-model="createForm.sender_id" type="number" min="1" required />
        </label>
        <label class="field">
          <span>Receiver ID</span>
          <input v-model="createForm.receiver_id" type="number" min="1" required />
        </label>
        <label class="field">
          <span>Subject</span>
          <input v-model="createForm.subject" type="text" required />
        </label>
        <label class="field">
          <span>Content</span>
          <textarea v-model="createForm.text" rows="4" required />
        </label>
        <button class="button" :disabled="loading">Create</button>
      </form>
    </FormSection>

    <FormSection title="Update message" description="Обновление сообщения по ID.">
      <form class="form-grid" @submit.prevent="handleUpdate">
        <label class="field">
          <span>Message ID</span>
          <input v-model="updateForm.id" type="text" required />
        </label>
        <label class="field">
          <span>External ID</span>
          <input v-model="updateForm.external_id" type="text" required />
        </label>
        <label class="field">
          <span>Sender ID</span>
          <input v-model="updateForm.sender_id" type="number" min="1" required />
        </label>
        <label class="field">
          <span>Receiver ID</span>
          <input v-model="updateForm.receiver_id" type="number" min="1" required />
        </label>
        <label class="field">
          <span>Subject</span>
          <input v-model="updateForm.subject" type="text" required />
        </label>
        <label class="field">
          <span>Content</span>
          <textarea v-model="updateForm.text" rows="4" required />
        </label>
        <button class="button" :disabled="loading">Update</button>
      </form>
    </FormSection>

    <FormSection title="Delete message" description="Удаление сообщения по ID.">
      <form class="form-grid" @submit.prevent="handleDelete">
        <label class="field">
          <span>Message ID</span>
          <input v-model="deleteForm.id" type="text" required />
        </label>
        <button class="button" :disabled="loading">Delete</button>
      </form>
    </FormSection>

    <ErrorAlert :message="errorMessage" />
    <DetailCard :value="created" />
    <DataTable :rows="rows" />
  </section>
</template>
