<template>
  <n-space vertical :size="16">
    <PageHeader title="游戏配置" subtitle="修改 PalWorldSettings.ini，带“需重启”标记的项需重启生效" />

    <n-alert type="info" :show-icon="false">
      修改游戏服配置（PalWorldSettings.ini）。带 <n-text strong>需重启</n-text> 标记的项需重启游戏服生效，
      保存时可选择"保存并重启"。
    </n-alert>

    <n-alert v-if="exists === false" type="warning" :show-icon="false">
      尚未检测到 PalWorldSettings.ini。游戏安装完成并首次启动后会自动生成该文件；
      现在保存的配置会在游戏启动时写入并生效。
    </n-alert>

    <n-input v-model:value="keyword" placeholder="搜索配置项（名称 / 字段名）" clearable>
      <template #prefix>
        <n-icon :component="SearchOutline" />
      </template>
    </n-input>

    <n-tabs type="line" animated default-value="服务器" v-model:value="activeGroup">
      <n-tab-pane v-for="g in groups" :key="g" :name="g">
        <template #tab>
          <n-icon :component="groupIcon(g)" style="vertical-align:-2px;margin-right:4px" />
          {{ g }}
        </template>
        <n-card size="small" :title="g">
          <n-form label-placement="top" v-if="filteredFields[g]?.length">
            <n-grid cols="1 s:2 m:2" responsive="screen" :x-gap="16">
              <n-gi v-for="f in filteredFields[g]" :key="f.key">
                <n-form-item>
                  <template #label>
                    <span class="field-label">
                      {{ fieldLabel(f) }}
                      <n-tooltip v-if="f.description" trigger="hover">
                        <template #trigger>
                          <n-icon class="help-icon" :component="HelpCircleOutline" />
                        </template>
                        <span class="help-text">{{ f.description }}</span>
                      </n-tooltip>
                    </span>
                  </template>
                  <n-switch v-if="f.type === 'bool'"
                    :value="form[f.key] === 'True'"
                    @update:value="(v: boolean) => { form[f.key] = v ? 'True' : 'False'; markCustom(f.key) }" />
                  <n-select v-else-if="f.type === 'enum'"
                    :value="form[f.key]"
                    :options="f.options || []"
                    @update:value="(v: string) => onEnumChange(f.key, v)" />
                  <n-select v-else-if="f.type === 'multi'"
                    :value="(form[f.key] || '').split(',').filter(Boolean)"
                    :options="f.options || []"
                    multiple
                    @update:value="(v: string[]) => { form[f.key] = v.join(','); markCustom(f.key) }" />
                  <n-input-number v-else-if="f.type === 'int'"
                    :value="toNum(form[f.key])"
                    :min="f.min ?? undefined"
                    :max="f.max && f.max > 0 ? f.max : undefined"
                    :step="f.step ?? 1"
                    @update:value="(v: number | null) => { form[f.key] = String(v ?? 0); markCustom(f.key) }"
                    style="width:100%" />
                  <n-input-number v-else-if="f.type === 'float'"
                    :value="toNum(form[f.key])"
                    :min="f.min ?? 0"
                    :max="f.max && f.max > 0 ? f.max : undefined"
                    :step="f.step ?? 0.1"
                    @update:value="(v: number | null) => { form[f.key] = String(v ?? 0); markCustom(f.key) }"
                    style="width:100%" />
                  <n-select v-else-if="f.key === 'DenyTechnologyList'"
                    :value="parseTechList(form[f.key])"
                    :options="techOptions"
                    multiple filterable
                    max-tag-count="responsive"
                    @update:value="(v: string[]) => { form[f.key] = formatTechList(v); markCustom(f.key) }" />
                  <n-input v-else v-model:value="form[f.key]"
                    :type="isSecret(f.key) ? 'password' : 'text'"
                    show-password-on="click"
                    :placeholder="f.default"
                    @update:value="() => markCustom(f.key)" />
                </n-form-item>
              </n-gi>
            </n-grid>
          </n-form>
          <n-text v-else depth="3" style="font-size:13px">没有匹配的配置项</n-text>
        </n-card>
      </n-tab-pane>
    </n-tabs>

    <div class="action-bar">
      <div class="action-bar__left">
        <PlatformBadge />
        <n-text depth="3" style="font-size:12px">
          配置文件：{{ data?.path || iniPath }}
        </n-text>
      </div>
      <n-space :size="8" wrap>
        <n-button @click="load">重置</n-button>
        <n-button type="primary" :loading="saving" @click="save(false)">保存</n-button>
        <n-button type="error" :loading="saving" @click="save(true)">保存并重启</n-button>
      </n-space>
    </div>
  </n-space>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  NSpace, NAlert, NTabs, NTabPane, NCard, NForm, NFormItem, NGrid, NGi,
  NSwitch, NSelect, NInputNumber, NInput, NButton, NText, NTooltip, NIcon,
  useMessage,
} from 'naive-ui'
import {
  HelpCircleOutline, SearchOutline, ServerOutline, GlobeOutline,
  GameControllerOutline, PeopleOutline, HomeOutline, CashOutline,
  FlashOutline, PeopleCircleOutline, SettingsOutline,
} from '@vicons/ionicons5'
import type { Component } from 'vue'
import { gameSettingsApi, type ConfigField, type GameSettingsData } from '@/api/gamesettings'
import PageHeader from '@/components/PageHeader.vue'
import PlatformBadge from '@/components/PlatformBadge.vue'

const message = useMessage()
const schema = ref<ConfigField[]>([])
const data = ref<GameSettingsData | null>(null)
const iniPath = ref('')
const exists = ref<boolean | null>(null)
const form = reactive<Record<string, string>>({})
const saving = ref(false)
const keyword = ref('')
const activeGroup = ref('服务器')
const techOptions = ref<{ label: string, value: string }[]>([])

fetch('/data/tech_list.json').then(r => r.json()).then((data: any[]) => {
  techOptions.value = data.map(t => ({ label: t.name, value: t.id }))
}).catch(() => {})

function parseTechList(v: string): string[] {
  return (v || '').replace(/[()]/g, '').split(',').map(s => s.trim()).filter(Boolean)
}
function formatTechList(v: string[]): string {
  return '(' + v.join(',') + ')'
}

const GROUP_ICONS: Record<string, Component> = {
  '服务器': ServerOutline,
  '世界': GlobeOutline,
  '帕鲁': GameControllerOutline,
  '玩家': PeopleOutline,
  '据点': HomeOutline,
  '经济': CashOutline,
  '战斗': FlashOutline,
  '公会': PeopleCircleOutline,
  '高级': SettingsOutline,
}
function groupIcon(g: string): Component {
  return GROUP_ICONS[g] || SettingsOutline
}

const groups = computed(() => {
  const s = new Set<string>()
  schema.value.forEach((f) => s.add(f.group))
  return Array.from(s)
})

const fields = computed(() => {
  const m: Record<string, ConfigField[]> = {}
  schema.value.forEach((f) => {
    if (!m[f.group]) m[f.group] = []
    m[f.group].push(f)
  })
  return m
})

const filteredFields = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  const out: Record<string, ConfigField[]> = {}
  Object.entries(fields.value).forEach(([g, list]) => {
    if (!q) { out[g] = list; return }
    out[g] = list.filter(
      (f) => f.label.toLowerCase().includes(q) || f.key.toLowerCase().includes(q)
    )
  })
  return out
})

function fieldLabel(f: ConfigField) {
  return f.requires_restart ? `${f.label} （需重启）` : f.label
}
function isSecret(k: string) {
  return /password/i.test(k)
}
function toNum(v: string) {
  const n = parseFloat(v)
  return isNaN(n) ? 0 : n
}

const difficultyPresets: Record<string, Record<string, string>> = {
  Easy: {
    DayTimeSpeedRate: '1.0', NightTimeSpeedRate: '1.0', ExpRate: '2.0',
    PalCaptureRate: '2.0', PalSpawnNumRate: '1.0', PalDamageRateAttack: '0.8',
    PalDamageRateDefense: '0.8', PlayerDamageRateAttack: '0.5', PlayerDamageRateDefense: '0.5',
    BuildObjectDamageRate: '0.5', BuildObjectDeteriorationDamageRate: '0.0',
    CollectionDropRate: '2.0', CollectionObjectHpRate: '2.0', CollectionObjectRespawnSpeedRate: '2.0',
    EnemyDropItemRate: '2.0',
  },
  Normal: {
    DayTimeSpeedRate: '1.0', NightTimeSpeedRate: '1.0', ExpRate: '1.0',
    PalCaptureRate: '1.0', PalSpawnNumRate: '1.0', PalDamageRateAttack: '1.0',
    PalDamageRateDefense: '1.0', PlayerDamageRateAttack: '1.0', PlayerDamageRateDefense: '1.0',
    BuildObjectDamageRate: '1.0', BuildObjectDeteriorationDamageRate: '1.0',
    CollectionDropRate: '1.0', CollectionObjectHpRate: '1.0', CollectionObjectRespawnSpeedRate: '1.0',
    EnemyDropItemRate: '1.0',
  },
  Hard: {
    DayTimeSpeedRate: '1.0', NightTimeSpeedRate: '1.0', ExpRate: '0.5',
    PalCaptureRate: '0.5', PalSpawnNumRate: '1.0', PalDamageRateAttack: '1.5',
    PalDamageRateDefense: '1.5', PlayerDamageRateAttack: '2.0', PlayerDamageRateDefense: '2.0',
    BuildObjectDamageRate: '1.5', BuildObjectDeteriorationDamageRate: '2.0',
    CollectionDropRate: '0.5', CollectionObjectHpRate: '0.5', CollectionObjectRespawnSpeedRate: '0.5',
    EnemyDropItemRate: '0.5',
  },
}

function markCustom(key: string) {
  const difficultyKeys = Object.keys(difficultyPresets.Easy)
  if (difficultyKeys.includes(key) && form['Difficulty'] && form['Difficulty'] !== 'None') {
    form['Difficulty'] = 'None'
  }
}

function onEnumChange(key: string, v: string) {
  form[key] = v
  if (key === 'Difficulty' && v !== 'None' && difficultyPresets[v]) {
    Object.entries(difficultyPresets[v]).forEach(([k, val]) => {
      form[k] = val
    })
  }
}

async function load() {
  const [s, d] = await Promise.all([gameSettingsApi.schema(), gameSettingsApi.get()])
  schema.value = s.fields
  iniPath.value = s.iniPath
  data.value = d
  exists.value = d.exists
  Object.keys(form).forEach((k) => delete form[k])
  Object.entries(d.settings).forEach(([k, v]) => (form[k] = v))
  if (groups.value.length && !groups.value.includes(activeGroup.value)) {
    activeGroup.value = groups.value[0]
  }
}

async function save(restart: boolean) {
  saving.value = true
  try {
    const res = await gameSettingsApi.save(form, restart)
    message.success(res.message || '已保存')
  } catch (e: any) {
    message.error(e.message)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.field-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.help-icon {
  font-size: 15px;
  color: var(--n-text-color-3, #999);
  cursor: help;
}
.help-text {
  display: block;
  max-width: 260px;
  line-height: 1.5;
}
.action-bar {
  position: sticky;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 12px 16px;
  margin: 4px -8px 0;
  background: var(--n-color, #fff);
  border: 1px solid var(--n-border-color, #efeff5);
  border-radius: 10px;
  box-shadow: 0 -4px 16px rgba(0, 0, 0, 0.05);
  z-index: 10;
}
.action-bar__left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
:deep(.n-tooltip) {
  background: #1f1f1f !important;
  color: #fff !important;
}
</style>
