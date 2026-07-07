import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

export default defineConfig({
  site: "https://polygolem.trebuchetdynamics.com",
  integrations: [
    starlight({
      title: "Polygolem",
      description: "Safe Polymarket SDK and CLI for Go — V2 deposit wallet (POLY_1271)",
      logo: {
        src: "./src/assets/logo.svg",
      },
      social: {
        github: "https://github.com/TrebuchetDynamics/polygolem",
      },
      sidebar: [
        {
          label: "Start here",
          items: [
            { label: "Introduction", link: "/docs/" },
            { label: "Docs Map", link: "/docs/getting-started/docs-map" },
            { label: "Installation", link: "/docs/getting-started/installation" },
            { label: "Quick Start", link: "/docs/getting-started/quickstart" },
          ],
        },
        {
          label: "Read-only workflows",
          items: [
            { label: "Market Discovery", link: "/docs/guides/market-discovery" },
            { label: "Orderbook Data", link: "/docs/guides/orderbook-data" },
            { label: "Wallet Intelligence", link: "/docs/guides/wallet-intelligence" },
            { label: "Paper Trading", link: "/docs/guides/paper-trading" },
          ],
        },
        {
          label: "Trade safely",
          items: [
            { label: "Safety Model", link: "/docs/concepts/safety" },
            { label: "Deposit Wallet Lifecycle", link: "/docs/guides/deposit-wallet-lifecycle" },
            { label: "Builder & Relayer Keys", link: "/docs/guides/builder-auto" },
            { label: "Headless Enable Trading", link: "/docs/guides/enable-trading-headless" },
            { label: "Bridge & Funding", link: "/docs/guides/bridge-funding" },
            { label: "Redeem Winners", link: "/docs/guides/redeem-winners" },
          ],
        },
        {
          label: "Developers",
          items: [
            { label: "Universal Client", link: "/docs/guides/universal-client" },
            { label: "Go-Bot Integration", link: "/docs/guides/go-bot-integration" },
            { label: "MCP and OpenAPI", link: "/docs/reference/mcp-openapi" },
            { label: "Go SDK Contracts", link: "/docs/reference/sdk" },
            { label: "CLI Commands", link: "/docs/reference/cli" },
          ],
        },
        {
          label: "Protocol concepts",
          items: [
            { label: "Polymarket API Overview", link: "/docs/concepts/polymarket-api" },
            { label: "Markets, Events & Tokens", link: "/docs/concepts/markets-events-tokens" },
            { label: "Deposit Wallets (POLY_1271)", link: "/docs/concepts/deposit-wallets" },
            { label: "POLY_1271 Signing Chain", link: "/docs/concepts/poly-1271-signing" },
            { label: "Smart Contracts", link: "/docs/concepts/contracts" },
            { label: "Secrets Management", link: "/docs/concepts/secrets-management" },
            { label: "Architecture", link: "/docs/concepts/architecture" },
          ],
        },
        {
          label: "API reference",
          items: [
            { label: "Gamma API", link: "/docs/reference/gamma-api" },
            { label: "CLOB V2 API", link: "/docs/reference/clob-api" },
            { label: "Data API", link: "/docs/reference/data-api" },
            { label: "Bridge API", link: "/docs/reference/bridge-api" },
            { label: "Relayer API", link: "/docs/reference/relayer-api" },
            { label: "Stream API", link: "/docs/reference/stream-api" },
            { label: "Protocol Types", link: "/docs/reference/polytypes" },
            { label: "Internal Packages", link: "/docs/reference/internal-packages" },
            { label: "Contracts Registry", link: "/docs/concepts/contracts" },
            { label: "Coverage Matrix", link: "/docs/reference/coverage-matrix" },
          ],
        },
      ],
      customCss: ["./src/styles/custom.css"],
    }),
  ],
});
