export {}

declare module 'axios' {
  interface AxiosRequestConfig {
    skipErrorToast?: boolean
  }
}
