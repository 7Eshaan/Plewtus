import type { Metadata, Viewport } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import { ThemeProvider } from "@/components/theme-provider";
import "./globals.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
  display: "swap",
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains-mono",
  subsets: ["latin"],
  display: "swap",
});

const description =
  "Federated metasearch for AI agents: DuckDuckGo, Bing, and Yahoo in one API, with deduplication, consensus ranking, page extraction, and MCP support.";

export const metadata: Metadata = {
  title: "Plewtus | Federated Search for AI Agents",
  description,
  applicationName: "Plewtus",
  keywords: ["metasearch", "search API", "AI agents", "MCP", "Model Context Protocol", "RAG"],
  openGraph: {
    title: "Plewtus | Federated Search for AI Agents",
    description,
    type: "website",
    siteName: "Plewtus",
  },
  twitter: {
    card: "summary",
    title: "Plewtus | Federated Search for AI Agents",
    description,
  },
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#ffffff" },
    { media: "(prefers-color-scheme: dark)", color: "#09090b" },
  ],
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning className={`${inter.variable} ${jetbrainsMono.variable}`}>
      <body className="min-h-screen bg-[var(--background)] text-[var(--foreground)] antialiased flex flex-col">
        <a
          href="#demo"
          className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-50 focus:px-3 focus:py-2 focus:rounded-md focus:bg-brand focus:text-white focus:text-sm"
        >
          Skip to search
        </a>
        <ThemeProvider attribute="class" defaultTheme="dark" enableSystem={false} disableTransitionOnChange>
          {children}
        </ThemeProvider>
      </body>
    </html>
  );
}
