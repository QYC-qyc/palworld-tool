<template>
  <n-space vertical :size="16">
    <n-card title="服务器连接" size="small">
      <n-alert type="info" :show-icon="false" style="margin-bottom:12px">
        面板本地启停游戏服时，REST/RCON 地址与管理员密码会在「游戏配置」保存网络项时自动同步为本机地址；
        仅当游戏服部署在其他机器时才需手动修改此处。
      </n-alert>
      <n-form label-placement="left" label-width="140">
        <n-form-item label="REST 地址">
          <n-input v-model:value="form['rest.address']" placeholder="http://palworld:8212" />
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
          <n-input v-model:value="form['rcon.address']" placeholder="palworld:25575" />
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
            placeholder="/game/Saved/SaveGames/0/<GUID>" />
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
      </n-form>
    </n-card>

    <n-card title="面板" size="small">
      <n-form label-placement="left" label-width="140">
        <n-form-item label="面板新密码">
          <n-input v-model:value="webPwd" type="password" show-password-on="click"
            :placeholder="form['web.password__set'] === 'true' ? '已设置（留空不修改）' : '未设置'" />
        </n-form-item>
      </n-form>
    </n-card>

    <n-card size="small">
      <n-space>
        <n-button type="primary" :loading="saving" @click="save">保存设置</n-button>
        <n-tag :bordered="false">端口 {{ form['web.port'] }}（静态，需重启修改）</n-tag>
        <n-text depth="3" style="font-size:12px">
          密码类字段留空表示不修改；连接、反作弊、存档路径等保存后即时生效。
        </n-text>
      </n-space>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import {
  NSpace, NCard, NForm, NFormItem, NInput, NSwitch, NGrid, NGi,
  NSelect, NButton, NTag, NText, NAlert, useMessage,
} from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const form = reactive<Record<string, any>>({})
const saving = ref(false)
const restPwd = ref('')
const rconPwd = ref('')
const webPwd = ref('')

const processModes = [
  { label: '不控制（手动停服）', value: 'noop' },
  { label: 'systemd', value: 'systemd' },
  { label: 'docker', value: 'docker' },
]
const acModes = [
  { label: '外置（存档扫描，纯 Linux 原生）', value: 'external' },
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
    if (restPwd.value) payload['rest.password'] = restPwd.value
    if (rconPwd.value) payload['rcon.password'] = rconPwd.value
    if (webPwd.value) payload['web.password'] = webPwd.value
    await api.saveSettings(payload)
    message.success('设置已保存')
    restPwd.value = ''
    rconPwd.value = ''
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
