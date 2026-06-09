import React from 'react'
import { X, User } from 'lucide-react'
import { Button } from './ui/button'
import { useAppStore } from '../store/useAppStore'

interface ProfileDialogProps {
  isOpen: boolean
  onClose: () => void
}

export const ProfileDialog: React.FC<ProfileDialogProps> = ({ isOpen, onClose }) => {
  const user = useAppStore((state) => state.user)

  if (!isOpen || !user) return null

  const chatPercentage = Math.min(100, Math.round((user.chats_used / user.max_chats) * 100))
  const messagePercentage = Math.min(100, Math.round((user.messages_used / user.max_messages) * 100))

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div 
        className="bg-background border border-border w-full max-w-md rounded-xl shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-200"
        role="dialog"
        aria-modal="true"
        aria-labelledby="profile-dialog-title"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border">
          <h2 id="profile-dialog-title" className="text-lg font-semibold text-foreground">
            User Profile
          </h2>
          <Button variant="ghost" size="icon" onClick={onClose} className="h-8 w-8 text-muted-foreground hover:text-foreground">
            <X size={18} />
          </Button>
        </div>

        {/* Body */}
        <div className="p-6 space-y-6">
          {/* User Info */}
          <div className="flex items-center gap-4">
            <div className="h-16 w-16 rounded-full bg-primary/10 flex items-center justify-center text-primary border border-primary/20">
              <User size={32} />
            </div>
            <div>
              <p className="font-medium text-lg text-foreground">{user.name || 'Anonymous User'}</p>
              <p className="text-sm text-muted-foreground">{user.email}</p>
            </div>
          </div>

          <div className="space-y-4 pt-2">
            <h3 className="text-sm font-medium text-foreground uppercase tracking-wider">Usage Limits (Daily)</h3>
            
            {/* Chats Progress */}
            <div className="space-y-1.5">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Chats Created</span>
                <span className="font-medium text-foreground">
                  {user.chats_used} / {user.max_chats}
                </span>
              </div>
              <div className="h-2 w-full bg-muted rounded-full overflow-hidden">
                <div 
                  className={`h-full rounded-full transition-all ${chatPercentage >= 100 ? 'bg-red-500' : chatPercentage >= 80 ? 'bg-amber-500' : 'bg-primary'}`} 
                  style={{ width: `${chatPercentage}%` }} 
                />
              </div>
            </div>

            {/* Messages Progress */}
            <div className="space-y-1.5">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Messages Sent</span>
                <span className="font-medium text-foreground">
                  {user.messages_used} / {user.max_messages}
                </span>
              </div>
              <div className="h-2 w-full bg-muted rounded-full overflow-hidden">
                <div 
                  className={`h-full rounded-full transition-all ${messagePercentage >= 100 ? 'bg-red-500' : messagePercentage >= 80 ? 'bg-amber-500' : 'bg-primary'}`} 
                  style={{ width: `${messagePercentage}%` }} 
                />
              </div>
            </div>

            {(chatPercentage >= 100 || messagePercentage >= 100) && (
              <p className="text-xs text-red-500 mt-2 bg-red-500/10 p-2 rounded">
                Message quota exhausted.
              </p>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-border bg-muted/30 flex justify-end">
          <Button onClick={onClose} variant="outline">
            Close
          </Button>
        </div>
      </div>
    </div>
  )
}
