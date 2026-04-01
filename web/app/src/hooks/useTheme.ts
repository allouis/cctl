import { useState, useEffect, useCallback } from "react";
import type { Theme } from "../types";

function getStored(): Theme {
  const v = localStorage.getItem("theme");
  if (v === "light" || v === "dark" || v === "system") return v;
  return "system";
}

function applyTheme(choice: Theme) {
  const prefersDark = window.matchMedia(
    "(prefers-color-scheme: dark)",
  ).matches;
  const isDark = choice === "dark" || (choice === "system" && prefersDark);
  document.documentElement.classList.toggle("dark", isDark);
  const meta = document.querySelector('meta[name="theme-color"]');
  if (meta) meta.setAttribute("content", isDark ? "#030712" : "#f9fafb");
}

export function useTheme() {
  const [theme, setThemeState] = useState<Theme>(getStored);

  const setTheme = useCallback((t: Theme) => {
    localStorage.setItem("theme", t);
    setThemeState(t);
    applyTheme(t);
  }, []);

  useEffect(() => {
    applyTheme(theme);

    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = () => {
      if (getStored() === "system") applyTheme("system");
    };
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [theme]);

  return { theme, setTheme };
}
