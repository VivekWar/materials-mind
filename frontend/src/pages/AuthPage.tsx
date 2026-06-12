import React from 'react'
import { Layers } from 'lucide-react'
import { Button } from '../components/ui/button'

const AuthPage: React.FC = () => {
  const loginUrl = `${import.meta.env.VITE_API_URL || 'http://localhost:8080/api'}/auth/google/login`

  return (
    <div className="flex min-h-screen bg-background">
      {/* ── Left Pane: Visual/Brand ───────────────────────────────────── */}
      <div className="hidden lg:flex flex-1 relative bg-zinc-950 dark:bg-zinc-950 overflow-hidden items-center justify-center border-r border-border/50">
        <div className="absolute inset-0 bg-[url('https://grainy-gradients.vercel.app/noise.svg')] opacity-20 mix-blend-overlay"></div>
        <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-white/10 to-transparent"></div>
        
        {/* Abstract structural visualization */}
        <div className="relative z-10 p-12 max-w-xl text-zinc-300">
          <div className="w-12 h-12 rounded-lg bg-white/10 border border-white/20 flex items-center justify-center shadow-2xl mb-8">
            <Layers size={24} className="text-white" />
          </div>
          <h1 className="text-4xl font-semibold tracking-tight text-white mb-6">
            Engineered Precision.
          </h1>
          <p className="text-lg text-zinc-400 leading-relaxed font-light">
            Materials Mind is a technical intelligence console. We translate complex engineering constraints into deterministic, physics-grounded material recommendations.
          </p>
          
          <div className="mt-12 space-y-4 font-mono text-xs text-zinc-500">
            <div className="flex items-center gap-4">
              <span className="w-16">SYS</span>
              <span className="text-zinc-300">Vector matching active</span>
            </div>
            <div className="flex items-center gap-4">
              <span className="w-16">DB</span>
              <span className="text-zinc-300">Extensive material database</span>
            </div>
            <div className="flex items-center gap-4">
              <span className="w-16">LATENCY</span>
              <span className="text-green-400">Sub-second retrieval latency</span>
            </div>
          </div>
        </div>

        {/* Decorative structural lines */}
        <div className="absolute right-0 top-1/4 bottom-1/4 w-px bg-gradient-to-b from-transparent via-white/10 to-transparent"></div>
        <div className="absolute left-1/4 right-1/4 bottom-0 h-px bg-gradient-to-r from-transparent via-white/10 to-transparent"></div>
      </div>

      {/* ── Right Pane: Minimal Login ──────────────────────────────────── */}
      <div className="flex-1 flex flex-col justify-center px-6 sm:px-12 lg:px-24 xl:px-32 relative bg-background">
        <div className="mx-auto w-full max-w-[360px] space-y-8">
          <div className="flex items-center gap-3 lg:hidden mb-12">
            <div className="w-8 h-8 rounded-md bg-primary flex items-center justify-center">
              <Layers size={16} className="text-primary-foreground" />
            </div>
            <span className="font-semibold tracking-tight">Materials Mind</span>
          </div>

          <div>
            <h2 className="text-2xl font-semibold tracking-tight text-foreground">Sign in</h2>
            <p className="text-sm text-muted-foreground mt-2">
              Continue with your Google workspace account to access the intelligence console.
            </p>
          </div>

          <div className="pt-2">
            <Button
              id="btn-google-signin"
              variant="outline"
              onClick={() => window.open(loginUrl, "Google Login", "width=500,height=600")}
              className="w-full h-10 text-sm shadow-sm transition-all duration-200 gap-2.5 font-medium border-border/80 hover:bg-muted/50"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
                <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
                <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
                <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
              </svg>
              Continue with Google
            </Button>
          </div>

          <div className="text-[11px] text-muted-foreground text-center">
            By continuing, you agree to our Terms of Service and Privacy Policy. Secure access is restricted to authorized personnel.
          </div>
        </div>
      </div>
    </div>
  )
}

export default AuthPage
