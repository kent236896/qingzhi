<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <t-card :bordered="false">
    <div class="page-status">
      <p>当前地址没有对应页面。</p>
      <t-button theme="primary" @click="router.push({ name: 'home' })">返回首页</t-button>
    </div>
  </t-card>
</template>
