<template>
  <div class="empty-state">
    <n-icon size="48" :component="icon || CubeOutline" class="empty-state__icon" />
    <h3 class="empty-state__title">{{ title }}</h3>
    <p v-if="description" class="empty-state__desc">{{ description }}</p>
    <div v-if="$slots.default" class="empty-state__actions">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { NIcon } from 'naive-ui'
import { CubeOutline } from '@vicons/ionicons5'
import type { Component } from 'vue'

defineProps<{
  title: string
  description?: string
  icon?: Component
}>()
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 56px 20px;
}
.empty-state__icon {
  color: var(--n-text-color-3, #999);
  opacity: 0.6;
  margin-bottom: 16px;
}
.empty-state__title {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 6px;
}
.empty-state__desc {
  font-size: 13px;
  color: var(--n-text-color-3, #999);
  margin: 0 0 16px;
  max-width: 420px;
  line-height: 1.6;
}
.empty-state__actions {
  display: flex;
  gap: 8px;
}
</style>
