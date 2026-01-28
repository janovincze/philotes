import Link from "next/link"
import { FileQuestion } from "lucide-react"
import { Button } from "@/components/ui/button"

export default function NotFound() {
  return (
    <div className="flex min-h-[400px] flex-col items-center justify-center gap-4 p-4">
      <FileQuestion className="h-16 w-16 text-muted-foreground" />
      <div className="text-center">
        <h2 className="text-2xl font-bold">Page Not Found</h2>
        <p className="text-muted-foreground">
          The page you&apos;re looking for doesn&apos;t exist.
        </p>
      </div>
      <Button asChild>
        <Link href="/">Go to Dashboard</Link>
      </Button>
    </div>
  )
}
