<script setup lang="ts">
import SettingsPageHeader from '@/components/settings/SettingsPageHeader.vue'
import SettingsRow from '@/components/settings/SettingsRow.vue'
import { useSettingsStore, type FontSizePreference } from '@/stores/settings'

const settings = useSettingsStore()

const sizes: { value: FontSizePreference; label: string }[] = [
  { value: 'small', label: '小' },
  { value: 'normal', label: '正常' },
  { value: 'large', label: '大' },
]
</script>

<template>
  <div>
    <SettingsPageHeader title="常规设置" description="配置界面字体等基础选项" />
    <!-- 语言 / 主题：项目尚未接入 i18n 与暗色主题，按约束不展示占位行 -->
    <SettingsRow title="字体大小" description="调整工作台整体字号，偏好保存在本机">
      <div class="font-size">
        <button
          v-for="item in sizes"
          :key="item.value"
          type="button"
          class="font-size__btn"
          :class="{ 'font-size__btn--active': settings.fontSize === item.value }"
          @click="settings.setFontSize(item.value)"
        >
          {{ item.label }}
        </button>
      </div>
    </SettingsRow>
  </div>
</template>

<style scoped>
.font-size {
  display: inline-flex;
  border: 1px solid var(--wb-border);
  border-radius: 8px;
  overflow: hidden;
  background: var(--wb-surface);
}

.font-size__btn {
  border: 0;
  background: transparent;
  min-width: 64px;
  height: 32px;
  padding: 0 12px;
  font-size: 13px;
  color: var(--wb-text);
  cursor: pointer;
}

.font-size__btn + .font-size__btn {
  border-left: 1px solid var(--wb-border);
}

.font-size__btn--active {
  background: var(--wb-primary);
  color: #fff;
}
</style>
