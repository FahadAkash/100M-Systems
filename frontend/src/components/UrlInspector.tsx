import React, { useState } from "react"
import { Search, Loader2, Calendar, ExternalLink, Copy, Check, FileSearch, AlertCircle } from "lucide-react"
import { Button, buttonVariants } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { getUrlMetadata, URLMetadata } from "@/lib/api"
import { toast } from "sonner"

export const UrlInspector: React.FC = () => {
  const [code, setCode] = useState("")
  const [loading, setLoading] = useState(false)
  const [metadata, setMetadata] = useState<URLMetadata | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const handleInspect = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setMetadata(null)

    const cleanCode = code.trim().replace(/^.*\/([a-zA-Z0-9_-]+)$/, "$1")
    if (!cleanCode) {
      setError("Please enter a short code or full short URL")
      return
    }

    setLoading(true)
    try {
      const data = await getUrlMetadata(cleanCode)
      setMetadata(data)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Code not found"
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  const copyOriginal = () => {
    if (!metadata) return
    navigator.clipboard.writeText(metadata.original_url)
    setCopied(true)
    toast.success("Destination copied!")
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Card className="border-border/80 bg-card/80 backdrop-blur-xl shadow-xl">
      <CardHeader>
        <div className="flex items-center gap-2 text-primary font-semibold text-sm">
          <FileSearch className="w-4 h-4" />
          <span>Code Inspector</span>
        </div>
        <CardTitle className="text-xl">Inspect Short URL Metadata</CardTitle>
        <CardDescription>
          Check the destination, creation date, and status of any short code without triggering a redirect.
        </CardDescription>
      </CardHeader>

      <CardContent className="space-y-4">
        <form onSubmit={handleInspect} className="flex gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-3.5 h-4 w-4 text-muted-foreground" />
            <Input
              type="text"
              placeholder="Paste code or short URL (e.g., 0PTsAyag)"
              value={code}
              onChange={(e) => {
                setCode(e.target.value)
                if (error) setError(null)
              }}
              className="pl-9 h-11"
              disabled={loading}
            />
          </div>

          <Button type="submit" disabled={loading || !code.trim()} className="gap-2 px-5">
            {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : "Lookup"}
          </Button>
        </form>

        {error && (
          <div className="flex items-center gap-2 p-3 text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-lg animate-in fade-in">
            <AlertCircle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {metadata && (
          <div className="mt-4 p-4 rounded-xl border border-border bg-background/60 space-y-3 animate-in fade-in slide-in-from-top-2">
            <div className="flex items-center justify-between">
              <Badge variant="outline" className="font-mono text-xs bg-primary/10 text-primary border-primary/30">
                Code: {metadata.code}
              </Badge>
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <Calendar className="w-3.5 h-3.5" />
                <span>{new Date(metadata.created_at).toLocaleString()}</span>
              </div>
            </div>

            <div>
              <div className="text-xs text-muted-foreground mb-1">Target Destination:</div>
              <div className="text-sm font-medium text-foreground break-all bg-card/60 p-2.5 rounded-lg border border-border/50">
                {metadata.original_url}
              </div>
            </div>

            <div className="flex items-center gap-2 pt-1">
              <Button size="sm" variant="secondary" onClick={copyOriginal} className="gap-1.5 text-xs">
                {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                <span>{copied ? "Copied" : "Copy Target"}</span>
              </Button>

              <a
                href={metadata.original_url}
                target="_blank"
                rel="noreferrer"
                className={buttonVariants({ variant: "outline", size: "sm", className: "gap-1.5 text-xs" })}
              >
                <ExternalLink className="w-3.5 h-3.5" />
                <span>Visit Target</span>
              </a>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

