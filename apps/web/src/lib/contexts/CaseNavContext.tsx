"use client";

import { createContext, useCallback, useContext, useState, type ReactNode } from "react";

interface CaseNavContextValue {
  action: ReactNode;
  setAction: (node: ReactNode) => void;
}

const CaseNavContext = createContext<CaseNavContextValue>({
  action: null,
  setAction: () => {},
});

export function CaseNavProvider({ children }: { children: ReactNode }) {
  const [action, setActionState] = useState<ReactNode>(null);

  const setAction = useCallback((node: ReactNode) => {
    setActionState(node);
  }, []);

  return (
    <CaseNavContext.Provider value={{ action, setAction }}>
      {children}
    </CaseNavContext.Provider>
  );
}

export const useCaseNav = () => useContext(CaseNavContext);
