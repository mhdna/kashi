import { API_BASE } from '@/config'
import { branchMethods } from '@/utils/paymentEntry'
import { authFetch } from './useApi'
import { useBranchPaymentMethods } from './useBranchPaymentMethods'
import { useBranchSettings } from './useBranchSettings'
import { useCurrencies } from './useCurrencies'

// What the payment calculator (PaymentEntryPanel) needs from kashi for one
// branch, and waiting on the till to run what it sent.

async function requestJSON (url) {
  const res = await authFetch(url)
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Request failed with status ${res.status}`)
  }
  return res.json().catch(() => null)
}

export function usePaymentEntry () {
  const { listBranchPaymentMethods } = useBranchPaymentMethods()
  const { fetchCurrencies } = useCurrencies()
  const { listBranchCommands } = useBranchSettings()

  // The branch's payment methods (branchMethods) and every currency's rate.
  async function loadPaymentSetup (branchId) {
    const [methods, currencies] = await Promise.all([listBranchPaymentMethods(branchId), fetchCurrencies()])
    return { methods: branchMethods(methods), currencies: currencies ?? [] }
  }

  // { variantId: cents } at the branch; a variant with no price is missing.
  async function variantPrices (branchId, variantIds) {
    if (variantIds.length === 0) {return {}}
    const data = await requestJSON(`${API_BASE}/branches/${branchId}/variant_prices?ids=${variantIds.join(',')}`)
    return data?.prices ?? {}
  }

  // Polls the command until the till ran it (success or failed) or ~15s pass
  // (pending: the till is offline and runs it when it's back). shouldStop
  // ends it early, when the dialog closed.
  async function waitForCommand (branchId, commandId, shouldStop = () => false) {
    if (!commandId) {return { status: 'pending', error: '' }}
    for (let attempt = 0; attempt < 10; attempt++) {
      if (attempt > 0) {await new Promise(resolve => setTimeout(resolve, 1500))}
      if (shouldStop()) {return null}
      let commands = []
      try {
        commands = await listBranchCommands(branchId)
      } catch {
        commands = []
      }
      const command = commands.find(c => c.id === commandId)
      if (command && command.status !== 'pending') {
        return { status: command.status, error: command.error?.Valid ? command.error.String : '' }
      }
    }
    return { status: 'pending', error: '' }
  }

  return { loadPaymentSetup, variantPrices, waitForCommand }
}
