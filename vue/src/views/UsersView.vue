<script setup lang="ts">
import { onMounted, onUnmounted, reactive, ref } from 'vue'
import { createUser, deleteUserById, getUserById, listUsers, updateUserById } from '../api/users'
import DataTable from '../components/DataTable.vue'
import DetailCard from '../components/DetailCard.vue'
import ErrorAlert from '../components/ErrorAlert.vue'
import FormSection from '../components/FormSection.vue'

const createForm = reactive({
  email: '',
})

const getForm = reactive({
  id: '',
})

const updateForm = reactive({
  id: '',
  email: '',
})

const deleteForm = reactive({
  id: '',
})

const result = ref<Record<string, unknown> | null>(null)
const rows = ref<Record<string, unknown>[]>([])
const errorMessage = ref<string | null>(null)
const loading = ref(false)
let pollTimer: number | undefined

async function refreshUsers() {
  try {
    rows.value = (await listUsers()) ?? []
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'List users failed'
  }
}

async function handleCreate() {
  loading.value = true
  errorMessage.value = null

  try {
    result.value = await createUser({ email: createForm.email })
    await refreshUsers()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Create user failed'
  } finally {
    loading.value = false
  }
}

async function handleGet() {
  loading.value = true
  errorMessage.value = null

  try {
    result.value = await getUserById(getForm.id)
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Get user failed'
  } finally {
    loading.value = false
  }
}

async function handleDelete() {
  loading.value = true
  errorMessage.value = null

  try {
    await deleteUserById(deleteForm.id)
    result.value = null
    await refreshUsers()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Delete user failed'
  } finally {
    loading.value = false
  }
}

async function handleUpdate() {
  loading.value = true
  errorMessage.value = null

  try {
    result.value = await updateUserById(updateForm.id, updateForm.email)
    await refreshUsers()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Update user failed'
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await refreshUsers()
  pollTimer = window.setInterval(() => {
    void refreshUsers()
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
    <FormSection title="Create user" description="Создание пользователя. ID генерирует backend.">
      <form class="form-grid" @submit.prevent="handleCreate">
        <label class="field">
          <span>Email</span>
          <input v-model="createForm.email" type="email" required />
        </label>
        <button class="button" :disabled="loading">Create</button>
      </form>
    </FormSection>

    <FormSection title="Get user" description="Чтение пользователя по ID.">
      <form class="form-grid" @submit.prevent="handleGet">
        <label class="field">
          <span>User ID</span>
          <input v-model="getForm.id" type="text" required />
        </label>
        <button class="button" :disabled="loading">Get</button>
      </form>
    </FormSection>

    <FormSection title="Update user" description="Обновление пользователя по ID.">
      <form class="form-grid" @submit.prevent="handleUpdate">
        <label class="field">
          <span>User ID</span>
          <input v-model="updateForm.id" type="text" required />
        </label>
        <label class="field">
          <span>Email</span>
          <input v-model="updateForm.email" type="email" required />
        </label>
        <button class="button" :disabled="loading">Update</button>
      </form>
    </FormSection>

    <FormSection title="Delete user" description="Удаление пользователя по ID.">
      <form class="form-grid" @submit.prevent="handleDelete">
        <label class="field">
          <span>User ID</span>
          <input v-model="deleteForm.id" type="text" required />
        </label>
        <button class="button" :disabled="loading">Delete</button>
      </form>
    </FormSection>

    <ErrorAlert :message="errorMessage" />
    <DetailCard :value="result" />
    <DataTable :rows="rows" />
  </section>
</template>
