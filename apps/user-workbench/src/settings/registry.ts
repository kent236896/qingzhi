import type { Component } from 'vue'
import {
  LockOnIcon,
  SettingIcon,
  UserCircleIcon,
  UserIcon,
  LoginIcon,
} from 'tdesign-icons-vue-next'
import GeneralSettingsPanel from '@/components/settings/GeneralSettingsPanel.vue'
import ProfileSettingsPanel from '@/components/settings/ProfileSettingsPanel.vue'
import AccountSettingsPanel from '@/components/settings/AccountSettingsPanel.vue'
import PasswordSettingsPanel from '@/components/settings/PasswordSettingsPanel.vue'
import MemorySettingsPanel from '@/components/settings/MemorySettingsPanel.vue'

export type SettingsSectionId =
  | 'general'
  | 'profile'
  | 'account'
  | 'password'
  | 'memory'

export interface SettingsNavItem {
  id: SettingsSectionId
  title: string
  description: string
  icon: Component
  component: Component
}

export interface SettingsNavGroup {
  id: string
  title: string
  items: SettingsNavItem[]
}

export const SETTINGS_NAV_GROUPS: SettingsNavGroup[] = [
  {
    id: 'account',
    title: '账户',
    items: [
      {
        id: 'general',
        title: '常规设置',
        description: '配置界面字体等基础选项',
        icon: SettingIcon,
        component: GeneralSettingsPanel,
      },
      {
        id: 'profile',
        title: '用户信息',
        description: '管理头像、昵称与联系方式',
        icon: UserIcon,
        component: ProfileSettingsPanel,
      },
      {
        id: 'account',
        title: '账号与登录方式',
        description: '查看账号类型与当前登录方式',
        icon: LoginIcon,
        component: AccountSettingsPanel,
      },
      {
        id: 'password',
        title: '密码管理',
        description: '修改本平台本地账号密码',
        icon: LockOnIcon,
        component: PasswordSettingsPanel,
      },
      {
        id: 'memory',
        title: '我的记忆',
        description: '管理个人长期记忆',
        icon: UserCircleIcon,
        component: MemorySettingsPanel,
      },
    ],
  },
]

export const DEFAULT_SETTINGS_SECTION: SettingsSectionId = 'profile'

const SECTION_IDS = new Set<string>(
  SETTINGS_NAV_GROUPS.flatMap((group) => group.items.map((item) => item.id)),
)

export function isSettingsSection(value: unknown): value is SettingsSectionId {
  return typeof value === 'string' && SECTION_IDS.has(value)
}

export function findSettingsItem(id: SettingsSectionId): SettingsNavItem | undefined {
  for (const group of SETTINGS_NAV_GROUPS) {
    const found = group.items.find((item) => item.id === id)
    if (found) return found
  }
  return undefined
}
