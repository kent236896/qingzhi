import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  const collapsed = ref(false)
  const pageTitle = ref('首页')

  const sidebarWidth = computed(() => (collapsed.value ? '64px' : '220px'))

  function setCollapsed(value: boolean) {
    collapsed.value = value
  }

  function toggleCollapsed() {
    collapsed.value = !collapsed.value
  }

  function setPageTitle(title: string) {
    pageTitle.value = title
  }

  return {
    collapsed,
    pageTitle,
    sidebarWidth,
    setCollapsed,
    toggleCollapsed,
    setPageTitle,
  }
})
