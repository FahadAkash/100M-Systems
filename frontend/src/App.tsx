import { useState, useEffect } from "react"
import { Navbar } from "@/components/Navbar"
import { ShortenForm } from "@/components/ShortenForm"
import { ResultCard } from "@/components/ResultCard"
import { RecentLinks, StoredLink } from "@/components/RecentLinks"
import { UrlInspector } from "@/components/UrlInspector"
import { AnalyticsModal } from "@/components/AnalyticsModal"
import { Button } from "@/components/ui/button"
import { ShortenResponse } from "@/lib/api"
import { Zap, Search, Database, ShieldCheck } from "lucide-react"

const STORAGE_KEY = "100m_recent_urls"

export function App() {
  const [activeTab, setActiveTab] = useState<"shorten" | "inspect">("shorten")
  const [currentResult, setCurrentResult] = useState<ShortenResponse | null>(null)
  const [recentLinks, setRecentLinks] = useState<StoredLink[]>([])
  const [analyticsCode, setAnalyticsCode] = useState<string | null>(null)

  useEffect(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored) {
        setRecentLinks(JSON.parse(stored))
      }
    } catch {
      // Ignore local storage parse error
    }
  }, [])

  const handleShortenSuccess = (result: ShortenResponse) => {
    setCurrentResult(result)
    const newEntry: StoredLink = {
      ...result,
      timestamp: Date.now(),
    }
    const updated = [newEntry, ...recentLinks.filter((l) => l.code !== result.code)].slice(0, 15)
    setRecentLinks(updated)
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(updated))
    } catch {
      // Ignore storage set errors
    }
  }

  const handleClearHistory = () => {
    setRecentLinks([])
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch {
      // Ignore
    }
  }

  const handleRemoveLink = (code: string) => {
    const updated = recentLinks.filter((l) => l.code !== code)
    setRecentLinks(updated)
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(updated))
    } catch {
      // Ignore
    }
  }

  return (
    <div className="min-h-screen flex flex-col bg-background selection:bg-primary/20 selection:text-primary">
      <Navbar />

      <main className="flex-1 container max-w-4xl px-4 py-8 sm:py-12 space-y-8">
        {/* Tab Buttons */}
        <div className="flex justify-center">
          <div className="inline-flex p-1 rounded-xl bg-muted/60 border border-border/80 backdrop-blur-md">
            <Button
              variant={activeTab === "shorten" ? "default" : "ghost"}
              size="sm"
              onClick={() => setActiveTab("shorten")}
              className="gap-2 rounded-lg text-xs sm:text-sm font-medium px-4"
            >
              <Zap className="w-4 h-4" />
              <span>Shorten URL</span>
            </Button>

            <Button
              variant={activeTab === "inspect" ? "default" : "ghost"}
              size="sm"
              onClick={() => setActiveTab("inspect")}
              className="gap-2 rounded-lg text-xs sm:text-sm font-medium px-4"
            >
              <Search className="w-4 h-4" />
              <span>Inspect Code</span>
            </Button>
          </div>
        </div>

        {/* Tab Content */}
        {activeTab === "shorten" ? (
          <div className="space-y-6">
            <ShortenForm onSuccess={handleShortenSuccess} />

            {currentResult && (
              <ResultCard
                result={currentResult}
                onViewAnalytics={(code) => setAnalyticsCode(code)}
              />
            )}

            <RecentLinks
              links={recentLinks}
              onClear={handleClearHistory}
              onRemove={handleRemoveLink}
              onViewAnalytics={(code) => setAnalyticsCode(code)}
            />
          </div>
        ) : (
          <div className="space-y-6 max-w-2xl mx-auto">
            <UrlInspector />
          </div>
        )}

        {/* Feature Highlights Grid */}
        <section className="pt-8 border-t border-border/40 grid grid-cols-1 sm:grid-cols-3 gap-4 text-center sm:text-left">
          <div className="p-4 rounded-xl border border-border/40 bg-card/40 space-y-2">
            <div className="flex items-center gap-2 text-primary font-semibold text-sm">
              <Zap className="w-4 h-4" />
              <span>L1 Memory + L2 Redis</span>
            </div>
            <p className="text-xs text-muted-foreground">
              Hot viral links resolve in nanoseconds with zero network hops, protected by singleflight stampede defense.
            </p>
          </div>

          <div className="p-4 rounded-xl border border-border/40 bg-card/40 space-y-2">
            <div className="flex items-center gap-2 text-indigo-400 font-semibold text-sm">
              <Database className="w-4 h-4" />
              <span>Persistent Storage</span>
            </div>
            <p className="text-xs text-muted-foreground">
              Guaranteed durability backed by PostgreSQL with vanity alias support and optional TTL link expiration.
            </p>
          </div>

          <div className="p-4 rounded-xl border border-border/40 bg-card/40 space-y-2">
            <div className="flex items-center gap-2 text-sky-400 font-semibold text-sm">
              <ShieldCheck className="w-4 h-4" />
              <span>Streaming Analytics</span>
            </div>
            <p className="text-xs text-muted-foreground">
              Asynchronous streaming pipeline tracks click counts, devices, operating systems, and top referrers.
            </p>
          </div>
        </section>
      </main>

      {/* Analytics Modal Dialog */}
      <AnalyticsModal
        code={analyticsCode}
        onClose={() => setAnalyticsCode(null)}
      />

      {/* Footer */}
      <footer className="border-t border-border/40 py-6 text-center text-xs text-muted-foreground">
        <div className="container max-w-4xl px-4 flex flex-col sm:flex-row items-center justify-between gap-3">
          <div className="flex items-center gap-1.5">
            <span>Built for high scale with</span>
            <span className="font-semibold text-foreground">Go, PostgreSQL, Redis & React</span>
          </div>
          <div className="flex items-center gap-1 text-[11px]">
            <span>100M-Systems URL Shortener</span>
          </div>
        </div>
      </footer>
    </div>
  )
}

export default App
