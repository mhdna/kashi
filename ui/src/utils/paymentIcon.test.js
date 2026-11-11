import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { paymentIcon } from './paymentIcon'

describe('paymentIcon', () => {
  it('the four methods of the till each look like what they are', () => {
    expect(paymentIcon('Cash $$')).toBe('mdi-cash-multiple')
    expect(paymentIcon('Cash LL')).toBe('mdi-cash')
    expect(paymentIcon('Visa')).toBe('mdi-credit-card-outline')
    expect(paymentIcon('Whish')).toBe('mdi-cellphone')
    expect(new Set(['Cash $$', 'Cash LL', 'Visa', 'Whish'].map(n => paymentIcon(n))).size).toBe(4)
  })
  it('reads any casing or spelling', () => {
    for (const n of ['VISA', 'visa debit', 'Master Card', 'Credit card']) {expect(paymentIcon(n), n).toBe('mdi-credit-card-outline')}
    for (const n of ['WHISH', 'Whish Money']) {expect(paymentIcon(n), n).toBe('mdi-cellphone')}
    for (const n of ['Cash LBP', 'LL', 'Lebanese Lira']) {expect(paymentIcon(n), n).toBe('mdi-cash')}
    for (const n of ['Bank', 'Bank transfer', 'Cheque']) {expect(paymentIcon(n), n).toBe('mdi-bank')}
  })
  it('anything else, empty or missing is plain cash', () => {
    for (const n of ['Cash', 'Cash USD', 'Other', '', null, undefined, 7]) {expect(paymentIcon(n), String(n)).toBe('mdi-cash-multiple')}
    expect(paymentIcon('Bella')).toBe('mdi-cash-multiple')
  })
  it('every icon it returns is registered in icons.js (CLAUDE.md: treeshaken)', () => {
    const icons = readFileSync(resolve(process.cwd(), 'src/plugins/icons.js'), 'utf8')
    for (const n of ['Visa', 'Whish', 'Bank', 'Cash LL', 'Cash $$']) {expect(icons, n).toContain(`'${paymentIcon(n)}':`)}
  })
})
