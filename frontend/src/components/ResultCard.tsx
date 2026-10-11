import React, { useState, useRef } from "react"
import { Copy, Check, ExternalLink, QrCode, Download, Share2, BarChart3, Clock } from "lucide-react"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { QRCodeSVG } from "qrcode.react"
import { ShortenResponse } from "@/lib/api"
import { toast } from "sonner"

interface ResultCardProps {
  result: ShortenResponse
  onViewAnalytics?: (code: string) => void
}

export const ResultCard: React.FC<ResultCardProps> = ({ result, onViewAnalytics }) => {
  const [copied, setCopied] = useState(false)
  const [showQR, setShowQR] = useState(false)
  const qrRef = useRef<HTMLDivElement>(null)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(result.short_url)
      setCopied(true)
      toast.success("Copied to clipboard!", {
        description: result.short_url,
      })
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error("Failed to copy URL")
    }
  }

  const downloadQR = () => {
    if (!qrRef.current) return
    const svg = qrRef.current.querySelector("svg")
    if (!svg) return

    const svgData = new XMLSerializer().serializeToString(svg)
    const canvas = document.createElement("canvas")
    const ctx = canvas.getContext("2d")
    const img = new Image()

    img.onload = () => {
      canvas.width = img.width + 40
      canvas.height = img.height + 40
      if (!ctx) return
      ctx.fillStyle = "#ffffff"
      ctx.fillRect(0, 0, canvas.width, canvas.height)
      ctx.drawImage(img, 20, 20)
      const pngFile = canvas.toDataURL("image/png")
      const downloadLink = document.createElement("a")
      downloadLink.download = `qr-${result.code}.png`
      downloadLink.href = pngFile
      downloadLink.click()
      toast.success("QR Code downloaded!")
    }

    img.src = "data:image/svg+xml;base64," + btoa(svgData)
  }

  return (
    <Card className="border-primary/40 bg-card/90 shadow-2xl relative overflow-hidden animate-in fade-in slide-in-from-bottom-4 duration-300">
      <div className="h-1.5 w-full bg-gradient-to-r from-primary via-indigo-500 to-sky-400" />
      <CardContent className="p-6">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div className="space-y-1.5 flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <Badge variant="outline" className="border-primary/40 text-primary bg-primary/10 font-mono text-xs">
                Code: /{result.code}
              </Badge>
              {result.expires_at && (
                <Badge variant="secondary" className="text-[11px] gap-1 font-normal text-muted-foreground">
                  <Clock className="w-3 h-3 text-indigo-400" />
                  Expires: {new Date(result.expires_at).toLocaleDateString()}
                </Badge>
              )}
              <span className="text-xs text-muted-foreground flex items-center gap-1">
                <Share2 className="w-3 h-3" /> Ready
              </span>
            </div>

            <div className="text-xl sm:text-2xl font-bold font-mono text-foreground tracking-tight break-all">
              {result.short_url}
            </div>

            <div className="text-xs text-muted-foreground truncate max-w-lg">
              Destination: <span className="text-foreground/80 hover:underline">{result.original_url}</span>
            </div>
          </div>

          <div className="flex items-center gap-2 w-full sm:w-auto shrink-0 flex-wrap">
            <Button
              onClick={handleCopy}
              variant={copied ? "default" : "secondary"}
              className="flex-1 sm:flex-none gap-2"
            >
              {copied ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
              <span>{copied ? "Copied" : "Copy"}</span>
            </Button>

            {onViewAnalytics && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => onViewAnalytics(result.code)}
                className="gap-1.5 h-10"
                title="View Click Analytics"
              >
                <BarChart3 className="w-4 h-4 text-primary" />
                <span className="hidden sm:inline">Analytics</span>
              </Button>
            )}

            <a
              href={result.short_url}
              target="_blank"
              rel="noreferrer"
              className={buttonVariants({ variant: "outline", className: "gap-1.5 h-10" })}
            >
              <ExternalLink className="w-4 h-4" />
              <span className="hidden sm:inline">Visit</span>
            </a>

            <Button
              variant="outline"
              size="icon"
              onClick={() => setShowQR(!showQR)}
              title="Show QR Code"
              className={`h-10 w-10 ${showQR ? "bg-accent text-accent-foreground" : ""}`}
            >
              <QrCode className="w-4 h-4" />
            </Button>
          </div>
        </div>

        {showQR && (
          <div className="mt-6 pt-6 border-t border-border/50 flex flex-col sm:flex-row items-center justify-center gap-6 animate-in fade-in duration-200">
            <div
              ref={qrRef}
              className="p-3 bg-white rounded-xl shadow-md border border-zinc-200"
            >
              <QRCodeSVG
                value={result.short_url}
                size={140}
                level="M"
                includeMargin={false}
              />
            </div>

            <div className="space-y-3 text-center sm:text-left">
              <div>
                <h4 className="font-semibold text-sm">Scan with mobile device</h4>
                <p className="text-xs text-muted-foreground max-w-xs">
                  Directs instantly to the destination URL with Redis-accelerated redirection.
                </p>
              </div>

              <Button
                variant="outline"
                size="sm"
                onClick={downloadQR}
                className="gap-2 text-xs"
              >
                <Download className="w-3.5 h-3.5" />
                Download PNG
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

