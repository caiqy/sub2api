<template>
  <TurnstileWidget
    v-if="turnstileEnabled && turnstileSiteKey"
    ref="turnstileRef"
    :site-key="turnstileSiteKey"
    @verify="onTurnstileVerify"
    @expire="onTurnstileExpire"
    @error="onTurnstileError"
  />
  <TencentCaptchaGate
    v-else-if="tencentEnabled && tencentAppId"
    ref="tencentRef"
    :app-id="tencentAppId"
  />
  <AliyunCaptchaWidget
    v-else-if="aliyunEnabled && aliyunSceneId && aliyunPrefix"
    ref="aliyunRef"
    :scene-id="aliyunSceneId"
    :prefix="aliyunPrefix"
    :region="aliyunRegion === 'sgp' ? 'sgp' : 'cn'"
    @verify="(param: string) => emit('verify', param, '')"
    @expire="emit('expire')"
    @error="emit('error')"
  />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import TurnstileWidget from '@/components/TurnstileWidget.vue'
import TencentCaptchaGate from '@/components/TencentCaptchaGate.vue'
import AliyunCaptchaWidget from '@/components/AliyunCaptchaWidget.vue'

// ActionCaptchaResult 动作触发式验证（腾讯/阿里云弹窗）的结果：
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

// verifyAction 返回当前已完成的 Turnstile token，或弹出当前动作触发式验证码并等待结果。
async function verifyAction(): Promise<ActionCaptchaResult | null> {
  if (props.turnstileEnabled && props.turnstileSiteKey) {
    return turnstileToken.value ? { token: turnstileToken.value, randstr: '' } : null
  }
  if (props.tencentEnabled && props.tencentAppId) {
    try {
      const proof = (await tencentRef.value?.verify()) ?? null
      if (!proof) return null
      return { token: proof.ticket, randstr: proof.randstr }
    } catch {
      emit('error')
      return null
    }
  }
  if (props.aliyunEnabled && props.aliyunSceneId && props.aliyunPrefix) {
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
