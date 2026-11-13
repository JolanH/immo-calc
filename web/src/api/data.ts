import { request } from './client'

export interface DataInfo {
  path: string
  schemaVersion: number
  counts: { properties: number; loans: number }
  warning?: string
}

export function getDataInfo(): Promise<DataInfo> {
  return request<DataInfo>('GET', '/api/data/info')
}
