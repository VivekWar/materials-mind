import React, { useEffect, useState } from 'react'
import { Circle, Menu, Plus, Sparkles, X, LogOut, User } from 'lucide-react'
import { Button } from '../components/ui/button'
import { ChatPanel } from '../components/ChatPanel'
import { ChatHistory } from '../components/ChatHistory'
import { ProfileDialog } from '../components/ProfileDialog'
import { useChat } from '../hooks/useChat'
import { useAppStore } from '../store/useAppStore'
import { getMe, pingStatus, logout, setAuthToken } from '../api/client'
import AuthPage from './AuthPage'

const navigateTo = (path: string) => {
  if (window.location.pathname !== path) {
    window.history.pushState({}, '', path)
    window.dispatchEvent(new PopStateEvent('popstate'))
  }
}

const ChatPage: React.FC = () => {
  const user = useAppStore((state) => state.user)
  const setUser = useAppStore((state) => state.setUser)
  const apiStatus = useAppStore((state) => state.apiStatus)
  const setApiStatus = useAppStore((state) => state.setApiStatus)

  const [authChecked, setAuthChecked] = useState(false)
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)
  const [isMobileViewport, setIsMobileViewport] = useState(false)
  const [isProfileOpen, setIsProfileOpen] = useState(false)

  const {
    sessions,
    activeSessionId,
    activeSession,
    loading,
    sendMessage,
    stopGeneration,
    selectSession,
    createNewSession,
  } = useChat()

  // ── Auth check ─────────────────────────────────────────────────────────────
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const token = params.get('token')
    if (token) {
      setAuthToken(token)
      // Remove token from URL to keep it clean
      const newUrl = window.location.pathname + window.location.hash
      window.history.replaceState({}, '', newUrl)

      // If we are in a popup window (due to Cross-Origin-Opener-Policy disconnecting opener), close it.
      // The main window will pick up the token via the 'storage' event listener we added.
      if (window.opener !== undefined || window.name === 'Google Login') {
        window.close()
      }
    }

    getMe()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setAuthChecked(true))
  }, [setUser])

  // ── API health polling ─────────────────────────────────────────────────────
  useEffect(() => {
    let mounted = true
    const check = async () => {
      if (document.hidden) return
      const ok = await pingStatus()
      if (mounted) setApiStatus(ok ? 'online' : 'offline')
    }
    check()
    const timer = window.setInterval(check, 45_000)
    return () => {
      mounted = false
      window.clearInterval(timer)
    }
  }, [setApiStatus])

  // ── Responsive sidebar ────────────────────────────────────────────────────
  useEffect(() => {
    const syncViewport = () => {
      const isMobile = window.innerWidth <= 980
      setIsMobileViewport(isMobile)
      if (isMobile) {
        setIsSidebarOpen(false)
      } else {
        const stored = localStorage.getItem('isSidebarOpen')
        setIsSidebarOpen(stored !== 'false')
      }
    }
    syncViewport()
    window.addEventListener('resize', syncViewport)
    return () => window.removeEventListener('resize', syncViewport)
  }, [])

  const handleToggleSidebar = () => {
    setIsSidebarOpen((v) => {
      const next = !v
      localStorage.setItem('isSidebarOpen', String(next))
      return next
    })
  }

  const handleSelectSession = (id: string) => {
    selectSession(id)
    if (window.innerWidth <= 980) setIsSidebarOpen(false)
  }

  const handleCreateSession = () => {
    void createNewSession()
    if (window.innerWidth <= 980) setIsSidebarOpen(false)
  }

  const handleLogout = async () => {
    try {
      await logout()
      setUser(null)
      navigateTo('/')
    } catch (e) {
      console.error('Failed to logout', e)
    }
  }

  // ── Guards ─────────────────────────────────────────────────────────────────
  if (!authChecked) {
    return <div className="min-h-screen bg-background" aria-busy="true" />
  }
  if (!user) {
    return <AuthPage />
  }

  // ── API status pill config ─────────────────────────────────────────────────
  const statusConfig = {
    checking: { dot: 'bg-muted-foreground animate-pulse', text: 'Checking',    wrap: 'bg-muted text-muted-foreground' },
    online:   { dot: 'bg-green-500',                     text: 'API Ready',    wrap: 'bg-green-500/10 text-green-600 dark:text-green-400' },
    offline:  { dot: 'bg-red-500',                       text: 'Unavailable',  wrap: 'bg-red-500/10 text-red-500' },
  }
  const sc = statusConfig[apiStatus]

  return (
    <div className="flex h-screen bg-background text-foreground overflow-hidden font-sans selection:bg-zinc-200 dark:selection:bg-zinc-800">
      {/* ── Sidebar ─────────────────────────────────────────────────────── */}
      <aside
        aria-label="Chat sessions"
        className={`shrink-0 border-r border-border bg-muted/20 transition-all duration-300 ease-in-out overflow-hidden ${
          isSidebarOpen ? 'w-64' : 'w-0'
        } ${isMobileViewport && isSidebarOpen ? 'fixed inset-y-0 left-0 z-50 shadow-2xl bg-background' : 'relative'}`}
      >
        <ChatHistory
          sessions={sessions}
          activeSessionId={activeSessionId}
          onSelectSession={handleSelectSession}
          onCreateNewSession={handleCreateSession}
        />
      </aside>

      {/* ── Main ────────────────────────────────────────────────────────── */}
      <div className="flex-1 flex flex-col min-w-0 bg-background">
        {/* Nav bar */}
        <nav
          aria-label="Chat workspace navigation"
          className="flex items-center justify-between border-b border-border px-4 py-2.5 bg-background z-10 shrink-0"
        >
          <div className="flex items-center gap-3">
            <Button
              id="btn-toggle-sidebar"
              variant="ghost"
              size="icon"
              className="text-muted-foreground hover:text-foreground h-8 w-8"
              onClick={handleToggleSidebar}
              aria-label={isSidebarOpen ? 'Close session history' : 'Open session history'}
              aria-expanded={isSidebarOpen}
              aria-controls="chat-sidebar"
            >
              {isSidebarOpen ? <X size={14} aria-hidden="true" /> : <Menu size={14} aria-hidden="true" />}
            </Button>

            <button
              type="button"
              id="btn-home-nav"
              className="flex items-center gap-2.5 hover:opacity-80 transition-opacity"
              onClick={() => navigateTo('/')}
            >
              <div className="flex items-center justify-center w-7 h-7 rounded bg-foreground text-background">
                <Sparkles size={12} aria-hidden="true" />
              </div>
              <div className="text-left hidden sm:block">
                <div className="text-xs font-medium leading-none tracking-tight">Materials Mind</div>
              </div>
            </button>
          </div>

          <div className="flex items-center gap-2.5">
            {/* API status */}
            <div
              className={`flex items-center gap-1.5 text-[10px] font-mono uppercase tracking-widest px-2.5 py-1 rounded-sm ${sc.wrap}`}
              role="status"
              aria-live="polite"
              aria-label={`API status: ${sc.text}`}
            >
              <Circle
                size={6}
                className={`${sc.dot} rounded-full fill-current`}
                aria-hidden="true"
              />
              <span className="hidden sm:inline">{sc.text}</span>
            </div>

            <Button
              id="btn-new-chat"
              size="sm"
              variant="outline"
              onClick={handleCreateSession}
              title="Start a new chat"
              className="gap-1.5 h-8 border-border/80"
            >
              <Plus size={12} aria-hidden="true" /> New Chat
            </Button>

            <Button
              id="btn-profile"
              size="sm"
              variant="ghost"
              onClick={() => setIsProfileOpen(true)}
              title="User Profile & Limits"
              className="text-muted-foreground hover:text-foreground h-8 px-2"
            >
              <User size={14} aria-hidden="true" />
            </Button>

            <Button
              id="btn-logout"
              size="sm"
              variant="ghost"
              onClick={handleLogout}
              title="Sign Out"
              className="text-muted-foreground hover:text-foreground h-8 px-2"
            >
              <LogOut size={14} aria-hidden="true" />
            </Button>
          </div>
        </nav>

        {/* Chat panel area */}
        <main id="main-content" className="flex-1 overflow-hidden relative" role="main">
          {activeSession ? (
            <div className="absolute inset-0 w-full h-full bg-background">
              <ChatPanel
                messages={activeSession.messages}
                onSendMessage={sendMessage}
                onStopGeneration={stopGeneration}
                loading={loading}
              />
            </div>
          ) : (
            /* Empty panel — no session selected */
            <div className="flex flex-col items-center justify-center h-full gap-4 text-center px-4">
              <div className="w-12 h-12 rounded bg-muted flex items-center justify-center border border-border">
                <Sparkles size={20} className="text-muted-foreground" aria-hidden="true" />
              </div>
              <div>
                <h2 className="text-lg font-medium text-foreground tracking-tight mb-1">
                  Workspace Initialized
                </h2>
                <p className="text-sm text-muted-foreground font-light">
                  Select a session from the sidebar or start a new analysis.
                </p>
              </div>
              <Button
                id="btn-start-chat-empty"
                onClick={handleCreateSession}
                className="mt-2 gap-2"
              >
                <Plus size={14} aria-hidden="true" /> Start Analysis
              </Button>
            </div>
          )}
        </main>
      </div>

      {/* Mobile sidebar overlay */}
      {isMobileViewport && isSidebarOpen && (
        <button
          className="fixed inset-0 z-40 bg-background/80 backdrop-blur-sm"
          onClick={() => setIsSidebarOpen(false)}
          aria-label="Close session history panel"
          tabIndex={0}
        />
      )}

      {/* Modals */}
      <ProfileDialog 
        isOpen={isProfileOpen} 
        onClose={() => setIsProfileOpen(false)} 
      />
    </div>
  )
}

export default ChatPage
