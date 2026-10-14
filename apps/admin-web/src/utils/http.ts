import axios, {
  AxiosError,
  AxiosHeaders,
  type AxiosAdapter,
  type AxiosRequestConfig,
  type AxiosResponse,
} from 'axios'
import type { ApiEnvelope } from '@/types/api'

export const REQUEST_ID_HEADER = 'X-Request-ID'

export type ApiErrorKind = 'timeout' | 'network' | 'http' | 'business'

export class ApiError extends Error {
  readonly kind: ApiErrorKind
  readonly code?: number
  readonly httpStatus?: number
  readonly requestId?: string
  readonly data?: unknown

  constructor(
    message: string,
    options: {
      kind: ApiErrorKind
      code?: number
      httpStatus?: number
      requestId?: string
      data?: unknown
    },
  ) {
    super(message)
    this.name = 'ApiError'
    this.kind = options.kind
    this.code = options.code
    this.httpStatus = options.httpStatus
    this.requestId = options.requestId
    this.data = options.data
  }
}

export interface HttpClientOptions {
  baseURL: string
  timeout: number
  onError?: (error: ApiError) => void
  generateRequestId?: () => string
  adapter?: AxiosAdapter
}

interface EnvelopeBody {
  code: number
  message: string
  data: unknown
  requestId: string
}

export function createHttpClient(options: HttpClientOptions) {
  const instance = axios.create({
    baseURL: options.baseURL,
    timeout: options.timeout,
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    adapter: options.adapter,
  })

  instance.interceptors.request.use((config) => {
    const headers = AxiosHeaders.from(config.headers)
    if (!headers.get(REQUEST_ID_HEADER)) {
      headers.set(REQUEST_ID_HEADER, (options.generateRequestId ?? createRequestId)())
    }
    config.headers = headers
    return config
  })

  async function request<T>(config: AxiosRequestConfig): Promise<ApiEnvelope<T>> {
    try {
      const response = await instance.request<unknown>(config)
      return parseSuccess<T>(response)
    } catch (error) {
      const apiError = normalizeAxiosError(error)
      if (!config.skipErrorToast) {
        options.onError?.(apiError)
      }
      throw apiError
    }
  }

  return {
    request,
    get<T>(url: string, config?: AxiosRequestConfig) {
      return request<T>({ ...config, method: 'GET', url })
    },
  }
}

export function createRequestId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `req-${Date.now()}`
}

export function normalizeAxiosError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error
  }
  if (!axios.isAxiosError(error)) {
    return new ApiError('请求失败', { kind: 'network' })
  }
  if (error.code === AxiosError.ECONNABORTED || error.code === AxiosError.ETIMEDOUT) {
    return new ApiError('请求超时，请稍后重试', { kind: 'timeout' })
  }
  if (!error.response) {
    return new ApiError('网络异常，无法连接服务器', { kind: 'network' })
  }

  const httpStatus = error.response.status
  const envelope = readEnvelope(error.response.data)
  const requestId = envelope?.requestId || headerRequestId(error.response)
  if (envelope) {
    return new ApiError(envelope.message || statusMessage(httpStatus), {
      kind: envelope.code === 0 ? 'http' : 'business',
      code: envelope.code,
      httpStatus,
      requestId,
      data: envelope.data,
    })
  }
  return new ApiError(statusMessage(httpStatus), {
    kind: 'http',
    httpStatus,
    requestId,
  })
}

function parseSuccess<T>(response: AxiosResponse<unknown>): ApiEnvelope<T> {
  const envelope = readEnvelope(response.data)
  const requestId = envelope?.requestId || headerRequestId(response)
  if (!envelope || envelope.code !== 0) {
    throw new ApiError(envelope?.message || statusMessage(response.status), {
      kind: 'business',
      code: envelope?.code,
      httpStatus: response.status,
      requestId,
      data: envelope?.data,
    })
  }
  return {
    code: 0,
    message: envelope.message || 'ok',
    data: envelope.data as T,
    requestId,
  }
}

function readEnvelope(data: unknown): EnvelopeBody | undefined {
  if (!data || typeof data !== 'object') {
    return undefined
  }
  const body = data as Record<string, unknown>
  if (typeof body.code !== 'number') {
    return undefined
  }
  return {
    code: body.code,
    message: typeof body.message === 'string' ? body.message : '',
    data: body.data,
    requestId: typeof body.request_id === 'string' ? body.request_id : '',
  }
}

function headerRequestId(response: AxiosResponse<unknown>): string {
  const value = response.headers?.[REQUEST_ID_HEADER.toLowerCase()] ?? response.headers?.[REQUEST_ID_HEADER]
  return typeof value === 'string' ? value : ''
}

function statusMessage(status: number): string {
  switch (status) {
    case 400:
      return '请求参数不合法'
    case 401:
      return '未授权'
    case 403:
      return '没有访问权限'
    case 404:
      return '资源不存在'
    case 405:
      return '请求方法不允许'
    case 500:
      return '服务器内部错误'
    case 503:
      return '服务暂不可用'
    default:
      return '请求失败'
  }
}
