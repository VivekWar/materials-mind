import React, { useState, useRef, useEffect } from 'react'
import { ArrowRight, Check, Copy, Layers, Target, AlertTriangle, ShieldCheck, Square, User, Database, Download } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ChatMessage, StructuredRecommendation } from '../api/client'
import { Button } from './ui/button'
import { Skeleton } from './ui/skeleton'
import { useAppStore } from '../store/useAppStore'

// ── Auto-resizing textarea ───────────────────────────────────────────────────
const Textarea = React.forwardRef<HTMLTextAreaElement, React.TextareaHTMLAttributes<HTMLTextAreaElement>>(
  (props, ref) => (
    <textarea
      ref={ref}
      className="flex min-h-[44px] w-full bg-transparent px-3 py-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 resize-none leading-relaxed"
      {...props}
    />
  ),
)
Textarea.displayName = 'Textarea'

// ── Typing animation dots ────────────────────────────────────────────────────
const TypingDots: React.FC = () => (
  <span className="inline-flex items-center gap-1 px-1" role="status" aria-label="Generating">
    <span className="typing-dot" />
    <span className="typing-dot" />
    <span className="typing-dot" />
  </span>
)

// ── Code Block with Copy ─────────────────────────────────────────────────────
const CodeBlock: React.FC<any> = ({ node, inline, className, children, ...props }) => {
  const [copied, setCopied] = useState(false)
  const match = /language-(\w+)/.exec(className || '')
  
  const handleCopy = () => {
    navigator.clipboard.writeText(String(children).replace(/\n$/, ''))
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  if (!inline && match) {
    return (
      <div className="relative group rounded-md overflow-hidden bg-muted/40 my-4 border border-border">
        <div className="flex items-center justify-between px-3 py-1.5 bg-muted/80 border-b border-border">
          <span className="text-[10px] font-mono uppercase tracking-widest text-muted-foreground">{match[1]}</span>
          <button
            onClick={handleCopy}
            className="text-[10px] uppercase tracking-widest flex items-center gap-1 text-muted-foreground hover:text-foreground opacity-0 group-hover:opacity-100 transition-opacity"
            aria-label="Copy code"
          >
            {copied ? <Check size={10} className="text-green-500" /> : <Copy size={10} />}
            {copied ? 'Copied' : 'Copy'}
          </button>
        </div>
        <pre className="p-4 overflow-x-auto text-sm m-0" {...props}>
          <code className={className}>{children}</code>
        </pre>
      </div>
    )
  }
  
  return (
    <code className={className} {...props}>
      {children}
    </code>
  )
}

// ── Loading skeleton ─────────────────────────────────────────────────────────
const AssistantLoadingSkeleton: React.FC<{ streamingContent: string }> = ({ streamingContent }) => {
  const hasContent = streamingContent.length > 0

  return (
    <div className="flex gap-4" role="status" aria-live="polite" aria-label="Generating response">
      <div className="w-7 h-7 shrink-0 rounded bg-foreground text-background flex items-center justify-center" aria-hidden="true">
        <Layers size={14} />
      </div>

      <div className="flex-1 min-w-0 space-y-3 py-0.5">
        <div className="flex items-center gap-2">
          <span className="font-semibold text-xs text-foreground">Materials Mind</span>
          {!hasContent && (
            <span className="text-[10px] font-mono text-muted-foreground flex items-center gap-1.5 uppercase tracking-widest">
              processing <TypingDots />
            </span>
          )}
        </div>

        {hasContent ? (
          <div className="prose prose-sm dark:prose-invert max-w-none font-sans">
            <ReactMarkdown remarkPlugins={[remarkGfm]} components={{ code: CodeBlock }}>
              {streamingContent}
            </ReactMarkdown>
            <span className="streaming-cursor" aria-hidden="true" />
          </div>
        ) : (
          <div className="space-y-2.5 max-w-lg opacity-60" aria-hidden="true">
            <Skeleton className="h-3 w-3/4 rounded-sm" />
            <Skeleton className="h-3 w-5/6 rounded-sm" />
            <Skeleton className="h-3 w-1/2 rounded-sm" />
          </div>
        )}
      </div>
    </div>
  )
}

// ── Data Sheet Pane (Right Side) ─────────────────────────────────────────────
const DataSheetPane: React.FC<{ structured?: StructuredRecommendation; streaming?: boolean }> = ({ structured, streaming }) => {
  if (!structured && !streaming) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-center p-6 text-muted-foreground">
        <div className="w-10 h-10 rounded border border-border border-dashed flex items-center justify-center mb-4 opacity-50">
          <Target size={16} />
        </div>
        <p className="text-xs font-medium uppercase tracking-widest mb-1">No Active Target</p>
        <p className="text-xs font-light opacity-70">Material properties will appear here.</p>
      </div>
    )
  }

  if (!structured && streaming) {
    return (
      <div className="p-6 space-y-6 opacity-50 animate-pulse">
        <div className="space-y-2">
          <div className="h-2 w-16 bg-muted rounded"></div>
          <div className="h-6 w-3/4 bg-muted rounded"></div>
        </div>
        <div className="space-y-2">
          <div className="h-3 w-full bg-muted rounded"></div>
          <div className="h-3 w-5/6 bg-muted rounded"></div>
          <div className="h-3 w-4/6 bg-muted rounded"></div>
        </div>
      </div>
    )
  }

  if (!structured) return null

  return (
    <div className="h-full overflow-y-auto custom-scrollbar p-6 space-y-8 bg-zinc-50 dark:bg-zinc-900/30">
      <div>
        <div className="text-[10px] font-mono uppercase tracking-widest text-muted-foreground mb-1">Primary Candidate</div>
        <h3 className="text-xl font-semibold tracking-tight text-foreground">{structured.recommended_material}</h3>
        {structured.confidence && (
          <div className="mt-3 inline-flex items-center gap-1.5 px-2 py-1 rounded-sm border border-border bg-background text-[10px] font-mono uppercase tracking-widest">
            <ShieldCheck size={12} className="text-primary" />
            Confidence: {Math.round((structured.confidence_score || 0.9) * 100)}%
          </div>
        )}
      </div>

      <div className="space-y-4">
        <div>
          <div className="flex items-center gap-1.5 text-xs font-semibold text-foreground mb-2">
            <Check size={12} className="text-green-500" /> Validation
          </div>
          <ul className="space-y-2">
            {(structured.why_it_matches || []).map((why, i) => (
              <li key={i} className="text-xs text-muted-foreground leading-relaxed pl-4 relative">
                <span className="absolute left-0 top-1.5 w-1 h-1 rounded-full bg-border"></span>
                {why}
              </li>
            ))}
          </ul>
        </div>

        <div>
          <div className="flex items-center gap-1.5 text-xs font-semibold text-foreground mb-2 mt-6">
            <AlertTriangle size={12} className="text-amber-500" /> Trade-offs
          </div>
          <ul className="space-y-2">
            {(structured.trade_offs || []).map((trade, i) => (
              <li key={i} className="text-xs text-muted-foreground leading-relaxed pl-4 relative">
                <span className="absolute left-0 top-1.5 w-1 h-1 rounded-full bg-border"></span>
                {trade}
              </li>
            ))}
          </ul>
        </div>
      </div>

      {structured.candidates && structured.candidates.length > 0 && (
        <div className="mt-8 border-t border-border pt-6">
          <div className="flex items-center justify-between mb-4">
            <div className="text-[10px] font-mono uppercase tracking-widest text-muted-foreground">Top Candidates</div>
          </div>
          <div className="space-y-3">
            {structured.candidates.map(candidate => (
              <div key={candidate.id} className="p-3 rounded-md bg-background border border-border text-xs">
                <div className="font-semibold text-foreground mb-1">{candidate.name}</div>
                <div className="text-muted-foreground grid grid-cols-2 gap-y-1 mt-2">
                  {candidate.yield_strength && <div>Yield: {candidate.yield_strength} MPa</div>}
                  {candidate.density && <div>Density: {candidate.density} g/cm³</div>}
                  {candidate.tensile_strength && <div>Tensile: {candidate.tensile_strength} MPa</div>}
                  {candidate.thermal_conductivity && <div>Cond: {candidate.thermal_conductivity} W/m·K</div>}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// ── Props ────────────────────────────────────────────────────────────────────
interface ChatPanelProps {
  messages: ChatMessage[]
  onSendMessage: (query: string) => Promise<void> | void
  onStopGeneration?: () => void
  loading?: boolean
}

// ── Main component ───────────────────────────────────────────────────────────
export const ChatPanel: React.FC<ChatPanelProps> = ({
  messages,
  onSendMessage,
  onStopGeneration,
  loading = false,
}) => {
  const [query, setQuery] = useState('')
  const [isSending, setIsSending] = useState(false)
  const [copiedMessageId, setCopiedMessageId] = useState<string | null>(null)

  const messagesEndRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const streamingContent = useAppStore((state) => state.streamingContent)

  // Auto-scroll
  useEffect(() => {
    let rafId: number
    const scrollToBottom = () => {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' })
    }
    rafId = requestAnimationFrame(scrollToBottom)
    return () => cancelAnimationFrame(rafId)
  }, [messages, loading, streamingContent])

  // Auto-resize
  useEffect(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = '0px'
    el.style.height = `${Math.min(el.scrollHeight, 200)}px`
  }, [query])

  const copyMessage = async (msg: ChatMessage) => {
    const text = msg.type === 'user' ? (msg.originalQuery || msg.query || '') : (msg.response?.report || '')
    if (!text) return
    try {
      await navigator.clipboard.writeText(text)
      setCopiedMessageId(msg.id)
      window.setTimeout(() => setCopiedMessageId(null), 1500)
    } catch {}
  }

  const send = async () => {
    const text = query.trim()
    if (!text || loading || isSending) return
    setIsSending(true)
    try {
      await onSendMessage(text)
      setQuery('')
      requestAnimationFrame(() => textareaRef.current?.focus())
    } finally {
      setIsSending(false)
    }
  }

  const isDisabled = !query.trim() || loading || isSending

  // Determine what to show in the right pane (latest assistant structured result)
  const latestAssistantMsg = [...messages].reverse().find(m => m.type === 'assistant')
  const latestStructured = latestAssistantMsg?.response?.structured_result
  // If loading and we have NO messages yet in this session, show streaming state in right pane
  const isInitialStream = loading && messages.length <= 1

  return (
    <div className="flex h-full bg-background relative overflow-hidden">
      
      {/* ── Left Pane: Conversation ───────────────────────────────────────── */}
      <div className="flex-1 flex flex-col min-w-0 relative h-full">
        <div className="flex-1 overflow-y-auto custom-scrollbar px-6 lg:px-12 pb-40 pt-8">
          {messages.length === 0 && !loading ? (
            /* Empty state */
            <div className="flex flex-col justify-center h-full max-w-lg mx-auto opacity-90 mt-12">
              <div className="w-10 h-10 rounded border border-border flex items-center justify-center mb-6 shadow-sm">
                <Target size={18} className="text-foreground" />
              </div>
              <h2 className="text-2xl font-semibold tracking-tight text-foreground mb-3">
                Specify constraints.
              </h2>
              <p className="text-sm text-muted-foreground leading-relaxed mb-8 font-light">
                Describe your operational environment, load requirements, and manufacturing process. The inference engine will cross-check properties against our database.
              </p>
              <div className="grid gap-2">
                {[
                  'High-strength alloy for aerospace fatigue loads, density < 4.5 g/cm³',
                  'Lightweight polymer for FDM printing, service temp > 130°C',
                  'Corrosion-resistant metal for marine saltwater environments',
                ].map((prompt) => (
                  <button
                    key={prompt}
                    type="button"
                    onClick={() => setQuery(prompt)}
                    className="text-xs text-left bg-muted/30 hover:bg-muted text-muted-foreground hover:text-foreground px-4 py-3 rounded-md transition-colors border border-border hover:border-border/80"
                  >
                    <span className="font-mono opacity-50 mr-2">&gt;</span> {prompt}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <div className="max-w-3xl mx-auto space-y-10">
              {messages.map((msg) => (
                <div key={msg.id} className="flex gap-4 group">
                  {/* Avatar */}
                  <div className={`w-7 h-7 shrink-0 rounded flex items-center justify-center shadow-sm ${
                      msg.type === 'user' ? 'bg-muted border border-border text-muted-foreground' : 'bg-foreground text-background'
                    }`}
                    aria-hidden="true"
                  >
                    {msg.type === 'user' ? <User size={13} /> : <Layers size={13} />}
                  </div>

                  {/* Content */}
                  <div className="flex-1 min-w-0 space-y-1.5">
                    <div className="flex items-center justify-between min-h-[24px]">
                      <span className="font-semibold text-xs text-foreground">
                        {msg.type === 'user' ? 'User' : 'Materials Mind'}
                      </span>
                      <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                        <button
                          className="h-6 px-2 flex items-center gap-1 text-[10px] uppercase tracking-widest font-mono text-muted-foreground hover:text-foreground"
                          onClick={() => { void copyMessage(msg) }}
                          title="Copy message"
                        >
                          {copiedMessageId === msg.id ? <Check size={10} className="text-green-500" /> : <Copy size={10} />}
                          {copiedMessageId === msg.id ? 'Copied' : 'Copy'}
                        </button>
                      </div>
                    </div>

                    {msg.type === 'user' ? (
                      <div className="text-foreground whitespace-pre-wrap text-sm leading-relaxed font-light">
                        {msg.originalQuery}
                      </div>
                    ) : (
                      <div className="prose prose-sm dark:prose-invert max-w-none font-sans font-light">
                        {/* We removed inline StructuredBlock because it renders in the right pane */}
                        <ReactMarkdown remarkPlugins={[remarkGfm]} components={{ code: CodeBlock }}>
                          {msg.response?.report || ''}
                        </ReactMarkdown>
                      </div>
                    )}
                  </div>
                </div>
              ))}

              {loading && <AssistantLoadingSkeleton streamingContent={streamingContent} />}
            </div>
          )}
          <div ref={messagesEndRef} className="h-4" aria-hidden="true" />
        </div>

        {/* ── Input bar ────────────────────────────────────────────────────── */}
        <div className="absolute bottom-0 inset-x-0 bg-gradient-to-t from-background via-background/95 to-transparent pt-12 pb-6 px-6 lg:px-12">
          <div className="max-w-3xl mx-auto">
            {loading && onStopGeneration && (
              <div className="flex justify-center mb-3">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onStopGeneration}
                  className="h-7 text-xs gap-1.5 rounded-sm shadow-sm bg-background hover:bg-muted font-mono tracking-widest uppercase"
                >
                  <Square size={9} className="fill-current text-foreground" />
                  Stop
                </Button>
              </div>
            )}
            
            <div className={`relative rounded-lg bg-background overflow-hidden transition-all duration-200 border ${
                isSending || loading ? 'border-primary/30 shadow-[0_0_15px_rgba(0,0,0,0.05)]' : 'border-border hover:border-border/80 focus-within:border-foreground/30 focus-within:shadow-[0_0_15px_rgba(0,0,0,0.03)]'
              }`}
            >
              <Textarea
                id="chat-input"
                ref={textareaRef}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Initialize query..."
                aria-label="Message input"
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault()
                    void send()
                  }
                }}
              />
              <div className="absolute bottom-2.5 right-2.5 flex items-center gap-3">
                <span className="text-[10px] font-mono text-muted-foreground hidden sm:block select-none opacity-50">
                  {loading ? 'PROCESSING...' : 'SHIFT+ENTER TO BREAK'}
                </span>
                <Button
                  id="btn-send-message"
                  type="button"
                  size="icon"
                  className={`h-7 w-7 rounded-[4px] shadow-sm transition-all ${isDisabled ? 'opacity-40' : ''}`}
                  onClick={() => { void send() }}
                  disabled={isDisabled}
                  title="Send message"
                >
                  <ArrowRight size={13} />
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* ── Right Pane: Data Sheet (Visible on large screens) ───────────── */}
      <div className="hidden lg:block w-80 shrink-0 border-l border-border bg-muted/10 h-full relative z-10">
        <div className="h-12 border-b border-border flex items-center px-6">
          <span className="text-[10px] font-mono uppercase tracking-widest text-muted-foreground flex items-center gap-2">
            <Database size={10} />
            Data Sheet
          </span>
        </div>
        <div className="absolute top-12 bottom-0 left-0 right-0">
           <DataSheetPane structured={latestStructured} streaming={isInitialStream} />
        </div>
      </div>

    </div>
  )
}
