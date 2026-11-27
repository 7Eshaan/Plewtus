import type { Metadata, Viewport } from "next";
import { Roboto_Flex, JetBrains_Mono } from "next/font/google";
import { ThemeProvider } from "@/components/theme-provider";
import { SnackbarProvider } from "@/components/ui/snackbar";
import "./globals.css";

// Material 3's variable typeface. The optical-size axis lets large display
// text and small labels each use the cut drawn for their size.
const robotoFlex = Roboto_Flex({
  variable: "--font-roboto-flex",
  subsets: ["latin"],
  axes: ["opsz"],
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
  // M3 surface colors (see m3-theme.css).
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#fff7ff" },
    { media: "(prefers-color-scheme: dark)", color: "#000000" },
  ],
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning className={`${robotoFlex.variable} ${jetbrainsMono.variable}`}>
      <body className="flex min-h-screen flex-col bg-surface text-on-surface antialiased">
        <a
          href="#demo"
          className="sr-only focus:not-sr-only focus:fixed focus:left-2 focus:top-2 focus:z-50 focus:rounded-corner-full focus:bg-primary-container focus:px-4 focus:py-2 focus:text-label-lg focus:text-on-primary-container"
        >
          Skip to search
        </a>
        <ThemeProvider attribute="class" defaultTheme="dark" enableSystem={false} disableTransitionOnChange>
          <SnackbarProvider>{children}</SnackbarProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
