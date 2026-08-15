<template>
  <n-space vertical :size="16">
    <n-alert type="info" :show-icon="false">
      修改游戏服配置（PalWorldSettings.ini）。带 <n-text strong>需重启</n-text> 标记的项需重启游戏服生效，
      保存时可勾选"保存并重启"。
    </n-alert>

    <n-tabs type="line" animated default-value="服务器">
      <n-tab-pane v-for="g in groups" :key="g" :tab="g" :name="g">
        <n-card size="small">
          <n-form label-placement="top" v-if="fields[g]">
            <n-grid cols="1 s:2 m:2" responsive="screen" :x-gap="16">
              <n-gi v-for="f in fields[g]" :key="f.key">
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
                  <!-- 布尔开关 -->
                  <n-switch v-if="f.type === 'bool'"
                    :value="form[f.key] === 'True'"
                    @update:value="(v: boolean) => { form[f.key] = v ? 'True' : 'False'; markCustom(f.key) }" />
                  <!-- 枚举下拉 -->
                  <n-select v-else-if="f.type === 'enum'"
                    :value="form[f.key]"
                    :options="f.options || []"
                    @update:value="(v: string) => onEnumChange(f.key, v)" />
                  <!-- 多选下拉 -->
                  <n-select v-else-if="f.type === 'multi'"
                    :value="(form[f.key] || '').split(',').filter(Boolean)"
                    :options="f.options || []"
                    multiple
                    @update:value="(v: string[]) => { form[f.key] = v.join(','); markCustom(f.key) }" />
                  <!-- 数字 -->
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
                  <!-- 禁用科技列表（多选） -->
                  <n-select v-else-if="f.key === 'DenyTechnologyList'"
                    :value="parseTechList(form[f.key])"
                    :options="techOptions"
                    multiple filterable
                    max-tag-count="responsive"
                    @update:value="(v: string[]) => { form[f.key] = formatTechList(v); markCustom(f.key) }" />
                  <!-- 字符串 -->
                  <n-input v-else v-model:value="form[f.key]"
                    :type="isSecret(f.key) ? 'password' : 'text'"
                    show-password-on="click"
                    :placeholder="f.default"
                    @update:value="() => markCustom(f.key)" />
                </n-form-item>
              </n-gi>
            </n-grid>
          </n-form>
        </n-card>
      </n-tab-pane>
    </n-tabs>

    <n-card size="small">
      <n-space>
        <n-button type="primary" :loading="saving" @click="save(false)">保存配置</n-button>
        <n-button type="error" :loading="saving" @click="save(true)">
          保存并重启游戏服
        </n-button>
        <n-button @click="load">重置/重新加载</n-button>
        <n-text depth="3" style="font-size:12px">
          配置文件：{{ data?.path || iniPath }}
        </n-text>
      </n-space>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  NSpace, NAlert, NTabs, NTabPane, NCard, NForm, NFormItem, NGrid, NGi,
  NSwitch, NSelect, NInputNumber, NInput, NButton, NText, NTooltip, NIcon,
  useMessage,
} from 'naive-ui'
import { HelpCircleOutline } from '@vicons/ionicons5'
import { gameSettingsApi, type ConfigField, type GameSettingsData } from '@/api/gamesettings'

const message = useMessage()
const schema = ref<ConfigField[]>([])
const data = ref<GameSettingsData | null>(null)
const iniPath = ref('')
const form = reactive<Record<string, string>>({})
const saving = ref(false)
const techOptions = ref<{label: string, value: string}[]>([])

// 加载科技列表
fetch('/data/tech_list.json').then(r => r.json()).then((data: any[]) => {
  techOptions.value = data.map(t => ({ label: t.name, value: t.id }))
}).catch(() => {})

function parseTechList(v: string): string[] {
  return (v || '').replace(/[()]/g, '').split(',').map(s => s.trim()).filter(Boolean)
}
function formatTechList(v: string[]): string {
  return '(' + v.join(',') + ')'
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

// 官方难度预设值（来自 DefaultPalWorldSettings.ini）
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

// 修改倍率字段时，如果难度不是自定义，自动切回自定义
function markCustom(key: string) {
  const difficultyKeys = Object.keys(difficultyPresets.Easy)
  if (difficultyKeys.includes(key) && form['Difficulty'] && form['Difficulty'] !== 'None') {
    form['Difficulty'] = 'None'
  }
}

// 难度选择变化时应用预设
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
  Object.keys(form).forEach((k) => delete form[k])
  Object.entries(d.settings).forEach(([k, v]) => (form[k] = v))
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
:deep(.n-tooltip) {
  background: #1f1f1f !important;
  color: #fff !important;
}
</style>
