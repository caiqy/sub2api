import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AppSidebar from '../AppSidebar.vue'

const {
  routeState,
  authState,
  appState,
  adminSettingsState,
  onboardingState,
  fetchMock
} = vi.hoisted(() => ({
  routeState: { path: '/dashboard' },
  authState: {
    isAdmin: false,
    isSimpleMode: false
  },
  appState: {
    sidebarCollapsed: false,
    mobileOpen: false,
    backendModeEnabled: false,
    siteName: 'Sub2API',
    siteLogo: '/logo.png',
    siteVersion: '1.0.0',
    publicSettingsLoaded: true,
    cachedPublicSettings: {
      payment_enabled: false,
      custom_menu_items: []
    }
  },
  adminSettingsState: {
    opsMonitoringEnabled: false,
    paymentEnabled: false,
    customMenuItems: [] as Array<Record<string, unknown>>
  },
  onboardingState: {
    isCurrentStep: vi.fn(() => false),
    nextStep: vi.fn()
  },
  fetchMock: vi.fn()
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => routeState,
    useRouter: () => ({
      push: vi.fn()
    })
  }
})

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/stores', () => ({
  useAuthStore: () => authState,
  useAppStore: () => ({
    ...appState,
    toggleSidebar: vi.fn(),
    setMobileOpen: vi.fn()
  }),
  useAdminSettingsStore: () => ({
    ...adminSettingsState,
    fetch: fetchMock
  }),
  useOnboardingStore: () => onboardingState
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appState
}))

vi.mock('@/utils/sanitize', () => ({
  sanitizeSvg: (svg: string) => svg
}))

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

const RouterLinkStub = defineComponent({
  name: 'RouterLinkStub',
  props: {
    to: {
      type: [String, Object],
      required: true
    }
  },
  setup(props, { slots, attrs }) {
    return () => h('a', {
      ...attrs,
      'data-to': typeof props.to === 'string' ? props.to : (props.to as { path?: string }).path
    }, slots.default?.())
  }
})

function mountSidebar() {
  return mount(AppSidebar, {
    global: {
      stubs: {
        VersionBadge: {
          template: '<div data-testid="version-badge" />'
        },
        RouterLink: RouterLinkStub,
        transition: false
      }
    }
  })
}

function setScenario(options: {
  isAdmin: boolean
  isSimpleMode: boolean
  backendModeEnabled?: boolean
}) {
  routeState.path = '/dashboard'
  authState.isAdmin = options.isAdmin
  authState.isSimpleMode = options.isSimpleMode
  appState.backendModeEnabled = options.backendModeEnabled ?? false
  appState.sidebarCollapsed = false
  appState.mobileOpen = false
  appState.cachedPublicSettings = {
    payment_enabled: false,
    custom_menu_items: []
  }
  adminSettingsState.opsMonitoringEnabled = false
  adminSettingsState.paymentEnabled = false
  adminSettingsState.customMenuItems = []
}

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})

describe('AppSidebar images entry wiring', () => {
  beforeEach(() => {
    fetchMock.mockReset()
    onboardingState.isCurrentStep.mockClear()
    onboardingState.nextStep.mockClear()
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn()
    })))
    localStorage.clear()
    document.documentElement.classList.remove('dark')
  })

  it('renders /images for a regular authenticated user', () => {
    setScenario({ isAdmin: false, isSimpleMode: false })

    const wrapper = mountSidebar()

    const imagesLinks = wrapper.findAll('[data-to="/images"]')
    expect(imagesLinks).toHaveLength(1)
    expect(imagesLinks[0].text()).toContain('nav.aiImages')
  })

  it('renders /images inside the admin personal menu', () => {
    setScenario({ isAdmin: true, isSimpleMode: false })

    const wrapper = mountSidebar()

    expect(wrapper.text()).toContain('nav.myAccount')
    const imagesLinks = wrapper.findAll('[data-to="/images"]')
    expect(imagesLinks).toHaveLength(1)
    expect(imagesLinks[0].text()).toContain('nav.aiImages')
  })

  it('renders /images in admin simple mode through the admin branch', () => {
    setScenario({ isAdmin: true, isSimpleMode: true })

    const wrapper = mountSidebar()

    expect(wrapper.text()).not.toContain('nav.myAccount')
    const imagesLinks = wrapper.findAll('[data-to="/images"]')
    expect(imagesLinks).toHaveLength(1)
    expect(imagesLinks[0].text()).toContain('nav.aiImages')
    expect(fetchMock).toHaveBeenCalled()
  })
})
