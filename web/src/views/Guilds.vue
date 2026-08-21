<template>
  <n-space vertical :size="16">
    <PageHeader title="公会" subtitle="查看公会成员、据点与等级" />

    <n-card size="small">
      <n-space vertical :size="12">
        <n-input v-model:value="search" placeholder="搜索公会名 / 会长" clearable style="max-width:320px">
          <template #prefix><n-icon :component="SearchOutline" /></template>
        </n-input>
        <n-data-table
          :columns="cols"
          :data="filtered"
          :bordered="false"
          size="small"
          :row-props="rowProps"
          :pagination="{ pageSize: 20, prefix: ({ itemCount }) => `共 ${itemCount} 个公会` }"
        />
      </n-space>
    </n-card>

    <n-drawer v-model:show="showDetail" :width="isMobile ? '100%' : 560" placement="right">
      <n-drawer-content closable>
        <template #header>
          <span style="font-size:16px;font-weight:600">公会详情 - {{ detail?.name || '' }}</span>
        </template>
        <template v-if="detail">
          <n-grid cols="2" :x-gap="12" :y-gap="12" responsive="screen" class="stat-grid">
            <div class="stat-box">
              <div class="stat-box__label">会长 UID</div>
              <div class="stat-box__value">{{ detail.admin_player_uid }}</div>
            </div>
            <div class="stat-box">
              <div class="stat-box__label">据点等级</div>
              <div class="stat-box__value">{{ detail.base_camp_level }}</div>
            </div>
            <div class="stat-box">
              <div class="stat-box__label">成员数</div>
              <div class="stat-box__value">{{ detail.players?.length || 0 }}</div>
            </div>
            <div class="stat-box">
              <div class="stat-box__label">据点数</div>
              <div class="stat-box__value">{{ detail.base_ids?.length || 0 }}</div>
            </div>
          </n-grid>

          <n-divider>成员列表</n-divider>
          <n-data-table
            v-if="detail.players?.length"
            :columns="playerCols"
            :data="detail.players"
            size="small"
            :bordered="false"
          />
          <n-text v-else depth="3" style="font-size:12px">暂无成员数据</n-text>
        </template>
      </n-drawer-content>
    </n-drawer>
  </n-space>
</template>

<script setup lang="ts">
import { h, onMounted, ref, computed } from 'vue'
import {
  NSpace, NCard, NDataTable, NDrawer, NDrawerContent, NGrid, NGi,
  NDivider, NInput, NIcon, NText,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { SearchOutline } from '@vicons/ionicons5'
import { api } from '@/api'
import PageHeader from '@/components/PageHeader.vue'

const isMobile = ref(window.innerWidth < 768)
const guilds = ref<any[]>([])
const showDetail = ref(false)
const detail = ref<any>(null)
const search = ref('')
const guildPlayers = ref<Record<string, any>>({})

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return guilds.value
  return guilds.value.filter((g) => {
    const adminName = guildPlayers.value[g.admin_player_uid]?.nickname || ''
    return (
      String(g.name || '').toLowerCase().includes(q) ||
      String(g.admin_player_uid || '').toLowerCase().includes(q) ||
      adminName.toLowerCase().includes(q)
    )
  })
})

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

function rowProps(row: any) {
  return {
    style: 'cursor: pointer',
    onClick: async () => {
      try {
        detail.value = await api.getGuild(row.admin_player_uid)
        showDetail.value = true
      } catch (e: any) {
        // 已由全局处理
      }
    },
  }
}

onMounted(async () => {
  // 并行请求，避免游戏未运行时串行等待超时
  const [g, p] = await Promise.allSettled([api.getGuilds(), api.getPlayers()])
  if (g.status === 'fulfilled') guilds.value = g.value
  if (p.status === 'fulfilled') {
    const m: Record<string, any> = {}
    p.value.forEach((pl: any) => { m[pl.player_uid] = pl })
    guildPlayers.value = m
  }
})
</script>

<style scoped>
.stat-grid {
  margin-top: 4px;
}
.stat-box {
  padding: 12px 14px;
  border-radius: 10px;
  background: var(--n-color-embedded, rgba(127,127,127,0.06));
  border: 1px solid var(--n-border-color, #eee);
}
.stat-box__label {
  font-size: 12px;
  color: var(--n-text-color-3, #999);
}
.stat-box__value {
  font-size: 18px;
  font-weight: 600;
  margin-top: 4px;
}
</style>
