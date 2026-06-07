import React, { useState, useEffect } from 'react'
import {
  ArrowRight,
  Database,
  Layers,
  Search,
  ShieldCheck,
  Zap,
  Activity,
  Server,
  Cpu,
  Lock
} from 'lucide-react'
import { Button } from './ui/button'

interface HomePageProps {
  onStartChat: () => void
}

export const HomePage: React.FC<HomePageProps> = ({ onStartChat }) => {
  const [activeStep, setActiveStep] = useState(0)
  const loginUrl = `${import.meta.env.VITE_API_URL || 'http://localhost:8080/api'}/auth/google/login`

  // Rotate through architecture steps automatically
  useEffect(() => {
    const timer = setInterval(() => {
      setActiveStep((prev) => (prev + 1) % 4)
    }, 3500)
    return () => clearInterval(timer)
  }, [])

  return (
    <div className="min-h-screen bg-background text-foreground font-sans selection:bg-zinc-200 dark:selection:bg-zinc-800">
      {/* ── Nav ─────────────────────────────────────────────────────────── */}
      <header className="fixed top-0 inset-x-0 z-50 border-b border-border/50 bg-background/80 backdrop-blur-md">
        <div className="max-w-[1400px] mx-auto px-6 h-14 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-7 h-7 rounded bg-foreground flex items-center justify-center">
              <Layers size={14} className="text-background" />
            </div>
            <div className="text-sm font-medium tracking-tight">Materials Mind</div>
          </div>
          <div className="flex items-center gap-4">
            <a href="#architecture" className="text-xs font-mono uppercase tracking-widest text-muted-foreground hover:text-foreground transition-colors hidden sm:block">
              Architecture
            </a>
            <a href={loginUrl} style={{ textDecoration: 'none' }}>
              <Button
                id="btn-nav-login"
                size="sm"
                variant="outline"
                className="h-8 text-xs font-medium rounded-md shadow-sm border-border/80"
              >
                Sign In
              </Button>
            </a>
            <Button
              id="btn-nav-start"
              size="sm"
              onClick={onStartChat}
              className="h-8 text-xs font-medium rounded-md"
            >
              Console <ArrowRight size={12} className="ml-1.5 opacity-70" />
            </Button>
          </div>
        </div>
      </header>

      <main>
        {/* ── Hero ────────────────────────────────────────────────────────── */}
        <section className="relative pt-32 pb-24 sm:pt-40 sm:pb-32 px-6 overflow-hidden">
          <div className="absolute inset-0 bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-[size:24px_24px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]"></div>
          
          <div className="relative max-w-[1400px] mx-auto grid lg:grid-cols-[1fr_500px] gap-16 items-center">
            
            <div className="max-w-2xl">
              <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md bg-muted text-[10px] font-mono font-medium text-muted-foreground uppercase tracking-widest mb-6 border border-border">
                <span className="w-1.5 h-1.5 rounded-full bg-foreground animate-pulse" />
                System v2.4 Online
              </div>

              <h1 className="text-5xl sm:text-6xl lg:text-[72px] font-semibold tracking-tighter leading-[1.05] mb-6 text-foreground">
                Engineered <br />
                <span className="text-muted-foreground">material selection.</span>
              </h1>
              
              <p className="text-lg sm:text-xl text-muted-foreground font-light leading-relaxed mb-10 max-w-xl">
                A technical intelligence console for hardware engineering. Translate complex physical constraints into deterministic material recommendations backed by structured data.
              </p>

              <div className="flex flex-col sm:flex-row gap-4">
                <Button
                  id="btn-hero-start"
                  onClick={onStartChat}
                  className="h-12 px-6 text-sm font-medium"
                >
                  Enter Console <ArrowRight size={14} className="ml-2" />
                </Button>
                <Button
                  id="btn-hero-docs"
                  variant="outline"
                  className="h-12 px-6 text-sm font-medium border-border/80 bg-transparent hover:bg-muted/50"
                  onClick={() => document.getElementById('architecture')?.scrollIntoView({ behavior: 'smooth' })}
                >
                  View Architecture
                </Button>
              </div>
            </div>

            {/* Right: Technical Preview Terminal */}
            <div className="hidden lg:flex justify-end perspective-1000">
              <div className="w-[500px] rounded-xl border border-border bg-card shadow-2xl shadow-black/5 overflow-hidden transform rotate-y-[-5deg] rotate-x-[2deg]">
                <div className="flex items-center gap-2 px-4 py-2.5 bg-muted/50 border-b border-border">
                  <div className="flex gap-1.5">
                    <div className="w-2.5 h-2.5 rounded-full bg-border"></div>
                    <div className="w-2.5 h-2.5 rounded-full bg-border"></div>
                    <div className="w-2.5 h-2.5 rounded-full bg-border"></div>
                  </div>
                  <div className="mx-auto text-[10px] font-mono text-muted-foreground">inference_pipeline.log</div>
                </div>
                <div className="p-5 font-mono text-xs leading-relaxed">
                  <div className="text-muted-foreground mb-4">
                    <span className="text-foreground">~$</span> input --query "Lightweight polymer for FDM bracket, &gt;130C service temp, vibration resistant"
                  </div>
                  
                  <div className="text-zinc-500 mb-2">» extracting_constraints... [OK]</div>
                  <div className="pl-4 border-l border-border mb-4 space-y-1">
                    <div><span className="text-muted-foreground">category:</span> "Polymer"</div>
                    <div><span className="text-muted-foreground">hdt_min:</span> 130 °C</div>
                    <div><span className="text-muted-foreground">process:</span> "FDM"</div>
                    <div><span className="text-muted-foreground">fatigue:</span> "high"</div>
                  </div>

                  <div className="text-zinc-500 mb-2">» vector_search... [14ms]</div>
                  <div className="text-zinc-500 mb-4">» ranking_candidates... [OK]</div>

                  <div className="text-foreground mb-1">
                    <span className="text-green-500">✓</span> MATCH: PEEK (Polyether ether ketone)
                  </div>
                  <div className="pl-6 text-muted-foreground">Confidence: 91.4%</div>
                  <div className="pl-6 text-muted-foreground">Density: 1.32 g/cm³</div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* ── Divider ────────────────────────────────────────────────────── */}
        <div className="w-full h-px bg-gradient-to-r from-transparent via-border to-transparent"></div>

        {/* ── Interactive Architecture Pipeline ───────────────────────────── */}
        <section id="architecture" className="py-24 sm:py-32 px-6 bg-zinc-50 dark:bg-zinc-900/20">
          <div className="max-w-[1200px] mx-auto">
            <div className="mb-16 text-center max-w-2xl mx-auto">
              <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-md bg-muted text-[10px] font-mono font-medium text-muted-foreground uppercase tracking-widest mb-6 border border-border">
                <Activity size={12} className="text-primary" /> System Topology
              </div>
              <h2 className="text-3xl lg:text-4xl font-semibold tracking-tight text-foreground mb-4">
                Zero Hallucination Architecture.
              </h2>
              <p className="text-muted-foreground text-lg font-light leading-relaxed">
                Queries are processed through a deterministic pipeline. Constraints are mathematically verified against structured properties before the LLM generates a response.
              </p>
            </div>

            {/* Interactive Pipeline Diagram */}
            <div className="relative mt-12 bg-background border border-border rounded-xl shadow-sm p-8 overflow-hidden">
              {/* Background wiring logic */}
              <div className="absolute inset-0 opacity-10 pointer-events-none">
                <svg className="w-full h-full" xmlns="http://www.w3.org/2000/svg">
                  <path d="M150 100 L300 100 M450 100 L600 100 M750 100 L900 100" stroke="currentColor" strokeWidth="2" strokeDasharray="4 4" />
                </svg>
              </div>

              <div className="grid md:grid-cols-4 gap-4 relative z-10">
                {[
                  {
                    id: 0,
                    icon: Search,
                    title: "Constraint Parsing",
                    sys: "NER / NLP",
                    desc: "Natural language is parsed into hard physical boundaries (e.g. Tensile Strength > 400MPa)."
                  },
                  {
                    id: 1,
                    icon: Database,
                    title: "Vector Retrieval",
                    sys: "pgvector",
                    desc: "High-dimensional embeddings are used to query a catalog of 12,000+ proprietary material sheets."
                  },
                  {
                    id: 2,
                    icon: ShieldCheck,
                    title: "Physics Engine",
                    sys: "Validation",
                    desc: "Candidates failing the extracted physical limits are systematically pruned from the context window."
                  },
                  {
                    id: 3,
                    icon: Cpu,
                    title: "Structured Synth",
                    sys: "LLM Orchestration",
                    desc: "The final surviving candidates are streamed to the client with precise trade-off JSON payloads."
                  }
                ].map((step) => (
                  <button
                    key={step.id}
                    onClick={() => setActiveStep(step.id)}
                    className={`relative p-5 rounded-lg border text-left transition-all duration-300 ${
                      activeStep === step.id 
                        ? 'border-foreground bg-muted/30 shadow-[0_4px_20px_rgba(0,0,0,0.05)]' 
                        : 'border-border bg-background hover:border-foreground/30 hover:bg-muted/10'
                    }`}
                  >
                    {activeStep === step.id && (
                      <div className="absolute -top-px left-4 right-4 h-[2px] bg-foreground shadow-[0_0_10px_currentColor]"></div>
                    )}
                    <div className={`w-8 h-8 rounded-md flex items-center justify-center mb-4 transition-colors ${
                      activeStep === step.id ? 'bg-foreground text-background' : 'bg-muted text-muted-foreground'
                    }`}>
                      <step.icon size={16} />
                    </div>
                    <div className="text-[10px] font-mono uppercase tracking-widest text-muted-foreground mb-1">{step.sys}</div>
                    <div className="text-sm font-semibold text-foreground mb-2">{step.title}</div>
                    <div className={`text-xs leading-relaxed transition-opacity ${
                      activeStep === step.id ? 'text-muted-foreground opacity-100' : 'text-muted-foreground opacity-50'
                    }`}>
                      {step.desc}
                    </div>
                  </button>
                ))}
              </div>

              {/* Data Flow Indicator */}
              <div className="mt-8 pt-6 border-t border-border flex items-center justify-between text-[10px] font-mono text-muted-foreground">
                <div className="flex items-center gap-2">
                  <Server size={12} />
                  <span>US-East-1 Edge</span>
                </div>
                <div className="flex items-center gap-2">
                  <Lock size={12} />
                  <span>AES-256 Encrypted Stream</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="w-1.5 h-1.5 bg-green-500 rounded-full animate-pulse" />
                  <span>Telemetry Active</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* ── Bottom CTA ─────────────────────────────────────────────────── */}
        <section className="py-24 px-6 border-t border-border bg-background">
          <div className="max-w-2xl mx-auto text-center">
            <div className="w-12 h-12 rounded bg-foreground flex items-center justify-center mx-auto mb-6 shadow-md shadow-black/10">
              <Zap size={20} className="text-background" />
            </div>
            <h2 className="text-3xl font-semibold tracking-tight text-foreground mb-4">
              Initialize workspace.
            </h2>
            <p className="text-muted-foreground mb-8 font-light">
              Access is currently restricted to authorized Google workspace accounts. Sign in to start your first analysis.
            </p>
            <div className="flex items-center justify-center gap-4">
              <a href={loginUrl} style={{ textDecoration: 'none' }}>
                <Button id="btn-cta-login" variant="outline" className="h-11 px-8 border-border/80 bg-background">
                  Sign In with Google
                </Button>
              </a>
              <Button id="btn-cta-start" onClick={onStartChat} className="h-11 px-8">
                Console Access
              </Button>
            </div>
          </div>
        </section>
      </main>

      {/* ── Footer ──────────────────────────────────────────────────────── */}
      <footer className="border-t border-border py-8 px-6 bg-background">
        <div className="max-w-[1400px] mx-auto flex flex-col sm:flex-row items-center justify-between gap-4 text-xs font-mono text-muted-foreground">
          <div className="flex items-center gap-2">
            <Layers size={12} />
            <span>Materials Mind © 2026</span>
          </div>
          <div className="flex gap-4">
            <span>Status: Operational</span>
            <span>v2.4.0</span>
          </div>
        </div>
      </footer>
    </div>
  )
}
