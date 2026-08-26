/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{html,js}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        // Paleta estilo Twenty (dark nativo)
        base: "#0f1115",
        panel: "#16181d",
        panel2: "#1b1e24",
        border: "#262a31",
        border2: "#2f343d",
        ink: "#e6e8eb",
        muted: "#9aa3af",
        faint: "#6b7280",
        accent: "#6366f1",
        accent2: "#818cf8",
      },
      fontFamily: {
        sans: ["Inter", "ui-sans-serif", "system-ui", "sans-serif"],
      },
      borderRadius: {
        xl2: "14px",
      },
    },
  },
  plugins: [],
};
