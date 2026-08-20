<template>
  <n-space vertical :size="16">
    <PageHeader title="操作审计" subtitle="查看面板与游戏服操作记录" />

    <n-card size="small">
      <n-space vertical :size="12">
        <n-space align="center" :size="12" wrap>
          <n-select
            v-model:value="sourceFilter"
            :options="sourceOptions"
            placeholder="来源"
            clearable
            style="width:180px"
          />
          <n-input v-model:value="keyword" placeholder="搜索操作 / 目标 / 详情" clearable style="max-width:320px">
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <n-text depth="3" style="font-size:12px">共 {{ filtered.length }} 条</n-text>
        </n-space>
        <n-data-table
          :columns="cols"
          :data="filtered"
          :bordered="false"
          size="small"
          :pagination="{ pageSize: 30, prefix: ({ itemCount }) => `共 ${itemCount} 条` }"
        />
      </n-space>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { h, onMounted, ref, computed } from 'vue'
import { NCard, NDataTable, NTag, NSelect, NInput, NIcon, NSpace, NText } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { SearchOutline } from '@vicons/ionicons5'
import PageHeader from '@/components/PageHeader.vue'

const records = ref<any[]>([])
const sourceFilter = ref<string | null>(null)
const keyword = ref('')

const sourceOptions = computed(() => {
  const set = new Set<string>()
  records.value.forEach((r) => { if (r.source) set.add(r.source) })
  return Array.from(set).map((s) => ({ label: s, value: s }))
})

const filtered = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return records.value.filter((r) => {
    if (sourceFilter.value && r.source !== sourceFilter.value) return false
    if (!q) return true
    return (
      String(r.action || '').toLowerCase().includes(q) ||
      String(r.target || '').toLowerCase().includes(q) ||
      String(r.detail || '').toLowerCase().includes(q)
    )
  })
})

function fmt(t: any) {
  if (!t) return '-'
  try {
    const d = new Date(t)
    if (isNaN(d.getTime())) return String(t)
    return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch { return '-' }
}

const cols: DataTableColumns<any> = [
  { title: '时间', key: 'created_at', width: 180, render: (r) => fmt(r.created_at) },
  { title: '来源', key: 'source', width: 110 },
  { title: '操作', key: 'action', width: 140 },
  { title: '目标', key: 'target', width: 200, ellipsis: { tooltip: true } },
  { title: '详情', key: 'detail', ellipsis: { tooltip: true } },
  {
    title: '结果', key: 'result', width: 90,
    render: (r) => h(NTag, {
      size: 'small',
      type: r.result === 'success' ? 'success' : 'error',
      round: true,
      bordered: false,
    }, { default: () => r.result || '-' }),
  },
]

onMounted(async () => {
  try {
    const token = localStorage.getItem('paladmin_token') || ''
    const resp = await fetch('/api/audit', {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (resp.ok) {
      const data = await resp.json()
      records.value = Array.isArray(data) ? data : []
    }
  } catch {
    records.value = []
  }
})
</script>
