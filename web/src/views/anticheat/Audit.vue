<template>
  <n-card title="审计日志" size="small">
    <n-data-table :columns="cols" :data="audits" :bordered="false" size="small" />
  </n-card>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { NCard, NDataTable } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { api } from '@/api'

const audits = ref<any[]>([])

const cols: DataTableColumns<any> = [
  { title: '时间', key: 'created_at', width: 180, render: (r) => new Date(r.created_at).toLocaleString() },
  { title: '操作者', key: 'actor', width: 120 },
  { title: '动作', key: 'action', width: 130 },
  { title: '目标', key: 'target', width: 180 },
  { title: '详情', key: 'detail' },
  { title: '结果', key: 'result', width: 100 },
]

onMounted(async () => {
  try { audits.value = await api.getAudit() } catch {}
})
</script>
