import { create } from 'zustand'
import { AuthUser, ChatSession } from '../api/client'

interface AppState {
  // ── Auth ────────────────────────────────────────────────────────────────────
  user: AuthUser | null
  setUser: (user: AuthUser | null) => void

  // ── Sessions ─────────────────────────────────────────────────────────────────
  sessions: ChatSession[]
  setSessions: (sessions: ChatSession[] | ((prev: ChatSession[]) => ChatSession[])) => void

  activeSessionId: string | null
  setActiveSessionId: (id: string | null) => void

  // ── UI states ────────────────────────────────────────────────────────────────
  loading: boolean
  setLoading: (loading: boolean) => void

  /** Live text accumulating from the current SSE stream. Empty string when idle. */
  streamingContent: string
  setStreamingContent: (text: string) => void
  appendStreamingContent: (chunk: string) => void

  // ── API health ───────────────────────────────────────────────────────────────
  apiStatus: 'checking' | 'online' | 'offline'
  setApiStatus: (status: 'checking' | 'online' | 'offline') => void

  // ── Generation control ───────────────────────────────────────────────────────
  /**
   * The AbortController for the currently in-flight SSE request.
   * Calling `abortController.abort()` cancels the stream mid-flight.
   * Null when no request is in flight.
   */
  abortController: AbortController | null
  setAbortController: (controller: AbortController | null) => void
}

export const useAppStore = create<AppState>((set) => ({
  // Auth
  user: null,
  setUser: (user) => set({ user }),

  // Sessions
  sessions: [],
  setSessions: (sessions) =>
    set((state) => ({
      sessions: typeof sessions === 'function' ? sessions(state.sessions) : sessions,
    })),

  activeSessionId: null,
  setActiveSessionId: (id) => set({ activeSessionId: id }),

  // UI
  loading: false,
  setLoading: (loading) => set({ loading }),

  streamingContent: '',
  setStreamingContent: (streamingContent) => set({ streamingContent }),
  appendStreamingContent: (chunk) =>
    set((state) => ({ streamingContent: state.streamingContent + chunk })),

  // API health
  apiStatus: 'checking',
  setApiStatus: (apiStatus) => set({ apiStatus }),

  // Generation control
  abortController: null,
  setAbortController: (abortController) => set({ abortController }),
}))
