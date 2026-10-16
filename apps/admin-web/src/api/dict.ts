import { http } from '@/api/client'
import type { ListParams, PageResult, RecordStatus } from '@/api/identity'

export interface DictType {
  id: string
  name: string
  type: string
  status: RecordStatus
  remark: string
  creator: string
  create_time: string
  updater: string
  update_time: string
  deleted: boolean
  deleted_time: string | null
}

export interface DictData {
  id: string
  sort: number
  label: string
  value: string
  dict_type: string
  status: RecordStatus
  color_type: string
  css_class: string
  remark: string
  creator: string
  create_time: string
  updater: string
  update_time: string
  deleted: boolean
}

export const colorTypeOptions = [
  { label: '默认', value: 'default' },
  { label: '主要', value: 'primary' },
  { label: '成功', value: 'success' },
  { label: '信息', value: 'info' },
  { label: '警告', value: 'warning' },
  { label: '危险', value: 'danger' },
]

export function listDictTypes(params: ListParams = {}) {
  return http.get<PageResult<DictType>>('/api/v1/dict-types', { params })
}

export function createDictType(body: { name: string; type: string; status?: RecordStatus; remark?: string }) {
  return http.post<DictType>('/api/v1/dict-types', body)
}

export function updateDictType(
  id: string,
  body: { name?: string; type?: string; status?: RecordStatus; remark?: string },
) {
  return http.patch<DictType>(`/api/v1/dict-types/${id}`, body)
}

export function deleteDictType(id: string) {
  return http.delete<null>(`/api/v1/dict-types/${id}`)
}

export function listDictData(dictType: string, params: ListParams = {}) {
  return http.get<PageResult<DictData>>('/api/v1/dict-data', { params: { ...params, dict_type: dictType } })
}

export function createDictData(body: {
  sort?: number
  label: string
  value: string
  dict_type: string
  status?: RecordStatus
  color_type?: string
  css_class?: string
  remark?: string
}) {
  return http.post<DictData>('/api/v1/dict-data', body)
}

export function updateDictData(
  id: string,
  body: {
    sort?: number
    label?: string
    value?: string
    status?: RecordStatus
    color_type?: string
    css_class?: string
    remark?: string
  },
) {
  return http.patch<DictData>(`/api/v1/dict-data/${id}`, body)
}

export function deleteDictData(id: string) {
  return http.delete<null>(`/api/v1/dict-data/${id}`)
}
