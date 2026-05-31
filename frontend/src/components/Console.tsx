import { useRef, useEffect, useState } from 'react'
import { useWebSocket } from '../hooks/useWebSocket'

type Filter = 'ALL' | 'INFO' | 'WARN' | 'ERROR'

export function Console() {
    const { messages, connected, sendMessage } = useWebSocket()
    const [input, setInput] = useState('')
    const [autoScroll, setAutoScroll] = useState(true)
    const [filter, setFilter] = useState<Filter>('ALL')
    const consoleRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        if (autoScroll && consoleRef.current) {
            consoleRef.current.scrollTop = consoleRef.current.scrollHeight
        }
    }, [messages, autoScroll])

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault()
        if (!input.trim()) return
        sendMessage(input.trim())
        setInput('')
    }

    const getLevel = (line: string): Filter => {
        if (line.includes('ERROR')) return 'ERROR'
        if (line.includes('WARN')) return 'WARN'
        return 'INFO'
    }

    const clearConsole = () => {
        if (consoleRef.current) consoleRef.current.innerHTML = ''
    }

    const filteredMessages = messages.filter((msg: string) =>
        filter === 'ALL' ? true : getLevel(msg) === filter
    )

    const filterBtn = (label: Filter, color: string) => (
        <button
            onClick={() => setFilter(label)}
            style={{
                fontFamily: '"VT323", monospace',
                fontSize: '16px',
                padding: '3px 10px 1px',
                border: 'none',
                cursor: 'pointer',
                color: filter === label ? '#000' : color,
                background: filter === label ? color : 'transparent',
                boxShadow: filter === label
                    ? 'inset 2px 2px 0 rgba(255,255,255,0.3), inset -2px -2px 0 rgba(0,0,0,0.4)'
                    : 'none',
            }}
        >
            {label}
        </button>
    )

    return (
        <div className="panel console-panel">
            <div className="console-head">
                <h2>Console</h2>
                <div className="opts">
                    {/* Filtres */}
                    <div style={{ display: 'flex', gap: '4px', alignItems: 'center' }}>
                        {filterBtn('ALL', '#a9a39a')}
                        {filterBtn('INFO', '#cfcfcf')}
                        {filterBtn('WARN', '#f0a821')}
                        {filterBtn('ERROR', '#d24b3e')}
                    </div>
                    <label>
                        <input
                            type="checkbox"
                            checked={autoScroll}
                            onChange={e => setAutoScroll(e.target.checked)}
                        />
                        auto-défilement
                    </label>
                    <button id="clearConsole" onClick={clearConsole}>effacer</button>
                    <span style={{ color: connected ? '#6cbf3a' : '#d24b3e', fontSize: '16px' }}>
                        {connected ? '● connecté' : '● déconnecté'}
                    </span>
                </div>
            </div>

            <div id="console" ref={consoleRef}>
                {filteredMessages.map((msg: string, i: number) => (
                    <div key={i} className="cl-line" data-level={getLevel(msg)}>
                        <span className="cl-msg">{msg}</span>
                    </div>
                ))}
            </div>

            <form id="rconForm" className="rcon-form" onSubmit={handleSubmit}>
                <span className="pr">&gt;</span>
                <input
                    id="rconInput"
                    type="text"
                    value={input}
                    onChange={e => setInput(e.target.value)}
                    disabled={!connected}
                    placeholder="list · say bonjour · time set day"
                    autoComplete="off"
                    spellCheck={false}
                />
                <button type="submit" className="send" disabled={!connected}>
                    Envoyer
                </button>
            </form>
        </div>
    )
}