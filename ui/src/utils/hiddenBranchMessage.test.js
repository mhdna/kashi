import { describe, expect, it } from 'vitest'
import { hiddenBranchMessage } from './hiddenBranchMessage'

describe('hiddenBranchMessage', () => {
  it('the reported case: hiding Test Branch 2 says its clients leave the other tills', () => {
    expect(hiddenBranchMessage('Test Branch 2', true, 4)).toBe('Test Branch 2 is out of reports; its 4 clients will leave the other tills')
  })
  it('one client is singular', () => {
    expect(hiddenBranchMessage('T', true, 1)).toBe('T is out of reports; its 1 client will leave the other tills')
    expect(hiddenBranchMessage('T', false, 1)).toBe('T is back in reports; its 1 client will return to the other tills')
  })
  it('no clients, or an older server that sends no count', () => {
    expect(hiddenBranchMessage('T', true, 0)).toBe('T is out of reports and keeps its clients to itself')
    expect(hiddenBranchMessage('T', false, undefined)).toBe('T is back in reports')
  })
})
