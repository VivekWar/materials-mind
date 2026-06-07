import React, { useState, useRef, useEffect } from 'react'
import { ArrowUp, Bot, Check, Copy, Sparkles, User } from 'lucide-react'
import { ChatMessage } from '../api/client'
import { Button, Textarea } from './ui'
import '../styles/chat.css'

interface ChatPanelProps {
  messages: ChatMessage[]
  onSendMessage: (query: string) => Promise<void> | void
  loading?: boolean
}

export const ChatPanel: React.FC<ChatPanelProps> = ({
  messages,
  onSendMessage,
  loading = false,
}) => {
  const [query, setQuery] = useState('')
  const [isSending, setIsSending] = useState(false)
  const [copiedMessageId, setCopiedMessageId] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  useEffect(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = '0px'
    el.style.height = `${Math.min(el.scrollHeight, 200)}px`
  }, [query])

  const getAssistantCopyText = (msg: ChatMessage): string => {
    if (!msg.response) return ''
    const topThree = (msg.response.recommendations || []).slice(0, 3)
    const lines: string[] = []
    if (topThree.length > 0) {
      lines.push('Top 3 recommendations:')
      topThree.forEach((item: any, idx: number) => {
        lines.push(`${idx + 1}. ${item.name}${item.category ? ` (${item.category})` : ''}`)
      })
      lines.push('')
    }
    lines.push(msg.response.report || '')
    return lines.join('\n')
  }

  const copyMessage = async (msg: ChatMessage) => {
    const text = msg.type === 'user' ? (msg.originalQuery || msg.query || '') : getAssistantCopyText(msg)
    if (!text) return
    try {
      await navigator.clipboard.writeText(text)
      setCopiedMessageId(msg.id)
      window.setTimeout(() => {
        setCopiedMessageId((current) => (current === msg.id ? null : current))
      }, 1500)
    } catch {
      // no-op for unsupported clipboard permissions
    }
  }

  const renderInlineText = (text: string) => {
    return text.split(/(\*\*.*?\*\*)/g).map((part, idx) => {
      if (part.startsWith('**') && part.endsWith('**')) {
        return <strong key={idx}>{part.slice(2, -2)}</strong>
      }
      return <React.Fragment key={idx}>{part}</React.Fragment>
    })
  }

  const cleanMarkdownText = (text: string) => {
    return text
      .replace(/^#{1,6}\s*/g, '')
      .replace(/\*\*(.*?)\*\*/g, '$1')
      .replace(/\s+/g, ' ')
      .trim()
  }

  const extractLeadText = (report: string) => {
    const lines = report
      .split('\n')
      .map((line) => cleanMarkdownText(line))
      .filter(Boolean)

    return (
      lines.find((line) => !/^[-*]/.test(line) && !/^\d+\./.test(line) && line.length > 24) ||
      lines[0] ||
      ''
    )
  }

  const extractQuickFacts = (report: string) => {
    return report
      .split('\n')
      .map((line) => cleanMarkdownText(line))
      .filter((line) => Boolean(line) && (/^[-*]/.test(line) || /^\d+\./.test(line) || line.includes(':')))
      .map((line) => line.replace(/^[-*]\s*/, '').replace(/^\d+\.\s*/, ''))
      .slice(0, 3)
  }

  const renderReport = (report: string) => {
    return report.split('\n').map((rawLine, idx) => {
      const line = rawLine.trim()
      if (!line) {
        return <div key={idx} className="report-space" />
      }

      const heading = line.match(/^#{1,6}\s+(.+)$/)
      if (heading) {
        return <h3 key={idx} className="report-heading">{renderInlineText(heading[1])}</h3>
      }

      const bullet = line.match(/^[-*]\s+(.+)$/)
      if (bullet) {
        return (
          <div key={idx} className="report-bullet">
            <span aria-hidden="true" />
            <p>{renderInlineText(bullet[1])}</p>
          </div>
        )
      }

      const numbered = line.match(/^(\d+)\.\s+(.+)$/)
      if (numbered) {
        return (
          <div key={idx} className="report-numbered">
            <span>{numbered[1]}</span>
            <p>{renderInlineText(numbered[2])}</p>
          </div>
        )
      }

      return <p key={idx} className="report-paragraph">{renderInlineText(line)}</p>
    })
  }

  const renderStructuredBlock = (structured: any) => {
    if (!structured) {
      return null
    }

    const why = Array.isArray(structured.why_it_matches) ? structured.why_it_matches : []
    const tradeOffs = Array.isArray(structured.trade_offs) ? structured.trade_offs : []
    const sources = Array.isArray(structured.sources) ? structured.sources : []
    const confidenceScore = typeof structured.confidence_score === 'number'
      ? Math.max(0, Math.min(1, structured.confidence_score))
      : undefined

    return (
      <div className="recommendation-summary recommendation-summary--compact">
        <div className="assistant-highlight">
          <div className="assistant-highlight-label">Recommended material</div>
          <div className="assistant-highlight-title">{structured.recommended_material || 'N/A'}</div>
          <p className="assistant-highlight-copy">
            Confidence: {structured.confidence || 'Unknown'}
            {confidenceScore !== undefined ? ` (${Math.round(confidenceScore * 100)}%)` : ''}
          </p>
        </div>

        {why.length > 0 && (
          <>
            <strong>Why it matches</strong>
            <div className="assistant-facts">
              {why.map((item: string, idx: number) => (
                <div key={`why-${idx}`} className="assistant-fact">{item}</div>
              ))}
            </div>
          </>
        )}

        {tradeOffs.length > 0 && (
          <>
            <strong>Trade-offs</strong>
            <div className="assistant-facts">
              {tradeOffs.map((item: string, idx: number) => (
                <div key={`trade-${idx}`} className="assistant-fact">{item}</div>
              ))}
            </div>
          </>
        )}

        {sources.length > 0 && (
          <p className="assistant-highlight-copy">Sources: {sources.map((id: number) => `[ID:${id}]`).join(', ')}</p>
        )}
      </div>
    )
  }

  const send = async () => {
    const text = query.trim()
    if (!text || loading || isSending) {
      return
    }

    setIsSending(true)
    try {
      await onSendMessage(text)
      setQuery('')
    } finally {
      setIsSending(false)
    }
  }

  return (
    <div className="chat-panel chat-panel--full">
      <div className="chat-messages">
        {messages.length === 0 ? (
          <div className="chat-empty-state chat-empty-state--minimal">
            <div className="chat-empty-icon chat-empty-icon--spark">
              <Sparkles size={26} />
            </div>
            <h2>What are we building today?</h2>
            <p>Describe your constraints, process, and performance goals to start.</p>
          </div>
        ) : (
          messages.map((msg) => (
            <div key={msg.id} className={`chat-message-row chat-message-row-${msg.type}`}>
              <div className="chat-avatar" aria-hidden="true">
                {msg.type === 'user' ? <User size={14} /> : <Bot size={14} />}
              </div>

              <div className={`chat-message chat-message-${msg.type}`}>
              <div className="chat-message-header">
                <span className="chat-message-role">{msg.type === 'user' ? 'You' : 'Material Assistant'}</span>
                <div className="chat-header-right">
                  <Button
                    className="message-action"
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      void copyMessage(msg)
                    }}
                    title="Copy message"
                    aria-label="Copy message"
                  >
                    {copiedMessageId === msg.id ? <Check size={13} /> : <Copy size={13} />}
                    {copiedMessageId === msg.id ? 'Copied' : 'Copy'}
                  </Button>
                  <span className="chat-message-time">
                    {new Date(msg.timestamp).toLocaleTimeString()}
                  </span>
                </div>
              </div>

              {msg.type === 'user' ? (
                <div className="chat-message-query">{msg.originalQuery}</div>
              ) : (
                <>
                  {msg.response && (
                    <>
                      <div className="chat-message-response">
                        {msg.response.structured_result && renderStructuredBlock(msg.response.structured_result)}

                        {msg.response.recommendations && msg.response.recommendations.length > 0 && (
                          <div className="recommendation-summary recommendation-summary--compact">
                            <div className="assistant-highlight">
                              <div className="assistant-highlight-label">Best match</div>
                              <div className="assistant-highlight-title">
                                {msg.response.final_recommendation?.name || msg.response.recommendations[0]?.name}
                              </div>
                              {extractLeadText(msg.response.report) && (
                                <p className="assistant-highlight-copy">{extractLeadText(msg.response.report)}</p>
                              )}
                              {extractQuickFacts(msg.response.report).length > 0 && (
                                <div className="assistant-facts">
                                  {extractQuickFacts(msg.response.report).map((fact, idx) => (
                                    <div key={`${msg.id}-fact-${idx}`} className="assistant-fact">
                                      {fact}
                                    </div>
                                  ))}
                                </div>
                              )}
                            </div>

                            <strong>Shortlist</strong>
                            <div className="top3-grid">
                              {msg.response.recommendations.slice(0, 3).map((item: any, idx: number) => (
                                <div key={`${msg.id}-${item.id ?? item.name ?? idx}`} className={`top3-card ${idx === 0 ? 'top3-card-best' : ''}`}>
                                  <div className="top3-rank">#{idx + 1}</div>
                                  <div className="top3-name">{item.name}</div>
                                  <div className="chip-row">
                                    <span className="preview-chip">{item.category}</span>
                                    {item.subcategory && (
                                      <span className="preview-chip preview-chip--muted">{item.subcategory}</span>
                                    )}
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}

                        <div className="report-excerpt">
                          {renderReport(msg.response.report)}
                        </div>
                      </div>
                    </>
                  )}
                </>
              )}
              </div>
            </div>
          ))
        )}
        {loading && (
          <div className="chat-message-row chat-message-row-assistant">
            <div className="chat-avatar" aria-hidden="true">
              <Bot size={14} />
            </div>
            <div className="chat-message chat-message-assistant">
              <div className="chat-message-header">
                <span className="chat-message-role">Material Assistant</span>
                <span className="chat-message-time">Analyzing...</span>
              </div>
              <div className="assistant-thinking">
                <span />
                <span />
                <span />
              </div>
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      <div className="chat-composer-shell">
        <div className="chat-composer-frame">
        <div className="chat-composer">
          <Textarea
            ref={textareaRef}
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Message Material Assistant..."
            onKeyDown={(event) => {
              if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault()
                void send()
              }
            }}
          />

          <Button
            type="button"
            className="chat-send-button"
            size="icon"
            onClick={() => {
              void send()
            }}
            disabled={!query.trim() || loading || isSending}
            title="Send"
          >
            <ArrowUp size={16} />
          </Button>
        </div>
        </div>
      </div>
    </div>
  )
}
