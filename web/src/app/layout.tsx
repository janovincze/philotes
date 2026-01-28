import type { Metadata } from "next"
import { Inter } from "next/font/google"
import "./globals.css"

import { ThemeProvider } from "@/components/theme-provider"
import { Providers } from "@/lib/providers"
import { Sidebar } from "@/components/layout/sidebar"
import { Header } from "@/components/layout/header"
import { MainContent } from "@/components/layout/main-content"
import { Toaster } from "@/components/ui/sonner"

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
})

export const metadata: Metadata = {
  title: "Philotes - CDC Data Platform",
  description: "Change Data Capture platform for PostgreSQL to Apache Iceberg",
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${inter.variable} font-sans antialiased`}>
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          <Providers>
            <div className="relative min-h-screen">
              <Sidebar />
              <Header />
              <MainContent>{children}</MainContent>
            </div>
            <Toaster />
          </Providers>
        </ThemeProvider>
      </body>
    </html>
  )
}
