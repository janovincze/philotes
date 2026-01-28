import { Bell } from "lucide-react"
import { Card, CardContent } from "@/components/ui/card"

export default function AlertsPage() {
  return (
    <div className="space-y-6">
      {/* Page header */}
      <div>
        <h1 className="text-3xl font-bold">Alerts</h1>
        <p className="text-muted-foreground">
          Configure and manage alert rules and notifications
        </p>
      </div>

      {/* Placeholder */}
      <Card>
        <CardContent className="py-12 text-center">
          <Bell className="mx-auto h-12 w-12 text-muted-foreground" />
          <h3 className="mt-4 text-lg font-medium">Alerts coming soon</h3>
          <p className="mt-2 text-muted-foreground">
            Alert configuration and notification management will be available in a future release.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
