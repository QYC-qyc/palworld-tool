<template>
  <n-space vertical>
    <n-card title="白名单" size="small">
      <template #header-extra>
        <n-button type="primary" @click="showAdd = true">添加</n-button>
      </template>
      <n-data-table :columns="cols" :data="list" :bordered="false" size="small" />
    </n-card>

    <n-modal v-model:show="showAdd" preset="card" title="添加白名单" style="max-width: 460px">
      <n-space vertical>
        <n-input v-model:value="form.name" placeholder="昵称" />
        <n-input v-model:value="form.steam_id" placeholder="SteamID（纯数字）" />
        <n-input v-model:value="form.player_uid" placeholder="PlayerUID（可选）" />
      </n-space>
      <template #footer>
        <n-button type="primary" @click="add">保存</n-button>
      </template>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  NSpace, NCard, NDataTable, NButton, NModal, NInput, useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const list = ref<any[]>([])
const showAdd = ref(false)
const form = ref({ name: '', steam_id: '', player_uid: '' })

const cols: DataTableColumns<any> = [
  { title: '昵称', key: 'name' },
  { title: 'SteamID', key: 'steam_id' },
  { title: 'PlayerUID', key: 'player_uid' },
  {
    title: '操作', key: 'actions', width: 90,
    render: (r) => h(NButton, { size: 'small', type: 'error', ghost: true, onClick: () => remove(r) }, { default: () => '移除' }),
  },
]

async function load() {
  try { list.value = await api.getWhitelist() } catch {}
}
async function add() {
  try {
    await api.addWhitelist(form.value)
    message.success('已添加')
    showAdd.value = false
    form.value = { name: '', steam_id: '', player_uid: '' }
    load()
  } catch (e: any) { message.error(e.message) }
}
async function remove(r: any) {
  try {
    await api.removeWhitelist({ name: r.name, steam_id: r.steam_id, player_uid: r.player_uid })
    message.success('已移除')
    load()
  } catch (e: any) { message.error(e.message) }
}

import { h } from 'vue'
onMounted(load)
</script>
