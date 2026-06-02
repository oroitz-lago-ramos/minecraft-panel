import type { ServerStatus, Player, SystemStats, Uptime, World } from '../types'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

const authHeaders = () => ({
  'Content-Type': 'application/json',
  Authorization: `Bearer ${localStorage.getItem('token') ?? ''}`,
})

const apiFetch = async (url: string, options?: RequestInit): Promise<Response> => {
  const res = await fetch(url, options)
  if (res.status === 401) {
    localStorage.removeItem('token')
    window.location.href = '/'
  }
  return res
}

export const login = async (username: string, password: string): Promise<string> => {
  const res = await fetch(`${BASE_URL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  })
  if (!res.ok) throw new Error('Login invalide')
  const data = await res.json()
  return data.token
}

export const fetchStatus = async (): Promise<ServerStatus> => {
  const res = await apiFetch(`${BASE_URL}/api/server/status`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Erreur status')
  return res.json()
}

export const fetchPlayers = async (): Promise<Player[]> => {
  const res = await apiFetch(`${BASE_URL}/api/server/players`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Erreur players')
  return res.json()
}

export const fetchStats = async (): Promise<SystemStats> => {
  const res = await apiFetch(`${BASE_URL}/api/stats`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Erreur stats')
  return res.json()
}

export const fetchUptime = async (): Promise<Uptime> => {
  const res = await apiFetch(`${BASE_URL}/api/server/uptime`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Erreur uptime')
  return res.json()
}

export const startServer = async (): Promise<void> => {
  const res = await apiFetch(`${BASE_URL}/api/server/start`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error('Erreur start')
}

export const stopServer = async (): Promise<void> => {
  const res = await apiFetch(`${BASE_URL}/api/server/stop`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error('Erreur stop')
}

export const sendCommand = async (command: string): Promise<string> => {
  const res = await apiFetch(`${BASE_URL}/api/server/command`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ command }),
  })
  if (!res.ok) throw new Error('Erreur commande')
  const data = await res.json()
  return data.response
}

export const fetchWorlds = async (): Promise<World[]> => {
  const res = await apiFetch(`${BASE_URL}/api/server/worlds`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Erreur worlds')
  return res.json()
}

export const switchWorld = async (name: string): Promise<void> => {
  const res = await apiFetch(`${BASE_URL}/api/server/worlds/switch`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ name }),
  })
  if (!res.ok) throw new Error('Erreur switch')
}

export const uploadWorld = async (file: File): Promise<void> => {
  const form = new FormData()
  form.append('world', file)
  const res = await apiFetch(`${BASE_URL}/api/server/worlds/upload`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${localStorage.getItem('token') ?? ''}` },
    body: form,
  })
  if (!res.ok) throw new Error('Erreur upload')
}

export const deleteWorld = async (name: string): Promise<void> => {
  const res = await apiFetch(`${BASE_URL}/api/server/worlds/${name}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error('Erreur delete')
}

export const backupWorld = async (name: string): Promise<void> => {
  const res = await apiFetch(`${BASE_URL}/api/server/worlds/${name}/backup`, {
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error('Erreur backup')
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${name}.zip`
  a.click()
  URL.revokeObjectURL(url)
}