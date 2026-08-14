<template>
  <n-card title="操作审计" size="small">
    <n-data-table :columns="cols" :data="records" :bordered="false" size="small" />
  </n-card>
</template>

<script setup lang="ts">
import { onMounted, ref, h } from 'vue'
import { NCard, NDataTable, NTag } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { api } from '@/api'

const records = ref<any[]>([])

const cols: DataTableColumns<any> = [
  { title: '时间', key: 'created_at', width: 170, render: (r) => new Date(r.created_at).toLocaleString() },
  { title: '来源', key: 'source', width: 90 },
  { title: '操作', key: 'action', width: 130 },
  { title: '目标', key: 'target', width: 180 },
  { title: '详情', key: 'detail' },
  {
    title: '结果', key: 'result', width: 90,
    render: (r) => h(NTag, {
      size: 'small',
      type: r.result === 'success' ? 'success' : 'error',
    }, { default: () => r.result }),
  },
]

onMounted(async () => {
  try {
    const token = localStorage.getItem('paladmin_token') || ''
    const resp = await fetch('/api/audit', {
      headers: { Authorization: `Bearer ${token}` },
    })
    records.value = await resp.json()
  } catch { /* ignore */ }
})
</script>
