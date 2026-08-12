<template>
  <n-card title="反作弊规则" size="small">
    <n-data-table :columns="cols" :data="rules" :bordered="false" size="small" />
  </n-card>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NCard, NDataTable, NTag, NSwitch, NButton, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const rules = ref<any[]>([])

async function toggleEnabled(row: any, enabled: boolean) {
  try {
    await api.updateRule(row.id, { enabled })
    row.enabled = enabled
    message.success('已更新')
  } catch (e: any) { message.error(e.message) }
}

const cols: DataTableColumns<any> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '名称', key: 'name', width: 160 },
  { title: '类别', key: 'category', width: 90 },
  {
    title: '启用', key: 'enabled', width: 80,
    render: (r) =>
      h(NSwitch, {
        value: r.enabled,
        size: 'small',
        onUpdateValue: (v: boolean) => toggleEnabled(r, v),
      }),
  },
  {
    title: '严重度', key: 'severity', width: 100,
    render: (r) =>
      h(NTag, {
        size: 'small',
        type: r.severity === 'critical' ? 'error' : r.severity === 'warn' ? 'warning' : 'info',
      }, { default: () => r.severity }),
  },
  {
    title: '处置动作', key: 'actions', width: 200,
    render: (r) => (r.actions || []).join(', '),
  },
  { title: '说明', key: 'reason' },
]

onMounted(async () => {
  try { rules.value = await api.getRules() } catch {}
})
</script>
