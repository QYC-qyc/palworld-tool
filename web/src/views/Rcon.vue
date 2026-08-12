<template>
  <n-grid cols="1 s:2" responsive="screen" :x-gap="12">
    <n-gi>
      <n-card title="RCON 命令" size="small">
        <n-space vertical>
          <n-select
            v-model:value="selectedCmd"
            :options="cmdOptions"
            placeholder="选择保存的命令"
            filterable
          />
          <n-input v-model:value="content" placeholder="参数（可选）" />
          <n-button type="primary" @click="send" :loading="sending">执行</n-button>
          <n-input
            v-model:value="customCmd"
            placeholder="或直接输入任意 RCON 命令，如 ShowPlayers"
          />
          <n-button @click="sendCustom" :loading="sending">执行自定义命令</n-button>
        </n-space>
      </n-card>
    </n-gi>
    <n-gi>
      <n-card title="响应" size="small">
        <n-input v-model:value="response" type="textarea" readonly rows="14" />
      </n-card>
    </n-gi>
  </n-grid>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  NGrid, NGi, NCard, NSpace, NSelect, NInput, NButton, useMessage,
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
  commands.value.map((c) => ({ label: `${c.command} — ${c.remark || ''}`, value: c.command }))
)

async function send() {
  if (!selectedCmd.value) return
  sending.value = true
  try {
    const res = await api.sendRcon(selectedCmd.value, content.value)
    response.value = res.response || '(无响应)'
  } catch (e: any) {
    response.value = '错误: ' + e.message
  } finally { sending.value = false }
}

async function sendCustom() {
  if (!customCmd.value) return
  sending.value = true
  try {
    const parts = customCmd.value.split(' ')
    const cmd = parts[0]
    const arg = parts.slice(1).join(' ')
    const res = await api.sendRcon(cmd, arg)
    response.value = res.response || '(无响应)'
  } catch (e: any) {
    response.value = '错误: ' + e.message
  } finally { sending.value = false }
}

onMounted(async () => {
  try { commands.value = await api.getRconCommands() } catch {}
})
</script>
