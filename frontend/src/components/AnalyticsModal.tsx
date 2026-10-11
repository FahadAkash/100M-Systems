import React, { useEffect, useState } from "react"
import {
  X,
  BarChart3,
  RefreshCw,
  ExternalLink,
  Smartphone,
  Monitor,
  Tablet,
  Globe,
  Clock,
  Activity,
  Layers,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { getUrlAnalytics, URLAnalytics } from "@/lib/api"
import { toast } from "sonner"

interface AnalyticsModalProps {
  code: string | null
  onClose: () => void
}

export const AnalyticsModal: React.FC<AnalyticsModalProps> = ({ code, onClose }) => {
  const [data, setData] = useState<URLAnalytics | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchAnalytics = async (shortCode: string) => {
    setLoading(true)
    setError(null)
    try {
      const res = await getUrlAnalytics(shortCode)
      setData(res)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to load analytics"
      setError(msg)
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (code) {
      fetchAnalytics(code)
    } else {
      setData(null)
      setError(null)
    }
  }, [code])

  if (!code) return null

  const renderBreakdown = (
    title: string,
    items?: Record<string, number>,
    icon?: React.ReactNode
  ) => {
    if (!items || Object.keys(items).length === 0) {
      return (
        <div className="p-4 rounded-xl border border-border/40 bg-card/40 text-xs text-muted-foreground text-center">
          No data recorded yet
        </div>
      )
    }

    const total = Object.values(items).reduce((acc, v) => acc + v, 0)
    const sorted = Object.entries(items).sort((a, b) => b[1] - a[1])

    return (
      <div className="p-4 rounded-xl border border-border/40 bg-card/60 space-y-3">
        <div className="flex items-center justify-between text-xs font-semibold text-foreground">
          <div className="flex items-center gap-1.5">
            {icon}
            <span>{title}</span>
          </div>
          <span className="text-muted-foreground font-mono">{total} clicks</span>
        </div>

        <div className="space-y-2">
          {sorted.map(([key, count]) => {
            const pct = total > 0 ? Math.round((count / total) * 100) : 0
            return (
              <div key={key} className="space-y-1 text-xs">
                <div className="flex justify-between items-center text-muted-foreground">
                  <span className="truncate max-w-[140px] text-foreground font-medium">{key}</span>
                  <span className="font-mono text-[11px]">{count} ({pct}%)</span>
                </div>
                <div className="h-1.5 w-full bg-muted/60 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-primary rounded-full transition-all duration-500"
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>
            )
          })}
        </div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-md animate-in fade-in duration-200">
      <div className="relative w-full max-w-3xl max-h-[90vh] flex flex-col rounded-2xl border border-border/80 bg-card shadow-2xl overflow-hidden">
        {/* Modal Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border/60 bg-muted/20">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-lg bg-primary/10 text-primary">
              <BarChart3 className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-lg font-bold tracking-tight">Streaming Click Analytics</h3>
              <p className="text-xs text-muted-foreground font-mono">Code: /{code}</p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchAnalytics(code)}
              disabled={loading}
              className="gap-1.5 text-xs h-8"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`} />
              <span className="hidden sm:inline">Refresh</span>
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={onClose}
              className="h-8 w-8 text-muted-foreground hover:text-foreground"
            >
              <X className="w-4 h-4" />
            </Button>
          </div>
        </div>

        {/* Modal Body */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {loading && !data && (
            <div className="py-16 text-center space-y-3">
              <RefreshCw className="w-8 h-8 animate-spin mx-auto text-primary" />
              <p className="text-sm text-muted-foreground">Loading real-time analytics...</p>
            </div>
          )}

          {error && !data && (
            <div className="p-6 text-center space-y-2 border border-destructive/20 bg-destructive/10 rounded-xl text-destructive">
              <p className="font-semibold text-sm">Failed to load analytics</p>
              <p className="text-xs">{error}</p>
            </div>
          )}

          {data && (
            <>
              {/* Summary Metrics Banner */}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <div className="p-4 rounded-xl border border-primary/30 bg-primary/5 space-y-1">
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Activity className="w-3.5 h-3.5 text-primary" />
                    <span>Total Recorded Clicks</span>
                  </div>
                  <div className="text-3xl font-extrabold text-primary font-mono">
                    {data.total_clicks}
                  </div>
                </div>

                <div className="p-4 rounded-xl border border-border/40 bg-card/40 space-y-1 sm:col-span-2">
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Clock className="w-3.5 h-3.5 text-indigo-400" />
                    <span>Last Redirect Event</span>
                  </div>
                  <div className="text-sm font-semibold text-foreground truncate">
                    {data.last_clicked
                      ? new Date(data.last_clicked).toLocaleString()
                      : "No clicks recorded yet"}
                  </div>
                  <div className="text-xs text-muted-foreground truncate">
                    Destination: {data.original_url}
                  </div>
                </div>
              </div>

              {/* Dimensional Breakdown Grid */}
              <div className="space-y-3">
                <h4 className="text-sm font-semibold flex items-center gap-1.5">
                  <Layers className="w-4 h-4 text-primary" />
                  <span>Traffic Demographics & Breakdowns</span>
                </h4>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  {renderBreakdown(
                    "Devices",
                    data.devices,
                    <Smartphone className="w-3.5 h-3.5 text-sky-400" />
                  )}
                  {renderBreakdown(
                    "Operating Systems",
                    data.operating_systems,
                    <Monitor className="w-3.5 h-3.5 text-indigo-400" />
                  )}
                  {renderBreakdown(
                    "Web Browsers",
                    data.browsers,
                    <Globe className="w-3.5 h-3.5 text-emerald-400" />
                  )}
                  {renderBreakdown(
                    "Top Referrers",
                    data.top_referrers,
                    <ExternalLink className="w-3.5 h-3.5 text-amber-400" />
                  )}
                </div>
              </div>

              {/* Recent Click Logs Table */}
              <div className="space-y-3 pt-2">
                <h4 className="text-sm font-semibold flex items-center gap-1.5">
                  <Clock className="w-4 h-4 text-primary" />
                  <span>Recent Click Stream (Last 20 Events)</span>
                </h4>

                {data.recent_clicks && data.recent_clicks.length > 0 ? (
                  <div className="border border-border/40 rounded-xl overflow-hidden bg-card/40">
                    <div className="max-h-60 overflow-y-auto">
                      <table className="w-full text-left text-xs">
                        <thead className="bg-muted/40 sticky top-0 border-b border-border/40 text-muted-foreground">
                          <tr>
                            <th className="py-2.5 px-3 font-semibold">IP Address</th>
                            <th className="py-2.5 px-3 font-semibold">Referrer</th>
                            <th className="py-2.5 px-3 font-semibold">Time</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-border/20">
                          {data.recent_clicks.map((c, idx) => (
                            <tr key={idx} className="hover:bg-muted/20 transition-colors">
                              <td className="py-2 px-3 font-mono text-[11px] text-foreground">
                                {c.ip || "Direct"}
                              </td>
                              <td className="py-2 px-3 text-muted-foreground truncate max-w-[150px]">
                                {c.referer || "Direct"}
                              </td>
                              <td className="py-2 px-3 text-muted-foreground whitespace-nowrap text-[11px]">
                                {new Date(c.timestamp).toLocaleTimeString()}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                ) : (
                  <div className="p-6 rounded-xl border border-border/40 bg-card/40 text-center text-xs text-muted-foreground">
                    No recent click records in database stream.
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

