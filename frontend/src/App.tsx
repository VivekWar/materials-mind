import React, { useEffect, useState } from 'react'
import { Circle, LogIn, Menu, Plus, Sparkles, X } from 'lucide-react'
import './styles/index.css'
import './styles/chat.css'
import './styles/chat-history.css'
import './styles/ui.css'
import { ChatPanel } from './components/ChatPanel'
import { ChatHistory } from './components/ChatHistory'
import { HomePage } from './components/HomePage'
import { Button } from './components/ui'
import { useChat } from './hooks/useChat'
import {
  AuthUser,
  getMe,
  googleLogin,
  pingStatus,
} from './api/client'

declare global {
  interface Window {
    google?: {
      accounts?: {
        id?: {
          initialize: (options: { client_id: string; callback: (response: { credential?: string }) => void }) => void
          renderButton: (element: HTMLElement, options: Record<string, unknown>) => void
        }
      }
    }
  }
}

type ApiStatus = 'checking' | 'online' | 'offline'

const CHAT_ROUTE = '/chat'

const getPathname = () => window.location.pathname

const navigateTo = (path: string) => {
  if (window.location.pathname !== path) {
    window.history.pushState({}, '', path)
    window.dispatchEvent(new PopStateEvent('popstate'))
  }
}

const AuthScreen: React.FC = () => {
  const loginUrl = `${import.meta.env.VITE_API_URL || 'http://localhost:8080/api'}/auth/google/login`

  return (
    <div className="home-page">
      <main className="home-main">
        <section className="home-hero">
          <div className="home-hero-copy">
            <div className="home-kicker">Secure workspace</div>
            <h1>Materials Mind</h1>
            <p>Sign in to keep your material decisions, reports, and follow-up context attached to your account.</p>
            <div className="home-hero-actions">
              <a href={loginUrl} className="home-cta" style={{ textDecoration: 'none' }}>
                <LogIn size={16} /> Login with Google
              </a>
            </div>
          </div>
        </section>
      </main>
    </div>
  )
}

const ChatWorkspace: React.FC = () => {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [authChecked, setAuthChecked] = useState(false)
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)
  const [isMobileViewport, setIsMobileViewport] = useState(false)
  const [apiStatus, setApiStatus] = useState<ApiStatus>('checking')

  const {
    sessions,
    activeSessionId,
    activeSession,
    loading,
    sendMessage,
    selectSession,
    createNewSession,
  } = useChat(!!user)

  useEffect(() => {
    getMe()
      .then((nextUser) => {
        setUser(nextUser)
      })
      .catch(() => {
        setUser(null)
      })
      .finally(() => setAuthChecked(true))
  }, [])

  useEffect(() => {
    let mounted = true
    const checkApiHealth = async () => {
      const ok = await pingStatus()
      if (mounted) {
        setApiStatus(ok ? 'online' : 'offline')
      }
    }

    checkApiHealth()
    const timer = window.setInterval(checkApiHealth, 45000)
    return () => {
      mounted = false
      window.clearInterval(timer)
    }
  }, [])

  useEffect(() => {
    const syncSidebarForViewport = () => {
      const isMobile = window.innerWidth <= 980
      setIsMobileViewport(isMobile)
      setIsSidebarOpen(!isMobile)
    }

    syncSidebarForViewport()
    window.addEventListener('resize', syncSidebarForViewport)
    return () => window.removeEventListener('resize', syncSidebarForViewport)
  }, [])

  const handleSelectSession = (id: string) => {
    selectSession(id)
    if (window.innerWidth <= 980) setIsSidebarOpen(false)
  }

  const handleCreateSession = () => {
    void createNewSession()
    if (window.innerWidth <= 980) setIsSidebarOpen(false)
  }

  if (!authChecked) {
    return <div className="home-page" />
  }

  if (!user) {
    return <AuthScreen />
  }

  return (
    <div className={`app-shell ${!isMobileViewport && !isSidebarOpen ? 'sidebar-collapsed' : ''}`}>
      <aside className={`app-sidebar ${isSidebarOpen ? 'is-open' : ''}`}>
        <ChatHistory
          sessions={sessions}
          activeSessionId={activeSessionId}
          onSelectSession={handleSelectSession}
          onCreateNewSession={handleCreateSession}
        />
      </aside>

      <main className="app-main">
        <nav className="top-nav">
          <div className="nav-brand">
            <Button
              className="icon-button sidebar-toggle"
              variant="ghost"
              size="icon"
              onClick={() => setIsSidebarOpen((current) => !current)}
              aria-label={isSidebarOpen ? 'Close session history' : 'Open session history'}
              title={isSidebarOpen ? 'Close history' : 'Open history'}
            >
              {isSidebarOpen ? <X size={16} /> : <Menu size={16} />}
            </Button>
            <button type="button" className="home-link-brand" onClick={() => navigateTo('/')}>
              <div className="brand-mark">
                <Sparkles size={18} />
              </div>
              <div>
                <div className="brand-title">Met-Quest Material Assistant</div>
                <div className="brand-subtitle">Tell your use-case, constraints, and manufacturing process.</div>
              </div>
            </button>
          </div>

          <div className="top-nav-actions">
            <div className={`nav-status nav-status-${apiStatus} nav-status-inline`}>
              <Circle size={9} />
              {apiStatus === 'checking' ? 'Checking API' : apiStatus === 'online' ? 'API Ready' : 'API Unavailable'}
            </div>
            <Button type="button" className="btn-new-chat btn-ghost" variant="ghost" onClick={() => navigateTo('/')}>
              Home
            </Button>
            <Button className="btn-new-chat" onClick={handleCreateSession} title="Start a new chat">
              <Plus size={14} /> New Chat
            </Button>
          </div>
        </nav>

        <section className="app-content">
          {activeSession && (
            <div className="chat-column chat-column--full">
              <ChatPanel
                messages={activeSession.messages}
                onSendMessage={sendMessage}
                loading={loading}
              />
            </div>
          )}
        </section>
      </main>

      {isMobileViewport && isSidebarOpen && (
        <button
          className="sidebar-backdrop"
          onClick={() => setIsSidebarOpen(false)}
          aria-label="Close session history panel"
        />
      )}
    </div>
  )
}

const App: React.FC = () => {
  const [pathname, setPathname] = useState(getPathname())

  useEffect(() => {
    const handlePopState = () => setPathname(getPathname())
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  if (pathname === CHAT_ROUTE) {
    return <ChatWorkspace />
  }

  return <HomePage onStartChat={() => navigateTo(CHAT_ROUTE)} />
}

export default App
