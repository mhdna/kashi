import { API_BASE } from '@/config'
import { authFetch } from './useApi'

async function requestJSON (url, options) {
  const res = await authFetch(url, options)
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Request failed with status ${res.status}`)
  }
  return res.json().catch(() => null)
}

function reportQuery (params) {
  const query = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== null && value !== undefined && value !== '') {
      query.set(key, String(value))
    }
  }
  return query.toString()
}

// Emailed invoice reports (api/email_report.go): a day, a date range or one
// shift's invoices as a CSV, sent through Resend. Settings live on
// app_settings.email_* (Settings > Email); every send is logged for the
// Emails page.
export function useEmailReports () {
  // { settings, columns: [{ key, header }], configured }
  async function getEmailSettings () {
    return requestJSON(`${API_BASE}/email_settings`)
  }

  async function updateEmailSettings (payload) {
    return requestJSON(`${API_BASE}/email_settings`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
  }

  // params: { scope: 'day' | 'range' | 'shift', from, to, branch_id, shift_id }
  async function previewEmailReport (params) {
    return requestJSON(`${API_BASE}/email_reports/preview?${reportQuery(params)}`)
  }

  async function sendEmailReport (params, recipients) {
    const data = await requestJSON(`${API_BASE}/email_reports`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...params, recipients }),
    })
    return data?.report ?? null
  }

  async function listEmailReportShifts (from, to, branchId) {
    const data = await requestJSON(`${API_BASE}/email_reports/shifts?${reportQuery({ from, to, branch_id: branchId })}`)
    return data?.shifts ?? []
  }

  // { recipients: every address sent to, newest first, last: the last send's }
  async function listEmailReportRecipients () {
    const data = await requestJSON(`${API_BASE}/email_reports/recipients`)
    return { recipients: data?.recipients ?? [], last: data?.last ?? [] }
  }

  // page_id is an offset, like the WhatsApp message list's.
  async function listEmailReports ({ status, pageSize, offset }) {
    const data = await requestJSON(`${API_BASE}/email_reports?${reportQuery({ status, page_size: pageSize, page_id: offset })}`)
    return { reports: data?.reports ?? [], total: data?.total ?? 0 }
  }

  return {
    getEmailSettings,
    updateEmailSettings,
    previewEmailReport,
    sendEmailReport,
    listEmailReportShifts,
    listEmailReportRecipients,
    listEmailReports,
  }
}
