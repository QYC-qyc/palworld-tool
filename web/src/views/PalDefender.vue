<template>
  <n-tabs v-model:value="activeTab" type="line" animated @update:value="onTab">
    <!-- Tab 1：安装与状态 -->
    <n-tab-pane name="install" tab="安装与状态">
      <n-space vertical :size="16">
        <n-card title="REST API 连接状态" size="small">
          <n-space align="center" :wrap="false">
            <n-tag :type="connTagType" size="small" :loading="connLoading">
              {{ connText }}
            </n-tag>
            <n-text v-if="connVersion" depth="3" style="font-size:12px">
              版本：{{ connVersion }}
            </n-text>
            <n-button size="small" :loading="connLoading" @click="testConnection">
              测试连接
            </n-button>
          </n-space>
          <n-text depth="3" style="font-size:12px;display:block;margin-top:8px">
            连接地址由「设置 - PalDefender 反作弊」中的 host/port/token 决定。DLL 加载且 RESTConfig 启用后才会在线。
          </n-text>
        </n-card>

        <n-card title="PalDefender 反作弊" size="small">
          <n-alert type="info" :show-icon="false" style="margin-bottom:16px">
            PalDefender 是进程级实时反作弊插件（Windows DLL），安装到游戏目录后拦截作弊行为。
            安装后会拦截玩家作弊操作（属性修改、非法物品、违禁科技等），实时性远高于外部检测。
          </n-alert>

          <n-descriptions :column="1" label-placement="left" bordered size="small">
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

        <n-card title="PalDefender 插件" size="small">
          <n-space vertical>
            <n-alert type="info" :show-icon="false" style="font-size:12px">
              安装或更新 PalDefender 反作弊 DLL 到游戏目录。安装后启动游戏服即会加载。
            </n-alert>
            <n-space>
              <n-button @click="refreshStatus" :loading="loading">刷新状态</n-button>
              <n-button type="primary" @click="install" :loading="installing">
                {{ status.d3d9_exists && status.pd_exists ? '更新 PalDefender' : '安装 PalDefender' }}
              </n-button>
              <n-popconfirm v-if="status.d3d9_exists || status.pd_exists" @positive-click="uninstall">
                <template #trigger>
                  <n-button type="error" ghost>卸载</n-button>
                </template>
                确定要卸载 PalDefender 吗？将删除 DLL 和配置目录。
              </n-popconfirm>
            </n-space>
            <n-alert v-if="!status.win64_path" type="info" :show-icon="false" style="font-size:12px">
              未找到 Win64 目录，安装时会自动创建
            </n-alert>
          </n-space>
        </n-card>

        <n-card title="反作弊连接配置" size="small">
          <n-alert type="info" :show-icon="false" style="margin-bottom:16px;font-size:12px">
            配置面板连接 PalDefender REST API 的地址。Docker 部署主机填 <code>gameserver</code>，
            端口默认 17993；生成 Token 后需保存才生效。
          </n-alert>
          <n-form label-placement="left" label-width="120">
            <n-form-item label="API 主机">
              <n-input v-model:value="pdForm.host" placeholder="gameserver（Docker）或 127.0.0.1">
                <template #suffix>
                  <n-icon :component="HelpCircleOutline"
                    title="PalDefender REST API 所在主机。Docker 部署填 gameserver（容器名）；游戏服与面板在同一机器的非 Docker 部署填 127.0.0.1。"
                    class="help-icon" />
                </template>
              </n-input>
            </n-form-item>
            <n-form-item label="API 端口">
              <n-input-number v-model:value="pdForm.port" :min="1" :max="65535" :step="1"
                style="width:100%" placeholder="17993" />
            </n-form-item>
            <n-form-item label="Token">
              <n-input-group>
                <n-input
                  v-model:value="pdForm.token"
                  type="password"
                  show-password-on="click"
                  :placeholder="pdTokenSet ? '已设置（留空不修改）' : '未设置'"
                >
                  <template #suffix>
                    <n-icon :component="HelpCircleOutline"
                      title="点击「生成 Token」自动写入 PalDefender 令牌，也可手动粘贴。请勿将 API 端口暴露到公网。"
                      class="help-icon" />
                  </template>
                </n-input>
                <n-button size="small" :loading="revealingToken"
                  style="flex-shrink:0" @click="revealPdToken">
                  <template #icon><n-icon :component="EyeOutline" /></template>
                </n-button>
                <n-button type="primary" size="small" :loading="generatingToken"
                  style="flex-shrink:0" @click="generateToken">
                  生成 Token
                </n-button>
              </n-input-group>
            </n-form-item>
          </n-form>
        </n-card>

        <n-card title="反作弊处置" size="small">
          <n-alert type="info" :show-icon="false" style="margin-bottom:16px;font-size:12px">
            设置检测到作弊时的处置方式。保存后写入 Config.json 并热重载，需先安装并启用 PalDefender。
          </n-alert>
          <n-form label-placement="left" label-width="160">
            <n-form-item label="启用反作弊">
              <n-switch :value="pdForm.anticheat_enabled !== 'false'"
                @update:value="(v) => (pdForm.anticheat_enabled = v ? 'true' : 'false')" />
            </n-form-item>
            <n-form-item label="检测到即踢出">
              <n-switch :value="pdForm.cheaters_kick === 'true'"
                @update:value="(v) => (pdForm.cheaters_kick = v ? 'true' : 'false')">
                <template #checked>踢出</template>
                <template #unchecked>不踢出</template>
              </n-switch>
            </n-form-item>
            <n-form-item label="检测到即封禁">
              <n-switch :value="pdForm.cheaters_ban === 'true'"
                @update:value="(v) => (pdForm.cheaters_ban = v ? 'true' : 'false')">
                <template #checked>封禁</template>
                <template #unchecked>不封禁</template>
              </n-switch>
            </n-form-item>
            <n-form-item label="同时封禁 IP">
              <n-switch :value="pdForm.cheaters_ipban === 'true'"
                @update:value="(v) => (pdForm.cheaters_ipban = v ? 'true' : 'false')">
                <template #checked>IP 封禁</template>
                <template #unchecked>不封 IP</template>
              </n-switch>
            </n-form-item>
            <n-button type="primary" :loading="pdSaving" @click="savePdSettings">保存配置</n-button>
          </n-form>
        </n-card>
      </n-space>
    </n-tab-pane>

    <!-- Tab 2：玩家管理 -->
    <n-tab-pane name="players" tab="玩家管理">
      <n-space vertical :size="12">
        <n-space align="center" :wrap="false">
          <n-input v-model:value="playerSearch" placeholder="按名称 / UserId 搜索" clearable style="max-width:280px" />
          <n-button @click="loadPlayers" :loading="playersLoading">刷新</n-button>
          <n-text depth="3" style="font-size:12px">
            共 {{ players.length }} 人，在线 {{ onlineCount }}
          </n-text>
        </n-space>
        <n-spin :show="playersLoading">
          <n-data-table
            :columns="playerCols"
            :data="filteredPlayers"
            :bordered="false"
            size="small"
            :pagination="{ pageSize: 20 }"
          />
        </n-spin>
      </n-space>
    </n-tab-pane>

    <!-- Tab 3：封禁列表 -->
    <n-tab-pane name="banlist" tab="封禁列表">
      <n-space vertical :size="12">
        <n-space align="center" :wrap="false">
          <n-tag size="small" :bordered="false">数据来源：PalDefender Banlist.json</n-tag>
          <n-button @click="loadBanlist" :loading="banLoading">刷新</n-button>
          <n-button type="error" ghost @click="openBanIP">手动封 IP</n-button>
        </n-space>
        <n-tabs v-model:value="banSubtab" type="segment" size="small" animated>
          <n-tab-pane name="user" :tab="`用户封禁 (${banlist?.UserEntries?.length || 0})`">
            <n-data-table
              :columns="userBanCols"
              :data="banlist?.UserEntries || []"
              :bordered="false"
              size="small"
              :pagination="{ pageSize: 20 }"
            />
          </n-tab-pane>
          <n-tab-pane name="ip" :tab="`IP 封禁 (${banlist?.IPEntries?.length || 0})`">
            <n-data-table
              :columns="ipBanCols"
              :data="banlist?.IPEntries || []"
              :bordered="false"
              size="small"
              :pagination="{ pageSize: 20 }"
            />
          </n-tab-pane>
        </n-tabs>
      </n-space>
    </n-tab-pane>

    <!-- Tab 4：广播与警报 -->
    <n-tab-pane name="message" tab="广播与警报">
      <n-space vertical :size="16">
        <n-card title="全服广播" size="small">
          <n-space vertical>
            <n-input v-model:value="broadcastMsg" type="textarea" :autosize="{ minRows: 2 }"
              placeholder="输入全服广播消息" />
            <n-button type="primary" @click="sendBroadcast">发送广播</n-button>
          </n-space>
        </n-card>

        <n-card title="高优先级警报" size="small">
          <n-space vertical>
            <n-alert type="warning" :show-icon="false" style="font-size:12px">
              警报为高优先级消息，会在游戏内以醒目方式提示玩家，请谨慎使用。
            </n-alert>
            <n-input v-model:value="alertMsg" type="textarea" :autosize="{ minRows: 2 }"
              placeholder="输入警报消息" />
            <n-button type="error" @click="sendAlert">发送警报</n-button>
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
              <n-button v-if="!playersLoaded" size="small" @click="loadPlayers">
                加载玩家列表
              </n-button>
            </n-space>
            <n-input v-model:value="msgContent" type="textarea" :autosize="{ minRows: 2 }"
              placeholder="输入私聊消息" />
            <n-button type="primary" @click="sendPrivateMsg">发送私聊</n-button>
          </n-space>
        </n-card>
      </n-space>
    </n-tab-pane>

    <!-- Tab 5：公会与据点 -->
    <n-tab-pane name="guild" tab="公会与据点">
      <n-space vertical :size="12">
        <n-space align="center" :wrap="false">
          <n-button @click="loadGuilds" :loading="guildsLoading">刷新</n-button>
          <n-text depth="3" style="font-size:12px">点击行查看公会成员与营地列表</n-text>
        </n-space>
        <n-data-table
          :columns="guildCols"
          :data="guilds"
          :bordered="false"
          size="small"
          :row-props="guildRowProps"
        />
      </n-space>
    </n-tab-pane>

    <!-- Tab 6：配置 -->
    <n-tab-pane name="config" tab="配置">
      <n-card title="PalDefender 配置热重载" size="small">
        <n-space vertical>
          <n-alert type="info" :show-icon="false" style="font-size:12px">
            重新加载 Banlist、ImportRules 等配置，无需重启游戏服。
          </n-alert>
          <n-popconfirm @positive-click="reloadConfig">
            <template #trigger>
              <n-button type="warning">热重载配置</n-button>
            </template>
            确定要热重载 PalDefender 配置吗？
          </n-popconfirm>
        </n-space>
      </n-card>
    </n-tab-pane>
  </n-tabs>

  <!-- 玩家操作弹窗（私聊 / 踢出 / 封禁） -->
  <n-modal v-model:show="actionModal.show" preset="card"
    :title="actionModal.title" style="max-width:440px" :mask-closable="false">
    <n-space vertical>
      <n-text style="font-size:13px">玩家：{{ actionModal.row?.Name }}（{{ actionModal.row?.UserId }}）</n-text>
      <template v-if="actionModal.mode === 'msg'">
        <n-input v-model:value="actionModal.reason" type="textarea" :autosize="{ minRows: 2 }"
          placeholder="输入私聊消息" />
      </template>
      <template v-else>
        <n-input v-model:value="actionModal.reason" type="textarea" :autosize="{ minRows: 2 }"
          :placeholder="actionModal.mode === 'ban' ? '封禁原因（可选）' : '踢出原因（可选）'" />
        <n-form-item v-if="actionModal.mode === 'ban'" label="同时封禁 IP"
          label-placement="left" style="margin-bottom:0">
          <n-switch v-model:value="actionModal.banIp" />
        </n-form-item>
      </template>
      <n-space justify="end">
        <n-button @click="actionModal.show = false">取消</n-button>
        <n-button :type="actionModal.mode === 'ban' ? 'error' : 'primary'"
          :loading="actionModal.loading" @click="confirmPlayerAction">
          确认
        </n-button>
      </n-space>
    </n-space>
  </n-modal>

  <!-- 手动封 IP 弹窗 -->
  <n-modal v-model:show="banIPModal.show" preset="card" title="手动封禁 IP"
    style="max-width:400px" :mask-closable="false">
    <n-space vertical>
      <n-input v-model:value="banIPModal.ip" placeholder="IP 地址，如 1.2.3.4" />
      <n-input v-model:value="banIPModal.reason" type="textarea" :autosize="{ minRows: 2 }"
        placeholder="封禁原因（可选）" />
      <n-space justify="end">
        <n-button @click="banIPModal.show = false">取消</n-button>
        <n-button type="error" :loading="banIPModal.loading" @click="confirmBanIP">封禁</n-button>
      </n-space>
    </n-space>
  </n-modal>

  <!-- 解封原因弹窗 -->
  <n-modal v-model:show="unbanModal.show" preset="card" :title="unbanModal.title"
    style="max-width:400px" :mask-closable="false">
    <n-space vertical>
      <n-input v-model:value="unbanModal.reason" type="textarea" :autosize="{ minRows: 2 }"
        placeholder="解封原因（可选）" />
      <n-space justify="end">
        <n-button @click="unbanModal.show = false">取消</n-button>
        <n-button type="primary" :loading="unbanModal.loading" @click="confirmUnban">确认解封</n-button>
      </n-space>
    </n-space>
  </n-modal>

  <!-- 公会详情弹窗 -->
  <n-modal v-model:show="guildModal.show" preset="card"
    :title="`公会详情 - ${guildName(guildDetail)}`" style="max-width:720px">
    <n-spin :show="guildModal.loading">
      <n-descriptions v-if="guildDetail" :column="2" bordered size="small" style="margin-bottom:12px">
        <n-descriptions-item label="公会 ID">{{ guildVal(guildDetail, ['GuildUUID', 'GuildId', 'guild_uuid']) }}</n-descriptions-item>
        <n-descriptions-item label="成员数">{{ guildMembers(guildDetail).length }}</n-descriptions-item>
      </n-descriptions>
      <n-divider style="margin:8px 0">成员列表</n-divider>
      <n-data-table
        :columns="guildMemberCols"
        :data="guildMembers(guildDetail)"
        size="small"
        :bordered="false"
        :pagination="{ pageSize: 10 }"
      />
      <n-divider style="margin:12px 0">营地列表</n-divider>
      <n-data-table
        :columns="campCols"
        :data="guildCamps(guildDetail)"
        size="small"
        :bordered="false"
        :pagination="{ pageSize: 10 }"
      />
    </n-spin>
  </n-modal>

  <!-- 删除据点弹窗 -->
  <n-modal v-model:show="deleteBaseModal.show" preset="card" title="删除据点（不可逆）"
    style="max-width:460px" :mask-closable="false">
    <n-space vertical>
      <n-alert type="error" :show-icon="false" style="font-size:12px">
        删除据点不可逆，营内建筑与帕鲁将被清除。请仔细核对 Camp ID。
      </n-alert>
      <n-text style="font-size:13px">目标 Camp ID：{{ deleteBaseModal.campId }}</n-text>
      <n-input v-model:value="deleteBaseModal.confirm" placeholder="请粘贴 Camp ID 以确认" />
      <n-space justify="end">
        <n-button @click="deleteBaseModal.show = false">取消</n-button>
        <n-button type="error" :loading="deleteBaseModal.loading"
          :disabled="deleteBaseModal.confirm !== deleteBaseModal.campId"
          @click="confirmDeleteBase">确认删除</n-button>
      </n-space>
    </n-space>
  </n-modal>

  <!-- PalDefender 安装进度弹窗 -->
  <n-modal v-model:show="showPdProgress" preset="card" title="安装 PalDefender"
    style="max-width:520px" :mask-closable="false">
    <n-space vertical>
      <n-progress type="line" :percentage="pdPercent"
        :status="pdDone ? (pdSuccess ? 'success' : 'error') : 'default'"
        :indicator-placement="'inside'" />
      <n-text>{{ pdMessage || '准备安装...' }}</n-text>
      <n-text v-if="pdError" depth="3" style="font-size:12px;color:#d03050">{{ pdError }}</n-text>
    </n-space>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import {
  NSpace, NCard, NAlert, NDescriptions, NDescriptionsItem, NTag,
  NButton, NText, NModal, NProgress, NPopconfirm, NTabs, NTabPane,
  NDataTable, NInput, NInputNumber, NSelect, NSwitch, NForm, NFormItem, NSpin, NDivider, NIcon,
  useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { ShieldOutline, SwapHorizontalOutline, HelpCircleOutline, EyeOutline } from '@vicons/ionicons5'
import { api } from '@/api'

const message = useMessage()

/* ============ 通用工具 ============ */
function formatTime(t: any): string {
  if (t === null || t === undefined || t === '') return '-'
  let d: Date
  if (typeof t === 'number') {
    d = new Date(t < 1e12 ? t * 1000 : t)
  } else if (typeof t === 'string') {
    const n = Number(t)
    d = isNaN(n) ? new Date(t) : new Date(n < 1e12 ? n * 1000 : n)
  } else {
    d = new Date(t)
  }
  if (isNaN(d.getTime())) return String(t)
  return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function pick(obj: any, keys: string[]): any {
  if (!obj) return undefined
  for (const k of keys) {
    if (obj[k] !== undefined && obj[k] !== null) return obj[k]
  }
  return undefined
}

/* ============ Tab 控制（懒加载） ============ */
const activeTab = ref('install')
const loaded = reactive<Record<string, boolean>>({})

function onTab(name: string) {
  if (name === 'players' && !loaded.players) loadPlayers()
  else if (name === 'banlist' && !loaded.banlist) loadBanlist()
  else if (name === 'guild' && !loaded.guild) loadGuilds()
}

/* ============ REST API 连接状态 ============ */
const connLoading = ref(false)
const connStatus = ref<'idle' | 'online' | 'failed' | 'unconfigured'>('idle')
const connVersion = ref('')

const connText = computed(() => {
  switch (connStatus.value) {
    case 'online': return '在线'
    case 'failed': return '连接失败'
    case 'unconfigured': return '未配置'
    default: return '未测试'
  }
})
const connTagType = computed(() => {
  switch (connStatus.value) {
    case 'online': return 'success'
    case 'failed': return 'error'
    case 'unconfigured': return 'warning'
    default: return 'default'
  }
})

async function testConnection() {
  connLoading.value = true
  try {
    const v = await api.pdVersion()
    connStatus.value = 'online'
    connVersion.value = v?.VersionLong || v?.Version || v?.version || ''
  } catch (e: any) {
    if (e.message && e.message.includes('未配置')) {
      connStatus.value = 'unconfigured'
    } else {
      connStatus.value = 'failed'
      message.error(e.message || '连接失败')
    }
    connVersion.value = ''
  } finally {
    connLoading.value = false
  }
}

/* ============ 安装状态 ============ */
const loading = ref(false)
const installing = ref(false)
const status = ref<any>({})

const showPdProgress = ref(false)
const pdPercent = ref(0)
const pdMessage = ref('')
const pdDone = ref(false)
const pdSuccess = ref(false)
const pdError = ref('')

const token = () => localStorage.getItem('palworld-panel_token') || ''

// PalDefender 连接与处置配置（从「设置」页迁移至此）
const pdForm = reactive({
  host: 'gameserver',
  port: 17993,
  token: '',
  anticheat_enabled: 'true',
  cheaters_kick: 'true',
  cheaters_ban: 'true',
  cheaters_ipban: 'false',
})
const pdTokenSet = ref(false)
const pdSaving = ref(false)
const generatingToken = ref(false)
const revealingToken = ref(false)

// 点击眼睛：获取已保存的 Token 明文
async function revealPdToken() {
  if (revealingToken.value) return
  revealingToken.value = true
  try {
    const res = await api.revealSecret('paldefender.token')
    pdForm.token = res.value || ''
  } catch (e: any) {
    message.error(e.message || '获取失败')
  } finally {
    revealingToken.value = false
  }
}

async function loadPdSettings() {
  try {
    const s = await api.getSettings()
    if (s['paldefender.host']) pdForm.host = s['paldefender.host']
    if (s['paldefender.port']) pdForm.port = parseInt(s['paldefender.port']) || 17993
    pdForm.anticheat_enabled = s['paldefender.anticheat_enabled'] ?? 'true'
    pdForm.cheaters_kick = s['paldefender.cheaters_kick'] ?? 'true'
    pdForm.cheaters_ban = s['paldefender.cheaters_ban'] ?? 'true'
    pdForm.cheaters_ipban = s['paldefender.cheaters_ipban'] ?? 'false'
    pdForm.token = ''
    pdTokenSet.value = s['paldefender.token__set'] === 'true'
  } catch (e) {
    // 设置接口可能需要初始化，忽略
  }
}

async function generateToken() {
  generatingToken.value = true
  try {
    const res = await api.createPalDefenderToken('PalWorldPanel')
    pdForm.token = res.token
    pdTokenSet.value = true
    message.success(`Token 已生成并写入 ${res.tokens_dir}，保存配置后生效`, { duration: 6000 })
  } catch (e: any) {
    message.error(e.message)
  } finally {
    generatingToken.value = false
  }
}

async function savePdSettings() {
  pdSaving.value = true
  try {
    const payload: Record<string, any> = {
      'paldefender.host': pdForm.host,
      'paldefender.port': String(pdForm.port),
      'paldefender.anticheat_enabled': pdForm.anticheat_enabled,
      'paldefender.cheaters_kick': pdForm.cheaters_kick,
      'paldefender.cheaters_ban': pdForm.cheaters_ban,
      'paldefender.cheaters_ipban': pdForm.cheaters_ipban,
    }
    if (pdForm.token) payload['paldefender.token'] = pdForm.token
    const resp = await fetch('/api/settings', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token()}` },
      body: JSON.stringify(payload),
    })
    const data = await resp.json()
    if (resp.ok) {
      message.success(data.message || '已保存')
      await loadPdSettings()
    } else {
      message.error(data.error || '保存失败')
    }
  } catch (e: any) {
    message.error(e.message)
  } finally {
    pdSaving.value = false
  }
}

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

/* ============ Tab 2：玩家管理 ============ */
const players = ref<any[]>([])
const playersLoading = ref(false)
const playersLoaded = ref(false)
const playerSearch = ref('')
const onlineCount = ref(0)

const filteredPlayers = computed(() => {
  const q = playerSearch.value.trim().toLowerCase()
  if (!q) return players.value
  return players.value.filter((p) =>
    String(p.Name || '').toLowerCase().includes(q) ||
    String(p.UserId || '').toLowerCase().includes(q) ||
    String(p.PlayerUID || '').toLowerCase().includes(q)
  )
})

const playerOptions = computed(() =>
  players.value.map((p) => ({
    label: `${p.Name || '-'} (${p.UserId})`,
    value: p.UserId,
  }))
)

function statusTagType(s: any) {
  const v = String(s || '').toLowerCase()
  if (v.includes('online') || v === '1' || v === 'true') return 'success'
  if (v.includes('offline') || v === '0') return 'default'
  return 'default'
}

function statusText(s: any) {
  const v = String(s || '').toLowerCase()
  if (v.includes('online') || v === '1' || v === 'true') return '在线'
  if (v.includes('offline') || v === '0') return '离线'
  return s || '-'
}

function locText(r: any) {
  if (r.MapLocation) return r.MapLocation
  const w = r.WorldLocation
  if (w && typeof w === 'object') {
    const x = w.X ?? w.x, y = w.Y ?? w.y
    if (x !== undefined && y !== undefined) return `${x}, ${y}`
  }
  return '-'
}

const playerCols: DataTableColumns<any> = [
  { title: '名称', key: 'Name', render: (r) => r.Name || '-' },
  { title: 'UserId', key: 'UserId', width: 180, ellipsis: { tooltip: true } },
  { title: 'PlayerUID', key: 'PlayerUID', width: 200, ellipsis: { tooltip: true } },
  { title: 'IP', key: 'IP', width: 130, render: (r) => r.IP || '-' },
  { title: '公会', key: 'GuildName', width: 140, render: (r) => r.GuildName || '-' },
  {
    title: '状态', key: 'Status', width: 80,
    render: (r) => h(NTag, { size: 'small', type: statusTagType(r.Status) }, { default: () => statusText(r.Status) }),
  },
  { title: '位置', key: 'loc', width: 160, render: locText },
  {
    title: '操作', key: 'actions', width: 200, fixed: 'right',
    render: (r) => h(NSpace, { size: 4 }, {
      default: () => [
        h(NButton, { size: 'tiny', onClick: () => openPlayerAction('msg', r) }, { default: () => '私聊' }),
        h(NButton, { size: 'tiny', type: 'warning', ghost: true, onClick: () => openPlayerAction('kick', r) }, { default: () => '踢出' }),
        h(NButton, { size: 'tiny', type: 'error', ghost: true, onClick: () => openPlayerAction('ban', r) }, { default: () => '封禁' }),
      ],
    }),
  },
]

const actionModal = reactive({
  show: false,
  mode: 'msg' as 'msg' | 'kick' | 'ban',
  title: '',
  row: null as any,
  reason: '',
  banIp: false,
  loading: false,
})

function openPlayerAction(mode: 'msg' | 'kick' | 'ban', row: any) {
  actionModal.mode = mode
  actionModal.row = row
  actionModal.reason = ''
  actionModal.banIp = false
  actionModal.title = mode === 'msg' ? '私聊玩家' : mode === 'kick' ? '踢出玩家' : '封禁玩家'
  actionModal.show = true
}

async function loadPlayers() {
  playersLoading.value = true
  try {
    const data = await api.pdGetPlayers()
    players.value = data?.Players || []
    onlineCount.value = data?.Meta?.OnlineCount ?? players.value.filter((p) => statusTagType(p.Status) === 'success').length
    playersLoaded.value = true
    loaded.players = true
  } catch (e: any) {
    message.error(e.message)
  } finally {
    playersLoading.value = false
  }
}

async function confirmPlayerAction() {
  const row = actionModal.row
  if (!row) return
  actionModal.loading = true
  try {
    if (actionModal.mode === 'msg') {
      if (!actionModal.reason.trim()) { message.warning('请输入消息'); actionModal.loading = false; return }
      await api.pdMessage(row.UserId, actionModal.reason)
      message.success('私聊已发送')
    } else if (actionModal.mode === 'kick') {
      await api.pdKick(row.UserId, actionModal.reason)
      message.success('玩家已踢出')
    } else {
      await api.pdBan(row.UserId, actionModal.reason, actionModal.banIp)
      message.success('玩家已封禁')
    }
    actionModal.show = false
    if (actionModal.mode !== 'msg') loadPlayers()
  } catch (e: any) {
    message.error(e.message)
  } finally {
    actionModal.loading = false
  }
}

/* ============ Tab 3：封禁列表 ============ */
const banlist = ref<any>(null)
const banLoading = ref(false)
const banSubtab = ref<'user' | 'ip'>('user')

function entryIssuer(e: any) {
  return pick(e, ['BannedBy'])?.NameValue || '-'
}
function entryIP(e: any) {
  return pick(e, ['BannedBy'])?.IP || e.IP || '-'
}
function entryReason(e: any) {
  return pick(e, ['BannedBy'])?.Reason || '-'
}
function entryTs(e: any) {
  return formatTime(pick(e, ['BannedBy'])?.Timestamp)
}

function activeTag(active: any) {
  const on = active === true || String(active).toLowerCase() === 'true'
  return h(NTag, { size: 'small', type: on ? 'error' : 'success' }, { default: () => on ? '生效中' : '已解除' })
}

const userBanCols: DataTableColumns<any> = [
  { title: 'UserId', key: 'UserId', ellipsis: { tooltip: true } },
  { title: '状态', key: 'Active', width: 90, render: (r) => activeTag(r.Active) },
  { title: '执行者', key: 'issuer', width: 130, render: entryIssuer },
  { title: 'IP', key: 'ip', width: 130, render: entryIP },
  { title: '原因', key: 'reason', render: entryReason },
  { title: '封禁时间', key: 'ts', width: 160, render: entryTs },
  {
    title: '操作', key: 'act', width: 90,
    render: (r) => {
      const on = r.Active === true || String(r.Active).toLowerCase() === 'true'
      if (!on) return null
      return h(NButton, {
        size: 'tiny', type: 'primary', ghost: true,
        onClick: () => openUnban('user', r.UserId),
      }, { default: () => '解封' })
    },
  },
]

const ipBanCols: DataTableColumns<any> = [
  { title: 'IP', key: 'IP', render: (r) => r.IP || '-' },
  { title: '状态', key: 'Active', width: 90, render: (r) => activeTag(r.Active) },
  { title: '执行者', key: 'issuer', width: 130, render: entryIssuer },
  { title: 'UserId', key: 'uid', width: 180, render: (r) => pick(r, ['BannedBy'])?.NameValue || r.UserId || '-' },
  { title: '原因', key: 'reason', render: entryReason },
  { title: '封禁时间', key: 'ts', width: 160, render: entryTs },
  {
    title: '操作', key: 'act', width: 90,
    render: (r) => {
      const on = r.Active === true || String(r.Active).toLowerCase() === 'true'
      if (!on) return null
      return h(NButton, {
        size: 'tiny', type: 'primary', ghost: true,
        onClick: () => openUnban('ip', r.IP),
      }, { default: () => '解封' })
    },
  },
]

const banIPModal = reactive({ show: false, ip: '', reason: '', loading: false })
const unbanModal = reactive({ show: false, kind: 'user' as 'user' | 'ip', target: '', reason: '', loading: false, title: '' })

function openBanIP() {
  banIPModal.ip = ''
  banIPModal.reason = ''
  banIPModal.show = true
}

async function confirmBanIP() {
  if (!banIPModal.ip.trim()) { message.warning('请输入 IP'); return }
  banIPModal.loading = true
  try {
    await api.pdBanIP(banIPModal.ip.trim(), banIPModal.reason)
    message.success('IP 已封禁')
    banIPModal.show = false
    loadBanlist()
  } catch (e: any) {
    message.error(e.message)
  } finally {
    banIPModal.loading = false
  }
}

function openUnban(kind: 'user' | 'ip', target: string) {
  unbanModal.kind = kind
  unbanModal.target = target
  unbanModal.reason = ''
  unbanModal.title = kind === 'user' ? `解封用户 ${target}` : `解封 IP ${target}`
  unbanModal.show = true
}

async function confirmUnban() {
  unbanModal.loading = true
  try {
    if (unbanModal.kind === 'user') {
      await api.pdUnban(unbanModal.target, unbanModal.reason)
    } else {
      await api.pdUnbanIP(unbanModal.target, unbanModal.reason)
    }
    message.success('已解封')
    unbanModal.show = false
    loadBanlist()
  } catch (e: any) {
    message.error(e.message)
  } finally {
    unbanModal.loading = false
  }
}

async function loadBanlist() {
  banLoading.value = true
  try {
    banlist.value = await api.pdBanlist()
    loaded.banlist = true
  } catch (e: any) {
    message.error(e.message)
  } finally {
    banLoading.value = false
  }
}

/* ============ Tab 4：广播与警报 ============ */
const broadcastMsg = ref('')
const alertMsg = ref('')
const msgTarget = ref<string | null>(null)
const msgContent = ref('')

async function sendBroadcast() {
  if (!broadcastMsg.value.trim()) { message.warning('请输入消息'); return }
  try {
    await api.pdBroadcast(broadcastMsg.value)
    message.success('广播已发送')
    broadcastMsg.value = ''
  } catch (e: any) {
    message.error(e.message)
  }
}

async function sendAlert() {
  if (!alertMsg.value.trim()) { message.warning('请输入消息'); return }
  try {
    await api.pdAlert(alertMsg.value)
    message.success('警报已发送')
    alertMsg.value = ''
  } catch (e: any) {
    message.error(e.message)
  }
}

async function sendPrivateMsg() {
  if (!msgTarget.value) { message.warning('请选择玩家'); return }
  if (!msgContent.value.trim()) { message.warning('请输入消息'); return }
  try {
    await api.pdMessage(msgTarget.value, msgContent.value)
    message.success('私聊已发送')
    msgContent.value = ''
  } catch (e: any) {
    message.error(e.message)
  }
}

/* ============ Tab 5：公会与据点 ============ */
const guilds = ref<any[]>([])
const guildsLoading = ref(false)
const guildModal = reactive({ show: false, loading: false })
const guildDetail = ref<any>(null)
const deleteBaseModal = reactive({ show: false, campId: '', confirm: '', loading: false })

function guildName(g: any) {
  return pick(g, ['GuildName', 'Name', 'guild_name']) || '-'
}
function guildVal(g: any, keys: string[]) {
  return pick(g, keys) ?? '-'
}
function guildMembers(g: any) {
  return pick(g, ['Players', 'Members', 'players', 'members']) || []
}
function guildCamps(g: any) {
  return pick(g, ['BaseCamps', 'Camps', 'Bases', 'base_camps']) || []
}
function campId(c: any) {
  return pick(c, ['CampID', 'CampId', 'ID', 'Id', 'id']) || '-'
}
function campName(c: any) {
  return pick(c, ['Name', 'CampName']) || '-'
}

const guildCols: DataTableColumns<any> = [
  { title: '公会名', key: 'GuildName', render: (r) => guildName(r) },
  { title: '公会 ID', key: 'GuildUUID', width: 220, ellipsis: { tooltip: true },
    render: (r) => guildVal(r, ['GuildUUID', 'GuildId', 'guild_uuid']) },
  { title: '成员数', key: 'members', width: 100,
    render: (r) => pick(r, ['PlayerCount', 'MemberCount', 'player_count']) ?? r.players?.length ?? r.Players?.length ?? '-' },
  { title: '据点数', key: 'camps', width: 100,
    render: (r) => pick(r, ['BaseCampCount', 'BaseCount', 'CampCount', 'base_camp_count']) ?? r.base_ids?.length ?? '-' },
]

const guildMemberCols: DataTableColumns<any> = [
  { title: '名称', key: 'Name', render: (r) => r.Name || r.PlayerName || r.nickname || '-' },
  { title: 'UserId', key: 'UserId', width: 180, ellipsis: { tooltip: true }, render: (r) => r.UserId || r.user_id || '-' },
  { title: 'PlayerUID', key: 'PlayerUID', width: 200, ellipsis: { tooltip: true }, render: (r) => r.PlayerUID || r.player_uid || '-' },
]

const campCols: DataTableColumns<any> = [
  { title: 'Camp ID', key: 'CampID', width: 240, ellipsis: { tooltip: true }, render: campId },
  { title: '名称', key: 'Name', render: campName },
  {
    title: '操作', key: 'act', width: 110,
    render: (r) => {
      const id = campId(r)
      if (id === '-') return null
      return h(NButton, {
        size: 'tiny', type: 'error', ghost: true,
        onClick: () => { deleteBaseModal.campId = String(id); deleteBaseModal.confirm = ''; deleteBaseModal.show = true },
      }, { default: () => '删除据点' })
    },
  },
]

function guildRowProps(row: any) {
  return {
    style: 'cursor: pointer',
    onClick: async () => {
      const id = pick(row, ['GuildUUID', 'GuildId', 'guild_uuid', 'ID', 'Id'])
      if (!id) return
      guildModal.show = true
      guildModal.loading = true
      guildDetail.value = null
      try {
        guildDetail.value = await api.pdGuild(id)
      } catch (e: any) {
        message.error(e.message)
        guildModal.show = false
      } finally {
        guildModal.loading = false
      }
    },
  }
}

async function loadGuilds() {
  guildsLoading.value = true
  try {
    const data = await api.pdGuilds()
    guilds.value = Array.isArray(data) ? data : (data?.Guilds || data?.guilds || [])
    loaded.guild = true
  } catch (e: any) {
    message.error(e.message)
  } finally {
    guildsLoading.value = false
  }
}

async function confirmDeleteBase() {
  deleteBaseModal.loading = true
  try {
    await api.pdDeleteBase(deleteBaseModal.campId)
    message.success('据点已删除')
    deleteBaseModal.show = false
    // 刷新公会详情
    const id = guildDetail.value ? pick(guildDetail.value, ['GuildUUID', 'GuildId', 'guild_uuid']) : null
    if (id) {
      try { guildDetail.value = await api.pdGuild(id) } catch {}
    }
  } catch (e: any) {
    message.error(e.message)
  } finally {
    deleteBaseModal.loading = false
  }
}

/* ============ Tab 6：配置 ============ */
async function reloadConfig() {
  try {
    await api.pdReloadConfig()
    message.success('配置已热重载')
  } catch (e: any) {
    message.error(e.message)
  }
}

/* ============ 挂载 ============ */
onMounted(() => {
  refreshStatus()
  testConnection()
  loadPdSettings()
})
</script>

<style scoped>
</style>
