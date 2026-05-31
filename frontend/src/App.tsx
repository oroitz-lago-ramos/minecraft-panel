import { useState, useEffect } from 'react'
import { fetchStatus, fetchPlayers, fetchStats, fetchUptime, login } from './api/server'
import { ServerCard } from './components/ServerCard'
import { StatsBar } from './components/StatsBar'
import { PlayerList } from './components/PlayerList'
import { Console } from './components/Console'
import type { ServerStatus, Player, SystemStats, Uptime } from './types'

export default function App() {
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'))
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loginError, setLoginError] = useState('')

  const [status, setStatus] = useState<ServerStatus | null>(null)
  const [players, setPlayers] = useState<Player[]>([])
  const [stats, setStats] = useState<SystemStats | null>(null)
  const [uptime, setUptime] = useState<Uptime | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchAll = async () => {
    try {
      const [s, p, st, up] = await Promise.all([
        fetchStatus(),
        fetchPlayers(),
        fetchStats(),
        fetchUptime(),
      ])
      setStatus(s)
      setPlayers(p)
      setStats(st)
      setUptime(up)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (!token) return
    fetchAll()
    const timer = setInterval(fetchAll, 5000)
    return () => clearInterval(timer)
  }, [token])

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const t = await login(username, password)
      localStorage.setItem('token', t)
      setToken(t)
      setLoginError('')
    } catch {
      setLoginError('Identifiants incorrects')
    }
  }

  const handleLogout = () => {
    localStorage.removeItem('token')
    setToken(null)
  }

  if (!token) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh' }}>
        <div className="panel" style={{ padding: '32px', minWidth: '320px' }}>
          <h1 className="px" style={{ fontSize: '14px', marginBottom: '24px', textAlign: 'center' }}>
            Minecraft Panel
          </h1>
          <form onSubmit={handleLogin} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
            <input
              type="text"
              placeholder="Username"
              value={username}
              onChange={e => setUsername(e.target.value)}
              style={{ background: '#1c1c1c', border: 'none', color: '#fff', padding: '10px', fontFamily: 'VT323, monospace', fontSize: '20px' }}
            />
            <input
              type="password"
              placeholder="Password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              style={{ background: '#1c1c1c', border: 'none', color: '#fff', padding: '10px', fontFamily: 'VT323, monospace', fontSize: '20px' }}
            />
            {loginError && <p style={{ color: '#d24b3e', margin: 0 }}>{loginError}</p>}
            <button type="submit" className="mc-btn btn-start">
              ▶ Connexion
            </button>
          </form>
        </div>
      </div>
    )
  }

  return (
    <div className="wrap">
      <header className="header">
        <div className="brand">
          <div className="logo">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round">
              <rect x="3" y="4" width="18" height="6" />
              <rect x="3" y="14" width="18" height="6" />
            </svg>
          </div>
          <div>
            <h1 className="px">mc.oroitzlagoramos.com</h1>
            <p>Paper 1.21.4 · eu-west</p>
          </div>
        </div>
        <button className="mc-btn" onClick={handleLogout} style={{ fontSize: '16px' }}>
          Déconnexion
        </button>
      </header>

      <ServerCard status={status} loading={loading} onRefresh={fetchAll} uptime={uptime} />
      <StatsBar stats={stats} />

      <section className="main">
        <PlayerList players={players} />
        <Console />
      </section>
    </div>
  )
}