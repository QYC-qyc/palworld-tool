<template>
  <n-space vertical :size="16">
    <PageHeader title="游戏服管理" :subtitle="status?.status?.install_dir ? `安装目录：${status.status.install_dir}` : '安装、启动、停止与更新游戏服'">
      <n-button @click="refreshAll" :loading="loading">
        <template #icon><n-icon :component="RefreshOutline" /></template>
        刷新
      </n-button>
    </PageHeader>

    <n-alert type="info" :show-icon="false">
      游戏服由你自行用 SteamCMD 安装。只需填写<strong>所在文件夹</strong>，面板会自动查找
      <code>steamcmd.sh</code>（Linux）/ <code>steamcmd.exe</code>（Windows）与游戏可执行文件。
    </n-alert>

    <!-- 运行状态 -->
    <n-card title="运行状态" size="small">
      <n-grid cols="1 s:2 m:4" responsive="screen" :x-gap="16" :y-gap="16" item-responsive>
        <n-gi>
          <div class="status-cell">
            <div class="status-cell__label">状态</div>
            <StatusTag :status="stateStatus" />
          </div>
        </n-gi>
        <n-gi>
          <div class="status-cell">
            <div class="status-cell__label">进程 PID</div>
            <div class="status-cell__value">{{ status?.status?.pid || '-' }}</div>
          </div>
        </n-gi>
        <n-gi>
          <div class="status-cell">
            <div class="status-cell__label">SteamCMD</div>
            <n-tag :type="status?.status?.steam_ready ? 'success' : 'error'" size="small" round :bordered="false">
              {{ status?.status?.steam_ready ? '已就绪' : '未找到' }}
            </n-tag>
          </div>
        </n-gi>
        <n-gi>
          <div class="status-cell">
            <div class="status-cell__label">当前模式</div>
            <PlatformBadge platform="current" :is-windows="!!status?.status?.wine_mode" />
          </div>
        </n-gi>
        <n-gi>
          <div class="status-cell">
            <div class="status-cell__label">Linux 原生版</div>
            <PlatformBadge platform="linux" :installed="!!status?.status?.linux_installed" />
          </div>
        </n-gi>
        <n-gi>
          <div class="status-cell">
            <div class="status-cell__label">Windows 版（Wine）</div>
            <PlatformBadge platform="windows" :installed="!!status?.status?.windows_installed" />
          </div>
        </n-gi>
      </n-grid>
    </n-card>

    <!-- 操作 -->
    <n-card title="操作" size="small">
      <n-space :size="10" wrap>
        <n-button type="success" :disabled="!canStart" :loading="acting === 'start'" @click="doStart">
          <template #icon><n-icon :component="PlayOutline" /></template>
          启动
        </n-button>
        <n-button type="error" :disabled="!isRunning" :loading="acting === 'stop'" @click="doStop">
          <template #icon><n-icon :component="StopOutline" /></template>
          停止
        </n-button>
        <n-button type="warning" :disabled="!isRunning" :loading="acting === 'restart'" @click="doRestart">
          <template #icon><n-icon :component="RefreshOutline" /></template>
          重启
        </n-button>
        <n-button type="info" :loading="acting === 'install'" @click="openInstall">
          <template #icon><n-icon :component="CloudDownloadOutline" /></template>
          安装 / 更新游戏服
        </n-button>
      </n-space>
      <n-text depth="3" style="font-size:12px;display:block;margin-top:10px">
        提示：首次安装可能需要较长时间，请在日志中查看 SteamCMD 下载进度。安装完成后再启动。
      </n-text>
    </n-card>

    <!-- 路径配置 -->
    <n-card title="路径配置" size="small">
      <n-form label-placement="top">
        <n-grid cols="1 s:2" :x-gap="16" responsive="screen">
          <n-gi>
            <n-form-item label="SteamCMD 目录">
              <n-input v-model:value="cfg.steamcmd_path" placeholder="文件夹，如 /root/steamcmd">
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
              <n-input v-model:value="cfg.install_dir" placeholder="文件夹，如 /root/PalServer">
                <template #suffix>
                  <n-button size="tiny" quaternary :loading="verifying === 'server'" @click="verifyPath('server')">
                    验证
                  </n-button>
                </template>
              </n-input>
            </n-form-item>
          </n-gi>
          <n-gi style="grid-column: 1 / -1">
            <n-form-item>
              <template #label>
                <span style="display:inline-flex;align-items:center;gap:4px">
                  启动额外参数（可选）
                  <n-tooltip trigger="hover" placement="top" :show-arrow="false">
                    <template #trigger>
                      <n-icon :component="HelpCircleOutline" class="help-icon" />
                    </template>
                    <div class="tooltip-content">
                      端口、REST API 等网络参数请到「游戏配置」页修改，无需填写 -port。<br/>
                      常用：<code>-publiclobby</code>；<code>-useperfthreads -NoAsyncLoadingThread -UseMultithreadForDS</code>
                    </div>
                  </n-tooltip>
                </span>
              </template>
              <n-input v-model:value="cfg.extra_args" type="textarea" :autosize="{ minRows: 2 }"
                placeholder="如 -publiclobby -useperfthreads -NoAsyncLoadingThread -UseMultithreadForDS" />
            </n-form-item>
          </n-gi>
        </n-grid>
        <n-space align="center" :size="12" wrap>
          <n-button type="primary" @click="saveConfig">保存配置</n-button>
          <n-button @click="verifyPath('all')" :loading="verifying === 'all'">验证路径</n-button>
          <n-text v-if="verifyResult.steamExe || verifyResult.serverExe" depth="3" style="font-size:12px">
            识别到：SteamCMD {{ verifyResult.steamExe || '-' }} ｜ 服务端 {{ verifyResult.serverExe || '-' }}
          </n-text>
          <n-text v-else-if="verifyResult.checked" depth="3" style="font-size:12px">
            未识别到 SteamCMD 或服务端，请检查路径
          </n-text>
        </n-space>
      </n-form>
    </n-card>

    <!-- 日志 -->
    <n-card size="small">
      <template #header>
        <n-space align="center" :size="8">
          <n-icon :component="DocumentTextOutline" />
          <span>日志</span>
        </n-space>
      </template>
      <template #header-extra>
        <n-space align="center" :size="12" :wrap="false">
          <n-checkbox v-model:checked="autoScroll" size="small">自动滚动到底部</n-checkbox>
          <n-button size="tiny" quaternary @click="clearLogs">清空显示</n-button>
          <n-button size="tiny" @click="loadLogs">刷新</n-button>
        </n-space>
      </template>
      <pre ref="logEl" class="logs">{{ logs || '暂无日志' }}</pre>
    </n-card>

    <!-- 选择安装版本 -->
    <n-modal v-model:show="showVersionModal" preset="card" title="选择游戏服版本"
      style="max-width:520px" :mask-closable="true">
      <n-space vertical :size="14">
        <n-text depth="3">请选择要安装/更新的游戏服版本。两版本文件相互独立，可同时安装、按需切换。</n-text>
        <n-radio-group v-model:value="installPlatform">
          <n-space vertical>
            <n-radio value="linux">
              原生 Linux 版 <n-text depth="3" style="font-size:12px">（默认，性能更好，装在安装目录根）</n-text>
            </n-radio>
            <n-radio value="windows">
              Windows 版（Wine / PalDefender）
              <n-text depth="3" style="font-size:12px">（用于 PalDefender 反作弊，需先安装 Wine，装在安装目录下的 PalServer-Win 子目录）</n-text>
            </n-radio>
          </n-space>
        </n-radio-group>
        <n-alert v-if="installPlatform === 'windows'" type="info" :show-icon="true" style="margin-top:4px">
          Windows 版将安装到 <b>{{ cfg.install_dir ? cfg.install_dir + '/PalServer-Win' : '安装目录/PalServer-Win' }}</b>，
          与 Linux 版隔离，互不影响。
        </n-alert>
        <n-space justify="end">
          <n-button @click="showVersionModal = false">取消</n-button>
          <n-button type="primary" :loading="acting === 'install'" @click="doInstall">
            开始安装
          </n-button>
        </n-space>
      </n-space>
    </n-modal>

    <!-- 安装/更新进度弹窗 -->
    <n-modal v-model:show="showInstallModal" preset="card" title="安装 / 更新游戏服"
      style="max-width:680px" :mask-closable="false">
      <n-space vertical :size="12">
        <n-text>正在通过 SteamCMD 下载/更新游戏服（{{ installPlatform === 'windows' ? 'Windows 版' : 'Linux 原生版' }}），首次安装可能需要几分钟，请耐心等待。</n-text>
        <pre class="logs logs-modal">{{ installLogs || '等待输出...' }}</pre>
        <n-space>
          <n-button size="small" @click="loadLogs">刷新日志</n-button>
          <n-button size="small" type="primary" @click="doStart"
            :disabled="!(status?.status?.installed)" :loading="acting === 'start'">
            安装完成，启动游戏服
          </n-button>
        </n-space>
      </n-space>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import {
  NSpace, NCard, NAlert, NTag, NForm, NFormItem, NInput, NButton, NGrid, NGi,
  NText, NModal, NTooltip, NIcon, NRadioGroup, NRadio, NCheckbox, useMessage,
} from 'naive-ui'
import {
  HelpCircleOutline, CloudDownloadOutline, RefreshOutline, PlayOutline, StopOutline,
  DocumentTextOutline,
} from '@vicons/ionicons5'
import { gameApi, type GameServerStatus, type GameServerConfig } from '@/api/gameserver'
import PageHeader from '@/components/PageHeader.vue'
import StatusTag from '@/components/StatusTag.vue'
import PlatformBadge from '@/components/PlatformBadge.vue'

const message = useMessage()
const status = ref<GameServerStatus | null>(null)
const logs = ref('')
const acting = ref('')
const loading = ref(false)
const showInstallModal = ref(false)
const installLogs = ref('')
let installTimer: number | null = null
const verifying = ref('')
const verifyResult = reactive<{
  steamExe: string
  serverExe: string
  checked: boolean
}>({ steamExe: '', serverExe: '', checked: false })
const cfg = reactive<GameServerConfig>({
  steamcmd_path: '',
  install_dir: '',
  extra_args: '',
})
const autoScroll = ref(true)
const logEl = ref<HTMLElement | null>(null)

const isRunning = computed(() => !!status.value?.status?.running)
const isInstalled = computed(() => !!status.value?.status?.installed)
const canStart = computed(() => isInstalled.value && !isRunning.value)
const isUpdating = computed(() => !!status.value?.status?.updating)

const stateStatus = computed<'running' | 'stopped' | 'updating' | 'error'>(() => {
  if (isUpdating.value) return 'updating'
  if (isRunning.value) return 'running'
  return 'stopped'
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
  verifyResult.checked = false
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
    verifyResult.checked = true
    const checkSteam = target === 'steam' || target === 'all'
    const checkServer = target === 'server' || target === 'all'
    if (checkSteam) {
      if (res.steam_ok) message.success(`SteamCMD 已找到：${res.steam_exe}`)
      else message.error('未在目录中找到 steamcmd.sh/steamcmd.exe')
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
    if (autoScroll.value) {
      await nextTick()
      scrollLog()
    }
  } catch { /* ignore */ }
}
function scrollLog() {
  if (logEl.value) logEl.value.scrollTop = logEl.value.scrollHeight
}
function clearLogs() {
  logs.value = ''
}

watch(logs, () => {
  if (autoScroll.value) {
    nextTick(scrollLog)
  }
})

const showVersionModal = ref(false)
const installPlatform = ref<'linux' | 'windows'>('linux')

async function refreshAll() {
  loading.value = true
  try {
    await Promise.all([loadStatus(), loadLogs()])
  } finally {
    loading.value = false
  }
}

function openInstall() {
  if (!cfg.steamcmd_path || !cfg.install_dir) {
    message.warning('请先填写 SteamCMD 路径和安装目录')
    return
  }
  installPlatform.value = status.value?.status?.wine_mode ? 'windows' : 'linux'
  showVersionModal.value = true
}

async function doInstall() {
  showVersionModal.value = false
  if (!cfg.steamcmd_path || !cfg.install_dir) {
    message.warning('请先填写 SteamCMD 路径和安装目录')
    return
  }
  await saveConfig()
  acting.value = 'install'
  showInstallModal.value = true
  installLogs.value = ''
  try {
    const r = await gameApi.install(installPlatform.value)
    message.info(r.message || '已开始安装')
    if (installTimer) clearInterval(installTimer)
    const poll = async () => {
      await loadLogs()
      installLogs.value = logs.value
      await loadStatus()
      if (!status.value?.status?.updating) {
        clearInterval(installTimer!)
        installTimer = null
        await loadStatus()
        if (status.value?.status?.installed) {
          message.success('游戏服安装/更新完成')
        } else {
          message.warning('安装过程已结束，查看日志确认是否成功')
        }
      }
    }
    poll()
    installTimer = window.setInterval(poll, 2000)
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

onMounted(async () => {
  await loadConfig()
  await loadStatus()
  await loadLogs()
})

onUnmounted(() => {
  if (installTimer) clearInterval(installTimer)
})
</script>

<style scoped>
.status-cell {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.status-cell__label {
  font-size: 12px;
  color: var(--n-text-color-3, #999);
}
.status-cell__value {
  font-size: 15px;
  font-weight: 500;
}
.logs {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 8px;
  font-size: 12px;
  max-height: 460px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
  margin: 0;
}
.logs-modal {
  max-height: 360px;
}
code {
  background: rgba(127, 127, 127, 0.15);
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 12px;
}
.help-icon {
  font-size: 15px;
  color: var(--n-text-color-3, #999);
  cursor: help;
}
.tooltip-content {
  max-width: 360px;
  line-height: 1.6;
}
.tooltip-content code {
  background: rgba(255,255,255,0.15);
  padding: 1px 5px;
  border-radius: 3px;
  font-size: 12px;
}
.verify-result {
  margin-top: 8px;
}
:deep(.n-tooltip) {
  background: #1f1f1f !important;
  color: #fff !important;
}
</style>
