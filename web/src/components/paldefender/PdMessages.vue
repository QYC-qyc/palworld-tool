<template>
  <n-space vertical :size="16">
    <n-card title="全服广播" size="small">
      <n-space vertical>
        <n-input v-model:value="broadcastMsg" type="textarea" :autosize="{ minRows: 2 }"
          placeholder="输入全服广播消息" />
        <n-button type="primary" :loading="sending" @click="onBroadcast">发送广播</n-button>
      </n-space>
    </n-card>

    <n-card title="高优先级警报" size="small">
      <n-space vertical>
        <n-alert type="warning" :show-icon="false" style="font-size:12px">
          警报为高优先级消息，会在游戏内以醒目方式提示玩家，请谨慎使用。
        </n-alert>
        <n-input v-model:value="alertMsg" type="textarea" :autosize="{ minRows: 2 }"
          placeholder="输入警报消息" />
        <n-button type="error" :loading="sending" @click="onAlert">发送警报</n-button>
      </n-space>
    </n-card>

    <n-card title="私聊玩家" size="small">
      <n-space vertical>
        <n-space align="center" :wrap="false">
          <n-select
            v-model:value="msgTarget"
            filterable
            placeholder="选择玩家"
            :options="playerOptions"
            style="max-width:320px"
          />
          <n-button v-if="!playersLoaded" size="small" @click="$emit('loadPlayers')">
            加载玩家列表
          </n-button>
        </n-space>
        <n-input v-model:value="msgContent" type="textarea" :autosize="{ minRows: 2 }"
          placeholder="输入私聊消息" />
        <n-button type="primary" :loading="sending" @click="onPrivateMsg">发送私聊</n-button>
      </n-space>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { NSpace, NCard, NAlert, NInput, NButton, NSelect, useMessage } from 'naive-ui'
import { api } from '@/api'

const props = defineProps<{
  playerOptions: { label: string; value: string }[]
  playersLoaded: boolean
}>()

defineEmits<{
  (e: 'loadPlayers'): void
}>()

const message = useMessage()
const broadcastMsg = ref('')
const alertMsg = ref('')
const msgTarget = ref<string | null>(null)
const msgContent = ref('')
const sending = ref(false)

async function onBroadcast() {
  if (!broadcastMsg.value.trim()) return
  sending.value = true
  try {
    await api.pdBroadcast(broadcastMsg.value)
    message.success('广播已发送')
    broadcastMsg.value = ''
  } catch (e: any) {
    message.error(e?.response?.data?.error || '广播失败')
  } finally {
    sending.value = false
  }
}

async function onAlert() {
  if (!alertMsg.value.trim()) return
  sending.value = true
  try {
    await api.pdAlert(alertMsg.value)
    message.success('警报已发送')
    alertMsg.value = ''
  } catch (e: any) {
    message.error(e?.response?.data?.error || '警报失败')
  } finally {
    sending.value = false
  }
}

async function onPrivateMsg() {
  if (!msgTarget.value || !msgContent.value.trim()) {
    message.warning('请选择玩家并输入消息')
    return
  }
  sending.value = true
  try {
    await api.pdMessage(msgTarget.value, msgContent.value)
    message.success('私聊已发送')
    msgContent.value = ''
  } catch (e: any) {
    message.error(e?.response?.data?.error || '私聊失败')
  } finally {
    sending.value = false
  }
}
</script>
