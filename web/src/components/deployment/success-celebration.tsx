"use client"

import { useEffect, useState } from "react"
import { CheckCircle, PartyPopper, Rocket } from "lucide-react"
import { cn } from "@/lib/utils"

interface SuccessCelebrationProps {
  show: boolean
  onComplete?: () => void
  className?: string
}

export function SuccessCelebration({ show, onComplete, className }: SuccessCelebrationProps) {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    if (show) {
      setVisible(true)

      // Dynamically import and trigger confetti
      import("canvas-confetti")
        .then((confettiModule) => {
          const confetti = confettiModule.default

          // Fire confetti from both sides
          const duration = 3000
          const end = Date.now() + duration

          const colors = ["#22c55e", "#3b82f6", "#8b5cf6", "#f59e0b"]

          const frame = () => {
            confetti({
              particleCount: 3,
              angle: 60,
              spread: 55,
              origin: { x: 0, y: 0.6 },
              colors,
            })
            confetti({
              particleCount: 3,
              angle: 120,
              spread: 55,
              origin: { x: 1, y: 0.6 },
              colors,
            })

            if (Date.now() < end) {
              requestAnimationFrame(frame)
            }
          }

          frame()

          // Also fire a big burst
          confetti({
            particleCount: 100,
            spread: 70,
            origin: { y: 0.6 },
            colors,
          })
        })
        .catch(() => {
          // Confetti not available, continue without it
          console.log("Confetti animation not available")
        })

      // Hide after animation
      const timer = setTimeout(() => {
        setVisible(false)
        onComplete?.()
      }, 5000)

      return () => clearTimeout(timer)
    }
  }, [show, onComplete])

  if (!show && !visible) return null

  return (
    <div
      className={cn(
        "fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm",
        "animate-in fade-in duration-300",
        !visible && "animate-out fade-out duration-500",
        className
      )}
    >
      <div className="flex flex-col items-center gap-4 text-center p-8">
        <div className="relative">
          <div className="absolute -inset-4 rounded-full bg-green-500/20 animate-pulse" />
          <div className="relative flex items-center justify-center w-24 h-24 rounded-full bg-green-500">
            <CheckCircle className="h-12 w-12 text-white" />
          </div>
        </div>

        <div className="space-y-2">
          <h2 className="text-3xl font-bold text-green-500 flex items-center gap-2">
            <PartyPopper className="h-8 w-8" />
            Deployment Complete!
            <Rocket className="h-8 w-8" />
          </h2>
          <p className="text-muted-foreground max-w-md">
            Your Philotes instance is now running and ready to use.
            You can access the dashboard and start creating pipelines.
          </p>
        </div>
      </div>
    </div>
  )
}
