/**
 * api/client.ts — HTTP + SSE client for Materials Mind.
 *
 * Circular-dependency note:
 *   This file previously imported useAppStore directly, creating a circular
 *   dependency (store → client → store). The fix: a module-level callback
 *   registered by the root App. Call setUnauthorizedHandler() once at startup.
 */
import axios from 'axios'

// ── 401 callback (avoids circular import with useAppStore) ───────────────────
let _unauthorizedHandler: (() => void) | null = null

export function setUnauthorizedHandler(handler: () => void): void {
  _unauthorizedHandler = handler
}

function handleUnauthorized() {
  setAuthToken(null)
  _unauthorizedHandler?.()
}

// ── Auth Token Management ────────────────────────────────────────────────────
let _authToken: string | null = localStorage.getItem('auth_token')

export function setAuthToken(token: string | null) {
  _authToken = token
  if (token) {
    localStorage.setItem('auth_token', token)
    api.defaults.headers.common['Authorization'] = `Bearer ${token}`
  } else {
    localStorage.removeItem('auth_token')
    delete api.defaults.headers.common['Authorization']
  }
}

// ── Axios instance ───────────────────────────────────────────────────────────
const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080/api',
  headers: { 'Content-Type': 'application/json' },
  timeout: 180000,
  withCredentials: true,
})

if (_authToken) {
  api.defaults.headers.common['Authorization'] = `Bearer ${_authToken}`
}

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      handleUnauthorized()
    }
    return Promise.reject(error)
  },
)

// ── Base URL helpers ─────────────────────────────────────────────────────────
function resolveBackendUrl(path: string): string {
  const baseURL = api.defaults.baseURL || ''
  const apiRoot = baseURL.endsWith('/api') ? baseURL : `${baseURL}/api`
  return `${apiRoot}${path}`
}

async function fetchWithAuth(url: string, options: RequestInit = {}): Promise<Response> {
  const headers = new Headers(options.headers || {})
  if (_authToken) {
    headers.set('Authorization', `Bearer ${_authToken}`)
  }
  options.headers = headers

  const res = await fetch(url, options)
  if (res.status === 401) {
    handleUnauthorized()
  }
  return res
}

// ── Type definitions ─────────────────────────────────────────────────────────
export interface Constraint {
  id?: string
  key: string
  operator: 'min' | 'max' | 'equals' | 'contains'
  value: string | number
  label?: string
}

export interface RangeFilter {
  min?: number
  max?: number
}

export interface IntentJSON {
  filters: Record<string, RangeFilter>
  category: string
  sort_by: string
  sort_dir: string
}

export interface Material {
  id: number
  name: string
  formula: string
  category: string
  subcategory?: string
  density?: number
  glass_transition_temp?: number
  heat_deflection_temp?: number
  melting_point?: number
  boiling_point?: number
  thermal_conductivity?: number
  specific_heat?: number
  thermal_expansion?: number
  electrical_resistivity?: number
  yield_strength?: number
  tensile_strength?: number
  youngs_modulus?: number
  hardness_vickers?: number
  poissons_ratio?: number
  processing_temp_min_c?: number
  processing_temp_max_c?: number
  crystallinity?: number
  source: string
}

export interface RecommendResponse {
  query: string
  extracted_intent: IntentJSON
  recommendations: Material[]
  final_recommendation?: Material
  top_recommendations?: Material[]
  routed_category?: string
  inline_alloy_prediction?: InlineAlloyPrediction
  structured_result?: StructuredRecommendation
  report: string
  tokens_used: number
}

export interface MaterialCandidate {
  id: number
  name: string
  formula?: string
  category: string
  subcategory?: string
  density?: number
  glass_transition_temp?: number
  heat_deflection_temp?: number
  melting_point?: number
  boiling_point?: number
  thermal_conductivity?: number
  specific_heat?: number
  thermal_expansion?: number
  electrical_resistivity?: number
  yield_strength?: number
  tensile_strength?: number
  youngs_modulus?: number
  hardness_vickers?: number
  poissons_ratio?: number
  processing_temp_min_c?: number
  processing_temp_max_c?: number
  crystallinity?: number
  crystal_system?: string
  fracture_toughness?: number
  weibull_modulus?: number
  interlaminar_shear_strength?: number
  fiber_volume_fraction?: number
  source?: string
}

export interface StructuredRecommendation {
  recommended_material: string
  why_it_matches: string[]
  trade_offs: string[]
  confidence: 'High' | 'Medium' | 'Low'
  confidence_score: number
  sources: number[]
  report: string
  candidates?: MaterialCandidate[]
}

export interface StructuredSearchResponse {
  structured_result?: StructuredRecommendation
  report: string
}

export interface InlineAlloyPrediction {
  summary: string
  key_findings?: Record<string, string>
  risk_flags?: string[]
  confidence?: string
  should_display: boolean
}

export interface PredictResponse {
  composition: Record<string, number>
  predicted_name: string
  baseline_properties?: Record<string, number>
  density?: number
  melting_point?: number
  thermal_conductivity?: number
  electrical_resistivity?: number
  yield_strength?: number
  youngs_modulus?: number
  scientific_explanation?: string
  method: string
  notes?: string
}

export interface ChatTurn {
  role: 'user' | 'assistant'
  content: string
}

export interface FollowUpChatRequest {
  message: string
  history?: ChatTurn[]
  initial_report?: string
  top_recommendations?: string[]
}

export interface FollowUpChatResponse {
  reply: string
  tokens_used: number
}

export interface AuthUser {
  user_id: string
  email: string
  name?: string
}

export interface ChatMessage {
  id: string
  type: 'user' | 'assistant'
  originalQuery?: string
  query?: string
  response?: any
  timestamp: number
  tokens?: number
}

export interface ChatSession {
  id: string
  title: string
  messages: ChatMessage[]
  createdAt: number
  updatedAt: number
}

interface ApiChat {
  id: number
  title: string
  created_at: string
  updated_at: string
}

interface ApiMessage {
  id: number
  sender_role: 'user' | 'assistant' | 'system'
  content: any
  content_text?: string
  tokens_used?: number
  created_at: string
}

// ── Auth ─────────────────────────────────────────────────────────────────────
export async function googleLogin(credential: string): Promise<AuthUser> {
  const { data } = await api.post<{ user: AuthUser }>('/auth/google', { credential })
  return data.user
}

export async function getMe(): Promise<AuthUser> {
  const res = await fetchWithAuth(resolveBackendUrl('/auth/me'), {
    credentials: 'include',
  })
  if (!res.ok) throw new Error(`me failed: HTTP ${res.status}`)
  const data = (await res.json()) as { user: AuthUser }
  return data.user
}

export async function logout(): Promise<void> {
  await fetchWithAuth(resolveBackendUrl('/auth/logout'), {
    method: 'POST',
    credentials: 'include',
  })
}

// ── Chat CRUD ─────────────────────────────────────────────────────────────────
function mapApiMessage(msg: ApiMessage): ChatMessage {
  const timestamp = new Date(msg.created_at).getTime()
  if (msg.sender_role === 'user') {
    const text =
      typeof msg.content?.text === 'string' ? msg.content.text : msg.content_text || ''
    return {
      id: String(msg.id),
      type: 'user',
      originalQuery: text,
      query: text,
      timestamp,
      tokens: msg.tokens_used || 0,
    }
  }
  return {
    id: String(msg.id),
    type: 'assistant',
    response: msg.content?.response || msg.content,
    timestamp,
    tokens: msg.tokens_used || 0,
  }
}

function mapApiChat(chat: ApiChat, messages: ChatMessage[] = []): ChatSession {
  return {
    id: String(chat.id),
    title: chat.title,
    messages,
    createdAt: new Date(chat.created_at).getTime(),
    updatedAt: new Date(chat.updated_at).getTime(),
  }
}

export async function createChat(title = 'New chat'): Promise<ChatSession> {
  const res = await fetchWithAuth(resolveBackendUrl('/chat/create'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ title }),
  })
  if (!res.ok) throw new Error(`create chat failed: HTTP ${res.status}`)
  const data = (await res.json()) as { chat: ApiChat }
  return mapApiChat(data.chat)
}

export async function generateChatTitle(chatId: string, query: string): Promise<string> {
  const res = await fetchWithAuth(resolveBackendUrl(`/chat/${chatId}/title/generate`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ query }),
  })
  if (!res.ok) throw new Error('Failed to generate title')
  const data = (await res.json()) as { title: string }
  return data.title
}

export async function listChats(): Promise<ChatSession[]> {
  const res = await fetchWithAuth(resolveBackendUrl('/chat/list'), {
    credentials: 'include',
  })
  if (!res.ok) throw new Error(`list chats failed: HTTP ${res.status}`)
  const data = (await res.json()) as { chats: ApiChat[] }
  return (data.chats || []).map((chat) => mapApiChat(chat))
}

export async function getChatMessages(chatId: string): Promise<ChatMessage[]> {
  const res = await fetchWithAuth(resolveBackendUrl(`/chat/${chatId}/messages`), {
    credentials: 'include',
  })
  if (!res.ok) throw new Error(`get messages failed: HTTP ${res.status}`)
  const data = (await res.json()) as { messages: ApiMessage[] }
  return (data.messages || []).map(mapApiMessage)
}

export async function addChatMessage(
  chatId: string,
  message: Pick<ChatMessage, 'type' | 'originalQuery' | 'query' | 'response' | 'tokens'>,
): Promise<ChatMessage> {
  const isUser = message.type === 'user'
  const text = isUser
    ? message.originalQuery || message.query || ''
    : message.response?.report || ''
  const content = isUser ? { text } : { response: message.response }

  const res = await fetchWithAuth(resolveBackendUrl(`/chat/${chatId}/messages`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({
      sender_role: isUser ? 'user' : 'assistant',
      content,
      content_text: text,
      tokens_used: message.tokens || 0,
    }),
  })
  if (!res.ok) throw new Error(`add message failed: HTTP ${res.status}`)
  return mapApiMessage((await res.json()) as ApiMessage)
}

// ── Recommendation (legacy, kept for reference) ───────────────────────────────
export async function recommend(
  query: string,
  domain: string,
  constraints?: Constraint[],
): Promise<RecommendResponse> {
  const payload: any = { query, domain }
  if (constraints && constraints.length > 0) {
    payload.constraints = constraints.map((c) => ({
      key: c.key,
      operator: c.operator,
      value: c.value,
    }))
  }
  let data: RecommendResponse
  try {
    const res = await api.post<RecommendResponse>('/recommend', payload)
    data = res.data
  } catch {
    const structured = await searchStructured(query)
    return {
      query,
      extracted_intent: { filters: {}, category: '', sort_by: '', sort_dir: '' },
      recommendations: [],
      structured_result: structured.structured_result,
      report: structured.report,
      tokens_used: 0,
    }
  }

  const recommendations = data.recommendations || []
  const topRecommendations = recommendations.slice(0, 3)

  return {
    query: data.query,
    extracted_intent: data.extracted_intent,
    recommendations,
    final_recommendation: data.final_recommendation || topRecommendations[0],
    top_recommendations: topRecommendations,
    routed_category: data.extracted_intent?.category,
    inline_alloy_prediction: data.inline_alloy_prediction,
    report: data.report,
    tokens_used: data.tokens_used || 0,
  }
}

export async function predict(
  composition: Record<string, number>,
): Promise<PredictResponse> {
  const { data } = await api.post<PredictResponse>('/predict', { composition })
  return data
}

// ── Follow-up chat ────────────────────────────────────────────────────────────
export async function chatFollowup(
  payload: FollowUpChatRequest,
): Promise<FollowUpChatResponse> {
  const res = await fetchWithAuth(resolveBackendUrl('/chat/followup'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const fallback = await res.text().catch(() => '')
    throw new Error(fallback || `follow-up endpoint failed: HTTP ${res.status}`)
  }
  return res.json() as Promise<FollowUpChatResponse>
}

// ── SSE streaming search ──────────────────────────────────────────────────────
function readSSEEvents(streamText: string): Array<{ event: string; data: string }> {
  const events: Array<{ event: string; data: string }> = []
  const blocks = streamText.split('\n\n')
  for (const block of blocks) {
    const lines = block.split('\n')
    let event = 'message'
    const dataLines: string[] = []
    for (const line of lines) {
      if (line.startsWith('event:')) {
        event = line.slice(6).trim()
        continue
      }
      if (line.startsWith('data:')) {
        dataLines.push(line.slice(5).trim())
      }
    }
    if (dataLines.length > 0) {
      events.push({ event, data: dataLines.join('\n') })
    }
  }
  return events
}

function parseStructuredEventData(data: string): StructuredRecommendation | undefined {
  const trimmed = data.trim()
  if (!trimmed) return undefined
  try {
    return JSON.parse(trimmed) as StructuredRecommendation
  } catch {
    try {
      const maybeString = JSON.parse(trimmed) as string
      return JSON.parse(maybeString) as StructuredRecommendation
    } catch {
      return undefined
    }
  }
}

export interface SearchStructuredOptions {
  /** Called with each SSE text chunk as it arrives (enables live streaming UI). */
  onChunk?: (chunk: string) => void
  /** AbortSignal to cancel the request mid-stream. */
  signal?: AbortSignal
}

export async function searchStructured(
  query: string,
  options: SearchStructuredOptions = {},
): Promise<StructuredSearchResponse> {
  const { onChunk, signal } = options
  const searchURL = resolveBackendUrl('/search')

  const res = await fetchWithAuth(searchURL, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ query }),
    signal,
  })

  if (!res.ok || !res.body) {
    throw new Error(`search endpoint failed: HTTP ${res.status}`)
  }

  const decoder = new TextDecoder()
  const reader = res.body.getReader()
  let buffer = ''
  let finalReport = ''
  let structured: StructuredRecommendation | undefined

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lastSep = buffer.lastIndexOf('\n\n')
      if (lastSep < 0) continue

      const chunk = buffer.slice(0, lastSep)
      buffer = buffer.slice(lastSep + 2)

      const events = readSSEEvents(chunk)
      for (const evt of events) {
        if (evt.event === 'error') {
          throw new Error(evt.data || 'search endpoint returned error')
        }
        if (evt.event === 'structured_result') {
          const parsed = parseStructuredEventData(evt.data)
          if (parsed) structured = parsed
        }
        if (evt.event === 'message') {
          finalReport += evt.data
          onChunk?.(evt.data)
        }
      }
    }
  } finally {
    reader.releaseLock()
  }

  if (!finalReport && structured?.report) {
    finalReport = structured.report
  }

  return { structured_result: structured, report: finalReport }
}

// ── Health check ──────────────────────────────────────────────────────────────
export async function pingStatus(): Promise<boolean> {
  try {
    const baseURL = api.defaults.baseURL || 'http://localhost:8080/api'
    const root = baseURL.endsWith('/api')
      ? baseURL.slice(0, -4)
      : baseURL.replace(/\/api\/v1$/, '')
    const res = await fetchWithAuth(`${root}/health`, { credentials: 'include' })
    return res.ok
  } catch {
    return false
  }
}
