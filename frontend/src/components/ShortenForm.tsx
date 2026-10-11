import React, { useState } from "react"
import { Link, ArrowRight, Loader2, Sparkles, AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { shortenUrl, ShortenResponse } from "@/lib/api"
import { toast } from "sonner"

interface ShortenFormProps {
  onSuccess: (result: ShortenResponse) => void
}

export const ShortenForm: React.FC<ShortenFormProps> = ({ onSuccess }) => {
  const [url, setUrl] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    const trimmed = url.trim()
    if (!trimmed) {
      setError("Please enter a URL")
      return
    }

    let parsedUrl = trimmed
    if (!/^https?:\/\//i.test(parsedUrl)) {
      parsedUrl = `https://${parsedUrl}`
    }

    try {
      new URL(parsedUrl)
    } catch {
      setError("Please enter a valid URL (e.g., https://example.com/page)")
      return
    }

    setLoading(true)
    try {
      const res = await shortenUrl(parsedUrl)
      toast.success("Short URL created successfully!", {
        description: `Code: ${res.code}`,
      })
      onSuccess(res)
      setUrl("")
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to shorten URL"
      setError(msg)
      toast.error("Could not shorten URL", {
        description: msg,
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card className="relative overflow-hidden border-border/80 shadow-2xl bg-card/80 backdrop-blur-xl">
      <div className="absolute top-0 right-0 -mr-16 -mt-16 w-48 h-48 bg-primary/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-0 left-0 -ml-16 -mb-16 w-48 h-48 bg-indigo-500/10 rounded-full blur-3xl pointer-events-none" />

      <CardHeader className="text-center pb-4">
        <div className="inline-flex items-center justify-center gap-1.5 self-center px-3 py-1 rounded-full bg-primary/10 border border-primary/20 text-xs font-medium text-primary mb-2">
          <Sparkles className="w-3.5 h-3.5" />
          <span>PostgreSQL + Redis Cached Engine</span>
        </div>
        <CardTitle className="text-2xl sm:text-3xl font-extrabold tracking-tight">
          Shorten Any Link in Seconds
        </CardTitle>
        <CardDescription className="text-sm sm:text-base max-w-md mx-auto">
          Transform long, messy URLs into clean, fast, high-performance short links with sub-millisecond Redis redirects.
        </CardDescription>
      </CardHeader>

      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="flex flex-col sm:flex-row gap-3">
            <div className="relative flex-1">
              <div className="absolute inset-y-0 left-0 flex items-center pl-3.5 pointer-events-none text-muted-foreground">
                <Link className="h-4 w-4" />
              </div>
              <Input
                type="text"
                placeholder="Enter long URL (e.g., https://github.com/FahadAkash/100M-Systems)"
                value={url}
                onChange={(e) => {
                  setUrl(e.target.value)
                  if (error) setError(null)
                }}
                disabled={loading}
                className="pl-10 h-12 bg-background/80 border-border/70 text-base"
              />
            </div>

            <Button
              type="submit"
              size="lg"
              disabled={loading || !url.trim()}
              className="h-12 px-6 font-semibold gap-2 shadow-lg shadow-primary/25 active:scale-95 transition-transform"
            >
              {loading ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span>Shortening...</span>
                </>
              ) : (
                <>
                  <span>Shorten URL</span>
                  <ArrowRight className="h-4 w-4" />
                </>
              )}
            </Button>
          </div>

          {error && (
            <div className="flex items-center gap-2 p-3 text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-lg">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}
        </form>
      </CardContent>
    </Card>
  )
}

