import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  addChatMessage,
  ChatMessage,
  ChatSession,
  chatFollowup,
  createChat,
  getChatMessages,
  listChats,
  searchStructured,
} from '../api/client'

const mergeSessionMessages = (sessions: ChatSession[], sessionId: string, messages: ChatMessage[]) =>
  sessions.map((session) => (
    session.id === sessionId
      ? { ...session, messages, updatedAt: messages[messages.length - 1]?.timestamp || session.updatedAt }
      : session
  ))

export const useChat = (isAuthenticated: boolean) => {
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const activeSession = useMemo(
    () => sessions.find((session) => session.id === activeSessionId) || null,
    [sessions, activeSessionId],
  )

  const refreshChats = useCallback(async () => {
    if (!isAuthenticated) return
    const loadedSessions = await listChats()
    if (loadedSessions.length > 0) {
      setSessions((current) => loadedSessions.map((session) => ({
        ...session,
        messages: current.find((existing) => existing.id === session.id)?.messages || session.messages,
      })))
      setActiveSessionId((current) => current || loadedSessions[0].id)
    } else {
      const created = await createChat('New chat')
      setSessions([created])
      setActiveSessionId(created.id)
    }
  }, [isAuthenticated])

  useEffect(() => {
    if (isAuthenticated) {
      void refreshChats()
    } else {
      setSessions([])
      setActiveSessionId(null)
    }
  }, [isAuthenticated, refreshChats])

  useEffect(() => {
    if (!activeSessionId) return
    getChatMessages(activeSessionId)
      .then((messages) => {
        setSessions((current) => mergeSessionMessages(current, activeSessionId, messages))
      })
      .catch(() => {
        setSessions((current) => mergeSessionMessages(current, activeSessionId, []))
      })
  }, [activeSessionId])

  const persistAndMergeMessage = useCallback(async (sessionId: string, message: Omit<ChatMessage, 'id' | 'timestamp'>) => {
    const saved = await addChatMessage(sessionId, message)
    setSessions((current) => current.map((session) => (
      session.id === sessionId
        ? { ...session, messages: [...session.messages, saved], updatedAt: saved.timestamp }
        : session
    )))
    return saved
  }, [])

  const sendMessage = useCallback(async (query: string) => {
    const text = query.trim()
    if (!text || loading || !activeSession) {
      return
    }

    const sessionId = activeSession.id
    const messagesBeforeSend = activeSession.messages
    await persistAndMergeMessage(sessionId, {
      type: 'user',
      originalQuery: text,
      query: text,
    })

    setLoading(true)
    try {
      const assistantMessages = messagesBeforeSend.filter((msg) => msg.type === 'assistant')
      const shouldRunFullRecommendation = assistantMessages.length === 0

      if (shouldRunFullRecommendation) {
        const structuredRes = await searchStructured(text)
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
        const history = messagesBeforeSend.slice(-10).map((msg) => ({
          role: msg.type,
          content: msg.type === 'user'
            ? (msg.originalQuery || msg.query || '')
            : (msg.response?.report || ''),
        }))
        const firstAssistant = assistantMessages[0]

        const follow = await chatFollowup({
          message: text,
          history,
          initial_report: firstAssistant?.response?.report || '',
          top_recommendations: [],
        })

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
    } catch {
      await persistAndMergeMessage(sessionId, {
        type: 'assistant',
        response: {
          recommendations: [],
          report: 'I could not reach the material assistant endpoint. Please try again in a moment.',
          tokens_used: 0,
        },
      })
    } finally {
      setLoading(false)
      void refreshChats()
    }
  }, [activeSession, loading, persistAndMergeMessage, refreshChats])

  const selectSession = useCallback((sessionId: string) => {
    setActiveSessionId(sessionId)
  }, [])

  const createNewSession = useCallback(async () => {
    const created = await createChat('New chat')
    setSessions((current) => [created, ...current])
    setActiveSessionId(created.id)
  }, [])

  return {
    sessions,
    activeSessionId,
    activeSession,
    loading,
    sendMessage,
    selectSession,
    createNewSession,
  }
}
