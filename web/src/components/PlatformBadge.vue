<template>
  <n-tag type="info" size="small" round :bordered="bordered">
    <n-icon :component="LogoWindows" style="vertical-align:-2px;margin-right:4px" />
    Proton / Windows
  </n-tag>
</template>

<script setup lang="ts">
import { NTag, NIcon } from 'naive-ui'
import { LogoWindows } from '@vicons/ionicons5'

withDefaults(
  defineProps<{
    bordered?: boolean
  }>(),
  { bordered: false }
)
</script>
