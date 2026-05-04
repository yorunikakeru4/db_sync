<script setup lang="ts">
import { onMounted, onUnmounted, reactive, ref } from 'vue'
import { createContact, deleteContactByIds, getContactsByUserId, listContacts } from '../api/contacts'
import DataTable from '../components/DataTable.vue'
import DetailCard from '../components/DetailCard.vue'
import ErrorAlert from '../components/ErrorAlert.vue'
import FormSection from '../components/FormSection.vue'

const createForm = reactive({
  user_id: 0,
  contact_id: 0,
  value: '',
  category: 0,
  importance: 0,
})

const getForm = reactive({
  user_id: '',
})

const deleteForm = reactive({
  user_id: '',
  contact_id: '',
})

const created = ref<Record<string, unknown> | null>(null)
const rows = ref<Record<string, unknown>[]>([])
const errorMessage = ref<string | null>(null)
const loading = ref(false)
let pollTimer: number | undefined

async function refreshContacts() {
  try {
    rows.value = (await listContacts()) ?? []
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'List contacts failed'
  }
}

async function handleCreate() {
  loading.value = true
  errorMessage.value = null

  try {
    created.value = await createContact({
      ...createForm,
      user_id: Number(createForm.user_id),
      contact_id: Number(createForm.contact_id) || undefined,
      category: Number(createForm.category),
      importance: Number(createForm.importance),
    })
    await refreshContacts()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Create contact failed'
  } finally {
    loading.value = false
  }
}

async function handleGet() {
  loading.value = true
  errorMessage.value = null

  try {
    rows.value = await getContactsByUserId(getForm.user_id)
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Get contacts failed'
  } finally {
    loading.value = false
  }
}

async function handleDelete() {
  loading.value = true
  errorMessage.value = null

  try {
    await deleteContactByIds(deleteForm.user_id, deleteForm.contact_id)
    created.value = null
    await refreshContacts()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Delete contact failed'
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await refreshContacts()
  pollTimer = window.setInterval(() => {
    void refreshContacts()
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
    <FormSection title="Create contact link" description="Привязка контакта к пользователю.">
      <form class="form-grid" @submit.prevent="handleCreate">
        <label class="field"><span>User ID</span><input v-model="createForm.user_id" type="number" min="1" required /></label>
        <label class="field"><span>Contact ID (optional)</span><input v-model="createForm.contact_id" type="number" min="0" /></label>
        <label class="field"><span>Value</span><input v-model="createForm.value" required /></label>
        <label class="field"><span>Category</span><input v-model="createForm.category" type="number" min="0" required /></label>
        <label class="field"><span>Importance</span><input v-model="createForm.importance" type="number" min="0" required /></label>
        <button class="button" :disabled="loading">Create</button>
      </form>
    </FormSection>

    <FormSection title="Get contacts" description="Чтение контактных связей пользователя.">
      <form class="form-grid" @submit.prevent="handleGet">
        <label class="field"><span>User ID</span><input v-model="getForm.user_id" required /></label>
        <button class="button" :disabled="loading">Get</button>
      </form>
    </FormSection>

    <FormSection title="Delete contact link" description="Удаление связи пользователя и контакта.">
      <form class="form-grid" @submit.prevent="handleDelete">
        <label class="field"><span>User ID</span><input v-model="deleteForm.user_id" required /></label>
        <label class="field"><span>Contact ID</span><input v-model="deleteForm.contact_id" required /></label>
        <button class="button" :disabled="loading">Delete</button>
      </form>
    </FormSection>

    <ErrorAlert :message="errorMessage" />
    <DetailCard :value="created" />
    <DataTable :rows="rows" />
  </section>
</template>
