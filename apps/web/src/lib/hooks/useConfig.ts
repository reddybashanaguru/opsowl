"use client";

import { useEffect } from "react";
import { useConfigStore, type AppConfig } from "@/lib/stores/config.store";

export function useConfig(): AppConfig {
  const { config, loaded, fetch } = useConfigStore();

  useEffect(() => {
    if (!loaded) {
      fetch();
    }
  }, [loaded, fetch]);

  return config;
}
