<template>
  <n-space vertical>
    <n-card title="PalDefender 集成状态" size="small">
      <n-descriptions :column="1" bordered size="small">
        <n-descriptions-item label="配置">
          <n-tag :type="status.available ? 'success' : 'default'">
            {{ status.available ? '已配置' : '未配置' }}
          </n-tag>
        </n-descriptions-item>
        <n-descriptions-item v-if="status.available" label="连接">
          <n-tag :type="status.connected ? 'success' : 'error'">
            {{ status.connected ? '已连接' : '未连接' }}
          </n-tag>
          <span v-if="status.error" style="color:#c00;margin-left:8px">{{ status.error }}</span>
        </n-descriptions-item>
        <n-descriptions-item v-if="status.version" label="版本信息">
          {{ JSON.stringify(status.version) }}
        </n-descriptions-item>
      </n-descriptions>
      <n-divider />
      <n-space>
        <n-button @click="load">刷新状态</n-button>
        <n-button @click="reload" type="primary" ghost>重载 PalDefender 配置</n-button>
      </n-space>
    </n-card>

    <n-card title="PalDefender 封禁列表" size="small">
      <n-data-table :columns="banCols" :data="bans" :bordered="false" size="small" />
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import {
  NSpace, NCard, NDescriptions, NDescriptionsItem, NTag, NDivider, NButton,
  NDataTable, useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const status = ref<any>({})
const bans = ref<any[]>([])

const banCols: DataTableColumns<any> = [
  { title: '类型', key: 'Type', width: 90 },
  { title: '标识符', key: 'Id' },
  { title: '原因', key: 'Reason' },
  { title: '时间', key: 'CreationTime', width: 180 },
]

async function load() {
  try { status.value = await api.pdStatus() } catch {}
  if (status.value.connected) {
    try { bans.value = await api.pdBanlist() } catch {}
  }
}

async function reload() {
  try {
    await api.pdReload()
    message.success('已触发重载')
    load()
  } catch (e: any) { message.error(e.message) }
}

onMounted(load)
</script>
