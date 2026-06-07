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
import { setUnauthorizedHandler } from './api/client'
import { useAppStore } from './store/useAppStore'
import { HomePage } from './components/HomePage'
import ChatPage from './pages/ChatPage'
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

  // Minimal client-side router
  useEffect(() => {
    const handlePopState = () => setPathname(window.location.pathname)
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  if (pathname === CHAT_ROUTE) {
    return <ChatPage />
  }

  return <HomePage onStartChat={() => navigateTo(CHAT_ROUTE)} />
}

export default App
