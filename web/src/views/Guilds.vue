<template>
  <n-space vertical>
    <n-card title="公会列表" size="small">
      <n-data-table
        :columns="cols"
        :data="guilds"
        :bordered="false"
        size="small"
        :row-props="rowProps"
      />
    </n-card>

    <n-drawer v-model:show="showDetail" :width="520">
      <n-drawer-content :title="`公会详情 - ${detail?.name || ''}`" closable>
        <n-descriptions v-if="detail" :column="1" bordered size="small">
          <n-descriptions-item label="会长UID">{{ detail.admin_player_uid }}</n-descriptions-item>
          <n-descriptions-item label="据点等级">{{ detail.base_camp_level }}</n-descriptions-item>
          <n-descriptions-item label="成员数">{{ detail.players?.length || 0 }}</n-descriptions-item>
          <n-descriptions-item label="据点数">{{ detail.base_ids?.length || 0 }}</n-descriptions-item>
        </n-descriptions>
        <n-divider>成员列表</n-divider>
        <n-data-table
          v-if="detail?.players?.length"
          :columns="playerCols"
          :data="detail.players"
          size="small"
          :bordered="false"
        />
      </n-drawer-content>
    </n-drawer>
  </n-space>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import {
  NSpace, NCard, NDataTable, NDrawer, NDrawerContent, NDescriptions, NDescriptionsItem,
  NDivider, NTag, useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const guilds = ref<any[]>([])
const showDetail = ref(false)
const detail = ref<any>(null)

const cols: DataTableColumns<any> = [
  { title: '公会名', key: 'name' },
  { title: '据点等级', key: 'base_camp_level', width: 100 },
  {
    title: '会长', key: 'admin_player_uid', width: 180,
    render: (r) => {
      const player = guildPlayers.value[r.admin_player_uid]
      return player?.nickname || r.admin_player_uid
    },
  },
  { title: '成员数', key: 'players', width: 90, render: (r) => r.players?.length ?? 0 },
  { title: '据点数', key: 'base_ids', width: 90, render: (r) => r.base_ids?.length ?? 0 },
]

const playerCols: DataTableColumns<any> = [
  { title: '玩家UID', key: 'player_uid' },
  {
    title: '昵称', key: 'nickname',
    render: (r) => {
      const p = guildPlayers.value[r.player_uid]
      return p?.nickname || r.nickname || '-'
    },
  },
]

const guildPlayers = ref<Record<string, any>>({})

function rowProps(row: any) {
  return {
    style: 'cursor: pointer',
    onClick: async () => {
      try {
        detail.value = await api.getGuild(row.admin_player_uid)
        showDetail.value = true
      } catch (e: any) {
        message.error(e.message)
      }
    },
  }
}

onMounted(async () => {
  try {
    guilds.value = await api.getGuilds()
    // 加载所有玩家数据用于显示昵称
    try {
      const players = await api.getPlayers()
      const m: Record<string, any> = {}
      players.forEach((p: any) => { m[p.player_uid] = p })
      guildPlayers.value = m
    } catch {}
  } catch {}
})
</script>
