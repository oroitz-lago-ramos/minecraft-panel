import type { ServerStatus, Player, SystemStats, Uptime } from '../types'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

const authHeaders = () => ({
  'Content-Type': 'application/json',
  Authorization: `Bearer ${localStorage.getItem('token') ?? ''}`,
})

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
  const res = await fetch(`${BASE_URL}/api/server/status`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Erreur status')
  return res.json()
}

export const fetchPlayers = async (): Promise<Player[]> => {
  const res = await fetch(`${BASE_URL}/api/server/players`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Erreur players')
  return res.json()
}

export const fetchStats = async (): Promise<SystemStats> => {
  const res = await fetch(`${BASE_URL}/api/stats`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Erreur stats')
  return res.json()
}

export const fetchUptime = async (): Promise<Uptime> => {
  const res = await fetch(`${BASE_URL}/api/server/uptime`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Erreur uptime')
  return res.json()
}

export const startServer = async (): Promise<void> => {
  const res = await fetch(`${BASE_URL}/api/server/start`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error('Erreur start')
}

export const stopServer = async (): Promise<void> => {
  const res = await fetch(`${BASE_URL}/api/server/stop`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error('Erreur stop')
}

export const sendCommand = async (command: string): Promise<string> => {
  const res = await fetch(`${BASE_URL}/api/server/command`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ command }),
  })
  if (!res.ok) throw new Error('Erreur commande')
  const data = await res.json()
  return data.response
}