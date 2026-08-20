<template>
  <n-space vertical :size="16">
    <PageHeader title="玩家" subtitle="在线玩家与存档玩家管理">
      <n-button @click="loadAll" :loading="loading">
        <template #icon><n-icon :component="RefreshOutline" /></template>
        刷新
      </n-button>
    </PageHeader>

    <n-card size="small">
      <n-space vertical :size="12">
        <n-input v-model:value="search" placeholder="搜索昵称 / SteamID" clearable style="max-width:320px">
          <template #prefix><n-icon :component="SearchOutline" /></template>
        </n-input>
        <n-tabs type="line" animated v-model:value="tab">
          <n-tab-pane name="online">
            <template #tab>
              在线玩家
              <n-badge :value="online.length" :max="999" type="success" style="margin-left:6px" />
            </template>
            <n-data-table
              :columns="onlineCols"
              :data="filteredOnline"
              :bordered="false"
              size="small"
              :row-props="rowProps"
              :pagination="{ pageSize: 20, prefix: ({ itemCount }) => `共 ${itemCount} 人` }"
            />
          </n-tab-pane>
          <n-tab-pane name="all">
            <template #tab>
              全部玩家
              <n-badge :value="players.length" :max="999" type="default" style="margin-left:6px" />
            </template>
            <n-data-table
              :columns="playerCols"
              :data="filteredPlayers"
              :bordered="false"
              size="small"
              :row-props="rowProps"
              :pagination="{ pageSize: 20, prefix: ({ itemCount }) => `共 ${itemCount} 人` }"
            />
          </n-tab-pane>
        </n-tabs>
      </n-space>
    </n-card>

    <n-drawer v-model:show="showDetail" :width="isMobile ? '100%' : 640" placement="right">
      <n-drawer-content closable>
        <template #header>
          <div class="drawer-title">
            <span>{{ detail?.nickname || '玩家详情' }}</span>
            <n-tag v-if="isOnline(detail)" size="small" type="success" round :bordered="false">在线</n-tag>
          </div>
        </template>

        <n-tabs type="line" animated v-model:value="detailTab">
          <n-tab-pane name="basic" tab="基本信息">
            <n-descriptions v-if="detail" :column="1" bordered size="small">
              <n-descriptions-item label="PlayerUID">{{ detail.player_uid }}</n-descriptions-item>
              <n-descriptions-item label="SteamID">{{ detail.steam_id }}</n-descriptions-item>
              <n-descriptions-item label="等级">{{ detail.level }}</n-descriptions-item>
              <n-descriptions-item label="HP">{{ detail.hp }} / {{ detail.max_hp }}</n-descriptions-item>
              <n-descriptions-item label="IP">{{ detail.ip }}</n-descriptions-item>
              <n-descriptions-item label="最后在线">{{ formatTime(detail.last_online || detail.save_last_online) }}</n-descriptions-item>
            </n-descriptions>
          </n-tab-pane>
          <n-tab-pane name="pals" :tab="`帕鲁 (${detail?.pals?.length || 0})`">
            <n-data-table
              v-if="detail?.pals?.length"
              :columns="palCols"
              :data="detail.pals"
              size="small"
              :bordered="false"
            />
            <n-text v-else depth="3" style="font-size:12px">暂无帕鲁数据</n-text>
          </n-tab-pane>
          <n-tab-pane name="items" :tab="`物品 (${itemCount})`">
            <n-data-table
              v-if="itemRows.length"
              :columns="itemCols"
              :data="itemRows"
              size="small"
              :bordered="false"
            />
            <n-text v-else depth="3" style="font-size:12px">暂无物品数据</n-text>
          </n-tab-pane>
        </n-tabs>

        <n-divider>管理操作</n-divider>
        <n-space wrap>
          <n-button type="warning" ghost @click="openAct('kick', detail)">踢出</n-button>
          <n-button type="error" ghost @click="openAct('ban', detail)">封禁</n-button>
          <n-button @click="openAct('unban', detail)">解封</n-button>
          <n-button type="error" @click="openAct('ipban', detail)">IP封禁</n-button>
        </n-space>
      </n-drawer-content>
    </n-drawer>

    <!-- 操作弹窗（固定走 PalDefender） -->
    <n-modal v-model:show="act.show" preset="card" :title="act.title" style="max-width:460px">
      <n-space vertical>
        <n-input v-if="act.kind === 'ipban'" v-model:value="act.ip" placeholder="IP 地址（默认使用该玩家 IP）" />
        <n-input v-model:value="act.reason" type="textarea" :autosize="{ minRows: 2 }"
          placeholder="原因（可选）" />
        <n-switch v-if="act.kind === 'ban'" v-model:value="act.ipBan">
          <template #checked>同时封禁 IP</template>
          <template #unchecked>仅封禁账号</template>
        </n-switch>
      </n-space>
      <template #footer>
        <n-space justify="end">
          <n-button @click="act.show = false">取消</n-button>
          <n-button type="primary" :loading="act.loading" @click="confirmAct">确定</n-button>
        </n-space>
      </template>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { h, onMounted, reactive, ref, computed } from 'vue'
import {
  NSpace, NCard, NDataTable, NDrawer, NDrawerContent, NDescriptions, NDescriptionsItem,
  NButton, NDivider, NTag, NTabs, NTabPane, NImage, NText, NModal, NInput, NSwitch,
  NBadge, NIcon, useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { SearchOutline, RefreshOutline } from '@vicons/ionicons5'
import { api } from '@/api'
import PageHeader from '@/components/PageHeader.vue'

const message = useMessage()
const isMobile = ref(window.innerWidth < 768)

const players = ref<any[]>([])
const online = ref<any[]>([])
const loading = ref(false)
const showDetail = ref(false)
const detail = ref<any>(null)
const itemMap = ref<Record<string, any>>({})
const search = ref('')
const tab = ref<'online' | 'all'>('online')
const detailTab = ref('basic')

const act = reactive({
  show: false,
  kind: '' as 'kick' | 'ban' | 'unban' | 'ipban' | '',
  title: '',
  reason: '',
  ip: '',
  ipBan: false,
  loading: false,
  target: null as any,
})

fetch('/data/item_list.json').then(r => r.json()).then((data: any[]) => {
  const m: Record<string, any> = {}
  data.forEach(i => { m[i.id] = i })
  itemMap.value = m
}).catch(() => {})

const itemRows = computed(() => {
  if (!detail.value?.items) return []
  const containers = [
    { key: 'CommonContainerId', name: '背包' },
    { key: 'EssentialContainerId', name: '关键物品' },
    { key: 'FoodEquipContainerId', name: '食物' },
    { key: 'PlayerEquipArmorContainerId', name: '装备护甲' },
    { key: 'WeaponLoadOutContainerId', name: '武器' },
    { key: 'DropSlotContainerId', name: '掉落' },
  ]
  const rows: any[] = []
  containers.forEach(c => {
    const items = detail.value.items?.[c.key] || []
    items.forEach((it: any) => {
      rows.push({ ...it, container: c.name, itemInfo: itemMap.value[it.ItemId] })
    })
  })
  return rows
})

const itemCount = computed(() => itemRows.value.length)

function matchSearch(p: any): boolean {
  const q = search.value.trim().toLowerCase()
  if (!q) return true
  return (
    String(p.nickname || '').toLowerCase().includes(q) ||
    String(p.steam_id || '').toLowerCase().includes(q)
  )
}
const filteredOnline = computed(() => online.value.filter(matchSearch))
const filteredPlayers = computed(() => players.value.filter(matchSearch))

function isOnline(p: any): boolean {
  if (!p) return false
  return online.value.some((o) => o.player_uid === p.player_uid || o.steam_id === p.steam_id)
}

function formatTime(t: string): string {
  if (!t) return '-'
  const d = new Date(t)
  if (isNaN(d.getTime())) return t
  const now = Date.now()
  const diff = now - d.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + ' 分钟前'
  if (diff < 86400000) return Math.floor(diff / 3600000) + ' 小时前'
  return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

const onlineCols: DataTableColumns<any> = [
  { title: '昵称', key: 'nickname' },
  { title: 'SteamID', key: 'steam_id', width: 160 },
  { title: 'IP', key: 'ip', width: 130 },
  { title: '等级', key: 'level', width: 70 },
  { title: 'Ping', key: 'ping', width: 80 },
]

const playerCols: DataTableColumns<any> = [
  { title: '昵称', key: 'nickname' },
  {
    title: '状态', key: 'online', width: 90,
    render: (r) => isOnline(r)
      ? h(NTag, { size: 'small', type: 'success', round: true, bordered: false }, { default: () => '在线' })
      : h(NTag, { size: 'small', round: true, bordered: false }, { default: () => '离线' }),
  },
  { title: '等级', key: 'level', width: 80 },
  { title: 'SteamID', key: 'steam_id', width: 170 },
  { title: '帕鲁数', key: 'pals', width: 90, render: (r) => r.pals?.length ?? 0 },
  { title: '最后在线', key: 'last_online', width: 180, render: (r) => formatTime(r.last_online || r.save_last_online) },
]

const palCols: DataTableColumns<any> = [
  { title: '等级', key: 'level', width: 70 },
  { title: '攻击', key: 'melee', width: 70 },
  { title: '防御', key: 'defense', width: 70 },
  {
    title: '标记', key: 'is_boss', width: 160,
    render: (r) => h('span', { style: 'display:flex;gap:4px' }, [
      r.is_boss ? h(NTag, { size: 'small', type: 'error' }, { default: () => 'Boss' }) : null,
      r.is_tower ? h(NTag, { size: 'small', type: 'warning' }, { default: () => '塔主' }) : null,
      r.is_lucky ? h(NTag, { size: 'small', type: 'success' }, { default: () => '闪光' }) : null,
    ]),
  },
]

const itemCols: DataTableColumns<any> = [
  {
    title: '图标', key: 'icon', width: 50,
    render: (r) => r.itemInfo?.icon
      ? h(NImage, { src: r.itemInfo.icon, width: 32, height: 32, objectFit: 'contain' })
      : null,
  },
  { title: '物品', key: 'ItemId', render: (r) => r.itemInfo?.name || r.ItemId },
  { title: '数量', key: 'StackCount', width: 80 },
  { title: '位置', key: 'container', width: 100 },
]

function rowProps(row: any) {
  return {
    style: 'cursor: pointer',
    onClick: async () => {
      try {
        detail.value = await api.getPlayer(row.player_uid)
        detailTab.value = 'basic'
        showDetail.value = true
      } catch (e: any) {
        message.error(e.message)
      }
    },
  }
}

// PalDefender 玩家标识：优先 steam_<steam_id>，缺失时回退 PlayerUID
function pdId(p: any): string {
  return p.steam_id ? `steam_${p.steam_id}` : p.player_uid
}

const ACT_TITLE: Record<string, string> = {
  kick: '踢出玩家',
  ban: '封禁玩家',
  unban: '解封玩家',
  ipban: 'IP 封禁',
}

function openAct(kind: 'kick' | 'ban' | 'unban' | 'ipban', p: any) {
  act.kind = kind
  act.title = ACT_TITLE[kind]
  act.reason = ''
  act.ip = p.ip || ''
  act.ipBan = false
  act.target = p
  act.show = true
}

async function confirmAct() {
  const p = act.target
  if (!p) return
  if (act.kind === 'ipban' && !act.ip.trim()) {
    message.warning('请输入 IP 地址')
    return
  }
  act.loading = true
  try {
    const reason = act.reason.trim()
    if (act.kind === 'kick') await api.pdKick(pdId(p), reason)
    else if (act.kind === 'ban') await api.pdBan(pdId(p), reason, act.ipBan)
    else if (act.kind === 'unban') await api.pdUnban(pdId(p), reason)
    else if (act.kind === 'ipban') await api.pdBanIP(act.ip.trim(), reason)
    message.success('操作成功')
    act.show = false
    await refreshAfterAction(p)
  } catch (e: any) {
    message.error(e.message)
  } finally {
    act.loading = false
  }
}

async function refreshAfterAction(p: any) {
  try { detail.value = await api.getPlayer(p.player_uid) } catch {}
  try { players.value = await api.getPlayers() } catch {}
  try { online.value = await api.getOnline() } catch {}
}

async function loadAll() {
  loading.value = true
  try {
    const [pl, on] = await Promise.all([
      api.getPlayers().catch(() => []),
      api.getOnline().catch(() => []),
    ])
    players.value = pl
    online.value = on
  } finally {
    loading.value = false
  }
}

onMounted(loadAll)
</script>

<style scoped>
.drawer-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 600;
}
</style>
