import type { RouteLocationRaw } from 'vue-router'

export {}

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    breadcrumb?: Array<string | { title: string; to?: RouteLocationRaw }>
  }
}
