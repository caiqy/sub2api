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

    await wrapper.setProps({ turnstileSiteKey: 'rotated-site-key' })
    await expect(exposed(wrapper).verifyAction()).resolves.toBeNull()

    await wrapper.get('[data-testid="verify"]').trigger('click')
    await wrapper.setProps({ turnstileEnabled: false })
    await wrapper.setProps({ turnstileEnabled: true })
    await expect(exposed(wrapper).verifyAction()).resolves.toBeNull()

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

  it('renders no provider and fails closed when multiple providers are enabled', async () => {
    turnstileResetMock.mockReset()
    const wrapper = mount(CaptchaChallenge, {
      props: {
        turnstileEnabled: true,
        turnstileSiteKey: 'site-key',
        tencentEnabled: true,
        tencentAppId: 'tencent-app-id',
        aliyunEnabled: true,
        aliyunSceneId: 'scene-id',
        aliyunPrefix: 'prefix'
      },
      global: {
        stubs: {
          TurnstileWidget: TurnstileWidgetStub,
          TencentCaptchaGate: true,
          AliyunCaptchaWidget: true
        }
      }
    })

    expect(wrapper.findComponent(TurnstileWidgetStub).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'TencentCaptchaGate' }).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'AliyunCaptchaWidget' }).exists()).toBe(false)
    await expect(exposed(wrapper).verifyAction()).resolves.toBeNull()
  })

  it('renders no provider and fails closed when one enabled provider is complete but another enabled provider is not', async () => {
    turnstileResetMock.mockReset()
    const wrapper = mount(CaptchaChallenge, {
      props: {
        turnstileEnabled: true,
        turnstileSiteKey: 'site-key',
        tencentEnabled: true,
        tencentAppId: '',
        aliyunEnabled: false
      },
      global: {
        stubs: {
          TurnstileWidget: TurnstileWidgetStub,
          TencentCaptchaGate: true,
          AliyunCaptchaWidget: true
        }
      }
    })

    expect(wrapper.findComponent(TurnstileWidgetStub).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'TencentCaptchaGate' }).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'AliyunCaptchaWidget' }).exists()).toBe(false)
    await expect(exposed(wrapper).verifyAction()).resolves.toBeNull()
  })

  it('renders no provider and fails closed when the only enabled provider is incompletely configured', async () => {
    turnstileResetMock.mockReset()
    const wrapper = mount(CaptchaChallenge, {
      props: {
        turnstileEnabled: false,
        turnstileSiteKey: '',
        tencentEnabled: true,
        tencentAppId: '',
        aliyunEnabled: true,
        aliyunSceneId: 'scene-id',
        aliyunPrefix: ''
      },
      global: {
        stubs: {
          TurnstileWidget: TurnstileWidgetStub,
          TencentCaptchaGate: true,
          AliyunCaptchaWidget: true
        }
      }
    })

    expect(wrapper.findComponent(TurnstileWidgetStub).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'TencentCaptchaGate' }).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'AliyunCaptchaWidget' }).exists()).toBe(false)
    await expect(exposed(wrapper).verifyAction()).resolves.toBeNull()
  })
})
