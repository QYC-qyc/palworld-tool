<template>
  <n-tag :type="tagType" size="small" round :bordered="false">
    <span class="dot" :class="`dot--${status}`" />
    {{ text }}
  </n-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'

const props = defineProps<{
  status: 'running' | 'stopped' | 'updating' | 'error'
}>()

const tagType = computed(() => {
  switch (props.status) {
    case 'running':
      return 'success'
    case 'updating':
      return 'warning'
    case 'error':
      return 'error'
    default:
      return 'default'
  }
})

const text = computed(() => {
  switch (props.status) {
    case 'running':
      return '运行中'
    case 'updating':
      return '更新中'
    case 'error':
      return '异常'
    default:
      return '已停止'
  }
})
</script>

<style scoped>
.dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  margin-right: 6px;
  vertical-align: middle;
}
.dot--running {
  background: #18a058;
  box-shadow: 0 0 0 3px rgba(24, 160, 88, 0.2);
}
.dot--updating {
  background: #f0a020;
}
.dot--error {
  background: #d03050;
}
.dot--stopped {
  background: #909399;
}
</style>
