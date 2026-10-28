import { MessagePlugin } from 'tdesign-vue-next'

export const feedback = {
  success(message: string) {
    void MessagePlugin.success(message)
  },
  warning(message: string) {
    void MessagePlugin.warning(message)
  },
  error(message: string) {
    void MessagePlugin.error(message)
  },
}
