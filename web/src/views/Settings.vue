<template>
  <n-space vertical :size="16">
    <n-card title="服务器连接" size="small">
      <n-alert type="info" :show-icon="false" style="margin-bottom:12px">
        面板本地启停游戏服时，地址与管理员密码会在「游戏配置」保存网络项时自动同步；
        仅当游戏服部署在其他机器时才需手动修改。REST 与 RCON 使用同一个管理员密码（AdminPassword）。
      </n-alert>
      <n-form label-placement="left" label-width="140">
        <n-form-item label="REST 地址">
          <n-input v-model:value="form['rest.address']" placeholder="http://127.0.0.1:8212">
            <template #suffix>
              <n-button size="tiny" quaternary :loading="testing === 'rest'" @click="testConn('rest')">
                测试
              </n-button>
            </template>
          </n-input>
        </n-form-item>
        <n-form-item label="RCON 地址">
          <n-input v-model:value="form['rcon.address']" placeholder="127.0.0.1:25575">
            <template #suffix>
              <n-button size="tiny" quaternary :loading="testing === 'rcon'" @click="testConn('rcon')">
                测试
              </n-button>
            </template>
          </n-input>
        </n-form-item>
        <n-form-item label="管理员密码">
          <n-input
            v-model:value="adminPwd"
            type="password"
            show-password-on="click"
            :placeholder="form['rest.password__set'] === 'true' ? '已设置（留空不修改）' : '未设置，需与游戏配置 AdminPassword 一致'"
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
            :placeholder="form['save.path_effective'] ? `留空自动使用：${form['save.path_effective']}` : '留空则从游戏安装目录自动查找'" />
          <n-text depth="3" style="font-size:12px">
            通常无需手动填写，留空时自动使用游戏安装目录下的 Pal/Saved
          </n-text>
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
const testing = ref<string>('')
const adminPwd = ref('')
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

async function testConn(type: 'rest' | 'rcon') {
  testing.value = type
  try {
    const address = type === 'rest' ? form['rest.address'] : form['rcon.address']
    if (!address) {
      message.warning('请先填写地址')
      return
    }
    // 优先使用刚输入的密码，否则空密码（已保存的密码后端自动使用）
    const password = adminPwd.value || ''
    const res = await fetch('/api/settings/test-connection', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${localStorage.getItem('paladmin_token')}`,
      },
      body: JSON.stringify({
        type,
        address,
        password,
        use_base64: form['rcon.use_base64'] === 'true',
      }),
    })
    const data = await res.json()
    if (data.success) {
      message.success(data.message + (data.version ? `（版本 ${data.version}）` : ''))
    } else {
      message.error(data.error || '连接失败')
    }
  } catch (e: any) {
    message.error(e.message || '连接失败')
  } finally {
    testing.value = ''
  }
}

async function save() {
  saving.value = true
  try {
    const payload: Record<string, any> = {}
    Object.keys(form).forEach((k) => {
      if (!k.endsWith('__set') && !k.endsWith('_effective')) payload[k] = form[k]
    })
    // REST 与 RCON 共用管理员密码，同时写入
    if (adminPwd.value) {
      payload['rest.password'] = adminPwd.value
      payload['rcon.password'] = adminPwd.value
    }
    if (webPwd.value) payload['web.password'] = webPwd.value
    await api.saveSettings(payload)
    message.success('设置已保存')
    adminPwd.value = ''
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
