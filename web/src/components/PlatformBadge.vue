<template>
  <n-tag :type="tagType" size="small" round :bordered="bordered">
    <n-icon v-if="platform === 'windows'" :component="LogoWindows" style="margin-right:4px" />
    <n-icon v-else :component="LogoTux" style="margin-right:4px" />
    {{ label }}
  </n-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NTag, NIcon } from 'naive-ui'
import { LogoWindows, LogoTux } from '@vicons/ionicons5'

const props = withDefaults(
  defineProps<{
    platform: 'linux' | 'windows' | 'current'
    isWindows?: boolean
    installed?: boolean
    bordered?: boolean
  }>(),
  { bordered: false }
)

const actualPlatform = computed<'linux' | 'windows'>(() => {
  if (props.platform !== 'current') return props.platform
  return props.isWindows ? 'windows' : 'linux'
})

const label = computed(() => {
  const name = actualPlatform.value === 'windows' ? 'Windows / Wine' : '原生 Linux'
  if (props.installed === true) return `${name} · 已安装`
  if (props.installed === false) return `${name} · 未安装`
  return name
})

const tagType = computed<'info' | 'success' | 'warning' | 'default'>(() => {
  if (props.installed === true) return 'success'
  if (props.installed === false) return 'default'
  return actualPlatform.value === 'windows' ? 'info' : 'success'
})
</script>
