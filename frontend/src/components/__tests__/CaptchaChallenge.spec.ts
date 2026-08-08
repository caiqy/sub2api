import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CaptchaChallenge from '@/components/CaptchaChallenge.vue'

const turnstileResetMock = vi.fn()

const TurnstileWidgetStub = defineComponent({
  emits: ['verify', 'expire', 'error'],
  setup(_, { emit, expose }) {
    expose({ reset: turnstileResetMock })
    return () => h('div', [
      h('button', { 'data-testid': 'verify', onClick: () => emit('verify', 'turnstile-token') }),
      h('button', { 'data-testid': 'expire', onClick: () => emit('expire') }),
      h('button', { 'data-testid': 'error', onClick: () => emit('error') })
    ])
  }
})

type CaptchaChallengeExposed = {
  reset: () => void
  verifyAction: () => Promise<{ token: string; randstr: string } | null>
}

function exposed(wrapper: ReturnType<typeof mount>): CaptchaChallengeExposed {
  return wrapper.vm as unknown as CaptchaChallengeExposed
}

describe('CaptchaChallenge', () => {
  it('returns a completed Turnstile token until it expires, errors, or resets', async () => {
    turnstileResetMock.mockReset()
    const wrapper = mount(CaptchaChallenge, {
      props: {
        turnstileEnabled: true,
        turnstileSiteKey: 'site-key',
        tencentEnabled: false,
        tencentAppId: ''
      },
      global: {
        stubs: {
          TurnstileWidget: TurnstileWidgetStub,
          TencentCaptchaGate: true,
          AliyunCaptchaWidget: true
        }
      }
    })

    await wrapper.get('[data-testid="verify"]').trigger('click')
    await expect(exposed(wrapper).verifyAction()).resolves.toEqual({
      token: 'turnstile-token',
      randstr: ''
    })

    await wrapper.get('[data-testid="expire"]').trigger('click')
    await expect(exposed(wrapper).verifyAction()).resolves.toBeNull()

    await wrapper.get('[data-testid="verify"]').trigger('click')
    await wrapper.get('[data-testid="error"]').trigger('click')
    await expect(exposed(wrapper).verifyAction()).resolves.toBeNull()

    await wrapper.get('[data-testid="verify"]').trigger('click')
    exposed(wrapper).reset()
    await expect(exposed(wrapper).verifyAction()).resolves.toBeNull()
    expect(turnstileResetMock).toHaveBeenCalledOnce()
  })
})
