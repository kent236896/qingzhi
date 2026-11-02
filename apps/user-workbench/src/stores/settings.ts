import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import {
  DEFAULT_SETTINGS_SECTION,
  isSettingsSection,
  type SettingsSectionId,
} from '@/settings/registry'

const FONT_SIZE_KEY = 'qz_wb_font_size'

export type FontSizePreference = 'small' | 'normal' | 'large'

function readFontSize(): FontSizePreference {
  const raw = localStorage.getItem(FONT_SIZE_KEY)
  if (raw === 'small' || raw === 'normal' || raw === 'large') return raw
  return 'normal'
}

function applyFontSize(size: FontSizePreference) {
  document.documentElement.dataset.fontSize = size
}

export const useSettingsStore = defineStore('settings', () => {
  const route = useRoute()
  const router = useRouter()

  const fontSize = ref<FontSizePreference>(readFontSize())
  applyFontSize(fontSize.value)

  const open = computed(() => isSettingsSection(route.query.settings))
  const section = computed<SettingsSectionId>(() => {
    const raw = route.query.settings
    return isSettingsSection(raw) ? raw : DEFAULT_SETTINGS_SECTION
  })

  function setFontSize(size: FontSizePreference) {
    fontSize.value = size
    localStorage.setItem(FONT_SIZE_KEY, size)
    applyFontSize(size)
  }

  async function openSettings(next: SettingsSectionId = DEFAULT_SETTINGS_SECTION) {
    await router.replace({
      query: {
        ...route.query,
        settings: next,
      },
    })
  }

  async function setSection(next: SettingsSectionId) {
    if (!open.value) {
      await openSettings(next)
      return
    }
    await router.replace({
      query: {
        ...route.query,
        settings: next,
      },
    })
  }

  async function closeSettings() {
    if (!('settings' in route.query)) return
    const query = { ...route.query }
    delete query.settings
    await router.replace({ query })
  }

  watch(
    fontSize,
    (value) => {
      applyFontSize(value)
    },
    { immediate: true },
  )

  return {
    open,
    section,
    fontSize,
    setFontSize,
    openSettings,
    setSection,
    closeSettings,
  }
})
