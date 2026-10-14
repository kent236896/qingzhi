import { defineStore } from 'pinia'

export const useAppStore = defineStore('app', () => {
  const title = import.meta.env.VITE_APP_TITLE || '擎知管理后台'
  const environment = import.meta.env.VITE_APP_ENV || 'development'
  return { title, environment }
})
