import { useState } from 'react'
import { startServer, stopServer } from '../api/server'
import type { ServerStatus, Uptime } from '../types'

interface Props {
    status: ServerStatus | null
    loading: boolean
    onRefresh: () => void
    uptime: Uptime | null
}

export function ServerCard({ status, loading, onRefresh, uptime }: Props) {
    const [acting, setActing] = useState(false)

    const handleStart = async () => {
        setActing(true)
        try {
            await startServer()
            setTimeout(onRefresh, 2000)
        } catch (err) {
            console.error(err)
        } finally {
            setActing(false)
        }
    }

    const handleStop = async () => {
        setActing(true)
        try {
            await stopServer()
            setTimeout(onRefresh, 2000)
        } catch (err) {
            console.error(err)
        } finally {
            setActing(false)
        }
    }

    if (loading) return <div style={{ color: '#a9a39a', padding: '18px' }}>Chargement...</div>

    const online = status?.online ?? false

    return (
        <section className="panel servercard">
            <div className="icon-box">
                <canvas id="serverIcon" width="52" height="52" />
            </div>

            <div className="ident">
                <div className="status-row">
                    <span className="status-dot" />
                    <span id="statusBadge">{online ? 'En ligne' : 'Hors ligne'}</span>
                </div>
                <div id="uptime" className="tab">
                    {uptime?.minecraft_uptime ?? '--'}
                </div>
                <div id="uptimeLabel">
                    {online ? 'Minecraft actif' : 'Serveur arrêté'}
                </div>
                <div style={{ fontSize: '14px', color: 'var(--sub)', marginTop: '6px' }}>
                    VPS : {uptime?.vps_uptime ?? '--'}
                </div>
            </div>

            <div className="players-mid">
                <div className="tab">
                    <span id="playerCount">{status?.players ?? 0}</span>
                    <span className="max">/{status?.maxPlayers ?? 20}</span>
                </div>
                <div className="cap">joueurs</div>
            </div>

            <div className="actions">
                <button
                    className="mc-btn btn-start"
                    onClick={handleStart}
                    disabled={acting || online}
                >
                    ▶ Démarrer
                </button>
                <button
                    className="mc-btn btn-stop"
                    onClick={handleStop}
                    disabled={acting || !online}
                >
                    ■ Arrêter
                </button>
            </div>
        </section>
    )
}