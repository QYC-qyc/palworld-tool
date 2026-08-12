<template>
  <n-space vertical>
    <n-card title="封禁列表" size="small">
      <n-data-table :columns="cols" :data="bans" :bordered="false" size="small" />
    </n-card>
    <n-card title="IP 封禁/解封" size="small">
      <n-space>
        <n-input v-model:value="ip" placeholder="输入 IP 地址" style="width: 240px" />
        <n-button type="error" @click="banIP">封禁 IP</n-button>
        <n-button @click="unbanIP">解封 IP</n-button>
      </n-space>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import {
  NSpace, NCard, NDataTable, NInput, NButton, NTag, useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const bans = ref<any[]>([])
const ip = ref('')

const cols: DataTableColumns<any> = [
  {
    title: '类型', key: 'type', width: 90,
    render: (r) => h(NTag, { type: r.type === 'ip' ? 'warning' : 'error', size: 'small' },
      { default: () => (r.type === 'ip' ? 'IP' : '用户') }),
  },
  { title: '标识符', key: 'identifier' },
  { title: '原因', key: 'reason' },
  { title: '操作者', key: 'issuer', width: 120 },
  {
    title: '时间', key: 'created_at', width: 180,
    render: (r) => new Date(r.created_at).toLocaleString(),
  },
]

async function load() {
  try { bans.value = await api.getBanlist() } catch {}
}
async function banIP() {
  if (!ip.value) return
  try { await api.banIP(ip.value); message.success('已封禁 IP'); ip.value = ''; load() }
  catch (e: any) { message.error(e.message) }
}
async function unbanIP() {
  if (!ip.value) return
  try { await api.unbanIP(ip.value); message.success('已解封 IP'); ip.value = ''; load() }
  catch (e: any) { message.error(e.message) }
}
onMounted(load)
</script>
