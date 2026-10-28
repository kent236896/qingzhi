<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

const modules = [
  {
    path: '/chat',
    title: 'AI 对话',
    description: '基于授权知识库的流式问答、会话历史与引用来源。',
  },
  {
    path: '/agents',
    title: '智能体',
    description: '浏览与使用企业智能体。功能将在后续阶段实现。',
  },
  {
    path: '/memory',
    title: '我的记忆',
    description: '管理个人长期记忆。当前因隔离验证未通过暂不开放。',
  },
  {
    path: '/profile',
    title: '个人中心',
    description: '查看与修改个人资料、了解登录方式、管理本地密码。',
  },
]

function openModule(path: string) {
  void router.push(path)
}
</script>

<template>
  <div class="home">
    <section class="home__intro">
      <h2 class="home__title">用户工作台</h2>
      <p class="home__lead">
        面向企业员工的知识与智能助手入口。已接入统一登录、个人中心与 AI 对话；智能体与个人记忆将在后续阶段实现。
      </p>
      <div class="home__status">
        <span class="home__badge">工程状态</span>
        <span>登录已接入 · 个人中心可用 · AI / 记忆业务未接入</span>
      </div>
    </section>

    <section class="home__modules" aria-label="功能入口">
      <article v-for="item in modules" :key="item.path" class="home__card">
        <h3>{{ item.title }}</h3>
        <p>{{ item.description }}</p>
        <button type="button" class="home__link" @click="openModule(item.path)">进入页面</button>
      </article>
    </section>
  </div>
</template>

<style scoped>
.home {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-width: 960px;
}

.home__intro {
  padding: 24px;
  background: var(--wb-surface);
  border: 1px solid var(--wb-border);
  border-radius: var(--wb-radius);
}

.home__title {
  margin: 0 0 8px;
  font-size: 22px;
  font-weight: 650;
}

.home__lead {
  margin: 0 0 16px;
  color: var(--wb-text-secondary);
  font-size: 14px;
}

.home__status {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--wb-text-secondary);
}

.home__badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 4px;
  background: #ecf2fe;
  color: var(--wb-primary);
  font-weight: 600;
}

.home__modules {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
}

.home__card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 20px;
  background: var(--wb-surface);
  border: 1px solid var(--wb-border);
  border-radius: var(--wb-radius);
}

.home__card h3 {
  margin: 0;
  font-size: 16px;
}

.home__card p {
  margin: 0;
  flex: 1;
  color: var(--wb-text-secondary);
  font-size: 13px;
}

.home__link {
  align-self: flex-start;
  margin-top: 8px;
  border: 1px solid var(--wb-border);
  background: #fff;
  color: var(--wb-primary);
  border-radius: 6px;
  padding: 6px 12px;
  font-size: 13px;
  cursor: pointer;
}

.home__link:hover {
  border-color: var(--wb-primary);
  background: #f3f8ff;
}
</style>
