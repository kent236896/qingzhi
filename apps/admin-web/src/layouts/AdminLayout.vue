<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const activeMenu = computed(() => (route.name === 'home' ? 'home' : ''))
const breadcrumbs = computed(() => route.meta.breadcrumb ?? [])

function onMenuChange(value: string | number) {
  if (value === 'home') {
    void router.push({ name: 'home' })
  }
}
</script>

<template>
  <t-layout class="admin-shell">
    <t-aside class="admin-aside" width="232px">
      <div class="brand">
        <img src="/favicon.svg" width="32" height="32" alt="" />
        <div>
          <h1>{{ appStore.title }}</h1>
          <p>工程基础</p>
        </div>
      </div>
      <t-menu class="admin-menu" theme="light" :value="activeMenu" @change="onMenuChange">
        <t-menu-item value="home">首页</t-menu-item>
      </t-menu>
    </t-aside>
    <t-layout>
      <t-header class="admin-header" height="64px">
        <t-breadcrumb>
          <t-breadcrumb-item v-for="item in breadcrumbs" :key="item">{{ item }}</t-breadcrumb-item>
        </t-breadcrumb>
        <div class="user-slot" aria-label="用户菜单占位">
          <t-avatar size="small">访</t-avatar>
          <span>访客</span>
        </div>
      </t-header>
      <t-content class="admin-content">
        <router-view />
      </t-content>
    </t-layout>
  </t-layout>
</template>
