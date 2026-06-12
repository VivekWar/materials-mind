/**
 * App.tsx — root router shell.
 *
 * Responsibilities:
 *   1. Register the 401 → logout redirect handler once on mount
 *   2. Route between the homepage and the chat workspace
 *
 * All page-level logic lives in src/pages/.
 */
import React, { useEffect, useState } from 'react'
import { setUnauthorizedHandler, getMe, setAuthToken } from './api/client'
import { useAppStore } from './store/useAppStore'
import { HomePage } from './components/HomePage'
import ChatPage from './pages/ChatPage'
import { ErrorBoundary } from './components/ErrorBoundary'
import './styles/index.css'

const CHAT_ROUTE = '/chat'

const navigateTo = (path: string) => {
  if (window.location.pathname !== path) {
    window.history.pushState({}, '', path)
    window.dispatchEvent(new PopStateEvent('popstate'))
  }
}

const App: React.FC = () => {
  const [pathname, setPathname] = useState(() => window.location.pathname)
  const setUser = useAppStore((state) => state.setUser)

  // Register 401 handler once — clears auth state and redirects to home
  useEffect(() => {
    setUnauthorizedHandler(() => {
      setUser(null)
      navigateTo('/')
    })
  }, [setUser])

  // Initial auth check for persistent sessions when users land directly on the app
  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    if (token && window.location.pathname !== CHAT_ROUTE) {
      getMe()
        .then(setUser)
        .catch(() => setUser(null))
    }
  }, [setUser])

  // Listen for secure authentication postMessage from popup
  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      if (event.data?.type === 'AUTH_SUCCESS' && event.data?.token) {
        setAuthToken(event.data.token)
        getMe()
          .then(setUser)
          .then(() => navigateTo(CHAT_ROUTE))
          .catch(() => setUser(null))
      }
    }
    window.addEventListener('message', handleMessage)
    return () => window.removeEventListener('message', handleMessage)
  }, [setUser])

  // Minimal client-side router
  useEffect(() => {
    const handlePopState = () => setPathname(window.location.pathname)
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  // Sanitize trailing slashes to ensure robust routing
  const currentPath = pathname.replace(/\/$/, '') || '/'

  return (
    <ErrorBoundary>
      {currentPath === CHAT_ROUTE ? (
        <ChatPage />
      ) : (
        <HomePage onStartChat={() => navigateTo(CHAT_ROUTE)} />
      )}
    </ErrorBoundary>
  )
}

export default App
