/** @type {import('tailwindcss').Config} */

module.exports = {
  darkMode: ["class", '[data-theme="dark"]'],
  content: ["./src/**/*.{js,jsx,ts,tsx,svg}", "./docusaurus.config.js"],
  theme: {
    extend: {
      animation: {
        scroll: "scroll 30s linear infinite",
      },
      keyframes: {
        scroll: {
          "0%": { transform: "translateX(0)" },
          "100%": { transform: "translateX(-50%)" },
        },
      },
      colors: {
        fontSize: {
          "5xl": "2.5rem",
        },
        brand: {
          900: "#1a237e", // Material Indigo 900
          850: "#202b85",
          800: "#283593", // Material Indigo 800
          700: "#303f9f", // Material Indigo 700
          650: "#3949ab", // Material Indigo 600
          600: "#3949ab",
          500: "#3f51b5", // Material Indigo 500 (Primary)
          400: "#5c6bc0", // Material Indigo 400
          300: "#7986cb", // Material Indigo 300
          200: "#9fa8da", // Material Indigo 200
          150: "#c5cae9",
          100: "#c5cae9", // Material Indigo 100
          50: "#e8eaf6",  // Material Indigo 50
        },
        gray: {
          950: "#121212", // Material Dark Background
          900: "#1B1D20",
          850: "#282B31",
          800: "#353A41",
          700: "#505661",
          600: "#6A7382",
          500: "#8590A2",
          400: "#9DA6B5",
          300: "#B6BCC7",
          200: "#CED3DA",
          150: "#DADEE3",
          100: "#E7E9EC",
          50: "#F9F9F9",
        },
        blue: {
          950: "#0c192b",
          900: "#14253D",
        },
      },
      screens: {
        xs: { max: "576px" },
      },
    },
    fontFamily: {
      sans: ['"DM Sans"', "system-ui"],
    },
  },
  plugins: [require("@tailwindcss/typography")],
};
