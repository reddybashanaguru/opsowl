import { create } from "zustand";

interface CopilotStore {
  open: boolean;
  setOpen: (open: boolean) => void;
}

export const useCopilotStore = create<CopilotStore>((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
}));
