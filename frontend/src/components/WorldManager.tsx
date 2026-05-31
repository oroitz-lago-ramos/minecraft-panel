import { useState, useEffect } from 'react'
import type { World } from '../types'
import { fetchWorlds, switchWorld, uploadWorld, deleteWorld, backupWorld } from '../api/server'

interface Props {
    onSwitch: () => void
}

export function WorldManager({ onSwitch }: Props) {
    const [worlds, setWorlds] = useState<World[]>([])
    const [loading, setLoading] = useState(true)
    const [switching, setSwitching] = useState<string | null>(null)
    const [uploading, setUploading] = useState(false)
    const [deleting, setDeleting] = useState<string | null>(null)
    const [backing, setBacking] = useState<string | null>(null)
    const [message, setMessage] = useState<{ text: string; ok: boolean } | null>(null)

    const showMessage = (text: string, ok: boolean) => {
        setMessage({ text, ok })
        setTimeout(() => setMessage(null), 5000)
    }

    const load = async () => {
        try {
            const data = await fetchWorlds()
            setWorlds(data)
        } catch (err) {
            console.error(err)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => { load() }, [])

    const handleSwitch = async (name: string) => {
        setSwitching(name)
        try {
            await switchWorld(name)
            showMessage(`✅ Map "${name}" activée — serveur en cours de redémarrage...`, true)
            await load()
            onSwitch()
        } catch {
            showMessage(`❌ Erreur lors du switch`, false)
        } finally {
            setSwitching(null)
        }
    }

    const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0]
        if (!file) return
        if (file.size > 500 * 1024 * 1024) {
            showMessage('❌ Fichier trop lourd (max 500 Mo) — utilisez SFTP pour les grosses maps', false)
            e.target.value = ''
            return
        }
        setUploading(true)
        try {
            await uploadWorld(file)
            showMessage(`✅ Map "${file.name}" uploadée !`, true)
            await load()
        } catch {
            showMessage(`❌ Erreur lors de l'upload`, false)
        } finally {
            setUploading(false)
            e.target.value = ''
        }
    }

    const handleDelete = async (name: string) => {
        if (!confirm(`Supprimer la map "${name}" ? Cette action est irréversible.`)) return
        setDeleting(name)
        try {
            await deleteWorld(name)
            showMessage(`✅ Map "${name}" supprimée`, true)
            await load()
        } catch {
            showMessage(`❌ Erreur lors de la suppression`, false)
        } finally {
            setDeleting(null)
        }
    }

    const handleBackup = async (name: string) => {
        setBacking(name)
        try {
            await backupWorld(name)
            showMessage(`✅ Backup de "${name}" téléchargé`, true)
        } catch {
            showMessage(`❌ Erreur lors du backup`, false)
        } finally {
            setBacking(null)
        }
    }

    return (
        <div className="panel" style={{ padding: '16px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                <h2 style={{ margin: 0, fontSize: '14px' }}>🗺️ MAPS</h2>
                <label style={{ cursor: uploading ? 'not-allowed' : 'pointer', opacity: uploading ? 0.5 : 1 }}>
                    <input
                        type="file"
                        accept=".zip"
                        onChange={handleUpload}
                        disabled={uploading}
                        style={{ display: 'none' }}
                    />
                    <span className="mc-btn" style={{ fontSize: '14px', padding: '6px 12px' }}>
                        {uploading ? '⏳ Upload...' : '📦 Importer ZIP'}
                    </span>
                </label>
            </div>

            {message && (
                <div style={{
                    padding: '8px 12px',
                    marginBottom: '12px',
                    background: message.ok ? '#1a2e1a' : '#2e1a1a',
                    color: message.ok ? '#6cbf3a' : '#d24b3e',
                    fontFamily: 'VT323, monospace',
                    fontSize: '16px',
                }}>
                    {message.text}
                </div>
            )}

            {loading ? (
                <div style={{ color: '#a9a39a' }}>Chargement...</div>
            ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                    {worlds.map(world => (
                        <div key={world.name} style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                            padding: '10px 14px',
                            background: world.active ? '#1a2e1a' : '#1c1c1c',
                            border: world.active ? '2px solid #6cbf3a' : '2px solid #333',
                        }}>
                            <div>
                                <span style={{
                                    fontFamily: 'VT323, monospace',
                                    fontSize: '20px',
                                    color: world.active ? '#6cbf3a' : '#cfcfcf',
                                }}>
                                    {world.active ? '▶ ' : '  '}{world.name}
                                </span>
                                {world.active && (
                                    <span style={{ marginLeft: '8px', fontSize: '12px', color: '#6cbf3a' }}>
                                        ACTIVE
                                    </span>
                                )}
                            </div>
                            <div style={{ display: 'flex', gap: '6px' }}>
                                {!world.active && (
                                    <button
                                        className="mc-btn btn-start"
                                        style={{ fontSize: '14px', padding: '4px 10px' }}
                                        onClick={() => handleSwitch(world.name)}
                                        disabled={switching !== null}
                                    >
                                        {switching === world.name ? '⏳' : 'Activer'}
                                    </button>
                                )}
                                <button
                                    className="mc-btn"
                                    style={{ fontSize: '14px', padding: '4px 10px', background: '#2a2a6a' }}
                                    onClick={() => handleBackup(world.name)}
                                    disabled={backing === world.name}
                                >
                                    {backing === world.name ? '⏳' : '💾 Backup'}
                                </button>
                                {!world.active && (
                                    <button
                                        className="mc-btn btn-stop"
                                        style={{ fontSize: '14px', padding: '4px 10px' }}
                                        onClick={() => handleDelete(world.name)}
                                        disabled={deleting === world.name}
                                    >
                                        {deleting === world.name ? '⏳' : '🗑️'}
                                    </button>
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    )
}