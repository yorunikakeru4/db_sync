<script setup lang="ts">
import { reactive, ref } from 'vue'
import { createUser, getUserById } from '../api/users'
import DetailCard from '../components/DetailCard.vue'
import ErrorAlert from '../components/ErrorAlert.vue'
import FormSection from '../components/FormSection.vue'

const createForm = reactive({
  email: '',
})

const getForm = reactive({
  id: '',
})

const result = ref<Record<string, unknown> | null>(null)
const errorMessage = ref<string | null>(null)
const loading = ref(false)

async function handleCreate() {
  loading.value = true
  errorMessage.value = null

  try {
    result.value = await createUser({ email: createForm.email })
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

    <ErrorAlert :message="errorMessage" />
    <DetailCard :value="result" />
  </section>
</template>
