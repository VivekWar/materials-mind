import { useCallback, useEffect, useMemo } from 'react'
import { useAppStore } from '../store/useAppStore'
import {
  addChatMessage,
  ChatMessage,
  ChatSession,
  chatFollowup,
  createChat,
  generateChatTitle,
  getChatMessages,
  listChats,
  searchStructured,
} from '../api/client'

const mergeSessionMessages = (
  sessions: ChatSession[],
  sessionId: string,
  messages: ChatMessage[],
) =>
  sessions.map((session) =>
    session.id === sessionId
      ? {
          ...session,
          messages,
          updatedAt: messages[messages.length - 1]?.timestamp || session.updatedAt,
        }
      : session,
  )

export const useChat = () => {
  const user = useAppStore((state) => state.user)
  const sessions = useAppStore((state) => state.sessions)
  const setSessions = useAppStore((state) => state.setSessions)
  const activeSessionId = useAppStore((state) => state.activeSessionId)
  const setActiveSessionId = useAppStore((state) => state.setActiveSessionId)
  const loading = useAppStore((state) => state.loading)
  const setLoading = useAppStore((state) => state.setLoading)
  const setStreamingContent = useAppStore((state) => state.setStreamingContent)
  const appendStreamingContent = useAppStore((state) => state.appendStreamingContent)
  const abortController = useAppStore((state) => state.abortController)
  const setAbortController = useAppStore((state) => state.setAbortController)

  const activeSession = useMemo(
    () => sessions.find((session) => session.id === activeSessionId) ?? null,
    [sessions, activeSessionId],
  )

  const isAuthenticated = !!user

  // ── Session list ────────────────────────────────────────────────────────────
  const refreshChats = useCallback(async () => {
    if (!isAuthenticated) return
    const loadedSessions = await listChats()
    if (loadedSessions.length > 0) {
      setSessions((current) =>
        loadedSessions.map((session) => ({
          ...session,
          messages:
            current.find((existing) => existing.id === session.id)?.messages ||
            session.messages,
        })),
      )
      // Only set active session if none is already selected
      const currentActiveId = useAppStore.getState().activeSessionId
      if (!currentActiveId) {
        setActiveSessionId(loadedSessions[0].id)
      }
    }
  }, [isAuthenticated, setSessions, setActiveSessionId])

  useEffect(() => {
    if (isAuthenticated) {
      void refreshChats()
    } else {
      setSessions([])
      setActiveSessionId(null)
    }
  }, [isAuthenticated, refreshChats, setSessions, setActiveSessionId])

  // Load messages when switching sessions
  useEffect(() => {
    if (!activeSessionId) return
    getChatMessages(activeSessionId)
      .then((messages) => {
        setSessions((current) =>
          mergeSessionMessages(current, activeSessionId, messages),
        )
      })
      .catch(() => {
        setSessions((current) =>
          mergeSessionMessages(current, activeSessionId, []),
        )
      })
  }, [activeSessionId, setSessions])

  // ── Message persistence ──────────────────────────────────────────────────────
  const persistAndMergeMessage = useCallback(
    async (
      sessionId: string,
      message: Omit<ChatMessage, 'id' | 'timestamp'>,
    ) => {
      const saved = await addChatMessage(sessionId, message)
      setSessions((current) =>
        current.map((session) =>
          session.id === sessionId
            ? {
                ...session,
                messages: [...session.messages, saved],
                updatedAt: saved.timestamp,
              }
            : session,
        ),
      )
      return saved
    },
    [setSessions],
  )

  // ── Session management ───────────────────────────────────────────────────────
  const createNewSession = useCallback(() => {
    if (!isAuthenticated) return
    const tempSession: ChatSession = {
      id: 'temp-new-chat',
      title: 'New chat',
      messages: [],
      updatedAt: Date.now(),
    }
    setSessions((current) => [tempSession, ...current.filter(s => s.id !== 'temp-new-chat')])
    setActiveSessionId('temp-new-chat')
  }, [isAuthenticated, setSessions, setActiveSessionId])

  const selectSession = useCallback(
    (sessionId: string) => {
      setActiveSessionId(sessionId)
    },
    [setActiveSessionId],
  )

  // ── Stop generation ──────────────────────────────────────────────────────────
  const stopGeneration = useCallback(() => {
    if (abortController) {
      abortController.abort()
      setAbortController(null)
    }
    setLoading(false)
    setStreamingContent('')
  }, [abortController, setAbortController, setLoading, setStreamingContent])

  // ── Send message ─────────────────────────────────────────────────────────────
  const sendMessage = useCallback(
    async (query: string) => {
      const text = query.trim()
      if (!text || loading || !activeSession) return

      let sessionId = activeSession.id
      
      // Lazily create session on backend when first message is sent
      if (sessionId === 'temp-new-chat') {
        const created = await createChat('New chat')
        sessionId = created.id
        setSessions((current) => [created, ...current.filter(s => s.id !== 'temp-new-chat')])
        setActiveSessionId(sessionId)
      }

      const currentActiveSession = useAppStore.getState().sessions.find(s => s.id === sessionId)
      const messagesBeforeSend = currentActiveSession?.messages || []

      await persistAndMergeMessage(sessionId, {
        type: 'user',
        originalQuery: text,
        query: text,
      })

      const controller = new AbortController()
      setAbortController(controller)
      setLoading(true)
      setStreamingContent('')

      let titleRefreshNeeded = false

      try {
        const assistantMessages = messagesBeforeSend.filter(
          (msg) => msg.type === 'assistant',
        )
        const isFirstTurn = assistantMessages.length === 0

        const simulateTypewriter = async (reportText: string, signal: AbortSignal) => {
          setStreamingContent('')
          const words = reportText.split(' ')
          for (let i = 0; i < words.length; i++) {
            if (signal.aborted) throw new DOMException('Aborted', 'AbortError')
            appendStreamingContent(words[i] + ' ')
            await new Promise(resolve => setTimeout(resolve, 80))
          }
        }

        if (isFirstTurn) {
          // Fire-and-forget title generation for new chats
          if (activeSession.title === 'New chat') {
              titleRefreshNeeded = true
              generateChatTitle(sessionId, text)
                .then(() => refreshChats())
                .catch(console.error)
            }

            const structuredRes = await searchStructured(text, {
              onChunk: () => {}, // Ignore actual chunks, use typewriter
              signal: controller.signal,
            })

            await simulateTypewriter(structuredRes.report, controller.signal)

            setStreamingContent('')

            await persistAndMergeMessage(sessionId, {
              type: 'assistant',
              response: {
                recommendations: [],
                report: structuredRes.report,
                structured_result: structuredRes.structured_result,
                tokens_used: 0,
              },
              tokens: 0,
            })
          } else {
            // Follow-up turns: context-aware
            const history = messagesBeforeSend.slice(-10).map((msg) => ({
              role: msg.type,
              content:
                msg.type === 'user'
                  ? msg.originalQuery || msg.query || ''
                  : msg.response?.report || '',
            }))
            const firstAssistant = assistantMessages[0]

            const follow = await chatFollowup({
              message: text,
              history,
              initial_report: firstAssistant?.response?.report || '',
              top_recommendations: [],
            })

            await simulateTypewriter(follow.reply, controller.signal)

            setStreamingContent('')

            await persistAndMergeMessage(sessionId, {
              type: 'assistant',
              response: {
                recommendations: [],
                report: follow.reply,
                tokens_used: follow.tokens_used || 0,
              },
              tokens: follow.tokens_used || 0,
            })
          }
        } catch (err: any) {
          setStreamingContent('')

        // AbortError = user clicked "Stop" — don't show an error message
        if (err?.name === 'AbortError') return

        await persistAndMergeMessage(sessionId, {
          type: 'assistant',
          response: {
            recommendations: [],
            report:
              'I could not reach the material assistant. Please check your connection and try again.',
            tokens_used: 0,
          },
        })
      } finally {
        setLoading(false)
        setAbortController(null)
        // Only refresh the sidebar list when a new title was generated
        if (titleRefreshNeeded) void refreshChats()
      }
    },
    [
      activeSession,
      loading,
      persistAndMergeMessage,
      refreshChats,
      setLoading,
      setStreamingContent,
      appendStreamingContent,
      setAbortController,
    ],
  )

  return {
    sessions,
    activeSessionId,
    activeSession,
    loading,
    sendMessage,
    stopGeneration,
    selectSession,
    createNewSession,
    refreshChats,
  }
}
