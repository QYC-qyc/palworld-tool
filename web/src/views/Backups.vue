<template>
  <n-space vertical :size="16">
    <PageHeader title="备份管理" subtitle="存档备份的查看、回档与删除">
      <n-button @click="load" :loading="loading">
        <template #icon><n-icon :component="RefreshOutline" /></template>
        刷新
      </n-button>
    </PageHeader>

    <n-card size="small">
      <n-data-table :columns="cols" :data="backups" :bordered="false" size="small" />
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NCard, NDataTable, NButton, NPopconfirm, NIcon, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { RefreshOutline, ArchiveOutline } from '@vicons/ionicons5'
import { api } from '@/api'
import PageHeader from '@/components/PageHeader.vue'

const message = useMessage()
const backups = ref<any[]>([])
const loading = ref(false)

const cols: DataTableColumns<any> = [
  {
    title: '备份文件', key: 'path',
    render: (r) =>
      h('span', { style: 'display:inline-flex;align-items:center;gap:6px' }, [
        h(NIcon, { component: ArchiveOutline, size: 16, style: 'color:#909399' }),
        h('span', {}, r.path || r.backup_id),
      ]),
  },
  {
    title: '时间', key: 'save_time', width: 200,
    render: (r) => {
      try { return r.save_time ? new Date(r.save_time).toLocaleString() : '-' } catch { return '-' }
    },
  },
  {
    title: '操作', key: 'actions', width: 200,
    render: (r) =>
      h('div', { style: 'display:flex;gap:8px' }, [
        h(
          NPopconfirm,
          {
            onPositiveClick: () => restore(r),
            positiveText: '确认回档',
            negativeText: '取消',
          },
          {
            trigger: () =>
              h(NButton, { size: 'small', type: 'warning', ghost: true }, { default: () => '回档' }),
            default: () => '回档会停服→恢复→启服，当前存档会先自动备份。确定？',
          }
        ),
        h(
          NPopconfirm,
          {
            onPositiveClick: () => remove(r),
            positiveText: '删除',
            negativeText: '取消',
          },
          {
            trigger: () =>
              h(NButton, { size: 'small', type: 'error', ghost: true }, { default: () => '删除' }),
            default: () => '确定删除该备份吗？',
          }
        ),
      ]),
  },
]

async function restore(r: any) {
  try {
    const res: any = await api.restoreBackup(r.backup_id)
    message.success(res.message || '回档已开始')
  } catch (e: any) {
    message.error(e.message)
  }
}

async function remove(r: any) {
  try {
    await api.deleteBackup(r.backup_id)
    message.success('已删除')
    load()
  } catch (e: any) {
    message.error(e.message)
  }
}

async function load() {
  loading.value = true
  try { backups.value = await api.getBackups() } catch {} finally { loading.value = false }
}
onMounted(load)
</script>
