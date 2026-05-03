<script setup lang="ts">
import { reactive, ref } from 'vue'
import { createEmail, getEmailsByUserId } from '../api/emails'
import DataTable from '../components/DataTable.vue'
import DetailCard from '../components/DetailCard.vue'
import ErrorAlert from '../components/ErrorAlert.vue'
import FormSection from '../components/FormSection.vue'

const createForm = reactive({
  user_id: '',
  address: '',
  category: '',
  importance: 0,
})

const getForm = reactive({
  user_id: '',
})

const created = ref<Record<string, unknown> | null>(null)
const rows = ref<Record<string, unknown>[]>([])
const errorMessage = ref<string | null>(null)
const loading = ref(false)

async function handleCreate() {
  loading.value = true
  errorMessage.value = null

  try {
    created.value = await createEmail({
      ...createForm,
      importance: Number(createForm.importance),
    })
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Create email failed'
  } finally {
    loading.value = false
  }
}

async function handleGet() {
  loading.value = true
  errorMessage.value = null

  try {
    rows.value = await getEmailsByUserId(getForm.user_id)
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Get emails failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="page-grid">
    <FormSection title="Create email link" description="Привязка email к пользователю.">
      <form class="form-grid" @submit.prevent="handleCreate">
        <label class="field"><span>User ID</span><input v-model="createForm.user_id" required /></label>
        <label class="field"><span>Address</span><input v-model="createForm.address" type="email" required /></label>
        <label class="field"><span>Category</span><input v-model="createForm.category" required /></label>
        <label class="field"><span>Importance</span><input v-model="createForm.importance" type="number" min="0" required /></label>
        <button class="button" :disabled="loading">Create</button>
      </form>
    </FormSection>

    <FormSection title="Get emails" description="Чтение email-связей пользователя.">
      <form class="form-grid" @submit.prevent="handleGet">
        <label class="field"><span>User ID</span><input v-model="getForm.user_id" required /></label>
        <button class="button" :disabled="loading">Get</button>
      </form>
    </FormSection>

    <ErrorAlert :message="errorMessage" />
    <DetailCard :value="created" />
    <DataTable :rows="rows" />
  </section>
</template>
