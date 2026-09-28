import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Restrained, institutional palette rather than a startup/fintech look.
        // Refined further once real UI work begins.
        brand: {
          50: "#f0f5f4",
          100: "#dbe7e4",
          600: "#2f6f63",
          700: "#255a51",
          900: "#173832",
        },
      },
    },
  },
  plugins: [],
};

export default config;
