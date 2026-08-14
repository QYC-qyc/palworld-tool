<template>
  <n-space vertical :size="16">
    <n-alert type="info" :show-icon="false">
      游戏服由你自行用 SteamCMD 安装。只需填写<strong>所在文件夹</strong>，面板会自动在其中查找
      <code>steamcmd.sh</code>（Linux）/ <code>steamcmd.exe</code>（Windows）与游戏可执行文件。
      点击「安装/更新」会执行：<code>steamcmd +force_install_dir &lt;游戏目录&gt; +login anonymous +app_update 2394010 validate +quit</code>
    </n-alert>

    <!-- 状态 -->
    <n-card title="运行状态" size="small">
      <n-descriptions v-if="status" bordered :column="2" label-placement="left" size="small">
        <n-descriptions-item label="状态">
          <n-tag :type="stateType" size="small">{{ stateText }}</n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="进程 PID">
          {{ status.status?.pid || '-' }}
        </n-descriptions-item>
        <n-descriptions-item label="SteamCMD">
          <n-tag :type="status.status?.steam_ready ? 'success' : 'error'" size="small">
            {{ status.status?.steam_ready ? '已就绪' : '未找到' }}
          </n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="服务端">
          <n-tag :type="status.status?.installed ? 'success' : 'warning'" size="small">
            {{ status.status?.installed ? '已安装' : '未安装' }}
          </n-tag>
        </n-descriptions-item>
      </n-descriptions>
    </n-card>

    <!-- 路径配置 -->
    <n-card title="路径配置" size="small">
      <n-form label-placement="top">
        <n-grid cols="1 s:2" :x-gap="16" responsive="screen">
          <n-gi>
            <n-form-item label="SteamCMD 目录">
              <n-input v-model:value="cfg.steamcmd_path"
                placeholder="文件夹，如 /root/steamcmd">
                <template #suffix>
                  <n-button size="tiny" quaternary :loading="verifying === 'steam'" @click="verifyPath('steam')">
                    验证
                  </n-button>
                </template>
              </n-input>
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="游戏安装目录">
              <n-input v-model:value="cfg.install_dir"
                placeholder="文件夹，如 /root/PalServer">
                <template #suffix>
                  <n-button size="tiny" quaternary :loading="verifying === 'server'" @click="verifyPath('server')">
                    验证
                  </n-button>
                </template>
              </n-input>
            </n-form-item>
          </n-gi>
          <n-gi style="grid-column: 1 / -1">
            <n-form-item label="启动额外参数（可选）">
              <n-input v-model:value="cfg.extra_args" type="textarea" :autosize="{ minRows: 2 }"
                placeholder="如 -publiclobby -useperfthreads -NoAsyncLoadingThread -UseMultithreadForDS" />
              <n-text depth="3" style="font-size:12px;line-height:1.6">
                端口、REST/RCON 等网络参数请到「游戏配置」页修改，无需在此填写 -port。常用参数：<br/>
                <code>-publiclobby</code> 设为社区服务器（公开列表）；
                <code>-useperfthreads -NoAsyncLoadingThread -UseMultithreadForDS</code> 多线程性能优化；
                <code>-NumberOfWorkerThreadsServer=X</code> 指定工作线程数；
                <code>-logformat=text</code> 日志格式（text/json）
              </n-text>
            </n-form-item>
          </n-gi>
        </n-grid>
        <n-button type="primary" @click="saveConfig">保存配置</n-button>
        <n-button @click="verifyPath('all')" :loading="verifying === 'all'" style="margin-left:8px">
          全部验证
        </n-button>
        <n-text depth="3" style="font-size:12px;margin-left:12px">
          已识别：SteamCMD {{ verifyResult.steamExe || status?.status?.steam_exe || '-' }}
          ｜ 服务端 {{ verifyResult.serverExe || status?.status?.server_exe || '-' }}
        </n-text>
      </n-form>
    </n-card>

    <!-- 操作 -->
    <n-card title="操作" size="small">
      <n-space>
        <n-button type="info" :loading="acting === 'install'" @click="doInstall">
          安装 / 更新游戏服
        </n-button>
        <n-button type="success" :disabled="!canStart" :loading="acting==='start'" @click="doStart">
          启动
        </n-button>
        <n-button type="warning" :disabled="!isRunning" :loading="acting==='stop'" @click="doStop">
          停止
        </n-button>
        <n-button :disabled="!isRunning" :loading="acting==='restart'" @click="doRestart">
          重启
        </n-button>
        <n-button @click="() => { loadLogs(); loadStatus(); }">刷新</n-button>
      </n-space>
      <n-text depth="3" style="font-size:12px;display:block;margin-top:8px">
        提示：首次安装可能需要较长时间，请在日志中查看 SteamCMD 下载进度。安装完成后再启动。
      </n-text>
    </n-card>

    <!-- 日志 -->
    <n-card title="日志" size="small">
      <n-button size="tiny" @click="loadLogs" style="margin-bottom:8px">刷新日志</n-button>
      <pre class="logs">{{ logs || '暂无日志' }}</pre>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  NSpace, NCard, NAlert, NDescriptions, NDescriptionsItem, NTag,
  NForm, NFormItem, NInput, NButton, NGrid, NGi, NText, useMessage,
} from 'naive-ui'
import { gameApi, type GameServerStatus, type GameServerConfig } from '@/api/gameserver'

const message = useMessage()
const status = ref<GameServerStatus | null>(null)
const logs = ref('')
const acting = ref('')
const verifying = ref('')
const verifyResult = reactive<{ steamExe: string; serverExe: string }>({ steamExe: '', serverExe: '' })
const cfg = reactive<GameServerConfig>({
  steamcmd_path: '',
  install_dir: '',
  extra_args: '',
})

const isRunning = computed(() => !!status.value?.status?.running)
const isInstalled = computed(() => !!status.value?.status?.installed)
const canStart = computed(() => isInstalled.value && !isRunning.value)
const isUpdating = computed(() => !!status.value?.status?.updating)

const stateText = computed(() => {
  if (isUpdating.value) return '更新中'
  if (isRunning.value) return '运行中'
  return '已停止'
})
const stateType = computed(() => {
  if (isUpdating.value) return 'warning'
  if (isRunning.value) return 'success'
  return 'default'
})

async function loadStatus() {
  status.value = await gameApi.status()
}
async function loadConfig() {
  try {
    const c = await gameApi.getConfig()
    Object.assign(cfg, c)
  } catch { /* 首次无配置 */ }
}
async function saveConfig() {
  await gameApi.saveConfig(cfg)
  message.success('配置已保存')
  verifyResult.steamExe = ''
  verifyResult.serverExe = ''
  await loadStatus()
}

async function verifyPath(target: 'steam' | 'server' | 'all') {
  if (!cfg.steamcmd_path && !cfg.install_dir) {
    message.warning('请先填写路径')
    return
  }
  verifying.value = target
  try {
    const res = await gameApi.verify(cfg)
    verifyResult.steamExe = res.steam_exe || ''
    verifyResult.serverExe = res.server_exe || ''
    const checkSteam = target === 'steam' || target === 'all'
    const checkServer = target === 'server' || target === 'all'
    if (checkSteam) {
      if (res.steam_ok) message.success(`SteamCMD 已找到：${res.steam_exe}`)
      else message.error(`未在目录中找到 steamcmd.sh/steamcmd.exe`)
    }
    if (checkServer) {
      if (res.server_ok) message.success(`游戏服务端已找到：${res.server_exe}`)
      else message.warning('未找到游戏服务端（未安装是正常的，安装后即可识别）')
    }
  } catch (e: any) {
    message.error(e.message)
  } finally {
    verifying.value = ''
  }
}
async function loadLogs() {
  try {
    const r = await gameApi.logs()
    logs.value = r.logs || ''
  } catch { /* ignore */ }
}

async function doInstall() {
  if (!cfg.steamcmd_path || !cfg.install_dir) {
    message.warning('请先填写 SteamCMD 路径和安装目录')
    return
  }
  await saveConfig()
  acting.value = 'install'
  try {
    const r = await gameApi.install()
    message.info(r.message || '已开始安装，请查看日志')
    pollStatus()
  } catch (e: any) { message.error(e.message) }
  finally { acting.value = '' }
}
async function doStart() {
  acting.value = 'start'
  try { await gameApi.start(); message.success('已启动'); await loadStatus() }
  catch (e: any) { message.error(e.message) }
  finally { acting.value = '' }
}
async function doStop() {
  acting.value = 'stop'
  try { await gameApi.stop(); message.success('已停止'); await loadStatus() }
  catch (e: any) { message.error(e.message) }
  finally { acting.value = '' }
}
async function doRestart() {
  acting.value = 'restart'
  try { await gameApi.restart(); message.success('已重启'); await loadStatus() }
  catch (e: any) { message.error(e.message) }
  finally { acting.value = '' }
}

// 更新期间轮询状态与日志
function pollStatus() {
  const t = setInterval(async () => {
    await loadStatus()
    await loadLogs()
    if (!isUpdating.value) {
      clearInterval(t)
      await loadStatus()
    }
  }, 3000)
}

onMounted(async () => {
  await loadConfig()
  await loadStatus()
  await loadLogs()
})
</script>

<style scoped>
.logs {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 6px;
  font-size: 12px;
  max-height: 420px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
code {
  background: #f0f0f0;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 12px;
}
</style>
