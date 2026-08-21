<template>
  <n-tag :type="tagType" size="small" round :bordered="false">
    <span class="dot" :class="`dot--${status}`" />
    {{ text }}
  </n-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'

type Status = 'running' | 'stopped' | 'pending' | 'updating' | 'installing' | 'starting' | 'error'
const props = defineProps<{
  status: Status
}>()

const tagType = computed(() => {
  switch (props.status) {
    case 'running':
      return 'success'
    case 'starting':
    case 'updating':
    case 'installing':
      return 'warning'
    case 'pending':
      return 'info'
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
    case 'pending':
      return '待启动'
    case 'updating':
      return '更新中'
    case 'installing':
      return '安装中'
    case 'starting':
      return '启动中'
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
.dot--starting,
.dot--updating,
.dot--installing {
  background: #f0a020;
  animation: pulse 1.2s ease-in-out infinite;
}
.dot--pending {
  background: #2080f0;
}
.dot--error {
  background: #d03050;
}
.dot--stopped {
  background: #909399;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
</style>
