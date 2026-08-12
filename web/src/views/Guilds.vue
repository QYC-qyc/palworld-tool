<template>
  <n-card title="公会列表" size="small">
    <n-data-table :columns="cols" :data="guilds" :bordered="false" size="small" />
  </n-card>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { NCard, NDataTable } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { api } from '@/api'

const guilds = ref<any[]>([])
const cols: DataTableColumns<any> = [
  { title: '公会名', key: 'name' },
  { title: '据点等级', key: 'base_camp_level', width: 100 },
  { title: '会长 UID', key: 'admin_player_uid', width: 160 },
  { title: '成员数', key: 'players', width: 90, render: (r) => r.players?.length ?? 0 },
  { title: '据点数', key: 'base_ids', width: 90, render: (r) => r.base_ids?.length ?? 0 },
]

onMounted(async () => {
  try { guilds.value = await api.getGuilds() } catch {}
})
</script>
