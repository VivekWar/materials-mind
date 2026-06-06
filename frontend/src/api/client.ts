import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1',
  headers: { 'Content-Type': 'application/json' },
  timeout: 180000,
})

function resolveBackendUrl(path: string): string {
  const baseURL = api.defaults.baseURL || ''
  const apiRoot = baseURL.endsWith('/api/v1') ? `${baseURL.slice(0, -7)}/api` : `${baseURL}/api`
  return `${apiRoot}${path}`
}

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

export interface StructuredRecommendation {
  recommended_material: string
  why_it_matches: string[]
  trade_offs: string[]
  confidence: 'High' | 'Medium' | 'Low'
  confidence_score: number
  sources: number[]
  report: string
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

export async function recommend(query: string, domain: string, constraints?: Constraint[]): Promise<RecommendResponse> {
  const payload: any = { query, domain }
  if (constraints && constraints.length > 0) {
    payload.constraints = constraints.map(c => ({
      key: c.key,
      operator: c.operator,
      value: c.value,
    }))
  }
  const { data } = await api.post<RecommendResponse>('/recommend', payload)

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

export async function chatFollowup(payload: FollowUpChatRequest): Promise<FollowUpChatResponse> {
  const res = await fetch(resolveBackendUrl('/chat/followup'), {
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
  if (!trimmed) {
    return undefined
  }

  try {
    return JSON.parse(trimmed) as StructuredRecommendation
  } catch {
    // Gin SSE may serialize payload as quoted JSON string.
    try {
      const maybeString = JSON.parse(trimmed) as string
      return JSON.parse(maybeString) as StructuredRecommendation
    } catch {
      return undefined
    }
  }
}

export async function searchStructured(query: string): Promise<StructuredSearchResponse> {
  const searchURL = resolveBackendUrl('/search')

  const res = await fetch(searchURL, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ query }),
  })

  if (!res.ok || !res.body) {
    throw new Error(`search endpoint failed: HTTP ${res.status}`)
  }

  const decoder = new TextDecoder()
  const reader = res.body.getReader()
  let buffer = ''
  let finalReport = ''
  let structured: StructuredRecommendation | undefined

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lastSep = buffer.lastIndexOf('\n\n')
    if (lastSep < 0) {
      continue
    }

    const chunk = buffer.slice(0, lastSep)
    buffer = buffer.slice(lastSep + 2)

    const events = readSSEEvents(chunk)
    for (const evt of events) {
      if (evt.event === 'error') {
        throw new Error(evt.data || 'search endpoint returned error')
      }
      if (evt.event === 'structured_result') {
        const parsed = parseStructuredEventData(evt.data)
        if (parsed) {
          structured = parsed
        }
      }
      if (evt.event === 'message') {
        finalReport += evt.data
      }
    }
  }

  if (!finalReport && structured?.report) {
    finalReport = structured.report
  }

  return {
    structured_result: structured,
    report: finalReport,
  }
}

export async function ping(): Promise<void> {
  await pingStatus()
}

export async function pingStatus(): Promise<boolean> {
  try {
    await api.options('/recommend', {
      validateStatus: (status) => status >= 200 && status < 500,
    })
    return true
  } catch {
    return false
  }
}
