<template>
  <n-space vertical :size="16">
    <n-alert type="info" :show-icon="false">
      修改游戏服配置（PalWorldSettings.ini）。带 <n-text strong>需重启</n-text> 标记的项需重启游戏服生效，
      保存时可勾选"保存并重启"。
    </n-alert>

    <n-tabs type="line" animated>
      <n-tab-pane v-for="g in groups" :key="g" :tab="g" :name="g">
        <n-card size="small">
          <n-form label-placement="top" v-if="fields[g]">
            <n-grid cols="1 s:2 m:2" responsive="screen" :x-gap="16">
              <n-gi v-for="f in fields[g]" :key="f.key">
                <n-form-item :label="fieldLabel(f)">
                  <!-- 布尔开关 -->
                  <n-switch v-if="f.type === 'bool'"
                    :value="form[f.key] === 'True'"
                    @update:value="(v: boolean) => (form[f.key] = v ? 'True' : 'False')" />
                  <!-- 枚举下拉 -->
                  <n-select v-else-if="f.type === 'enum'"
                    :value="form[f.key]"
                    :options="f.options!.map((o: string) => ({ label: o, value: o }))"
                    @update:value="(v: string) => (form[f.key] = v)" />
                  <!-- 数字 -->
                  <n-input-number v-else-if="f.type === 'int'"
                    :value="toNum(form[f.key])"
                    @update:value="(v: number | null) => (form[f.key] = String(v ?? 0))"
                    style="width:100%" />
                  <n-input-number v-else-if="f.type === 'float'"
                    :value="toNum(form[f.key])" :step="0.1"
                    @update:value="(v: number | null) => (form[f.key] = String(v ?? 0))"
                    style="width:100%" />
                  <!-- 字符串 -->
                  <n-input v-else v-model:value="form[f.key]"
                    :type="isSecret(f.key) ? 'password' : 'text'"
                    show-password-on="click"
                    :placeholder="f.default" />
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
  NSwitch, NSelect, NInputNumber, NInput, NButton, NText, useMessage,
} from 'naive-ui'
import { gameSettingsApi, type ConfigField, type GameSettingsData } from '@/api/gamesettings'

const message = useMessage()
const schema = ref<ConfigField[]>([])
const data = ref<GameSettingsData | null>(null)
const iniPath = ref('')
const form = reactive<Record<string, string>>({})
const saving = ref(false)

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
