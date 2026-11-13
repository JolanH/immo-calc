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

/**
 * Switches to another data file. Without a path the server shows a native
 * "Ouvrir" dialog. Resolves to undefined when the dialog was cancelled.
 */
export function openFile(path?: string): Promise<DataInfo | undefined> {
  return request<DataInfo | undefined>(
    'POST',
    '/api/data/open',
    path === undefined ? undefined : { path },
  )
}
