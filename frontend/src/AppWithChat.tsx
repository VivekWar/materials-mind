import React, { useState, useCallback, useEffect } from 'react'
import './styles/index.css'
import './styles/chat.css'
import './styles/chat-history.css'
import { QueryInput } from './components/QueryInput'
import { ReportCard } from './components/ReportCard'
import { ChatPanel } from './components/ChatPanel'
import { ChatHistory } from './components/ChatHistory'
import { RecommendResponse, ping, recommend, Constraint } from './api/client'
import { useChatStorage, ChatMessage, Constraint as StorageConstraint } from './hooks/useChatStorage'

const App: React.FC = () => {
  const [result, setResult] = useState<RecommendResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [showChat, setShowChat] = useState(true)
  const [currentConstraints, setCurrentConstraints] = useState<StorageConstraint[]>([])

  // Chat storage hook
  const chatStorage = useChatStorage()

  // Cold start mitigation
  useEffect(() => {
    ping()
  }, [])

  // Create initial session if none exists
  useEffect(() => {
    if (chatStorage.isLoaded && chatStorage.sessions.length === 0) {
      chatStorage.createSession()
    }
  }, [chatStorage.isLoaded, chatStorage.sessions.length, chatStorage])

  const activeSession = chatStorage.getActiveSession()

  const handleResult = useCallback((res: RecommendResponse) => {
    setResult(res)
    
    // Add assistant message to chat
    if (activeSession) {
      const assistantMsg: ChatMessage = {
        id: Date.now().toString(),
        type: 'assistant',
        response: res,
        timestamp: Date.now(),
        tokens: res.tokens_used,
      }
      chatStorage.addMessage(activeSession.id, assistantMsg)
    }

    // Smooth scroll to results
    setTimeout(() => {
      document.getElementById('report-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }, 100)
  }, [activeSession, chatStorage])

  const handleQuery = useCallback(async (query: string, domain: string) => {
    if (!activeSession) return

    // Add user message to chat
    const userMsg: ChatMessage = {
      id: Date.now().toString(),
      type: 'user',
      originalQuery: query,
      constraints: currentConstraints.length > 0 ? [...currentConstraints] : undefined,
      query: query,
      timestamp: Date.now(),
    }
    chatStorage.addMessage(activeSession.id, userMsg)

    // Call recommend with constraints
    setLoading(true)
    try {
      const constraints: Constraint[] = currentConstraints.map(c => ({
        key: c.key,
        operator: c.operator,
        value: c.value,
      }))

      const result = await recommend(query, domain, constraints)
      handleResult(result)
    } finally {
      setLoading(false)
    }
  }, [activeSession, chatStorage, currentConstraints, handleResult])

  const handleAddConstraint = useCallback((constraint: StorageConstraint) => {
    setCurrentConstraints(prev => [...prev, constraint])
  }, [])

  const handleRemoveConstraint = useCallback((constraintId: string) => {
    setCurrentConstraints(prev => prev.filter(c => c.id !== constraintId))
  }, [])

  const handleReQuery = useCallback((constraints: StorageConstraint[]) => {
    if (activeSession && activeSession.messages.length > 0) {
      // Find the last user message
      const lastUserMsg = [...activeSession.messages].reverse().find(m => m.type === 'user')
      if (lastUserMsg && lastUserMsg.originalQuery) {
        // Extract domain from previous query or use default
        const domain = 'Overall (Top 1000)'
        handleQuery(lastUserMsg.originalQuery, domain)
      }
    }
  }, [activeSession, handleQuery])

  return (
    <div style={{ display: 'flex', minHeight: '100vh', background: 'var(--color-bg)' }}>
      {/* ── Chat History Sidebar ─────────────────────────────────────────── */}
      {showChat && (
        <div style={{
          width: 300,
          borderRight: '1px solid var(--color-border)',
          padding: 12,
          overflow: 'hidden',
          background: 'rgba(15,20,25,0.5)',
          display: 'flex',
          flexDirection: 'column',
        }}>
          <ChatHistory
            sessions={chatStorage.sessions}
            activeSessionId={chatStorage.activeSessionId}
            onSelectSession={chatStorage.setActiveSessionId}
            onDeleteSession={chatStorage.deleteSession}
            onCreateNewSession={chatStorage.createSession}
            onRenameSession={chatStorage.renameSession}
            onClearAll={chatStorage.clearAllSessions}
          />
        </div>
      )}

      {/* ── Main Content ────────────────────────────────────────────────────────── */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* Navigation */}
        <nav className="top-nav" style={{
          position: 'sticky', top: 0, zIndex: 100,
          background: 'rgba(8,12,24,0.9)',
          backdropFilter: 'blur(16px)',
          borderBottom: '1px solid var(--color-border)',
          padding: '0 24px',
          display: 'flex',
          alignItems: 'center',
          height: 64,
        }}>
          <div className="nav-brand" style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <button
              onClick={() => setShowChat(!showChat)}
              style={{
                background: 'rgba(0,212,255,0.1)',
                border: '1px solid rgba(0,212,255,0.2)',
                color: 'var(--color-primary)',
                borderRadius: 8,
                width: 40,
                height: 40,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                cursor: 'pointer',
                fontSize: '1.1rem',
              }}
              title={showChat ? 'Hide chat history' : 'Show chat history'}
            >
              {showChat ? '◀' : '▶'}
            </button>
            <div style={{
              width: 36, height: 36, borderRadius: 9,
              background: 'linear-gradient(135deg, #00d4ff, #0080ff)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: '1.1rem', fontWeight: 800,
              boxShadow: '0 4px 16px rgba(0,212,255,0.35)',
            }}>⚛</div>
            <div>
              <div style={{ fontWeight: 800, fontSize: '0.95rem', letterSpacing: '-0.02em' }}>
                Smart Alloy Selector
              </div>
              <div className="text-xs text-dim" style={{ fontWeight: 500 }}>MET-QUEST '26</div>
            </div>
          </div>

          <div style={{ flex: 1 }} />

          {/* DB badge */}
          <div className="nav-badge" style={{
            display: 'flex', alignItems: 'center', gap: 8,
            padding: '6px 14px',
            background: 'rgba(0,255,159,0.08)',
            border: '1px solid rgba(0,255,159,0.2)',
            borderRadius: 20,
            fontSize: '0.75rem',
          }}>
            <span style={{
              width: 7, height: 7, borderRadius: '50%',
              background: '#00ff9f',
              boxShadow: '0 0 6px #00ff9f',
              display: 'inline-block',
            }} />
            <span style={{ color: '#00ff9f', fontWeight: 600 }}>8,759 Materials Loaded</span>
          </div>
        </nav>

        {/* Hero Section */}
        {!result && (
          <div style={{
            textAlign: 'center', padding: '72px 24px 48px',
            background: 'radial-gradient(ellipse at 50% 0%, rgba(0,212,255,0.07) 0%, transparent 65%)',
          }}>
            <div
              className="text-xs font-mono"
              style={{
                color: 'var(--color-primary)', letterSpacing: '0.15em',
                textTransform: 'uppercase', marginBottom: 20,
                display: 'inline-flex', alignItems: 'center', gap: 8,
                padding: '5px 16px',
                background: 'rgba(0,212,255,0.08)',
                border: '1px solid rgba(0,212,255,0.2)',
                borderRadius: 20,
              }}
            >
              <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'var(--color-primary)', display: 'inline-block', animation: 'pulse-glow 2s ease-in-out infinite' }} />
              Powered by Gemini + Local PostgreSQL RAG
            </div>

            <h1 style={{ maxWidth: 640, margin: '0 auto 16px' }}>
              <span className="gradient-text">AI-Powered</span> Material Selection
            </h1>
            <p style={{ maxWidth: 540, margin: '0 auto 40px', fontSize: '1.05rem', color: 'var(--color-text-muted)' }}>
              Describe your engineering challenge. Our AI extracts your requirements, queries 8,759+ materials,
              and delivers a <strong style={{ color: 'var(--color-text)' }}>Virtual Scientist report</strong> with deep technical analysis.
            </p>

            {/* Feature pills */}
            <div style={{ display: 'flex', justifyContent: 'center', gap: 12, flexWrap: 'wrap', marginBottom: 48 }}>
              {[
                ['🧠', 'Gemini Intent Extraction'],
                ['💾', 'Chat Storage & History'],
                ['🎯', 'Constraint Refinement'],
                ['📋', 'Virtual Scientist Report'],
              ].map(([icon, label]) => (
                <div key={label as string} style={{
                  display: 'flex', alignItems: 'center', gap: 8,
                  padding: '8px 16px',
                  background: 'rgba(255,255,255,0.03)',
                  border: '1px solid var(--color-border)',
                  borderRadius: 20,
                  fontSize: '0.8125rem',
                  color: 'var(--color-text-muted)',
                }}>
                  <span>{icon}</span>{label}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Main Content */}
        <div className="container" style={{ paddingBottom: 64, flex: 1 }}>
          <div style={{ maxWidth: 1200, margin: '0 auto' }}>
            <div style={{ display: 'grid', gridTemplateColumns: 'minmax(600px, 1fr) minmax(300px, 400px)', gap: 24 }}>
              {/* Left: Query & Results */}
              <div>
                <div style={{ marginBottom: 24 }}>
                  <QueryInputWithConstraints
                    onQuery={handleQuery}
                    onLoading={setLoading}
                    constraints={currentConstraints}
                  />
                </div>

                {/* Loading skeleton */}
                {loading && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                    {[160, 300, 80].map((h, i) => (
                      <div key={i} className="skeleton" style={{ height: h, borderRadius: 18 }} />
                    ))}
                  </div>
                )}

                {/* Results */}
                {result && !loading && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
                    <ReportCard result={result} />
                    <div style={{ textAlign: 'center' }}>
                      <button
                        onClick={() => {
                          setResult(null)
                          setCurrentConstraints([])
                        }}
                        style={{
                          padding: '10px 20px',
                          background: 'rgba(0,212,255,0.1)',
                          border: '1px solid rgba(0,212,255,0.2)',
                          color: 'var(--color-primary)',
                          borderRadius: 8,
                          cursor: 'pointer',
                          fontSize: '0.85rem',
                          fontWeight: 600,
                        }}
                      >
                        ↺ New Query
                      </button>
                    </div>
                  </div>
                )}
              </div>

              {/* Right: Chat Panel */}
              {activeSession && (
                <ChatPanel
                  messages={activeSession.messages}
                  onAddConstraint={handleAddConstraint}
                  onRemoveConstraint={handleRemoveConstraint}
                  onReQuery={handleReQuery}
                  currentConstraints={currentConstraints}
                  loading={loading}
                />
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

// Wrapper component that handles query with domain
interface QueryInputWithConstraintsProps {
  onQuery: (query: string, domain: string) => void
  onLoading: (loading: boolean) => void
  constraints: StorageConstraint[]
}

const QueryInputWithConstraints: React.FC<QueryInputWithConstraintsProps> = ({ onQuery, onLoading, constraints }) => {
  const [domain, setDomain] = useState('Overall (Top 1000)')

  const handleResult = useCallback((res: RecommendResponse) => {
    // This is called by QueryInput but we intercept and call onQuery instead
  }, [])

  return (
    <QueryInput
      onResult={handleResult}
      onLoading={onLoading}
      onQuery={(query, domain) => {
        onQuery(query, domain)
      }}
    />
  )
}

export default App
