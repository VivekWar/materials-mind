import React from 'react'
import { Layers, LogOut, MessageSquareText, Plus } from 'lucide-react'
import { ChatSession, logout } from '../api/client'
import { Button } from './ui/button'
import { useAppStore } from '../store/useAppStore'

interface ChatHistoryProps {
  sessions: ChatSession[]
  activeSessionId: string | null
  onSelectSession: (sessionId: string) => void
  onCreateNewSession: () => void
}

type GroupedSessions = {
  label: string
  sessions: ChatSession[]
}

function groupSessionsByDate(sessions: ChatSession[]): GroupedSessions[] {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)
  const sevenDaysAgo = new Date(today)
  sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 7)

  const groups: GroupedSessions[] = [
    { label: 'Today', sessions: [] },
    { label: 'Yesterday', sessions: [] },
    { label: 'Past 7 days', sessions: [] },
    { label: 'Earlier', sessions: [] },
  ]

  for (const session of sessions) {
    const d = new Date(session.updatedAt)
    d.setHours(0, 0, 0, 0)
    if (d >= today) {
      groups[0].sessions.push(session)
    } else if (d >= yesterday) {
      groups[1].sessions.push(session)
    } else if (d >= sevenDaysAgo) {
      groups[2].sessions.push(session)
    } else {
      groups[3].sessions.push(session)
    }
  }

  return groups.filter((g) => g.sessions.length > 0)
}



const getUserInitials = (name?: string, email?: string) => {
  const display = name || email || '?'
  return display
    .split(' ')
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? '')
    .join('')
}

export const ChatHistory: React.FC<ChatHistoryProps> = ({
  sessions,
  activeSessionId,
  onSelectSession,
  onCreateNewSession,
}) => {
  const user = useAppStore((state) => state.user)
  const setUser = useAppStore((state) => state.setUser)

  const handleLogout = async () => {
    try {
      await logout()
    } catch {
      // ignore errors — we clear state regardless
    }
    setUser(null)
  }

  // Filter out empty chats and temp-new-chat from sidebar
  const filteredSessions = sessions.filter(s => s.messages && s.messages.length > 0)
  const grouped = groupSessionsByDate(filteredSessions)

  return (
    <div className="flex flex-col h-full w-64 bg-background border-r border-border overflow-hidden">
      {/* ── Header ──────────────────────────────────────────────────────── */}
      <div className="p-4 border-b border-border shrink-0">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <div className="w-5 h-5 rounded-[4px] bg-foreground flex items-center justify-center">
              <Layers size={10} className="text-background" />
            </div>
            <span className="text-xs font-semibold text-foreground tracking-tight uppercase">History</span>
          </div>
          <Button
            id="btn-sidebar-new-chat"
            size="sm"
            variant="ghost"
            onClick={onCreateNewSession}
            title="New search"
            className="h-6 w-6 p-0 rounded-[4px] text-muted-foreground hover:text-foreground hover:bg-muted"
          >
            <Plus size={14} />
          </Button>
        </div>
      </div>

      {/* ── Session list ─────────────────────────────────────────────────── */}
      <div className="flex-1 overflow-y-auto custom-scrollbar py-2">
        {grouped.length === 0 ? (
          /* Empty state */
          <div className="flex flex-col items-center justify-center h-full px-5 py-10 text-center space-y-3 opacity-60">
            <MessageSquareText size={18} className="text-muted-foreground mb-2" />
            <p className="text-[11px] font-mono uppercase tracking-widest text-muted-foreground">No History</p>
          </div>
        ) : (
          /* Grouped sessions */
          <div className="space-y-4 px-3 mt-2">
            {grouped.map((group) => (
              <div key={group.label}>
                <div className="px-2 py-1 mb-1 text-[10px] font-mono uppercase tracking-widest text-muted-foreground/60 select-none">
                  {group.label}
                </div>
                <div className="space-y-0.5">
                  {group.sessions.map((session) => {
                    const isActive = activeSessionId === session.id
                    const lastMessage = session.messages[session.messages.length - 1]
                    const previewText = lastMessage?.type === 'user' 
                      ? lastMessage.originalQuery || lastMessage.query 
                      : lastMessage?.response?.report || 'New session'
                      
                    return (
                      <button
                        type="button"
                        key={session.id}
                        id={`btn-session-${session.id}`}
                        aria-current={isActive ? 'page' : undefined}
                        className={`w-full text-left px-2 py-2 rounded-md transition-all duration-100 group ${
                          isActive
                            ? 'bg-muted/50 text-foreground'
                            : 'text-muted-foreground hover:bg-muted/30 hover:text-foreground'
                        }`}
                        onClick={() => onSelectSession(session.id)}
                      >
                        <div className="flex items-start gap-2">
                          <div className="flex-1 min-w-0">
                            <div className={`text-xs font-medium line-clamp-1 leading-snug tracking-tight ${isActive ? 'text-foreground' : ''}`}>
                              {session.title}
                            </div>
                            <div className="text-[10px] line-clamp-1 mt-0.5 opacity-70 font-light">
                              {previewText}
                            </div>
                          </div>
                        </div>
                      </button>
                    )
                  })}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* ── User chip / footer ───────────────────────────────────────────── */}
      {user && (
        <div className="p-4 border-t border-border shrink-0">
          <div className="flex items-center gap-2">
            {/* Avatar */}
            <div className="w-6 h-6 rounded-[4px] bg-muted flex items-center justify-center shrink-0 border border-border">
              <span className="text-[9px] font-mono text-foreground">{getUserInitials(user.name, user.email)}</span>
            </div>
            {/* Name + email */}
            <div className="flex-1 min-w-0">
              <div className="text-[11px] font-medium text-foreground truncate tracking-tight">
                {user.name || 'Engineer'}
              </div>
            </div>
            {/* Logout */}
            <button
              type="button"
              id="btn-logout"
              title="End session"
              aria-label="End session"
              onClick={() => { void handleLogout() }}
              className="p-1.5 rounded-[4px] text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
            >
              <LogOut size={12} />
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
