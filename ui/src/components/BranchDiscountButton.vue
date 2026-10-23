<!--
  App-bar shortcut: set a product's discount on one branch without opening
  the Discount Lists page. It changes the product in the list that branch is
  discounted from (api/branch_discount.go) -- not just one invoice -- and
  the branch's till re-prices it in its open cart within moments.
-->
<template>
  <v-btn
    v-if="auth.canEdit('pricing')"
    icon="mdi-sale"
    title="Set a branch discount"
    variant="text"
    @click="open = true"
  />

  <v-dialog v-model="open" max-width="520">
    <v-card>
      <v-card-title>Branch discount</v-card-title>
      <v-card-text>
        <v-select
          v-model="branchId"
          density="compact"
          :items="branchItems"
          label="Branch"
          variant="outlined"
        />
        <v-text-field
          v-model="entry"
          autofocus
          class="mt-2"
          density="compact"
          hint="A code covers all its colours and sizes; a barcode just that one"
          label="Product code or barcode"
          :loading="looking"
          persistent-hint
          variant="outlined"
          @blur="lookUp"
          @keyup.enter="lookUp"
        />

        <template v-if="found">
          <div class="mt-4 text-subtitle-2">{{ found.variants[0].product_name }}</div>
          <v-table class="mt-1" density="compact">
            <thead>
              <tr>
                <th>Variant</th>
                <th>Barcode</th>
                <th class="text-end">Now</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="v in found.variants" :key="v.id">
                <td>{{ [v.color_name, v.size_name].filter(Boolean).join(' / ') || '—' }}</td>
                <td>{{ v.barcode }}</td>
                <td class="text-end">{{ v.discount ? `${v.discount}%` : '—' }}</td>
              </tr>
            </tbody>
          </v-table>

          <v-alert
            class="mt-3"
            density="compact"
            :type="found.list.is_default && found.list.branches > 1 ? 'warning' : 'info'"
            variant="tonal"
          >
            <template v-if="found.list.is_default && found.list.branches > 1">
              This branch uses the default list, <strong>{{ found.list.name }}</strong>: the discount
              applies to all {{ found.list.branches }} branches without a list of their own.
            </template>
            <template v-else>
              Goes into <strong>{{ found.list.name }}</strong>, this branch's discount list.
            </template>
          </v-alert>

          <v-text-field
            v-model="discount"
            class="mt-3"
            density="compact"
            label="Discount"
            max="100"
            min="0"
            style="max-width: 160px"
            suffix="%"
            type="number"
            variant="outlined"
            @keyup.enter="save"
          />
        </template>
        <v-alert
          v-else-if="notFound"
          class="mt-3"
          density="compact"
          type="warning"
          variant="tonal"
        >
          No product has that code or barcode.
        </v-alert>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn text="Close" variant="text" @click="open = false" />
        <v-btn
          color="primary"
          :disabled="!found || discount === ''"
          :loading="saving"
          text="Set discount"
          variant="flat"
          @click="save"
        />
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
  import { computed, ref, watch } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { useBranches } from '@/composables/useBranches'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'
  import { useAuthStore } from '@/stores/auth'

  const auth = useAuthStore()
  const notify = useNotify()
  const { branches, fetchBranches } = useBranches()

  const LAST_BRANCH_KEY = 'branchDiscount.branchId'

  const open = ref(false)
  const branchId = ref(null)
  const entry = ref('')
  const discount = ref('')
  const found = ref(null)
  const notFound = ref(false)
  const looking = ref(false)
  const saving = ref(false)
  let lookedUp = ''

  const branchItems = computed(() => branches.value
    .filter(b => b.is_active)
    .map(b => ({ title: b.name, value: b.id })))

  async function request (method, body) {
    const url = method === 'GET'
      ? `${API_BASE}/discount_lists/branch/${branchId.value}?${new URLSearchParams({ entry: entry.value.trim() })}`
      : `${API_BASE}/discount_lists/branch/${branchId.value}`
    const res = await authFetch(url, {
      method,
      headers: body ? { 'Content-Type': 'application/json' } : undefined,
      body: body ? JSON.stringify(body) : undefined,
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.error || `Request failed with status ${res.status}`)
    return data
  }

  // What the entry is discounted at on the branch now, and where a change goes.
  async function lookUp () {
    const key = `${branchId.value}:${entry.value.trim()}`
    if (!branchId.value || !entry.value.trim() || key === lookedUp) return
    lookedUp = key
    looking.value = true
    notFound.value = false
    try {
      const data = await request('GET')
      if (data.variants?.length) {
        found.value = data
        const current = new Set(data.variants.map(v => v.discount))
        discount.value = current.size === 1 ? String(data.variants[0].discount) : ''
      } else {
        found.value = null
        notFound.value = true
      }
    } catch (error) {
      found.value = null
      lookedUp = ''
      notify.error(error.message)
    } finally {
      looking.value = false
    }
  }

  async function save () {
    const value = Number(discount.value)
    if (!found.value || !Number.isInteger(value) || value < 0 || value > 100) {
      notify.error('Enter a discount from 0 to 100.')
      return
    }
    saving.value = true
    try {
      const data = await request('PUT', { entry: entry.value.trim(), discount: value })
      const branch = branchItems.value.find(b => b.value === branchId.value)?.title ?? 'the branch'
      notify.success(`${found.value.variants[0].product_name}: ${value}% on ${branch} (${data.list.name}). The till updates now.`)
      lookedUp = ''
      await lookUp()
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  watch(open, async value => {
    if (!value) return
    entry.value = ''
    discount.value = ''
    found.value = null
    notFound.value = false
    lookedUp = ''
    try {
      await fetchBranches()
    } catch (error) {
      notify.error(error.message)
    }
    let last = null
    try {
      last = Number(localStorage.getItem(LAST_BRANCH_KEY)) || null
    } catch {}
    branchId.value = branchItems.value.some(b => b.value === last) ? last : (branchItems.value[0]?.value ?? null)
  })

  // The branch is remembered between uses, and changing it looks the entry
  // up again for that branch.
  watch(branchId, value => {
    try {
      if (value) localStorage.setItem(LAST_BRANCH_KEY, String(value))
    } catch {}
    lookedUp = ''
    if (entry.value.trim()) lookUp()
  })
</script>
