<template>
  <TurnstileWidget
    v-if="validProvider === 'turnstile'"
    ref="turnstileRef"
    :site-key="turnstileSiteKey"
    @verify="onTurnstileVerify"
    @expire="onTurnstileExpire"
    @error="onTurnstileError"
  />
  <TencentCaptchaGate
    v-else-if="validProvider === 'tencent'"
    ref="tencentRef"
    :app-id="tencentAppId"
    :region="tencentRegion"
  />
  <AliyunCaptchaWidget
    v-else-if="validProvider === 'aliyun'"
    ref="aliyunRef"
    :scene-id="aliyunSceneId ?? ''"
    :prefix="aliyunPrefix ?? ''"
    :region="aliyunRegion === 'sgp' ? 'sgp' : 'cn'"
    @verify="(param: string) => emit('verify', param, '')"
    @expire="emit('expire')"
    @error="emit('error')"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import TurnstileWidget from '@/components/TurnstileWidget.vue'
import TencentCaptchaGate from '@/components/TencentCaptchaGate.vue'
import AliyunCaptchaWidget from '@/components/AliyunCaptchaWidget.vue'

// ActionCaptchaResult: Turnstile 复用已完成 token；腾讯/阿里云在动作时获取 proof。
// 腾讯 token=ticket、randstr 非空；阿里云 token=captchaVerifyParam、randstr 恒为空。
export interface ActionCaptchaResult {
  token: string
  randstr: string
}

const props = defineProps<{
  siteKey?: string
  turnstileEnabled: boolean
  turnstileSiteKey: string
  tencentEnabled: boolean
  tencentAppId: string
  tencentRegion?: string
  aliyunEnabled?: boolean
  aliyunSceneId?: string
  aliyunPrefix?: string
  aliyunRegion?: string
}>()

const emit = defineEmits<{
  verify: [tokenOrTicket: string, randstr: string]
  expire: []
  error: []
}>()

const turnstileRef = ref<InstanceType<typeof TurnstileWidget> | null>(null)
const tencentRef = ref<InstanceType<typeof TencentCaptchaGate> | null>(null)
const aliyunRef = ref<InstanceType<typeof AliyunCaptchaWidget> | null>(null)
const turnstileToken = ref('')

// 先独立统计三个 enabled 开关：数量不等于 1 时直接 fail-closed，
// 防止「一个 enabled 但配置残缺」被另一个 enabled 且完整的 provider 静默掩盖。
// 恰好一个 enabled 后，再校验该 provider 的必需配置字段，残缺同样 fail-closed。
// 模板渲染与 verifyAction 共用同一判定，避免多 provider 或残缺配置被部分消费。
type ProviderKind = 'turnstile' | 'tencent' | 'aliyun'
const validProvider = computed<ProviderKind | null>(() => {
  const enabled: ProviderKind[] = []
  if (props.turnstileEnabled) enabled.push('turnstile')
  if (props.tencentEnabled) enabled.push('tencent')
  if (props.aliyunEnabled) enabled.push('aliyun')
  if (enabled.length !== 1) return null
  const provider = enabled[0]
  if (provider === 'turnstile') return props.turnstileSiteKey ? provider : null
  if (provider === 'tencent') return props.tencentAppId ? provider : null
  return props.aliyunSceneId && props.aliyunPrefix ? provider : null
})

watch(
  () => [props.turnstileEnabled, props.turnstileSiteKey],
  () => {
    turnstileToken.value = ''
  }
)

function onTurnstileVerify(token: string): void {
  turnstileToken.value = token
  emit('verify', token, '')
}

function onTurnstileExpire(): void {
  turnstileToken.value = ''
  emit('expire')
}

function onTurnstileError(): void {
  turnstileToken.value = ''
  emit('error')
}

function reset(): void {
  turnstileToken.value = ''
  turnstileRef.value?.reset()
  tencentRef.value?.reset()
  aliyunRef.value?.reset()
}

// verifyAction 复用当前已完成的 Turnstile token，或获取腾讯/阿里云的动作 proof。
async function verifyAction(): Promise<ActionCaptchaResult | null> {
  if (validProvider.value === 'turnstile') {
    return turnstileToken.value ? { token: turnstileToken.value, randstr: '' } : null
  }
  if (validProvider.value === 'tencent') {
    try {
      const proof = (await tencentRef.value?.verify()) ?? null
      if (!proof) return null
      return { token: proof.ticket, randstr: proof.randstr }
    } catch {
      emit('error')
      return null
    }
  }
  if (validProvider.value === 'aliyun') {
    try {
      const param = (await aliyunRef.value?.verify()) ?? null
      if (!param) return null
      return { token: param, randstr: '' }
    } catch {
      emit('error')
      return null
    }
  }
  return null
}

defineExpose({ reset, verifyAction })
</script>
