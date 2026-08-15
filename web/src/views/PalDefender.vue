<template>
  <n-space vertical :size="16">
    <n-card title="PalDefender 反作弊" size="small">
      <n-alert type="info" :show-icon="false" style="margin-bottom:16px">
        PalDefender 是进程级实时反作弊插件（Windows DLL），通过 Wine 在 Linux 上运行。
        安装后会拦截玩家作弊操作（属性修改、非法物品、违禁科技等），实时性远高于外部检测。
      </n-alert>

      <n-descriptions :column="1" label-placement="left" bordered size="small">
        <n-descriptions-item label="Wine 状态">
          <n-tag v-if="status.wine_present" type="success" size="small">
            已安装 {{ status.wine_version }}
          </n-tag>
          <n-tag v-else type="error" size="small">未安装</n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="Wine 路径">
          {{ status.wine_path || '-' }}
        </n-descriptions-item>
        <n-descriptions-item label="游戏 Win64 目录">
          {{ status.win64_path || '未找到' }}
        </n-descriptions-item>
        <n-descriptions-item label="d3d9.dll">
          <n-tag :type="status.d3d9_exists ? 'success' : 'default'" size="small">
            {{ status.d3d9_exists ? '已存在' : '未安装' }}
          </n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="PalDefender.dll">
          <n-tag :type="status.pd_exists ? 'success' : 'default'" size="small">
            {{ status.pd_exists ? '已存在' : '未安装' }}
          </n-tag>
        </n-descriptions-item>
      </n-descriptions>
    </n-card>

    <!-- Wine 管理 -->
    <n-card title="Wine 运行环境" size="small">
      <n-space vertical>
        <n-space>
          <n-button @click="refreshStatus" :loading="loading">刷新状态</n-button>
          <n-button
            type="primary"
            @click="installWine"
            :loading="wineInstalling"
          >
            {{ status.wine_present ? '重新安装 / 更新 Wine' : '一键安装 Wine' }}
          </n-button>
          <n-popconfirm v-if="status.wine_present" @positive-click="uninstallWine">
            <template #trigger>
              <n-button type="error" ghost>卸载 Wine</n-button>
            </template>
            确定要卸载 Wine 吗？
          </n-popconfirm>
        </n-space>
        <n-text depth="3" style="font-size:12px">
          {{ status.wine_present
            ? 'Wine 已就绪，可以安装 PalDefender。'
            : 'PalDefender 需要 Wine 才能在 Linux 上运行。' }}
        </n-text>
      </n-space>
    </n-card>

    <!-- PalDefender 管理 -->
    <n-card title="PalDefender 插件" size="small">
      <n-space vertical>
        <n-space>
          <n-button
            type="primary"
            @click="install"
            :loading="installing"
            :disabled="!status.wine_present"
          >
            {{ status.d3d9_exists && status.pd_exists ? '更新 PalDefender' : '安装 PalDefender' }}
          </n-button>
          <n-popconfirm
            v-if="status.d3d9_exists || status.pd_exists"
            @positive-click="uninstall"
          >
            <template #trigger>
              <n-button type="error" ghost :disabled="!status.wine_present">卸载</n-button>
            </template>
            确定要卸载 PalDefender 吗？将删除 DLL 和配置目录。
          </n-popconfirm>
        </n-space>
        <n-alert v-if="!status.win64_path" type="info" :show-icon="false" style="font-size:12px">
          未找到 Win64 目录，安装时会自动创建
        </n-alert>
        <n-text depth="3" style="font-size:12px">
          安装后需通过 Wine 启动游戏服才能加载 DLL。
        </n-text>
      </n-space>
    </n-card>

    <!-- Wine 安装进度弹窗 -->
    <n-modal v-model:show="showWineProgress" preset="card" title="安装 Wine"
      style="max-width:600px" :mask-closable="false">
      <n-space vertical>
        <n-progress
          type="line"
          :percentage="winePercent"
          :status="wineDone ? (wineSuccess ? 'success' : 'error') : 'default'"
          :indicator-placement="'inside'"
        />
        <n-text>{{ wineMessage || '准备安装...' }}</n-text>
        <pre style="background:#1e1e1e;color:#d4d4d4;padding:12px;border-radius:6px;font-size:12px;max-height:300px;overflow:auto;white-space:pre-wrap">{{ wineLog || '等待输出...' }}</pre>
      </n-space>
    </n-modal>

    <!-- PalDefender 安装进度弹窗 -->
    <n-modal v-model:show="showPdProgress" preset="card" title="安装 PalDefender"
      style="max-width:520px" :mask-closable="false">
      <n-space vertical>
        <n-progress
          type="line"
          :percentage="pdPercent"
          :status="pdDone ? (pdSuccess ? 'success' : 'error') : 'default'"
          :indicator-placement="'inside'"
        />
        <n-text>{{ pdMessage || '准备安装...' }}</n-text>
        <n-text v-if="pdError" depth="3" style="font-size:12px;color:#d03050">{{ pdError }}</n-text>
      </n-space>
    </n-modal>
  </n-space>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  NSpace, NCard, NAlert, NDescriptions, NDescriptionsItem, NTag,
  NButton, NText, NModal, NProgress, NPopconfirm, useMessage,
} from 'naive-ui'

const message = useMessage()
const loading = ref(false)
const installing = ref(false)
const wineInstalling = ref(false)
const status = ref<any>({})

const showWineProgress = ref(false)
const wineLog = ref('')
const winePercent = ref(0)
const wineMessage = ref('')
const wineDone = ref(false)
const wineSuccess = ref(false)

const showPdProgress = ref(false)
const pdPercent = ref(0)
const pdMessage = ref('')
const pdDone = ref(false)
const pdSuccess = ref(false)
const pdError = ref('')

const token = () => localStorage.getItem('paladmin_token') || ''

async function refreshStatus() {
  loading.value = true
  try {
    const resp = await fetch('/api/paldefender/status', {
      headers: { Authorization: `Bearer ${token()}` },
    })
    status.value = await resp.json()
  } catch (e: any) {
    message.error(e.message)
  } finally {
    loading.value = false
  }
}

async function install() {
  installing.value = true
  showPdProgress.value = true
  pdPercent.value = 0
  pdMessage.value = '开始安装...'
  pdDone.value = false
  pdSuccess.value = false
  pdError.value = ''

  try {
    const resp = await fetch('/api/paldefender/install', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token()}` },
      body: JSON.stringify({ game_dir: '' }),
    })
    const data = await resp.json()
    if (!resp.ok || data.error) {
      pdError.value = data.error || '安装失败'
      pdDone.value = true
      installing.value = false
      return
    }

    const timer = setInterval(async () => {
      try {
        const r = await fetch('/api/paldefender/install-status', {
          headers: { Authorization: `Bearer ${token()}` },
        })
        const d = await r.json()
        if (typeof d.percent === 'number') pdPercent.value = d.percent
        if (d.message) pdMessage.value = d.message
        if (d.error) pdError.value = d.error
        if (d.done) {
          clearInterval(timer)
          pdDone.value = true
          pdSuccess.value = d.success
          installing.value = false
          if (d.success) {
            message.success('PalDefender 安装成功')
            await refreshStatus()
            setTimeout(() => { showPdProgress.value = false }, 3000)
          }
        }
      } catch { /* ignore */ }
    }, 1000)
  } catch (e: any) {
    pdError.value = e.message
    pdDone.value = true
    installing.value = false
  }
}

async function uninstall() {
  try {
    const resp = await fetch('/api/paldefender/uninstall', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token()}` },
      body: JSON.stringify({ game_dir: '' }),
    })
    const data = await resp.json()
    if (resp.ok && data.success) {
      message.success(data.message || '已卸载')
    } else {
      message.error(data.error || '卸载失败')
    }
    await refreshStatus()
  } catch (e: any) {
    message.error(e.message)
  }
}

async function installWine() {
  wineInstalling.value = true
  showWineProgress.value = true
  wineLog.value = ''
  winePercent.value = 0
  wineMessage.value = '准备安装...'
  wineDone.value = false
  wineSuccess.value = false

  try {
    await fetch('/api/paldefender/install-wine', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token()}` },
    })

    const timer = setInterval(async () => {
      try {
        const resp = await fetch('/api/paldefender/wine-status', {
          headers: { Authorization: `Bearer ${token()}` },
        })
        const data = await resp.json()
        wineLog.value = data.log || ''
        if (typeof data.percent === 'number') winePercent.value = data.percent
        if (data.message) wineMessage.value = data.message
        if (data.done) {
          clearInterval(timer)
          wineDone.value = true
          wineSuccess.value = data.success
          wineInstalling.value = false
          if (data.success) {
            message.success('Wine 安装成功')
            await refreshStatus()
            setTimeout(() => { showWineProgress.value = false }, 3000)
          } else {
            message.error(data.error || '安装失败')
          }
        }
      } catch { /* ignore */ }
    }, 2000)
  } catch (e: any) {
    wineInstalling.value = false
    message.error(e.message)
  }
}

async function uninstallWine() {
  try {
    const resp = await fetch('/api/paldefender/uninstall-wine', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token()}` },
    })
    const data = await resp.json()
    if (resp.ok && data.success) {
      message.success(data.message || 'Wine 已卸载')
    } else {
      message.error(data.error || '卸载失败')
    }
    await refreshStatus()
  } catch (e: any) {
    message.error(e.message)
  }
}

onMounted(refreshStatus)
</script>
