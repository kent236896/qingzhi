<script setup lang="ts">
import { useRouter } from 'vue-router'
import PageStatus from '@/components/common/PageStatus.vue'

const router = useRouter()
</script>

<template>
  <div class="not-found">
    <PageStatus title="页面不存在" description="请检查地址，或返回工作台首页。" kind="error">
      <button type="button" class="not-found__btn" @click="router.push('/')">返回首页</button>
    </PageStatus>
  </div>
</template>

<style scoped>
.not-found {
  max-width: 560px;
  margin: 48px auto;
  padding: 0 16px;
}

.not-found__btn {
  border: 1px solid var(--wb-border);
  background: #fff;
  color: var(--wb-primary);
  border-radius: 6px;
  padding: 6px 12px;
  font-size: 13px;
  cursor: pointer;
}
</style>
