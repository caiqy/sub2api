<template>
  <ChannelStatusV2View v-if="isV2" />
  <ChannelStatusV1View v-else />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { isChannelMonitorV2Mode } from '@/utils/featureFlags'
import ChannelStatusV1View from './ChannelStatusV1View.vue'
import ChannelStatusV2View from './ChannelStatusV2View.vue'

const isV2 = computed(() => isChannelMonitorV2Mode())
</script>
