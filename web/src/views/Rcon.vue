<template>
  <n-grid cols="1 s:2" responsive="screen" :x-gap="12">
    <n-gi>
      <n-card title="RCON 命令" size="small">
        <n-space vertical>
          <n-select
            v-model:value="selectedCmd"
            :options="cmdOptions"
            placeholder="选择内置命令"
            filterable
            @update:value="onSelectCmd"
          />
          <n-input v-model:value="content" :placeholder="placeholder || '参数（可选）'" />
          <n-button type="primary" @click="send" :loading="sending" :disabled="!selectedCmd">
            执行
          </n-button>
          <n-divider style="margin: 4px 0" />
          <n-text depth="3" style="font-size:12px">自定义命令</n-text>
          <n-input
            v-model:value="customCmd"
            placeholder="直接输入 RCON 命令，如 ShowPlayers"
            @keyup.enter="sendCustom"
          />
          <n-button @click="sendCustom" :loading="sending" :disabled="!customCmd">
            执行自定义命令
          </n-button>
        </n-space>
      </n-card>
    </n-gi>
    <n-gi>
      <n-card title="响应" size="small">
        <n-space vertical :size="8">
          <n-text depth="3" style="font-size:12px" v-if="selectedCmdInfo">
            {{ selectedCmdInfo.remark }}
            <template v-if="selectedCmdInfo.command === 'Shutdown'"> — 例如 <code>Shutdown 60 服务器将在60秒后关闭</code></template>
            <template v-else-if="selectedCmdInfo.command === 'Broadcast'"> — 例如 <code>Broadcast 服务器即将维护</code></template>
          </n-text>
          <n-input v-model:value="response" type="textarea" readonly :rows="16"
            :input-props="{ style: 'font-family: monospace; font-size: 12px' }" />
        </n-space>
      </n-card>
    </n-gi>
  </n-grid>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  NGrid, NGi, NCard, NSpace, NSelect, NInput, NButton, NDivider, NText, useMessage,
} from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const commands = ref<any[]>([])
const selectedCmd = ref<string | null>(null)
const content = ref('')
const customCmd = ref('')
const response = ref('')
const sending = ref(false)

const cmdOptions = computed(() =>
  commands.value.map((c) => ({
    label: c.remark ? `${c.command} — ${c.remark}` : c.command,
    value: c.command,
  }))
)

const selectedCmdInfo = computed(() => {
  if (!selectedCmd.value) return null
  return commands.value.find((c) => c.command === selectedCmd.value)
})

const placeholder = computed(() => selectedCmdInfo.value?.placeholder || '')

function onSelectCmd(cmd: string) {
  content.value = ''
  const info = commands.value.find((c) => c.command === cmd)
  if (info?.placeholder) {
    // 占位提示已显示在输入框 placeholder 中
  }
}

async function send() {
  if (!selectedCmd.value) return
  sending.value = true
  try {
    let cmd = selectedCmd.value
    if (content.value.trim()) cmd += ' ' + content.value.trim()
    const res = await api.sendRcon(cmd, '')
    response.value = res.response || '(无响应)'
  } catch (e: any) {
    response.value = '错误: ' + e.message
  } finally { sending.value = false }
}

async function sendCustom() {
  if (!customCmd.value) return
  sending.value = true
  try {
    const fullCmd = customCmd.value
    const res = await api.sendRcon(fullCmd, '')
    response.value = res.response || '(无响应)'
  } catch (e: any) {
    response.value = '错误: ' + e.message
  } finally { sending.value = false }
}

onMounted(async () => {
  try { commands.value = await api.getRconCommands() } catch {}
})
</script>
