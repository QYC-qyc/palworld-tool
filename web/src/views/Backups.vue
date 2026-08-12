<template>
  <n-card title="备份管理" size="small">
    <template #header-extra>
      <n-button @click="load">刷新</n-button>
    </template>
    <n-data-table :columns="cols" :data="backups" :bordered="false" size="small" />
  </n-card>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NCard, NDataTable, NButton, NPopconfirm, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const backups = ref<any[]>([])

const cols: DataTableColumns<any> = [
  { title: '备份文件', key: 'path' },
  {
    title: '时间', key: 'save_time', width: 200,
    render: (r) => new Date(r.save_time).toLocaleString(),
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
          NButton,
          {
            size: 'small', type: 'error', ghost: true,
            onClick: async () => {
              try { await api.deleteBackup(r.backup_id); message.success('已删除'); load() }
              catch (e: any) { message.error(e.message) }
            },
          },
          { default: () => '删除' }
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

async function load() {
  try { backups.value = await api.getBackups() } catch {}
}
onMounted(load)
</script>
