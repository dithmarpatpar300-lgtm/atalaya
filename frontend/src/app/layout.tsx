import type { Metadata, Viewport } from "next";
import { GeistMono } from "geist/font/mono";
import { GeistSans } from "geist/font/sans";
import { GUION_ARRANQUE } from "@/lib/tema-arranque";
import "./globals.css";

export const metadata: Metadata = {
  title: "Atalaya",
  description: "Los agentes de IA de tus proyectos, en vivo, y la memoria del centro de operaciones.",
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#fafafa" },
    { media: "(prefers-color-scheme: dark)", color: "#111111" },
  ],
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    // `suppressHydrationWarning`: el guion de abajo pone `data-tema` antes de que React hidrate.
    <html lang="es" className={`${GeistSans.variable} ${GeistMono.variable} h-full antialiased`} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: GUION_ARRANQUE }} />
      </head>
      <body className="min-h-full">{children}</body>
    </html>
  );
}
