export interface ServerStatus {
  online: boolean
  players: number
  maxPlayers: number
  uptime: number
}

export interface Player {
  name: string
}

export interface SystemStats {
  cpuPercent: number
  ramUsed: number
  ramTotal: number
  diskUsed: number
  diskTotal: number
  uptime: number
}

export interface Uptime {
  vps_uptime: string
  minecraft_uptime: string
}

export interface ConsoleMessage {
  timestamp: string
  level: 'INFO' | 'WARN' | 'ERROR'
  content: string
}