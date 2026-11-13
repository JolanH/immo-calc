import { request, requestText } from './client'

export interface DataInfo {
  path: string
  schemaVersion: number
  counts: { properties: number; loans: number }
  warning?: string
}

export function getDataInfo(): Promise<DataInfo> {
  return request<DataInfo>('GET', '/api/data/info')
}

/** Replaces all data with an exported YAML file; returns the new info. */
export function importData(yaml: string): Promise<DataInfo> {
  return requestText<DataInfo>('POST', '/api/data/import', yaml, 'application/yaml')
}
