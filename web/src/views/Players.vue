<template>
  <n-space vertical>
    <n-card title="在线玩家" size="small">
      <n-data-table :columns="onlineCols" :data="online" :bordered="false" size="small" />
    </n-card>
    <n-card title="全部玩家（存档）" size="small">
      <n-data-table
        :columns="playerCols"
        :data="players"
        :bordered="false"
        size="small"
        :row-props="rowProps"
      />
    </n-card>

    <n-drawer v-model:show="showDetail" :width="520">
      <n-drawer-content :title="`玩家详情 - ${detail?.nickname || ''}`" closable>
        <n-descriptions v-if="detail" :column="1" bordered size="small">
          <n-descriptions-item label="PlayerUID">{{ detail.player_uid }}</n-descriptions-item>
          <n-descriptions-item label="SteamID">{{ detail.steam_id }}</n-descriptions-item>
          <n-descriptions-item label="等级">{{ detail.level }}</n-descriptions-item>
          <n-descriptions-item label="HP">{{ detail.hp }} / {{ detail.max_hp }}</n-descriptions-item>
          <n-descriptions-item label="IP">{{ detail.ip }}</n-descriptions-item>
          <n-descriptions-item label="最后在线">{{ detail.last_online }}</n-descriptions-item>
        </n-descriptions>
        <n-divider>帕鲁 ({{ detail?.pals?.length || 0 }})</n-divider>
        <n-data-table
          v-if="detail?.pals?.length"
          :columns="palCols"
          :data="detail.pals"
          size="small"
          :bordered="false"
        />
        <n-divider>操作</n-divider>
        <n-space>
          <n-button type="warning" ghost @click="act('kick', detail)">踢出</n-button>
          <n-button type="error" ghost @click="act('ban', detail)">封禁</n-button>
          <n-button @click="act('unban', detail)">解封</n-button>
          <n-button type="error" @click="act('ipban', detail)">IP封禁</n-button>
        </n-space>
      </n-drawer-content>
    </n-drawer>
  </n-space>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import {
  NSpace, NCard, NDataTable, NDrawer, NDrawerContent, NDescriptions, NDescriptionsItem,
  NButton, NDivider, NTag, useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const players = ref<any[]>([])
const online = ref<any[]>([])
const showDetail = ref(false)
const detail = ref<any>(null)

const onlineCols: DataTableColumns<any> = [
  { title: '昵称', key: 'nickname' },
  { title: 'SteamID', key: 'steam_id', width: 160 },
  { title: 'IP', key: 'ip', width: 130 },
  { title: '等级', key: 'level', width: 70 },
  { title: 'Ping', key: 'ping', width: 80 },
]

const playerCols: DataTableColumns<any> = [
  { title: '昵称', key: 'nickname' },
  { title: '等级', key: 'level', width: 80 },
  { title: 'SteamID', key: 'steam_id', width: 170 },
  { title: '帕鲁数', key: 'pals', width: 90, render: (r) => r.pals?.length ?? 0 },
  { title: '最后在线', key: 'last_online', width: 180 },
]

const palCols: DataTableColumns<any> = [
  { title: '类型', key: 'type' },
  { title: '等级', key: 'level', width: 70 },
  { title: '攻击', key: 'melee', width: 70 },
  { title: '防御', key: 'defense', width: 70 },
  {
    title: '标记', key: 'is_boss', width: 120,
    render: (r) => h('span', [
      r.is_boss ? h(NTag, { size: 'small', type: 'error' }, { default: () => 'Boss' }) : '',
      r.is_tower ? h(NTag, { size: 'small', type: 'warning' }, { default: () => '塔主' }) : '',
      r.is_lucky ? h(NTag, { size: 'small', type: 'success' }, { default: () => '闪光' }) : '',
    ]),
  },
]

function rowProps(row: any) {
  return {
    style: 'cursor: pointer',
    onClick: async () => {
      try {
        detail.value = await api.getPlayer(row.player_uid)
        showDetail.value = true
      } catch (e: any) {
        message.error(e.message)
      }
    },
  }
}

async function act(kind: string, p: any) {
  try {
    if (kind === 'kick') await api.kickPlayer(p.player_uid)
    if (kind === 'ban') await api.banPlayer(p.player_uid)
    if (kind === 'unban') await api.unbanPlayer(p.player_uid)
    if (kind === 'ipban') await api.ipBanPlayer(p.player_uid)
    message.success('操作成功')
  } catch (e: any) {
    message.error(e.message)
  }
}

onMounted(async () => {
  try { players.value = await api.getPlayers() } catch {}
  try { online.value = await api.getOnline() } catch {}
})
</script>
