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

    <n-drawer v-model:show="showDetail" :width="640">
      <n-drawer-content :title="`玩家详情 - ${detail?.nickname || ''}`" closable>
        <n-descriptions v-if="detail" :column="1" bordered size="small">
          <n-descriptions-item label="PlayerUID">{{ detail.player_uid }}</n-descriptions-item>
          <n-descriptions-item label="SteamID">{{ detail.steam_id }}</n-descriptions-item>
          <n-descriptions-item label="等级">{{ detail.level }}</n-descriptions-item>
          <n-descriptions-item label="HP">{{ detail.hp }} / {{ detail.max_hp }}</n-descriptions-item>
          <n-descriptions-item label="IP">{{ detail.ip }}</n-descriptions-item>
          <n-descriptions-item label="最后在线">{{ formatTime(detail.last_online || detail.save_last_online) }}</n-descriptions-item>
        </n-descriptions>
        <n-tabs type="line" animated style="margin-top:16px">
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
        <n-divider>操作</n-divider>
        <n-space>
          <n-button type="warning" ghost @click="openAct('kick', detail)">踢出</n-button>
          <n-button type="error" ghost @click="openAct('ban', detail)">封禁</n-button>
          <n-button @click="openAct('unban', detail)">解封</n-button>
          <n-button type="error" @click="openAct('ipban', detail)">IP封禁</n-button>
        </n-space>
      </n-drawer-content>
    </n-drawer>

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
  useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const players = ref<any[]>([])
const online = ref<any[]>([])
const showDetail = ref(false)
const detail = ref<any>(null)
const itemMap = ref<Record<string, any>>({})

// 操作弹框
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

// 加载物品图标映射
fetch('/data/item_list.json').then(r => r.json()).then((data: any[]) => {
  const m: Record<string, any> = {}
  data.forEach(i => { m[i.id] = i })
  itemMap.value = m
}).catch(() => {})

// 物品列表（合并所有容器）
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

function formatTime(t: string): string {
  if (!t) return '-'
  // ISO时间或时间戳
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
    title: '标记', key: 'is_boss', width: 140,
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
      ? h(NImage, { src: r.itemInfo.icon, width: 32, height: 32, objectFit: 'contain', style: 'object-fit:contain' })
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
    // 刷新详情与列表
    try { detail.value = await api.getPlayer(p.player_uid) } catch {}
    try { players.value = await api.getPlayers() } catch {}
    try { online.value = await api.getOnline() } catch {}
  } catch (e: any) {
    message.error(e.message)
  } finally {
    act.loading = false
  }
}

onMounted(async () => {
  try { players.value = await api.getPlayers() } catch {}
  try { online.value = await api.getOnline() } catch {}
})
</script>
