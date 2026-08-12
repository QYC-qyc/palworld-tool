<template>
  <n-space vertical>
    <n-card title="反作弊告警" size="small">
      <template #header-extra>
        <n-space>
          <n-select
            v-model:value="statusFilter"
            :options="statusOptions"
            placeholder="状态"
            style="width: 140px"
            clearable
            @update:value="load"
          />
          <n-button @click="load">刷新</n-button>
          <n-button type="primary" @click="scan" :loading="scanning">立即扫描</n-button>
        </n-space>
      </template>
      <n-data-table :columns="cols" :data="alerts" :bordered="false" size="small" />
    </n-card>

    <n-drawer v-model:show="showDetail" :width="520">
      <n-drawer-content :title="`告警 #${current?.id}`" closable>
        <n-descriptions v-if="current" :column="1" bordered size="small">
          <n-descriptions-item label="规则">{{ current.rule_id }}</n-descriptions-item>
          <n-descriptions-item label="严重度">
            <n-tag :type="severityType(current.severity)">{{ current.severity }}</n-tag>
          </n-descriptions-item>
          <n-descriptions-item label="玩家">{{ current.nickname }} ({{ current.player_uid }})</n-descriptions-item>
          <n-descriptions-item label="SteamID">{{ current.steam_id }}</n-descriptions-item>
          <n-descriptions-item label="标题">{{ current.title }}</n-descriptions-item>
          <n-descriptions-item label="详情">
            <pre style="white-space: pre-wrap">{{ current.detail }}</pre>
          </n-descriptions-item>
          <n-descriptions-item label="时间">{{ new Date(current.created_at).toLocaleString() }}</n-descriptions-item>
          <n-descriptions-item label="状态">{{ current.status }}</n-descriptions-item>
        </n-descriptions>
        <n-divider>处理</n-divider>
        <n-space>
          <n-button type="success" @click="action('confirmed')">确认作弊</n-button>
          <n-button @click="action('ignored')">忽略</n-button>
          <n-button type="warning" @click="action('actioned')">已处置</n-button>
        </n-space>
      </n-drawer-content>
    </n-drawer>
  </n-space>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import {
  NSpace, NCard, NDataTable, NButton, NSelect, NTag,
  NDrawer, NDrawerContent, NDescriptions, NDescriptionsItem, NDivider, useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const alerts = ref<any[]>([])
const showDetail = ref(false)
const current = ref<any>(null)
const statusFilter = ref<string | null>(null)
const scanning = ref(false)

const statusOptions = [
  { label: '待处理', value: 'open' },
  { label: '已确认', value: 'confirmed' },
  { label: '已忽略', value: 'ignored' },
  { label: '已处置', value: 'actioned' },
]

const cols: DataTableColumns<any> = [
  {
    title: '严重度', key: 'severity', width: 90,
    render: (r) => h(NTag, { type: severityType(r.severity), size: 'small' }, { default: () => r.severity }),
  },
  { title: '规则', key: 'rule_id', width: 80 },
  { title: '玩家', key: 'nickname', width: 140 },
  { title: '标题', key: 'title' },
  { title: '来源', key: 'source', width: 110 },
  {
    title: '时间', key: 'created_at', width: 180,
    render: (r) => new Date(r.created_at).toLocaleString(),
  },
  {
    title: '状态', key: 'status', width: 90,
    render: (r) => h(NTag, { size: 'small', type: r.status === 'open' ? 'warning' : 'default' }, { default: () => r.status }),
  },
]

function severityType(s: string) {
  if (s === 'critical') return 'error'
  if (s === 'warn') return 'warning'
  return 'info'
}

async function load() {
  try {
    const params = new URLSearchParams()
    if (statusFilter.value) params.set('status', statusFilter.value)
    params.set('limit', '200')
    const res = await api.getAlerts(params.toString())
    alerts.value = res.alerts
  } catch {}
}

async function scan() {
  scanning.value = true
  try {
    await api.runScan()
    message.success('扫描已触发')
    setTimeout(load, 3000)
  } catch (e: any) { message.error(e.message) }
  finally { scanning.value = false }
}

async function action(status: string) {
  if (!current.value) return
  try {
    await api.alertAction(current.value.id, status)
    message.success('已更新')
    showDetail.value = false
    load()
  } catch (e: any) { message.error(e.message) }
}

onMounted(load)
</script>
