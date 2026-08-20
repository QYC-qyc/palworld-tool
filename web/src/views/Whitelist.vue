<template>
  <n-space vertical :size="16">
    <PageHeader title="白名单" subtitle="管理允许加入服务器的玩家">
      <n-button @click="showImport = true">
        <template #icon><n-icon :component="CloudUploadOutline" /></template>
        批量导入
      </n-button>
      <n-button type="primary" @click="openAdd">
        <template #icon><n-icon :component="AddOutline" /></template>
        添加
      </n-button>
    </PageHeader>

    <n-card size="small">
      <n-space vertical :size="12">
        <n-input v-model:value="search" placeholder="搜索昵称 / SteamID" clearable style="max-width:320px">
          <template #prefix><n-icon :component="SearchOutline" /></template>
        </n-input>
        <n-data-table :columns="cols" :data="filtered" :bordered="false" size="small" :pagination="{ pageSize: 30, prefix: ({ itemCount }) => `共 ${itemCount} 条` }" />
      </n-space>
    </n-card>

    <n-modal v-model:show="showAdd" preset="card" title="添加白名单" style="max-width: 460px">
      <n-space vertical>
        <n-input v-model:value="form.name" placeholder="昵称" />
        <n-input v-model:value="form.steam_id" placeholder="SteamID（纯数字）" />
        <n-input v-model:value="form.player_uid" placeholder="PlayerUID（可选）" />
      </n-space>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showAdd = false">取消</n-button>
          <n-button type="primary" @click="add">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal v-model:show="showImport" preset="card" title="批量导入白名单" style="max-width:520px">
      <n-space vertical>
        <n-alert type="info" :show-icon="false" style="font-size:12px">
          每行一条，格式：<code>昵称,SteamID,PlayerUID</code>（PlayerUID 可省略，用逗号或空格分隔）。
        </n-alert>
        <n-input v-model:value="importText" type="textarea" :autosize="{ minRows: 6 }"
          placeholder="张三,76561198xxxxxxx&#10;李四,76561198yyyyyyy,12345678" />
      </n-space>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showImport = false">取消</n-button>
          <n-button type="primary" :loading="importing" @click="doImport">导入</n-button>
        </n-space>
      </template>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { h, onMounted, ref, computed } from 'vue'
import {
  NSpace, NCard, NDataTable, NButton, NModal, NInput, NIcon,
  NAlert, useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { SearchOutline, AddOutline, CloudUploadOutline } from '@vicons/ionicons5'
import { api } from '@/api'
import PageHeader from '@/components/PageHeader.vue'

const message = useMessage()
const list = ref<any[]>([])
const showAdd = ref(false)
const showImport = ref(false)
const importing = ref(false)
const importText = ref('')
const form = ref({ name: '', steam_id: '', player_uid: '' })
const search = ref('')

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return list.value
  return list.value.filter(
    (r) =>
      String(r.name || '').toLowerCase().includes(q) ||
      String(r.steam_id || '').toLowerCase().includes(q)
  )
})

const cols: DataTableColumns<any> = [
  { title: '昵称', key: 'name' },
  { title: 'SteamID', key: 'steam_id' },
  { title: 'PlayerUID', key: 'player_uid' },
  {
    title: '操作', key: 'actions', width: 90,
    render: (r) => h(NButton, {
      size: 'small', type: 'error', ghost: true,
      onClick: () => remove(r),
    }, { default: () => '移除' }),
  },
]

function openAdd() {
  form.value = { name: '', steam_id: '', player_uid: '' }
  showAdd.value = true
}

async function load() {
  try { list.value = await api.getWhitelist() } catch {}
}
async function add() {
  if (!form.value.name || !form.value.steam_id) {
    message.warning('请填写昵称和 SteamID')
    return
  }
  try {
    await api.addWhitelist(form.value)
    message.success('已添加')
    showAdd.value = false
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

async function doImport() {
  const lines = importText.value
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter(Boolean)
  if (!lines.length) {
    message.warning('请输入要导入的数据')
    return
  }
  importing.value = true
  let ok = 0
  let fail = 0
  for (const line of lines) {
    const parts = line.split(/[,\s]+/).filter(Boolean)
    if (parts.length < 2) { fail++; continue }
    try {
      await api.addWhitelist({ name: parts[0], steam_id: parts[1], player_uid: parts[2] || '' })
      ok++
    } catch {
      fail++
    }
  }
  importing.value = false
  message.success(`导入完成：成功 ${ok} 条${fail ? `，失败 ${fail} 条` : ''}`)
  showImport.value = false
  importText.value = ''
  load()
}

onMounted(load)
</script>
