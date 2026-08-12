<template>
  <n-space vertical :size="16">
    <n-card title="服务器连接" size="small">
      <n-form label-placement="left" label-width="140">
        <n-form-item label="REST 地址">
          <n-input v-model:value="form['rest.address']" placeholder="http://127.0.0.1:8212" />
        </n-form-item>
        <n-form-item label="REST 用户名">
          <n-input v-model:value="form['rest.username']" placeholder="admin" />
        </n-form-item>
        <n-form-item label="Admin 密码">
          <n-input
            v-model:value="restPwd"
            type="password"
            show-password-on="click"
            :placeholder="form['rest.password__set'] === 'true' ? '已设置（留空不修改）' : '未设置'"
          />
        </n-form-item>
        <n-form-item label="RCON 地址">
          <n-input v-model:value="form['rcon.address']" placeholder="127.0.0.1:25575" />
        </n-form-item>
        <n-form-item label="RCON 密码">
          <n-input
            v-model:value="rconPwd"
            type="password"
            show-password-on="click"
            :placeholder="form['rcon.password__set'] === 'true' ? '已设置（留空不修改）' : '未设置'"
          />
        </n-form-item>
        <n-form-item label="RCON Base64">
          <n-switch :value="form['rcon.use_base64'] === 'true'"
            @update:value="(v) => (form['rcon.use_base64'] = v ? 'true' : 'false')" />
        </n-form-item>
      </n-form>
    </n-card>

    <n-card title="存档与进程" size="small">
      <n-form label-placement="left" label-width="140">
        <n-form-item label="存档 Saved 目录">
          <n-input v-model:value="form['save.path']"
            placeholder="/home/steam/Pal/Saved/SaveGames/0/<GUID>" />
        </n-form-item>
        <n-grid cols="1 s:3" :x-gap="12">
          <n-gi>
            <n-form-item label="进程控制" label-placement="top">
              <n-select v-model:value="form['process.mode']" :options="processModes" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="systemd 服务名" label-placement="top">
              <n-input v-model:value="form['process.service']" placeholder="palworld" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="docker 容器名" label-placement="top">
              <n-input v-model:value="form['process.container']" placeholder="palworld" />
            </n-form-item>
          </n-gi>
        </n-grid>
      </n-form>
    </n-card>

    <n-card title="反作弊" size="small">
      <n-form label-placement="left" label-width="140">
        <n-form-item label="启用反作弊">
          <n-switch :value="form['anticheat.enabled'] === 'true'"
            @update:value="(v) => (form['anticheat.enabled'] = v ? 'true' : 'false')" />
        </n-form-item>
        <n-form-item label="反作弊模式">
          <n-select v-model:value="form['anticheat.mode']" :options="acModes" />
        </n-form-item>
        <n-form-item label="PalDefender 地址">
          <n-input v-model:value="form['paldefender.address']"
            placeholder="http://127.0.0.1:17993（集成模式）" />
        </n-form-item>
        <n-form-item label="PalDefender Token">
          <n-input v-model:value="pdToken" type="password" show-password-on="click"
            :placeholder="form['paldefender.token__set'] === 'true' ? '已设置（留空不修改）' : '未设置'" />
        </n-form-item>
      </n-form>
    </n-card>

    <n-card title="面板与通知" size="small">
      <n-form label-placement="left" label-width="140">
        <n-form-item label="面板新密码">
          <n-input v-model:value="webPwd" type="password" show-password-on="click"
            :placeholder="form['web.password__set'] === 'true' ? '已设置（留空不修改）' : '未设置'" />
        </n-form-item>
        <n-form-item label="Webhook 地址">
          <n-input v-model:value="form['notify.webhook_url']" placeholder="钉钉/企业微信/Discord webhook" />
        </n-form-item>
        <n-form-item label="Webhook 类型">
          <n-select v-model:value="form['notify.webhook_type']" :options="webhookTypes" />
        </n-form-item>
      </n-form>
    </n-card>

    <n-card size="small">
      <n-space>
        <n-button type="primary" :loading="saving" @click="save">保存设置</n-button>
        <n-tag :bordered="false">
          端口 {{ form['web.port'] }}（静态，需重启修改）
        </n-tag>
        <n-text depth="3" style="font-size:12px">
          密码/Token 类字段留空表示不修改；连接、反作弊、存档路径等保存后即时生效；
          反作弊模式（external/integrated）切换需重启服务。
        </n-text>
      </n-space>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import {
  NSpace, NCard, NForm, NFormItem, NInput, NSwitch, NGrid, NGi,
  NSelect, NButton, NTag, NText, useMessage,
} from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const form = reactive<Record<string, any>>({})
const saving = ref(false)
const restPwd = ref('')
const rconPwd = ref('')
const pdToken = ref('')
const webPwd = ref('')

const processModes = [
  { label: '不控制（手动停服）', value: 'noop' },
  { label: 'systemd', value: 'systemd' },
  { label: 'docker', value: 'docker' },
]
const acModes = [
  { label: '外置（存档扫描，默认）', value: 'external' },
  { label: '集成 PalDefender', value: 'integrated' },
]
const webhookTypes = [
  { label: '通用 JSON', value: 'generic' },
  { label: '钉钉', value: 'dingtalk' },
  { label: '企业微信', value: 'wechat' },
  { label: 'Discord', value: 'discord' },
]

async function load() {
  const s = await api.getSettings()
  Object.keys(s).forEach((k) => (form[k] = s[k]))
}

async function save() {
  saving.value = true
  try {
    const payload: Record<string, any> = {}
    Object.keys(form).forEach((k) => {
      if (!k.endsWith('__set')) payload[k] = form[k]
    })
    // 仅当填写了新值才提交密码类字段
    if (restPwd.value) payload['rest.password'] = restPwd.value
    if (rconPwd.value) payload['rcon.password'] = rconPwd.value
    if (pdToken.value) payload['paldefender.token'] = pdToken.value
    if (webPwd.value) payload['web.password'] = webPwd.value
    await api.saveSettings(payload)
    message.success('设置已保存，部分更改需重启生效')
    restPwd.value = ''
    rconPwd.value = ''
    pdToken.value = ''
    webPwd.value = ''
    load()
  } catch (e: any) {
    message.error(e.message)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
