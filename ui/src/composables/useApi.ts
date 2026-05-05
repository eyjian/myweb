export interface StatusResponse {
  connected: boolean
  host?: string
  database?: string
  user?: string
}

export interface QueryResponse {
  columns?: string[]
  rows?: unknown[][]
  row_count?: number
  affected_rows?: number
  duration?: string
  is_query: boolean
  warning?: string
  error?: string
}

export interface DatabaseResponse {
  databases: string[]
}

export interface TableResponse {
  tables: string[]
}

export interface ColumnInfo {
  name: string
  type: string
  nullable: boolean
  key?: string
}

export interface ColumnResponse {
  columns: ColumnInfo[]
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const resp = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({ error: resp.statusText }))
    throw new Error(err.error || resp.statusText)
  }
  return resp.json()
}

export function useApi() {
  const getStatus = () => apiFetch<StatusResponse>('/api/status')

  const connect = (dsn: string) =>
    apiFetch<StatusResponse>('/api/connect', {
      method: 'POST',
      body: JSON.stringify({ dsn }),
    })

  const getDatabases = () => apiFetch<DatabaseResponse>('/api/databases')

  const getTables = (db?: string) =>
    apiFetch<TableResponse>(`/api/tables${db ? `?db=${db}` : ''}`)

  const getColumns = (table: string, db?: string) =>
    apiFetch<ColumnResponse>(`/api/columns?table=${table}${db ? `&db=${db}` : ''}`)

  const executeQuery = (sql: string) =>
    apiFetch<QueryResponse>('/api/query', {
      method: 'POST',
      body: JSON.stringify({ sql }),
    })

  return { getStatus, connect, getDatabases, getTables, getColumns, executeQuery }
}
