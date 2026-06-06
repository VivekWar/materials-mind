import React from 'react'
import { Clock3, MessageSquareText, Plus } from 'lucide-react'
import { ChatSession } from '../hooks/useChatStorage'
import { Badge, Button, Card, CardDescription, CardHeader, CardTitle, ScrollArea } from './ui'
import '../styles/chat-history.css'

interface ChatHistoryProps {
  sessions: ChatSession[]
  activeSessionId: string | null
  onSelectSession: (sessionId: string) => void
  onCreateNewSession: () => void
}

export const ChatHistory: React.FC<ChatHistoryProps> = ({
  sessions,
  activeSessionId,
  onSelectSession,
  onCreateNewSession,
}) => {
  const formatDate = (timestamp: number) => {
    const date = new Date(timestamp)
    const today = new Date()
    const yesterday = new Date(today)
    yesterday.setDate(yesterday.getDate() - 1)

    if (date.toDateString() === today.toDateString()) {
      return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
    }
    if (date.toDateString() === yesterday.toDateString()) {
      return 'Yesterday'
    }
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
  }

  return (
    <Card className="chat-history">
      <CardHeader className="chat-history-header">
        <div>
          <CardTitle>Sessions</CardTitle>
          <CardDescription>{sessions.length} total</CardDescription>
        </div>
        <Button className="btn-new-chat" onClick={onCreateNewSession} title="Start a new chat">
          <Plus size={14} /> New
        </Button>
      </CardHeader>

      {sessions.length === 0 ? (
        <div className="chat-history-empty">
          <div className="empty-icon">
            <MessageSquareText size={22} />
          </div>
          <p>No sessions yet</p>
          <span>Start a new chat to save your material exploration history.</span>
        </div>
      ) : (
        <ScrollArea className="chat-history-list">
          {sessions.map((session) => (
            <button
              type="button"
              key={session.id}
              className={`chat-history-item ${activeSessionId === session.id ? 'active' : ''}`}
              onClick={() => onSelectSession(session.id)}
            >
              <div className="chat-item-content">
                <div className="chat-item-title">{session.title}</div>
                <div className="chat-item-meta">
                  <Badge>
                    <MessageSquareText size={12} /> {session.messages.length}
                  </Badge>
                  <span><Clock3 size={12} /> {formatDate(session.updatedAt)}</span>
                </div>
              </div>
            </button>
          ))}
        </ScrollArea>
      )}
    </Card>
  )
}
