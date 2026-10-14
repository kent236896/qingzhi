import { AxiosError, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'
import { describe, expect, it, vi } from 'vitest'
import { ApiError, REQUEST_ID_HEADER, createHttpClient, normalizeAxiosError } from './http'

const liveBody = {
  code: 0,
  message: 'ok',
  data: { status: 'live' },
  request_id: 'server-request-id',
}

function responseOf(status: number, data: unknown): AxiosResponse {
  return {
    data,
    status,
    statusText: String(status),
    headers: { [REQUEST_ID_HEADER.toLowerCase()]: 'header-request-id' },
    config: {} as InternalAxiosRequestConfig,
  }
}

describe('http client', () => {
  it('unwraps a successful envelope and sends a request id', async () => {
    let sentId = ''
    const client = createHttpClient({
      baseURL: 'http://example.test',
      timeout: 1000,
      generateRequestId: () => 'client-request-id',
      adapter: async (config) => {
        sentId = String(config.headers.get(REQUEST_ID_HEADER))
        return responseOf(200, liveBody)
      },
    })

    const result = await client.get<{ status: string }>('/api/v1/health/live')
    expect(sentId).toBe('client-request-id')
    expect(result).toEqual({
      code: 0,
      message: 'ok',
      data: { status: 'live' },
      requestId: 'server-request-id',
    })
  })

  it('maps timeout and network failures without calling the toast handler when skipped', async () => {
    const onError = vi.fn()
    const timeoutClient = createHttpClient({
      baseURL: 'http://example.test',
      timeout: 1000,
      onError,
      adapter: () => Promise.reject(new AxiosError('timeout', AxiosError.ECONNABORTED)),
    })
    await expect(timeoutClient.get('/api/v1/health/live', { skipErrorToast: true })).rejects.toMatchObject({
      kind: 'timeout',
      message: '请求超时，请稍后重试',
    })
    expect(onError).not.toHaveBeenCalled()

    const networkError = new AxiosError('Network Error')
    expect(normalizeAxiosError(networkError)).toMatchObject({
      kind: 'network',
      message: '网络异常，无法连接服务器',
    })
  })

  it('uses the safe server message and keeps diagnostic fields out of the error text', async () => {
    const onError = vi.fn()
    const client = createHttpClient({
      baseURL: 'http://example.test',
      timeout: 1000,
      onError,
      adapter: () =>
        Promise.reject(
          new AxiosError('Request failed with status code 500', AxiosError.ERR_BAD_RESPONSE, undefined, undefined, {
            ...responseOf(500, {
              code: 50000,
              message: '服务器内部错误',
              data: null,
              request_id: 'req-500',
              stack: 'goroutine 1 [running]',
            }),
          }),
        ),
    })

    await expect(client.get('/api/v1/health/live')).rejects.toMatchObject({
      kind: 'business',
      code: 50000,
      httpStatus: 500,
      requestId: 'req-500',
      message: '服务器内部错误',
    })
    expect(onError).toHaveBeenCalledOnce()
    expect(onError.mock.calls[0]?.[0].message).not.toContain('goroutine')
  })

  it('preserves validation and readiness payloads', () => {
    const validation = normalizeAxiosError(
      new AxiosError('bad request', AxiosError.ERR_BAD_REQUEST, undefined, undefined, {
        ...responseOf(400, {
          code: 40000,
          message: '请求参数不合法',
          data: { fields: [{ field: 'name', message: 'name为必填项' }] },
          request_id: 'req-400',
        }),
      }),
    )
    expect(validation).toBeInstanceOf(ApiError)
    expect(validation).toMatchObject({
      kind: 'business',
      code: 40000,
      httpStatus: 400,
      requestId: 'req-400',
    })

    const notReady = normalizeAxiosError(
      new AxiosError('unavailable', AxiosError.ERR_BAD_RESPONSE, undefined, undefined, {
        ...responseOf(503, {
          code: 50301,
          message: '服务暂不可用',
          data: { status: 'not_ready', checks: { database: 'down', redis: 'up' } },
          request_id: 'req-503',
        }),
      }),
    )
    expect(notReady.data).toEqual({
      status: 'not_ready',
      checks: { database: 'down', redis: 'up' },
    })
  })
})
