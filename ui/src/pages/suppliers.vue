<template>
  <v-dialog v-model="dialog" max-width="560">
    <v-card class="px-4">
      <v-card-title>{{ t('Add a New Supplier') }}</v-card-title>
      <v-card-text>
        <form @submit.prevent="submit">
          <v-text-field
            v-model="name.value.value"
            density="compact"
            :error-messages="name.errorMessage.value"
            :label="t('Name')"
          />
          <v-text-field
            v-model="phone.value.value"
            density="compact"
            :error-messages="phone.errorMessage.value"
            :label="t('Phone')"
          />
          <v-text-field
            v-model="country.value.value"
            density="compact"
            :error-messages="country.errorMessage.value"
            :label="t('Country')"
          />
          <v-text-field
            v-model="address.value.value"
            density="compact"
            :error-messages="address.errorMessage.value"
            :label="t('Address')"
          />
          <v-row>
            <v-col cols="6">
              <v-text-field
                v-model.number="latitude.value.value"
                density="compact"
                :error-messages="latitude.errorMessage.value"
                :label="t('Latitude')"
                type="number"
              />
            </v-col>
            <v-col cols="6">
              <v-text-field
                v-model.number="longitude.value.value"
                density="compact"
                :error-messages="longitude.errorMessage.value"
                :label="t('Longitude')"
                type="number"
              />
            </v-col>
          </v-row>

          <v-card-actions class="px-0">
            <v-spacer />
            <v-btn :text="t('Cancel')" @click="closeDialog" />
            <v-btn color="primary" :loading="submitting" :text="t('Add Supplier')" type="submit" />
          </v-card-actions>
        </form>
      </v-card-text>
    </v-card>
  </v-dialog>

  <div class="page-root">
    <v-card class="suppliers-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-truck-delivery" />
        <span>{{ t('Suppliers') }}</span>
        <v-spacer />
        <v-text-field
          v-model="search"
          class="supplier-search"
          clearable
          density="compact"
          hide-details
          :label="t('Search suppliers')"
          prepend-inner-icon="mdi-magnify"
          variant="outlined"
        />
        <v-btn
          v-if="canEdit"
          color="primary"
          prepend-icon="mdi-plus"
          :text="t('Add Supplier')"
          variant="flat"
          @click="dialog = true"
        />
      </v-card-title>
      <v-divider />

      <ServerSideTable
        ref="tableRef"
        :api-u-r-l="apiURL"
        density="comfortable"
        :external-search="search"
        flush
        :headers="headers"
        root-key="suppliers"
        :show-search-icon="false"
      />
    </v-card>
  </div>
</template>

<script setup lang="ts">
  import { useField, useForm } from 'vee-validate'
  import { computed, ref } from 'vue'
  import { t } from '@/i18n'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { authFetch } from '@/composables/useApi'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()

  const apiURL = `${API_BASE}/suppliers`
  const headers = computed(() => [
    { title: t('ID'), key: 'id', align: 'start' },
    { title: t('Name'), key: 'name', align: 'start' },
    { title: t('Phone'), key: 'phone', align: 'start' },
    { title: t('Country'), key: 'country', align: 'start' },
    { title: t('Address'), key: 'address', align: 'start' },
    { title: t('Created At'), key: 'created_at', align: 'end' },
  ])

  const tableRef = ref(null)
  const search = ref('')
  const dialog = ref(false)
  const submitting = ref(false)
  const notify = useNotify()

  const { handleSubmit, handleReset } = useForm({
    validationSchema: {
      name (value) {
        if (value?.length >= 2) return true
        return t('Name needs to be at least 2 characters.')
      },
      phone (value) {
        if (/^[0-9-]{7,}$/.test(value)) return true
        return t('Phone number needs to be at least 7 digits.')
      },
      country (value) {
        if (value?.length >= 2) return true
        return t('Country is required.')
      },
      address (value) {
        if (value?.length >= 2) return true
        return t('Address is required.')
      },
      latitude (value) {
        if (value !== undefined && value !== null && value !== '') return true
        return t('Latitude is required.')
      },
      longitude (value) {
        if (value !== undefined && value !== null && value !== '') return true
        return t('Longitude is required.')
      },
    },
  })

  const name = useField('name')
  const phone = useField('phone')
  const country = useField('country')
  const address = useField('address')
  const latitude = useField('latitude')
  const longitude = useField('longitude')

  function closeDialog () {
    dialog.value = false
    handleReset()
  }

  const submit = handleSubmit(async values => {
    submitting.value = true
    try {
      const res = await authFetch(apiURL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: values.name,
          phone: values.phone,
          country: values.country,
          address: values.address,
          latitude: values.latitude,
          longitude: values.longitude,
        }),
      })

      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error || t('Request failed with status {status}', { status: res.status }))
      }

      closeDialog()
      tableRef.value?.reload()
    } catch (error) {
      notify.error(error.message)
    } finally {
      submitting.value = false
    }
  })
</script>

<style scoped>
.page-root {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
}

.suppliers-card {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
}

.supplier-search {
  flex: 0 1 320px;
}
</style>
