import { describe, expect, it } from 'vitest'
import { buildOpenWorkQrCodeUrl, normalizeOpenWorkAuthStatus } from '../../app/utils/openworkAuth'

describe('openworkAuth utils', () => {
  it('buildOpenWorkQrCodeUrl returns empty string for blank input', () => {
    expect(buildOpenWorkQrCodeUrl('')).toBe('')
    expect(buildOpenWorkQrCodeUrl('   ')).toBe('')
  })

  it('buildOpenWorkQrCodeUrl encodes authorize url', () => {
    const source = 'https://open.work.weixin.qq.com/3rdapp/install?suite_id=sid&state=s-1'
    const url = buildOpenWorkQrCodeUrl(source)
    expect(url).toContain('https://quickchart.io/qr?')
    expect(url).toContain(encodeURIComponent(source))
  })

  it('normalizeOpenWorkAuthStatus keeps known status and falls back to pending', () => {
    expect(normalizeOpenWorkAuthStatus('authorized')).toBe('authorized')
    expect(normalizeOpenWorkAuthStatus('failed')).toBe('failed')
    expect(normalizeOpenWorkAuthStatus('')).toBe('pending')
    expect(normalizeOpenWorkAuthStatus('unknown')).toBe('pending')
  })
})
