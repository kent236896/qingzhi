export {}

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    breadcrumb?: string[]
  }
}
