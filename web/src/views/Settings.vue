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

    <n-card title="面板" size="small">
      <n-form label-placement="left" label-width="140">
        <n-form-item label="面板新密码">
          <n-input v-model:value="webPwd" type="password" show-password-on="click"
            :placeholder="form['web.password__set'] === 'true' ? '已设置（留空不修改）' : '未设置'" />
        </n-form-item>
      </n-form>
    </n-card>

    <n-card title="面板更新" size="small">
      <n-space vertical :size="12">
        <n-space align="center" :wrap="false">
          <n-tag :bordered="false" :type="updateInfo.has_update ? 'warning' : 'success'">
            当前版本：{{ updateInfo.current || '未知' }}
          </n-tag>
          <n-text v-if="updateInfo.has_update" strong>
            发现新版本：{{ updateInfo.latest }}
          </n-text>
          <n-text v-else depth="3">已是最新版本</n-text>
          <n-button size="small" :loading="checking" @click="checkUpdate">检查更新</n-button>
          <n-button v-if="updateInfo.has_update" size="small" type="primary"
            :loading="updating" @click="doUpdate">
            立即更新
          </n-button>
        </n-space>
        <n-alert v-if="updateInfo.has_update && updateInfo.body" type="info" :show-icon="false"
          style="white-space:pre-wrap;max-height:200px;overflow:auto;font-size:12px">
          {{ updateInfo.body }}
        </n-alert>
        <n-text v-if="updateInfo.error" depth="3" style="font-size:12px;color:#d03050">
          {{ updateInfo.error }}
        </n-text>
        <n-text depth="3" style="font-size:12px">
          更新会自动下载最新二进制与前端资源并重启服务，期间面板短暂不可用。
        </n-text>
      </n-space>
    </n-card>

    <!-- 更新进度弹窗 -->
    <n-modal v-model:show="showUpdateProgress" preset="card" title="面板更新"
      style="max-width:520px" :mask-closable="false" :close-on-esc="false">
      <n-space vertical :size="16">
        <n-progress
          type="line"
          :percentage="updateProgress.percent"
          :status="updateProgress.stage === 'error' ? 'error' : updateProgress.stage === 'done' ? 'success' : 'default'"
          :indicator-placement="'inside'"
        />
        <n-text>{{ updateProgress.message }}</n-text>
        <n-text v-if="updateProgress.stage === 'done'" depth="3" style="font-size:12px">
          服务正在重启，稍后自动刷新...
        </n-text>
      </n-space>
    </n-modal>

    <n-card size="small">
      <n-space>
        <n-button type="primary" :loading="saving" @click="save">保存设置</n-button>
        <n-tag :bordered="false">端口 {{ form['web.port'] }}（静态，需重启修改）</n-tag>
        <n-text depth="3" style="font-size:12px">
          密码类字段留空表示不修改；连接、存档路径等保存后即时生效。
        </n-text>
      </n-space>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import {
  NSpace, NCard, NForm, NFormItem, NInput, NSwitch, NGrid, NGi,
  NSelect, NButton, NTag, NText, NAlert, NModal, NProgress, useMessage,
} from 'naive-ui'
import { api } from '@/api'

const message = useMessage()
const form = reactive<Record<string, any>>({})
const saving = ref(false)
const testing = ref<string>('')
const adminPwd = ref('')
const webPwd = ref('')
const checking = ref(false)
const updating = ref(false)
const showUpdateProgress = ref(false)
const updateProgress = reactive<{ stage: string; message: string; percent: number }>({
  stage: '',
  message: '',
  percent: 0,
})
const updateInfo = reactive<{
  current: string
  has_update: boolean
  latest?: string
  name?: string
  body?: string
  error?: string
}>({ current: '', has_update: false })

const processModes = [
  { label: '不控制（手动停服）', value: 'noop' },
  { label: 'systemd', value: 'systemd' },
  { label: 'docker', value: 'docker' },
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

onMounted(async () => {
  await load()
  checkUpdate()
})

async function checkUpdate() {
  checking.value = true
  try {
    const info = await api.checkUpdate()
    Object.assign(updateInfo, info)
  } catch (e: any) {
    updateInfo.error = e.message
  } finally {
    checking.value = false
  }
}

async function doUpdate() {
  updating.value = true
  showUpdateProgress.value = true
  updateProgress.percent = 0
  updateProgress.message = '正在准备更新...'

  const token = localStorage.getItem('paladmin_token') || ''
  try {
    // 触发更新
    const resp = await fetch('/api/updater/do', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
    })
    const data = await resp.json()
    if (!resp.ok || data.error) {
      updating.value = false
      message.error(data.error || '更新失败')
      return
    }

    // 先轮询 status 获取下载进度，直到 status 返回 done 或连接断开
    let elapsed = 0
    const pollStatus = setInterval(async () => {
      elapsed += 1
      try {
        const progResp = await fetch('/api/updater/status', {
          headers: { Authorization: `Bearer ${token}` },
          signal: AbortSignal.timeout(3000),
        })
        if (progResp.ok) {
          const prog = await progResp.json()
          if (prog.message) updateProgress.message = prog.message
          if (typeof prog.percent === 'number' && prog.percent > 0) {
            updateProgress.percent = prog.percent
          }
          // 后端报告下载完成（done=true），开始等待重启
          if (prog.done) {
            if (prog.error) {
              clearInterval(pollStatus)
              updating.value = false
              message.error(prog.error)
              return
            }
            // 下载完成，进入重启等待阶段
            clearInterval(pollStatus)
            waitForRestart()
          }
        }
      } catch {
        // status 连接失败，可能正在重启，转入 health 检测
        clearInterval(pollStatus)
        waitForRestart()
      }
      if (elapsed > 120) {
        clearInterval(pollStatus)
        // 超时也转入重启等待，服务可能已经重启了
        waitForRestart()
      }
    }, 1500)

    // 等待服务重启：先断开再恢复
    let restartElapsed = 0
    let wentDown = false
    function waitForRestart() {
      updateProgress.message = '服务正在重启...'
      updateProgress.percent = 98
      const t = setInterval(async () => {
        restartElapsed += 1
        try {
          const r = await fetch('/health', { signal: AbortSignal.timeout(2000) })
          if (!r.ok) {
            wentDown = true
          } else if (wentDown) {
            // 服务断开后恢复了
            clearInterval(t)
            updateProgress.percent = 100
            updateProgress.message = '更新完成，即将刷新...'
            message.success('面板更新完成')
            setTimeout(() => location.reload(), 2000)
          } else {
            // health 一直 OK，说明服务可能重启太快没检测到断开
            // 等 10 秒后直接刷新（版本应该已经更新）
            if (restartElapsed > 6) {
              clearInterval(t)
              updateProgress.percent = 100
              updateProgress.message = '更新完成，即将刷新...'
              setTimeout(() => location.reload(), 1500)
            }
          }
        } catch {
          wentDown = true
        }
        if (restartElapsed > 60) {
          clearInterval(t)
          updating.value = false
          message.error('更新超时，请手动刷新页面')
        }
      }, 1500)
    }
  } catch (e: any) {
    updating.value = false
    message.error(e.message || '更新失败')
  }
}
</script>
