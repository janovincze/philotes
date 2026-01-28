import { Settings } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      {/* Page header */}
      <div>
        <h1 className="text-3xl font-bold">Settings</h1>
        <p className="text-muted-foreground">
          Configure system settings and preferences
        </p>
      </div>

      {/* Placeholder */}
      <Card>
        <CardContent className="py-12 text-center">
          <Settings className="mx-auto h-12 w-12 text-muted-foreground" />
          <h3 className="mt-4 text-lg font-medium">Settings coming soon</h3>
          <p className="mt-2 text-muted-foreground">
            System configuration options will be available in a future release.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
