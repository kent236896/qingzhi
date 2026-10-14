import { MessagePlugin } from 'tdesign-vue-next'

export const feedback = {
  success(message: string) {
    return MessagePlugin.success(message)
  },
  error(message: string) {
    return MessagePlugin.error(message)
  },
  warning(message: string) {
    return MessagePlugin.warning(message)
  },
}
