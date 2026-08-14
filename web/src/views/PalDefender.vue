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

    <n-card title="安装与配置" size="small">
      <n-space vertical>
        <n-alert v-if="!status.wine_present" type="warning" :show-icon="false">
          <template #header>需要安装 Wine</template>
          <div style="margin-bottom:8px">PalDefender 需要 Wine 才能在 Linux 上运行。</div>
          <n-space>
            <n-button type="primary" size="small"
              @click="installWine" :loading="wineInstalling">
              一键安装 Wine
            </n-button>
            <n-button size="small" @click="copyWineCmd" quaternary>复制手动命令</n-button>
          </n-space>
        </n-alert>

        <n-space>
          <n-button @click="refreshStatus" :loading="loading">刷新状态</n-button>
          <n-button
            type="primary"
            @click="install"
            :loading="installing"
            :disabled="!status.wine_present"
          >
            {{ status.d3d9_exists && status.pd_exists ? '重新安装 / 更新' : '下载并安装 PalDefender' }}
          </n-button>
        </n-space>
        <n-alert v-if="!status.win64_path" type="info" :show-icon="false" style="font-size:12px">
          未找到 Win64 目录，安装时会自动在游戏目录下创建 <code>Pal/Binaries/Win64/</code>
        </n-alert>

        <n-alert v-if="!status.win64_path" type="warning" :show-icon="false">
          未找到 <code>Pal/Binaries/Win64</code> 目录。请先在「游戏服」页面正确填写游戏安装目录并完成安装。
        </n-alert>

        <n-text depth="3" style="font-size:12px">
          安装后请通过 Wine 方式启动游戏服（而非原生 Linux 版 PalServer.sh），DLL 才会被加载。
          首次启动会自动生成 PalDefender 配置目录。
        </n-text>
      </n-space>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  NSpace, NCard, NAlert, NDescriptions, NDescriptionsItem, NTag,
  NButton, NText, NModal, useMessage,
} from 'naive-ui'

const message = useMessage()
const loading = ref(false)
const installing = ref(false)
const wineInstalling = ref(false)
const status = ref<any>({})

async function refreshStatus() {
  loading.value = true
  try {
    const token = localStorage.getItem('paladmin_token') || ''
    const resp = await fetch('/api/paldefender/status', {
      headers: { Authorization: `Bearer ${token}` },
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
  try {
    const token = localStorage.getItem('paladmin_token') || ''
    const resp = await fetch('/api/paldefender/install', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ game_dir: '' }),
    })
    const data = await resp.json()
    if (resp.ok && data.success) {
      message.success(data.message || '安装成功')
      await refreshStatus()
    } else {
      message.error(data.error || '安装失败')
    }
  } catch (e: any) {
    message.error(e.message)
  } finally {
    installing.value = false
  }
}

async function installWine() {
  wineInstalling.value = true
  showWineProgress.value = true
  wineLog.value = ''
  let pollTimer: number | null = null
  try {
    const token = localStorage.getItem('paladmin_token') || ''
    // 触发后台安装
    await fetch('/api/paldefender/install-wine', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
    })
    // 轮询进度
    pollTimer = window.setInterval(async () => {
      try {
        const resp = await fetch('/api/paldefender/wine-status', {
          headers: { Authorization: `Bearer ${token}` },
        })
        const data = await resp.json()
        wineLog.value = data.log || ''
        if (data.done) {
          if (pollTimer) clearInterval(pollTimer)
          wineInstalling.value = false
          if (data.success) {
            message.success('Wine 安装成功')
            await refreshStatus()
            setTimeout(() => { showWineProgress.value = false }, 3000)
          } else {
            message.error(data.error || '安装失败')
          }
        }
      } catch { /* ignore poll errors */ }
    }, 2000)
  } catch (e: any) {
    wineInstalling.value = false
    message.error(e.message || '安装失败')
  }
}

function copyWineCmd() {
  const cmd = 'dpkg --add-architecture i386 && apt update && apt install -y wine64'
  navigator.clipboard?.writeText(cmd).then(() => message.success('命令已复制'))
}

onMounted(refreshStatus)
</script>
