import React, { useState } from "react"
import { History, Copy, Check, ExternalLink, Trash2 } from "lucide-react"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { ShortenResponse } from "@/lib/api"
import { toast } from "sonner"

export interface StoredLink extends ShortenResponse {
  timestamp: number
}

interface RecentLinksProps {
  links: StoredLink[]
  onClear: () => void
  onRemove: (code: string) => void
}

export const RecentLinks: React.FC<RecentLinksProps> = ({ links, onClear, onRemove }) => {
  const [copiedCode, setCopiedCode] = useState<string | null>(null)

  const copyUrl = (url: string, code: string) => {
    navigator.clipboard.writeText(url)
    setCopiedCode(code)
    toast.success("Copied to clipboard!", { description: url })
    setTimeout(() => setCopiedCode(null), 2000)
  }

  if (links.length === 0) {
    return null
  }

  return (
    <Card className="border-border/60 bg-card/60 backdrop-blur-md shadow-lg">
      <CardHeader className="flex flex-row items-center justify-between pb-4">
        <div>
          <div className="flex items-center gap-2 text-primary font-semibold text-sm">
            <History className="w-4 h-4" />
            <span>Session History</span>
          </div>
          <CardTitle className="text-lg">Recent Short Links</CardTitle>
          <CardDescription>Links generated during your current browser session</CardDescription>
        </div>

        <Button
          variant="ghost"
          size="sm"
          onClick={onClear}
          className="text-xs text-muted-foreground hover:text-destructive gap-1.5"
        >
          <Trash2 className="w-3.5 h-3.5" />
          Clear All
        </Button>
      </CardHeader>

      <CardContent>
        <div className="divide-y divide-border/40">
          {links.map((link) => (
            <div
              key={link.code}
              className="py-3 flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 first:pt-0 last:pb-0"
            >
              <div className="space-y-1 min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-mono font-medium text-sm text-foreground">
                    {link.short_url}
                  </span>
                  <Badge variant="outline" className="text-[10px] py-0 px-1.5">
                    {link.code}
                  </Badge>
                </div>
                <div className="text-xs text-muted-foreground truncate max-w-md">
                  {link.original_url}
                </div>
              </div>

              <div className="flex items-center gap-1.5 self-end sm:self-auto shrink-0">
                <Button
                  size="sm"
                  variant="secondary"
                  className="h-8 px-2.5 text-xs gap-1"
                  onClick={() => copyUrl(link.short_url, link.code)}
                >
                  {copiedCode === link.code ? (
                    <Check className="w-3.5 h-3.5 text-emerald-400" />
                  ) : (
                    <Copy className="w-3.5 h-3.5" />
                  )}
                  <span>{copiedCode === link.code ? "Copied" : "Copy"}</span>
                </Button>

                <a
                  href={link.short_url}
                  target="_blank"
                  rel="noreferrer"
                  className={buttonVariants({ variant: "outline", size: "icon", className: "h-8 w-8" })}
                >
                  <ExternalLink className="w-3.5 h-3.5" />
                </a>

                <Button
                  size="icon"
                  variant="ghost"
                  className="h-8 w-8 text-muted-foreground hover:text-destructive"
                  onClick={() => onRemove(link.code)}
                  title="Remove from history"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </Button>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

